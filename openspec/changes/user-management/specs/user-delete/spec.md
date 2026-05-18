## ADDED Requirements

### Requirement: Admin kann einen Nutzer löschen
Das System SHALL einen `DELETE /api/admin/users/{id}` Endpunkt bereitstellen, der einen Nutzer und alle zugehörigen Daten in einer Transaktion entfernt.

#### Scenario: Erfolgreiches Löschen
- **WHEN** ein Admin `DELETE /api/admin/users/{id}` mit einer gültigen Nutzer-ID aufruft
- **THEN** löscht das Backend den Nutzer inklusive seiner Refresh-Tokens, Family-Links, Duty-Assignments und Duty-Accounts in einer Transaktion und antwortet mit HTTP 204

#### Scenario: Nutzer nicht gefunden
- **WHEN** `DELETE /api/admin/users/{id}` mit einer nicht existierenden ID aufgerufen wird
- **THEN** antwortet das Backend mit HTTP 404

#### Scenario: Nur Admins dürfen Nutzer löschen
- **WHEN** ein Nutzer mit einer anderen Rolle als `admin` den Endpunkt aufruft
- **THEN** antwortet das Backend mit HTTP 403

### Requirement: Admin kann sich nicht selbst löschen
Das System SHALL verhindern, dass ein Admin seinen eigenen Account über die API löscht.

#### Scenario: Self-Delete-Versuch
- **WHEN** ein Admin `DELETE /api/admin/users/{id}` mit seiner eigenen ID aufruft
- **THEN** antwortet das Backend mit HTTP 400 und einer Fehlermeldung

### Requirement: Löschen erfordert Bestätigung im Frontend
Das Frontend SHALL vor dem Senden des Delete-Requests eine Bestätigungsaufforderung anzeigen.

#### Scenario: Admin bestätigt das Löschen
- **WHEN** ein Admin auf den Löschen-Button eines Nutzers klickt und den Bestätigungsdialog bestätigt
- **THEN** sendet das Frontend `DELETE /api/admin/users/{id}` und entfernt den Nutzer aus der Tabelle nach erfolgreichem Response

#### Scenario: Admin bricht das Löschen ab
- **WHEN** ein Admin auf den Löschen-Button eines Nutzers klickt und den Bestätigungsdialog abbricht
- **THEN** sendet das Frontend keinen Request und die Tabelle bleibt unverändert
