## MODIFIED Requirements

### Requirement: Visibility-Endpoint nutzt user_visibility wenn Kind Account hat

`PUT /api/profile/kind/{memberId}/visibility` MUST in `user_visibility` des Kindes schreiben (UPSERT), wenn `members.user_id IS NOT NULL`. Bei `user_id IS NULL` werden die Felder `phones_visible`, `address_visible`, `email_visible` in der `members`-Tabelle gesetzt (bisheriges Verhalten). Der Fallback für `photo_visible` entfällt — `photo_visible` lebt ausschließlich in `user_visibility`, da Fotos serverseitig nur an einen User-Account gebunden werden können.

#### Scenario: Visibility setzen — Kind mit Account

- **WHEN** `PUT /api/profile/kind/42/visibility` mit `{ "phones_visible": true, "photo_visible": true, ... }` aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** wird ein UPSERT auf `user_visibility` für `user_id = 7` ausgeführt

#### Scenario: Visibility setzen — Kind ohne Account

- **WHEN** `PUT /api/profile/kind/42/visibility` aufgerufen wird und Member 42 hat `user_id = NULL`
- **THEN** antwortet der Endpunkt mit HTTP 403 — kein User-Strang vorhanden, direkte members-Writes sind nicht erlaubt

#### Scenario: photo_visible ohne Account nicht setzbar

- **WHEN** ein Aufrufer versucht, `photo_visible` an einem Member ohne User-Account zu setzen (auf welchem Weg auch immer)
- **THEN** existiert kein Speicherplatz mehr in `members.photo_visible` und die Änderung wird abgelehnt (HTTP 403 auf dem Kind-Endpoint; Draft-/Approval-Pfade prüfen `user_id IS NOT NULL`)

## ADDED Requirements

### Requirement: Foto-Endpoint schreibt users.photo_path des Kindes

`POST /api/profile/kind/{memberId}/photo` und `DELETE /api/profile/kind/{memberId}/photo` MUST `users.photo_path` **des Kind-Users** aktualisieren, wenn `members.user_id IS NOT NULL`. Die Autorisierung MUST über `family_links` erfolgen (isParentOf-Check). Wenn `members.user_id IS NULL`, MUST der Endpunkt mit HTTP 409 antworten und **keinen** File-System-Write durchführen.

#### Scenario: Elternteil lädt Kinderfoto hoch — Kind mit Account

- **WHEN** `POST /api/profile/kind/42/photo` mit einer Bilddatei aufgerufen wird und Member 42 hat `user_id = 7`
- **THEN** wird die Datei gespeichert, `users.photo_path` für User 7 auf den Dateinamen gesetzt, HTTP 200 mit `{ "photo_url": "/api/uploads/..." }`
- **AND** `members.photo_path` wird nicht mehr geschrieben (Spalte existiert nach Migration 029 nicht)

#### Scenario: Elternteil löscht Kinderfoto — Kind mit Account

- **WHEN** `DELETE /api/profile/kind/42/photo` aufgerufen wird und Member 42 hat `user_id = 7` mit gesetztem `users.photo_path`
- **THEN** wird die Datei entfernt und `users.photo_path` für User 7 auf `NULL` gesetzt, HTTP 204

#### Scenario: Foto-Upload für Kind ohne Account

- **WHEN** `POST /api/profile/kind/42/photo` aufgerufen wird und Member 42 hat `user_id = NULL`
- **THEN** antwortet der Endpunkt mit HTTP 409 und Body `{"error": "member_has_no_user_account"}`
- **AND** es findet kein Datei-Write statt

#### Scenario: Kein family_links-Eintrag

- **WHEN** `POST /api/profile/kind/42/photo` von einem Nutzer aufgerufen wird, der nicht Elternteil von Member 42 ist
- **THEN** antwortet der Endpunkt mit HTTP 403

### Requirement: Kind-Profil-Response liefert Foto aus User-Strang

`GET /api/profile/kind/{memberId}` MUST das Feld `member.photo_url` aus `users.photo_path` des Kind-Users befüllen (via `members.user_id`-Join), nicht aus einer separaten `members.photo_path`-Spalte. Ist `members.user_id IS NULL` oder `users.photo_path IS NULL`, MUST `member.photo_url` fehlen oder `null` sein.

#### Scenario: Kind mit Account und Foto

- **WHEN** `GET /api/profile/kind/42` und Member 42 hat `user_id = 7`, User 7 hat `photo_path = "abc.jpg"`
- **THEN** enthält der Response `member.photo_url = "/api/uploads/abc.jpg"`

#### Scenario: Kind mit Account ohne Foto

- **WHEN** `GET /api/profile/kind/42` und Member 42 hat `user_id = 7`, User 7 hat `photo_path = NULL`
- **THEN** enthält `member` kein `photo_url`-Feld (oder `null`)

#### Scenario: Kind ohne Account

- **WHEN** `GET /api/profile/kind/42` und Member 42 hat `user_id = NULL`
- **THEN** enthält `member` kein `photo_url`-Feld (oder `null`)
