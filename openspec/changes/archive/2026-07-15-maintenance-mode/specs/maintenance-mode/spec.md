## ADDED Requirements

### Requirement: Persistenter Wartungsmodus-Zustand

Das System SHALL einen booleschen Wartungsmodus-Zustand über Server-Neustarts hinweg persistieren. Der Zustand wird in einer generischen `system_settings`-Tabelle als Key-Value-Paar `key='maintenance_mode', value='on'|'off'` gespeichert. Beim Erst-Migrationslauf MUSS die Default-Row idempotent mit `value='off'` angelegt werden, sodass bestehende Instanzen ohne manuellen Eingriff im deaktivierten Zustand starten.

#### Scenario: Migration angelegt Default-Row idempotent

- **WHEN** die Migration zweimal hintereinander auf derselben DB ausgeführt wird
- **THEN** existiert genau eine Row mit `key='maintenance_mode'` und `value='off'`, und die zweite Ausführung endet ohne Fehler

#### Scenario: Zustand überlebt Server-Neustart

- **WHEN** der Admin den Modus auf `on` schaltet und der Serverprozess neu gestartet wird
- **THEN** ist der Modus nach dem Neustart weiterhin `on`

### Requirement: Blockade von Mutations bei aktivem Modus

Bei aktivem Wartungsmodus SHALL das System jede Request-Methode aus `{POST, PUT, PATCH, DELETE}` mit HTTP-Status `503 Service Unavailable` beantworten, ausgenommen:
- Requests an Pfade unter dem Prefix `/api/auth/` (Login, Refresh, Logout);
- Requests, deren JWT-Claims die System-Rolle `admin` tragen.

Der 503-Response MUSS den Header `X-Maintenance-Mode: 1` sowie einen JSON-Body `{"error":"maintenance_mode","message":"…"}` enthalten. Requests mit Methoden aus `{GET, HEAD, OPTIONS}` werden vom Wartungsmodus nicht beeinflusst.

#### Scenario: Normale Mutation eines Nicht-Admin wird blockiert

- **WHEN** der Modus `on` ist und ein Nutzer mit Rolle `standard` (z. B. Vorstand) `POST /api/games` aufruft
- **THEN** liefert die Response Status 503, Header `X-Maintenance-Mode: 1` und Body enthält `"error":"maintenance_mode"`

#### Scenario: Admin darf trotz aktivem Modus mutieren

- **WHEN** der Modus `on` ist und ein Nutzer mit Rolle `admin` `POST /api/games` mit gültigem Payload aufruft
- **THEN** wird der Request an den regulären Handler weitergegeben und liefert eine reguläre Antwort (z. B. 201 bei Erfolg), nicht 503

#### Scenario: Auth-Routen bleiben erreichbar

- **WHEN** der Modus `on` ist und ein unauthentifizierter Nutzer `POST /api/auth/login` aufruft
- **THEN** liefert der reguläre Login-Handler seine übliche Antwort (200 bei korrekten Credentials, 400/401 sonst), nicht 503

#### Scenario: GET-Requests unbeeinflusst

- **WHEN** der Modus `on` ist und ein authentifizierter Nutzer `GET /api/games` aufruft
- **THEN** liefert die Response Status 200 mit dem regulären Payload und **ohne** `X-Maintenance-Mode`-Header

#### Scenario: OPTIONS-Preflight unbeeinflusst

- **WHEN** der Modus `on` ist und ein Browser einen `OPTIONS`-Preflight-Request auf eine beliebige API-Route sendet
- **THEN** wird der Preflight regulär beantwortet (CORS-Headers gesetzt), nicht 503

#### Scenario: Inaktiver Modus verändert kein Response-Verhalten

- **WHEN** der Modus `off` ist
- **THEN** wird kein `X-Maintenance-Mode`-Header gesetzt, und keine Request-Antwort weicht von der ohne den Wartungsmodus zu erwartenden Antwort ab

