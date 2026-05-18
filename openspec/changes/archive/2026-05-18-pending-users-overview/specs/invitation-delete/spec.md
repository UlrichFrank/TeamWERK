## ADDED Requirements

### Requirement: Admin kann eine Einladung widerrufen
Das System SHALL einen `DELETE /api/admin/invitations/{id}` Endpunkt bereitstellen, der den Einladungstoken aus der Datenbank entfernt. Der Link wird damit sofort ungültig.

#### Scenario: Erfolgreiches Löschen einer Einladung
- **WHEN** ein Admin `DELETE /api/admin/invitations/{id}` mit einer gültigen ID aufruft
- **THEN** löscht das Backend den Token und antwortet mit HTTP 204

#### Scenario: Einladung nicht gefunden
- **WHEN** `DELETE /api/admin/invitations/{id}` mit einer nicht existierenden ID aufgerufen wird
- **THEN** antwortet das Backend mit HTTP 404

#### Scenario: Nur Admins dürfen Einladungen löschen
- **WHEN** ein Nicht-Admin den Endpunkt aufruft
- **THEN** antwortet das Backend mit HTTP 403

### Requirement: Löschen einer Einladung erfordert Bestätigung im Frontend
Das Frontend SHALL vor dem Löschen einen Bestätigungsdialog anzeigen.

#### Scenario: Admin bestätigt das Löschen
- **WHEN** ein Admin auf „Löschen" bei einer Einladungszeile klickt und bestätigt
- **THEN** sendet das Frontend `DELETE /api/admin/invitations/{id}` und entfernt die Zeile aus der Tabelle

#### Scenario: Admin bricht ab
- **WHEN** ein Admin den Bestätigungsdialog abbricht
- **THEN** sendet das Frontend keinen Request und die Tabelle bleibt unverändert
