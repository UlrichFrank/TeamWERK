## MODIFIED Requirements

### Requirement: Elternteil kann User-Strang des Kindes aktualisieren

Das System MUST einen Endpunkt `PUT /api/profile/kind/{memberId}/account` bereitstellen, der `users.first_name`, `users.last_name`, `users.street`, `users.zip`, `users.city`, `users.date_of_birth` des Kindes sofort aktualisiert. Der Endpunkt DARF NUR aufgerufen werden, wenn `members.user_id IS NOT NULL`. Die Autorisierung MUST über `family_links` erfolgen (isParentOf-Check).

#### Scenario: Elternteil aktualisiert Kontodaten des Kindes mit Account

- **WHEN** `PUT /api/profile/kind/42/account` mit `{ "first_name": "Max", "last_name": "Muster", "street": "...", "zip": "...", "city": "...", "date_of_birth": "2016-03-01" }` aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** werden `users.first_name`, `last_name`, `street`, `zip`, `city`, `date_of_birth` für User 7 sofort aktualisiert, HTTP 204

#### Scenario: Endpoint bei Kind ohne User-Account nicht aufrufbar

- **WHEN** `PUT /api/profile/kind/42/account` aufgerufen wird und Member 42 hat `user_id = NULL`
- **THEN** antwortet der Endpunkt mit HTTP 404

#### Scenario: Kein family_links-Eintrag

- **WHEN** `PUT /api/profile/kind/42/account` von einem Nutzer aufgerufen wird, der nicht Elternteil von Member 42 ist
- **THEN** antwortet der Endpunkt mit HTTP 403

### Requirement: GET Kind-Profil liefert User-Strang-Daten wenn Kind Account hat

`GET /api/profile/kind/{memberId}` MUST zusätzlich die `users`-Kontaktdaten des Kindes zurückgeben, wenn `members.user_id IS NOT NULL`. Die Felder MUST als `user_contact`-Objekt im Response enthalten sein mit: `first_name`, `last_name`, `street`, `zip`, `city`, `date_of_birth`, `phones` (aus `user_phones`), `visibility` (aus `user_visibility`).

#### Scenario: Kind mit User-Account — Response enthält user_contact

- **WHEN** `GET /api/profile/kind/42` aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** enthält der Response `user_contact` mit Name, Adresse, Geburtsdatum, Telefonnummern und Sichtbarkeitseinstellungen des Users 7

#### Scenario: Kind ohne User-Account — kein user_contact im Response

- **WHEN** `GET /api/profile/kind/42` aufgerufen wird und Member 42 hat `user_id = NULL`
- **THEN** enthält der Response kein `user_contact`-Objekt (oder `null`)

#### Scenario: Kind mit User-Account — Geburtsdatum im user_contact noch nie gesetzt

- **WHEN** `GET /api/profile/kind/42` aufgerufen wird, Member 42 hat `user_id = 7` und `users.date_of_birth IS NULL` für User 7
- **THEN** enthält `user_contact.date_of_birth` einen leeren String (Frontend übernimmt den Fallback auf `member.date_of_birth`)
