## ADDED Requirements

### Requirement: ChangePassword Sicherheitsinvariante
Das System SHALL bei Passwortänderung das aktuelle Passwort prüfen und nach erfolgreicher Änderung alle Refresh-Tokens des Nutzers löschen (erzwungener Logout auf allen Geräten).

#### Scenario: Passwort ändern mit korrektem altem Passwort
- **WHEN** POST /api/profile/password mit korrektem current_password und neuem Passwort
- **THEN** HTTP 204, Passwort in DB geändert, alle refresh_tokens des Users gelöscht

#### Scenario: Passwort ändern mit falschem alten Passwort
- **WHEN** POST /api/profile/password mit falschem current_password
- **THEN** HTTP 403 Forbidden, Passwort unverändert

### Requirement: Mitgliedschaftsantrag-Workflow
Das System SHALL Mitgliedschaftsanträge nur im Status `pending` genehmigen oder ablehnen können. Bei Genehmigung wird ein Einladungstoken erstellt.

#### Scenario: Antrag genehmigen
- **WHEN** Trainer/Admin POST /api/membership-requests/{id}/approve für pending-Antrag
- **THEN** HTTP 204, `membership_requests.status='approved'`, `invitation_tokens`-Eintrag für die Antragssteller-E-Mail angelegt

#### Scenario: Antrag ablehnen
- **WHEN** Trainer/Admin POST /api/membership-requests/{id}/reject für pending-Antrag
- **THEN** HTTP 204, `membership_requests.status='rejected'`

#### Scenario: Nicht-pending-Antrag genehmigen
- **WHEN** Approve für Antrag mit status='approved' oder 'rejected'
- **THEN** HTTP 404

### Requirement: ListUsers Paginierung und Suche
Das System SHALL Admin-Nutzerlistung mit server-seitiger Paginierung und Suche bereitstellen.

#### Scenario: Paginierung
- **WHEN** Admin GET /api/admin/users?limit=5&offset=5 bei 12 Nutzern
- **THEN** 5 Einträge, total=12

#### Scenario: Suche nach Name
- **WHEN** Admin GET /api/admin/users?search=müller
- **THEN** Nur Nutzer mit „müller" in first_name, last_name oder email
