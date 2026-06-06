### Requirement: Elternteil kann User-Strang des Kindes aktualisieren

Das System MUSS einen Endpunkt `PUT /api/profile/kind/{memberId}/account` bereitstellen, der `users.first_name`, `users.last_name`, `users.street`, `users.zip`, `users.city` des Kindes sofort aktualisiert. Der Endpunkt DARF NUR aufgerufen werden, wenn `members.user_id IS NOT NULL`. Die Autorisierung MUSS über `family_links` erfolgen (isParentOf-Check).

#### Scenario: Elternteil aktualisiert Kontodaten des Kindes mit Account

- **WHEN** `PUT /api/profile/kind/42/account` mit `{ "first_name": "Max", "last_name": "Muster", "street": "...", "zip": "...", "city": "..." }` aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** werden `users.first_name`, `last_name`, `street`, `zip`, `city` für User 7 sofort aktualisiert, HTTP 204

#### Scenario: Endpoint bei Kind ohne User-Account nicht aufrufbar

- **WHEN** `PUT /api/profile/kind/42/account` aufgerufen wird und Member 42 hat `user_id = NULL`
- **THEN** antwortet der Endpunkt mit HTTP 404

#### Scenario: Kein family_links-Eintrag

- **WHEN** `PUT /api/profile/kind/42/account` von einem Nutzer aufgerufen wird, der nicht Elternteil von Member 42 ist
- **THEN** antwortet der Endpunkt mit HTTP 403

### Requirement: GET Kind-Profil liefert User-Strang-Daten wenn Kind Account hat

`GET /api/profile/kind/{memberId}` MUSS zusätzlich die `users`-Kontaktdaten des Kindes zurückgeben, wenn `members.user_id IS NOT NULL`. Die Felder MÜSSEN als `user_contact`-Objekt im Response enthalten sein mit: `first_name`, `last_name`, `street`, `zip`, `city`, `phones` (aus `user_phones`), `visibility` (aus `user_visibility`).

#### Scenario: Kind mit User-Account — Response enthält user_contact

- **WHEN** `GET /api/profile/kind/42` aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** enthält der Response `user_contact` mit Name, Adresse, Telefonnummern und Sichtbarkeitseinstellungen des Users 7

#### Scenario: Kind ohne User-Account — kein user_contact im Response

- **WHEN** `GET /api/profile/kind/42` aufgerufen wird und Member 42 hat `user_id = NULL`
- **THEN** enthält der Response kein `user_contact`-Objekt (oder `null`)

### Requirement: Phones-Endpunkte nutzen user_phones wenn Kind Account hat

`POST /api/profile/kind/{memberId}/phones` und `DELETE /api/profile/kind/{memberId}/phones/{phoneId}` MÜSSEN in `user_phones` des Kindes schreiben/löschen, wenn `members.user_id IS NOT NULL`. Bei `user_id IS NULL` bleibt `member_phones` das Ziel.

#### Scenario: Telefonnummer hinzufügen — Kind mit Account

- **WHEN** `POST /api/profile/kind/42/phones` aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** wird die Telefonnummer in `user_phones` mit `user_id = 7` gespeichert, nicht in `member_phones`

#### Scenario: Telefonnummer löschen — Kind mit Account

- **WHEN** `DELETE /api/profile/kind/42/phones/5` aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** wird der Eintrag aus `user_phones` mit `user_id = 7` gelöscht, nicht aus `member_phones`

#### Scenario: Telefonnummer hinzufügen — Kind ohne Account

- **WHEN** `POST /api/profile/kind/42/phones` aufgerufen wird und Member 42 hat `user_id = NULL`
- **THEN** antwortet der Endpunkt mit HTTP 403 — kein User-Strang vorhanden, direkte member_phones-Writes sind nicht erlaubt

### Requirement: Visibility-Endpoint nutzt user_visibility wenn Kind Account hat

`PUT /api/profile/kind/{memberId}/visibility` MUSS in `user_visibility` des Kindes schreiben (UPSERT), wenn `members.user_id IS NOT NULL`. Bei `user_id IS NULL` werden die Felder `phones_visible`, `address_visible`, `photo_visible`, `email_visible` in der `members`-Tabelle gesetzt (bisheriges Verhalten).

#### Scenario: Visibility setzen — Kind mit Account

- **WHEN** `PUT /api/profile/kind/42/visibility` mit `{ "phones_visible": true, ... }` aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** wird ein UPSERT auf `user_visibility` für `user_id = 7` ausgeführt

#### Scenario: Visibility setzen — Kind ohne Account

- **WHEN** `PUT /api/profile/kind/42/visibility` aufgerufen wird und Member 42 hat `user_id = NULL`
- **THEN** antwortet der Endpunkt mit HTTP 403 — kein User-Strang vorhanden, direkte members-Writes sind nicht erlaubt
