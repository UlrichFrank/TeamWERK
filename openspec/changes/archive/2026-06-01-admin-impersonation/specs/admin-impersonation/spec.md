## ADDED Requirements

### Requirement: Admin kann User impersonieren
Ein eingeloggter Admin SHALL in der Nutzerverwaltung einen beliebigen Standard-User auswählen und dessen Session-Sicht übernehmen können. Der Impersonation-Endpoint gibt ein kurzlebiges JWT mit den Claims des Ziel-Users zurück. Das Impersonieren eines anderen Admins ist nicht erlaubt.

#### Scenario: Impersonation starten
- **WHEN** Admin klickt "Testen als" bei einem Standard-User in der Nutzerverwaltung
- **THEN** sendet das Frontend `POST /api/admin/impersonate/{userId}`
- **THEN** gibt das Backend ein gültiges JWT mit role, club_functions und is_parent des Ziel-Users zurück
- **THEN** aktualisiert das Frontend den AuthContext mit dem neuen Token und dem impersonating-State

#### Scenario: Impersonation eines Admins wird abgelehnt
- **WHEN** Admin sendet `POST /api/admin/impersonate/{userId}` für einen User mit role=admin
- **THEN** antwortet das Backend mit HTTP 400

#### Scenario: Selbst-Impersonation wird abgelehnt
- **WHEN** Admin sendet `POST /api/admin/impersonate/{userId}` mit der eigenen userId
- **THEN** antwortet das Backend mit HTTP 400

### Requirement: Impersonation-Banner ist sichtbar
Während einer aktiven Impersonation SHALL im Hauptbereich der App ein deutlich sichtbarer Banner angezeigt werden, der Name und Rolle des impersonierten Users anzeigt.

#### Scenario: Banner erscheint bei aktiver Impersonation
- **WHEN** `impersonating`-State im AuthContext ist gesetzt
- **THEN** zeigt AppShell einen gelben Banner mit Name des Ziel-Users und einem "Beenden"-Button

#### Scenario: Banner ist für nicht-Admins unsichtbar
- **WHEN** ein Standard-User eingeloggt ist
- **THEN** ist kein Impersonation-Banner sichtbar

### Requirement: Admin kehrt zur eigenen Session zurück
Ein Admin SHALL die Impersonation jederzeit beenden können, ohne sich neu einloggen zu müssen. Die Rückkehr erfolgt über einen normalen Token-Refresh, da der Refresh-Cookie des Admins während der Impersonation unverändert bleibt.

#### Scenario: Impersonation beenden
- **WHEN** Admin klickt "Beenden" im Impersonation-Banner
- **THEN** ruft das Frontend `POST /api/auth/refresh` auf (Admin-Cookie unveränderter)
- **THEN** aktualisiert das Frontend den AuthContext mit dem Admin-Token
- **THEN** wird `impersonating` auf null gesetzt und der Banner verschwindet

### Requirement: UI verhält sich entsprechend der impersonierten Rolle
Während aktiver Impersonation SHALL die gesamte UI — Navigation, sichtbare Seiten, API-Antworten — exakt dem impersonierten User entsprechen.

#### Scenario: Trainer-Sicht zeigt korrekte Navigation
- **WHEN** Admin impersoniert einen User mit club_function=trainer
- **THEN** zeigt die Sidebar die gleichen Navigationspunkte wie für einen echten Trainer
- **THEN** sind Admin-only-Bereiche nicht sichtbar

#### Scenario: API-Calls laufen mit Ziel-JWT
- **WHEN** das Frontend während Impersonation einen API-Call absetzt
- **THEN** enthält der Authorization-Header das JWT des Ziel-Users (nicht das Admin-JWT)
- **THEN** antwortet das Backend entsprechend der Berechtigungen des Ziel-Users

### Requirement: "Testen als"-Button in der Nutzerverwaltung
In der Nutzerverwaltung SHALL pro User-Zeile ein "Testen als"-Button sichtbar sein, sofern der eingeloggte User Admin ist und es sich nicht um seinen eigenen Account handelt.

#### Scenario: Button erscheint für fremde Standard-User
- **WHEN** Admin öffnet die Nutzerverwaltung
- **THEN** hat jede User-Zeile (außer der eigenen) einen "Testen als"-Button

#### Scenario: Button erscheint auf Desktop und Mobile
- **WHEN** Admin öffnet die Nutzerverwaltung auf einem mobilen Gerät
- **THEN** ist der "Testen als"-Button im Action-Menu der Mobile-Card verfügbar
