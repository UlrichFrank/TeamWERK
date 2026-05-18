## ADDED Requirements

### Requirement: Admin kann eine Beitrittsanfrage löschen
Das System SHALL einen `DELETE /api/admin/membership-requests/{id}` Endpunkt bereitstellen, der eine Anfrage vollständig aus der Datenbank entfernt. Dies ergänzt die bestehenden Approve/Reject-Aktionen.

#### Scenario: Erfolgreiches Löschen einer Anfrage
- **WHEN** ein Admin `DELETE /api/admin/membership-requests/{id}` mit einer gültigen ID aufruft
- **THEN** löscht das Backend die Anfrage und antwortet mit HTTP 204

#### Scenario: Anfrage nicht gefunden
- **WHEN** `DELETE /api/admin/membership-requests/{id}` mit einer nicht existierenden ID aufgerufen wird
- **THEN** antwortet das Backend mit HTTP 404

#### Scenario: Nur Admins dürfen Anfragen löschen
- **WHEN** ein Nicht-Admin den Endpunkt aufruft
- **THEN** antwortet das Backend mit HTTP 403

### Requirement: Beitrittsanfragen erscheinen in der Nutzertabelle
Offene Beitrittsanfragen (`status = 'pending'`) SHALL in der Nutzertabelle der AdminUsersPage sichtbar sein, mit Approve-, Reject- und Delete-Aktionen pro Zeile.

#### Scenario: Anfrage in Tabelle erkennbar
- **WHEN** eine offene Beitrittsanfrage in der Tabelle angezeigt wird
- **THEN** erscheint der Status-Badge mit der Beschriftung „Anfrage", Name und E-Mail des Antragstellers sind sichtbar

#### Scenario: Anfrage nach Approve/Reject/Delete aus Tabelle entfernt
- **WHEN** ein Admin eine Anfrage genehmigt, ablehnt oder löscht
- **THEN** verschwindet die Zeile sofort aus der Tabelle (optimistisches Update)
