## REMOVED Requirements

### Requirement: Teamleiter sieht nur eigene Teammitglieder
**Reason**: Trainer haben keinen Lesezugriff mehr auf die Mitgliederliste. Die Verwaltung von Spielerzuordnungen erfolgt ausschließlich über das Kader-Modul (`/api/admin/kader`), das eigene Endpunkte mit eigenem Datenbankzugriff hat.
**Migration**: Trainer nutzen `/admin/kader` und `/api/admin/kader/{id}/member-suggestions` für alle teambezogenen Mitgliederoperationen.

## ADDED Requirements

### Requirement: Mitgliederliste nur für Vorstand und Admin
Das System SHALL `GET /api/members` und `GET /api/members/{id}` nur für Nutzer mit der Vereinsfunktion `vorstand` oder der Systemrolle `admin` zugänglich machen. Alle anderen authentifizierten Nutzer erhalten 403.

#### Scenario: Trainer ruft Mitgliederliste ab
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` (aber ohne `vorstand`) `GET /api/members` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Trainer ruft Mitgliederdetail ab
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` (aber ohne `vorstand`) `GET /api/members/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Vorstand ruft Mitgliederliste ab
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `GET /api/members` aufruft
- **THEN** antwortet der Server mit HTTP 200 und der vollständigen Mitgliederliste

#### Scenario: Admin ruft Mitgliederliste ab
- **WHEN** ein Nutzer mit Systemrolle `admin` `GET /api/members` aufruft
- **THEN** antwortet der Server mit HTTP 200 und der vollständigen Mitgliederliste

### Requirement: Mitglieder-Frontend-Route nur für Vorstand und Admin
Das System SHALL Direktzugriff auf `/mitglieder` und `/mitglieder/:id` per URL nur Nutzern mit Vereinsfunktion `vorstand` oder Systemrolle `admin` erlauben. Andere Nutzer werden zur Startseite weitergeleitet.

#### Scenario: Trainer öffnet /mitglieder direkt per URL
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` die URL `/mitglieder` direkt aufruft
- **THEN** wird er zur Startseite (`/`) weitergeleitet

#### Scenario: Trainer öffnet /mitglieder/:id direkt per URL
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` die URL `/mitglieder/6` direkt aufruft
- **THEN** wird er zur Startseite (`/`) weitergeleitet