### Requirement: Admin-Toggle-Endpoint

Das System SHALL einen HTTP-Endpoint `POST /api/admin/maintenance-mode` bereitstellen, der ausschließlich Nutzern der System-Rolle `admin` erlaubt, den Wartungsmodus umzuschalten. Der Request-Body MUSS `{"enabled": true|false}` enthalten. Bei erfolgreicher Umschaltung MUSS das System die `system_settings.value` aktualisieren, `updated_at` (aktueller Zeitstempel) und `updated_by` (User-ID des Admins) setzen und ein SSE-Event `settings-changed` an alle verbundenen Clients broadcasten.

#### Scenario: Admin schaltet Modus ein

- **WHEN** ein Admin `POST /api/admin/maintenance-mode` mit Body `{"enabled":true}` aufruft
- **THEN** ist die Response Status 200; die DB-Row `maintenance_mode` hat `value='on'`, `updated_by`=Admin-User-ID, und ein SSE-Event `settings-changed` wurde gebroadcastet

#### Scenario: Nicht-Admin kann nicht toggeln

- **WHEN** ein Nutzer mit Rolle `standard` (z. B. Vorstand oder Kassierer) `POST /api/admin/maintenance-mode` aufruft
- **THEN** ist die Response Status 403, und der Zustand bleibt unverändert

#### Scenario: Unauthentifizierter Aufruf abgewiesen

- **WHEN** ein Client ohne gültigen Auth-Header `POST /api/admin/maintenance-mode` aufruft
- **THEN** ist die Response Status 401, und der Zustand bleibt unverändert

### Requirement: Public Status-Endpoint

Das System SHALL einen HTTP-Endpoint `GET /api/maintenance-status` bereitstellen, der ohne Authentifizierung aufrufbar ist und ausschließlich das JSON `{"enabled": bool}` zurückgibt. Metadaten wie `updated_by` oder `updated_at` DÜRFEN NICHT über diesen Endpoint ausgeliefert werden.

#### Scenario: Unauthentifizierter Aufruf liefert Status

- **WHEN** ein Client ohne Auth-Header `GET /api/maintenance-status` aufruft
- **THEN** liefert die Response Status 200 und den Body `{"enabled": <bool>}` entsprechend dem aktuellen Zustand

#### Scenario: Kein Info-Leak über Metadaten

- **WHEN** die Response von `GET /api/maintenance-status` inspiziert wird
- **THEN** enthält der Body kein `updated_by`- und kein `updated_at`-Feld

### Requirement: CLI-Toggle als Ausfall-Sicherung

Das System SHALL das Umschalten des Wartungsmodus über einen CLI-Subcommand `teamwerk maintenance on|off` ermöglichen. Der Subcommand schreibt direkt in die DB (unter Verwendung des `--db`-Pfad-Flags analog zu bestehenden Subcommands wie `migrate`). Der Subcommand ist unabhängig von einem laufenden HTTP-Server und benötigt keine Authentifizierung (Zugriff wird über OS-Login/sudo auf dem VPS reguliert).

#### Scenario: CLI setzt Modus auf `on`

- **WHEN** `teamwerk maintenance on --db /var/lib/teamwerk/teamwerk.db` ausgeführt wird
- **THEN** hat die DB-Row `maintenance_mode` `value='on'`, und der Exit-Code ist 0

#### Scenario: CLI setzt Modus auf `off`

- **WHEN** `teamwerk maintenance off --db /var/lib/teamwerk/teamwerk.db` ausgeführt wird
- **THEN** hat die DB-Row `maintenance_mode` `value='off'`, und der Exit-Code ist 0

### Requirement: Frontend-Banner bei aktivem Modus

