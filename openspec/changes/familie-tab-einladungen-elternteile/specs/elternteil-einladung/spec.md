## ADDED Requirements

### Requirement: Einladung als Erziehungsberechtigter vormerken
Ein Admin SHALL eine ausstehende Einladung über `PUT /api/admin/invitations/{id}/parent-member` mit einem Mitglied verknüpfen können. Das setzt `invitation_tokens.parent_member_id` auf die Mitglieds-ID. Dieselbe Route mit `{ member_id: null }` entfernt die Verknüpfung.

#### Scenario: Einladung als Elternteil verknüpfen
- **WHEN** ein Admin `PUT /api/admin/invitations/{id}/parent-member` mit `{ "member_id": 50 }` aufruft
- **THEN** wird `invitation_tokens.parent_member_id = 50` gesetzt und HTTP 204 zurückgegeben

#### Scenario: Verknüpfung entfernen
- **WHEN** ein Admin `PUT /api/admin/invitations/{id}/parent-member` mit `{ "member_id": null }` aufruft
- **THEN** wird `invitation_tokens.parent_member_id = NULL` gesetzt und HTTP 204 zurückgegeben

#### Scenario: Nicht-existierende oder bereits verwendete Einladung
- **WHEN** die Einladung nicht existiert oder `used_at` gesetzt ist
- **THEN** gibt der Endpoint HTTP 404 zurück

### Requirement: Automatische family_links-Erstellung bei Registrierung
Wenn ein Nutzer sich über eine Einladung registriert, bei der `parent_member_id` gesetzt ist, SHALL das System automatisch einen Eintrag in `family_links` anlegen.

#### Scenario: Registrierung mit parent_member_id
- **WHEN** ein Nutzer sich über eine Einladung registriert, deren `parent_member_id = 50` gesetzt ist
- **THEN** wird nach der User-Erstellung `INSERT INTO family_links (parent_user_id, member_id) VALUES (newUserID, 50)` ausgeführt

#### Scenario: parent_member_id nicht gesetzt
- **WHEN** ein Nutzer sich über eine Einladung ohne `parent_member_id` registriert
- **THEN** wird kein `family_links`-Eintrag angelegt (kein Unterschied zum bisherigen Verhalten)

#### Scenario: family_link bereits vorhanden (Duplikat)
- **WHEN** beim INSERT in `family_links` ein UNIQUE-Constraint-Fehler auftritt
- **THEN** wird der Fehler ignoriert (IGNORE-Semantik), die Registrierung schlägt nicht fehl

### Requirement: Invitations-Endpoint gibt parent_member_id zurück
`GET /api/admin/invitations` SHALL das Feld `parent_member_id` (int oder null) pro Einladung zurückgeben.

#### Scenario: Einladung mit gesetztem parent_member_id
- **WHEN** `GET /api/admin/invitations` aufgerufen wird und eine Einladung hat `parent_member_id = 50`
- **THEN** enthält das entsprechende Objekt `"parent_member_id": 50`

#### Scenario: Einladung ohne parent_member_id
- **WHEN** `GET /api/admin/invitations` aufgerufen wird und eine Einladung hat `parent_member_id = NULL`
- **THEN** enthält das entsprechende Objekt `"parent_member_id": null`
