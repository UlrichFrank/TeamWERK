## MODIFIED Requirements

### Requirement: Explizite Vorlage bei Erstellung

`POST /api/admin/games` SHALL ein optionales Feld `template_id` akzeptieren. Wenn angegeben, wird `template_id` in der `games`-Tabelle gespeichert. Die Slot-Generierung nutzt weiterhin die Slots aus dem Request-Body (die durch den PreviewSlots-Endpunkt berechnet wurden).

#### Scenario: Vorlage explizit übergeben — wird gespeichert

- **WHEN** `POST /api/admin/games` mit `template_id: 5` aufgerufen wird
- **THEN** werden Slots aus dem `slots`-Array im Request-Body angelegt
- **THEN** wird `games.template_id = 5` in der DB gespeichert

#### Scenario: Kein template_id — NULL in DB

- **WHEN** `POST /api/admin/games` ohne `template_id` aufgerufen wird
- **THEN** wird das Spiel angelegt mit `template_id = NULL`

---

## ADDED Requirements

### Requirement: Game-Response enthält template_id

`GET /api/games/{id}` SHALL `template_id` im `game`-Objekt der Antwort zurückgeben (null wenn nicht gesetzt).

#### Scenario: Spiel mit gespeichertem Template

- **WHEN** `GET /api/games/42` aufgerufen wird
- **AND** `games.template_id = 3`
- **THEN** enthält die Antwort `"game": { ..., "template_id": 3 }`

#### Scenario: Spiel ohne Template

- **WHEN** `GET /api/games/42` aufgerufen wird
- **AND** `games.template_id` ist NULL
- **THEN** enthält die Antwort `"game": { ..., "template_id": null }`

---

### Requirement: Template-basierte Regenerierung

`POST /api/admin/games/{id}/regenerate` SHALL `template_id` aus dem Request-Body akzeptieren. Das Backend lädt Template-Items aus der DB, wendet Optimierungslogik an und generiert Duty-Slots. Ein `slots`-Array im Request-Body ist nicht mehr erforderlich. Der gespeicherte `games.template_id` wird auf den übergebenen Wert aktualisiert.

#### Scenario: Regenerierung mit template_id im Body

- **WHEN** `POST /api/admin/games/42/regenerate` mit `{"template_id": 2}` aufgerufen wird
- **THEN** werden alle unfüllten Duty-Slots des Spiels gelöscht
- **THEN** werden neue Slots aus Template 2 generiert
- **THEN** wird `games.template_id = 2` gespeichert
- **THEN** antwortet der Server mit HTTP 200 und `{"kept_slots": N}`

#### Scenario: Regenerierung ohne template_id — Fallback auf gespeichertes Template

- **WHEN** `POST /api/admin/games/42/regenerate` ohne `template_id` aufgerufen wird
- **AND** `games.template_id = 3`
- **THEN** wird Template 3 für die Regenerierung verwendet

#### Scenario: Regenerierung ohne template_id und ohne gespeichertes Template

- **WHEN** `POST /api/admin/games/42/regenerate` ohne `template_id` aufgerufen wird
- **AND** `games.template_id` ist NULL
- **THEN** antwortet der Server mit HTTP 400
