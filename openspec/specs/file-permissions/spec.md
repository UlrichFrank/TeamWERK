# file-permissions Specification

## Purpose
TBD - created by archiving change dateiablage. Update Purpose after archive.
## Requirements
### Requirement: Dynamische Ordnerverwaltung
Authentifizierte Nutzer mit `can_write` auf einen Ordner SHALL darin Unterordner via `POST /api/folders` anlegen können. Ein Ordner MUST einen `name` und eine `parent_id` (oder null für Wurzel) haben.

#### Scenario: Unterordner anlegen
- **WHEN** ein Nutzer mit `can_write` auf Ordner A `POST /api/folders` mit `parent_id = A` aufruft
- **THEN** wird ein neuer Unterordner angelegt und `201 Created` zurückgegeben

#### Scenario: Anlegen ohne Schreibrecht
- **WHEN** ein Nutzer ohne `can_write` auf den Elternordner einen Unterordner anlegen will
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Wurzelordner anlegen
- **WHEN** ein Admin `POST /api/folders` ohne `parent_id` aufruft
- **THEN** wird ein neuer Wurzelordner angelegt

### Requirement: Additive Berechtigungsvererbung
Das System SHALL Berechtigungen entlang des Ordnerpfads von der Wurzel zum Blatt auflösen. Ein Nutzer hat `can_read` oder `can_write` auf einen Ordner, wenn IRGENDEIN Ordner im Pfad (einschließlich des Ordners selbst) diese Berechtigung für den Nutzer gewährt. Ein Unterordner KANN NICHT Rechte entziehen die ein Vorfahren gewährt.

#### Scenario: Vererbtes Leserecht
- **WHEN** ein Nutzer `can_read` auf Ordner A hat und Ordner B ein Kind von A ist
- **THEN** hat der Nutzer auch `can_read` auf B (auch ohne explizite Permission auf B)

#### Scenario: Kein DENY möglich
- **WHEN** ein Nutzer `can_read` auf Ordner A hat
- **THEN** hat er `can_read` auf alle Nachkommen von A, unabhängig von deren ACL-Einträgen

### Requirement: ACL-Einträge verwalten
Nutzer mit `can_write` auf einen Ordner SHALL Berechtigungseinträge via `POST /api/folders/:id/permissions` anlegen können. Bestehende Einträge SHALL via `DELETE /api/folders/:id/permissions/:permId` entfernt werden können.

Ein Eintrag besteht aus `principal_type` (`everyone` | `role` | `club_function` | `user`), `principal_ref` (null bei `everyone`, sonst Rollenname / Funktionsname / user_id) sowie `can_read` und `can_write`.

#### Scenario: Berechtigung anlegen
- **WHEN** ein Nutzer mit `can_write` einen neuen ACL-Eintrag anlegt
- **THEN** wird der Eintrag gespeichert und ab sofort bei der Auflösung berücksichtigt

#### Scenario: Berechtigung lesen
- **WHEN** ein Nutzer mit `can_write` `GET /api/folders/:id/permissions` aufruft
- **THEN** erhält er alle direkten ACL-Einträge des Ordners (ohne geerbte)

#### Scenario: Berechtigung entfernen
- **WHEN** ein berechtigter Nutzer `DELETE /api/folders/:id/permissions/:permId` aufruft
- **THEN** wird der Eintrag gelöscht; geerbte Rechte bleiben unberührt

### Requirement: Anti-Eskalation
Das System SHALL verhindern, dass ein Nutzer mehr Rechte vergibt als er selbst auf den Ordner hat. Admin (`role = 'admin'`) ist ausgenommen und darf immer alle Rechte vergeben.

#### Scenario: Nur Leserecht — kann kein Schreibrecht vergeben
- **WHEN** ein Nutzer nur `can_read` (nicht `can_write`) auf einen Ordner hat und versucht einem anderen Nutzer `can_write` zu geben
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Schreibrecht — kann Lese- und Schreibrecht vergeben
- **WHEN** ein Nutzer `can_write` auf einen Ordner hat
- **THEN** darf er sowohl `can_read` als auch `can_write` an andere vergeben

#### Scenario: Admin darf alles vergeben
- **WHEN** ein Nutzer mit `role = 'admin'` einen ACL-Eintrag anlegt
- **THEN** wird der Eintrag ohne Eskalationsprüfung gespeichert

### Requirement: Principal-Typen
Das System SHALL vier Principal-Typen unterstützen, die bei der Berechtigungsauflösung in folgender Reihenfolge ausgewertet werden (spezifischste zuerst): `user` → `club_function` → `role` → `everyone`. Ein Treffer bei einem Typ reicht für die Gewährung.

#### Scenario: Everyone-Berechtigung
- **WHEN** ein Ordner einen ACL-Eintrag mit `principal_type = 'everyone'` und `can_read = 1` hat
- **THEN** hat jeder authentifizierte Nutzer `can_read` auf diesen Ordner

#### Scenario: Rollen-Berechtigung
- **WHEN** ein Ordner einen Eintrag mit `principal_type = 'role'`, `principal_ref = 'trainer'` und `can_write = 1` hat
- **THEN** haben alle Nutzer mit `role = 'trainer'` `can_write` auf diesen Ordner

#### Scenario: Vereinsfunktions-Berechtigung
- **WHEN** ein Ordner einen Eintrag mit `principal_type = 'club_function'` und `principal_ref = 'kassierer'` hat
- **THEN** haben alle Nutzer mit der Funktion `kassierer` in `ClubFunctions[]` die entsprechenden Rechte

#### Scenario: User-Berechtigung
- **WHEN** ein Ordner einen Eintrag mit `principal_type = 'user'` und `principal_ref = '42'` hat
- **THEN** hat nur Nutzer mit `id = 42` die entsprechenden Rechte