Das Frontend SHALL bei aktivem Wartungsmodus einen persistenten, nicht-schließbaren Banner am oberen Rand der Anwendungs-Shell (oberhalb des `TransitionalHostnameBanner`) anzeigen. Der Banner-Zustand wird über einen initialen Aufruf von `GET /api/maintenance-status` beim App-Start sowie über SSE-Events `settings-changed` (via `useLiveUpdates`) synchronisiert. Bei inaktivem Modus SHALL kein Banner sichtbar sein.

#### Scenario: Banner wird gerendert, wenn Modus aktiv ist

- **WHEN** `GET /api/maintenance-status` liefert `{"enabled": true}` und die App gerendert wird
- **THEN** ist der `MaintenanceBanner` im DOM vorhanden und sichtbar

#### Scenario: Banner verschwindet, wenn Modus deaktiviert wird

- **WHEN** der Modus während einer laufenden Session per SSE-Event `settings-changed` und anschließendem Refetch als `false` gemeldet wird
- **THEN** verschwindet der `MaintenanceBanner` aus dem DOM

#### Scenario: Banner rendert `null` bei inaktivem Modus initial

- **WHEN** die App startet und der Status als `{"enabled": false}` geliefert wird
- **THEN** ist der `MaintenanceBanner` nicht im DOM sichtbar

### Requirement: Frontend fängt 503-Maintenance-Response ab

Das Frontend SHALL einen Axios-Response-Interceptor implementieren, der Responses mit Status 503 **und** Header `X-Maintenance-Mode: 1` erkennt und dem Nutzer einen freundlichen Hinweis (Toast oder Modal) präsentiert („Wartungsmodus aktiv — bitte gleich noch einmal versuchen"). Reguläre 503-Responses (ohne den Header) SHALL vom Interceptor **nicht** verändert werden.

#### Scenario: Maintenance-503 zeigt freundlichen Hinweis

- **WHEN** eine Axios-Request eine Response mit Status 503 und Header `X-Maintenance-Mode: 1` erhält
- **THEN** wird die konfigurierte Nutzer-Hinweis-Funktion (z. B. Toast oder Dialog) aufgerufen und das Promise weiterhin rejected

#### Scenario: Regulärer 503 wird nicht abgefangen

- **WHEN** eine Axios-Request eine Response mit Status 503 **ohne** Header `X-Maintenance-Mode` erhält
- **THEN** wird die Nutzer-Hinweis-Funktion nicht aufgerufen; die Response propagiert unverändert

### Requirement: Admin-UI zur Toggle-Bedienung

Das Frontend SHALL eine Admin-only-Seite (Route unter `/admin/`) bereitstellen, die den aktuellen Zustand des Wartungsmodus anzeigt und einen Button zum Ein- bzw. Ausschalten bietet. Die Seite MUSS zusätzlich `updated_by` (Anzeige-Name des zuletzt umschaltenden Admins) und `updated_at` (Datum/Uhrzeit) anzeigen, sofern verfügbar. Die Seite ist über einen Nav-Eintrag erreichbar, der ausschließlich für Nutzer mit Rolle `admin` gerendert wird.

#### Scenario: Admin sieht aktuellen Zustand und kann toggeln

- **WHEN** ein Admin die Wartungsmodus-Seite öffnet
- **THEN** ist der aktuelle Zustand (`Ein` oder `Aus`) und ein Toggle-Button sichtbar

#### Scenario: Toggle-Button ruft POST-Endpoint auf

- **WHEN** der Admin auf den Toggle-Button klickt und der Modus vorher `off` war
- **THEN** wird `POST /api/admin/maintenance-mode` mit Body `{"enabled":true}` gesendet, und nach erfolgreicher Antwort zeigt die Seite den neuen Zustand `Ein`

#### Scenario: Nicht-Admin sieht keinen Nav-Eintrag

- **WHEN** ein Nutzer mit Rolle `standard` die Anwendungs-Shell rendert
- **THEN** ist der Nav-Eintrag zur Wartungsmodus-Seite nicht im DOM enthalten
