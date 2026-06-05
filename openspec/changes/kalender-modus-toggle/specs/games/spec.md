## ADDED Requirements

### Requirement: Spielplan für alle eingeloggten User lesbar

`GET /api/games` und `GET /api/games/{id}` SHALL ohne Rollen-Einschränkung für alle authentifizierten User zugänglich sein.

#### Scenario: Spieler ruft Spielplan ab
- **WHEN** ein User mit Rolle spieler oder elternteil `GET /api/games` aufruft
- **THEN** antwortet der Server mit HTTP 200 und der Spielplanliste

#### Scenario: Nav-Eintrag sichtbar
- **WHEN** ein User eingeloggt ist
- **THEN** ist der „Spielplan"-Menüeintrag in der Navigation sichtbar, unabhängig von der Rolle

---

### Requirement: Schreibzugriff für admin, vorstand und trainer

`POST`, `PUT` und `DELETE` auf `/api/admin/games/*` SHALL für die Rollen admin, vorstand und trainer zugänglich sein.

#### Scenario: Vorstand legt Event an
- **WHEN** ein User mit Rolle vorstand `POST /api/admin/games` aufruft
- **THEN** antwortet der Server mit HTTP 201

#### Scenario: Spieler kann kein Event anlegen
- **WHEN** ein User mit Rolle spieler oder elternteil `POST /api/admin/games` aufruft
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: Multi-Team-Zuordnung via `game_teams`

Ein Event SHALL einer oder mehreren Mannschaften zugeordnet sein, abgebildet über die Junction-Tabelle `game_teams`.

#### Scenario: Event mit mehreren Teams anlegen
- **WHEN** `POST /api/admin/games` mit `team_ids: [1, 2, 3]` aufgerufen wird
- **THEN** werden in `game_teams` drei Einträge angelegt
- **THEN** wird für jede Mannschaft ein identischer Satz Duty-Slots generiert

#### Scenario: Event ohne Team abgelehnt
- **WHEN** `POST /api/admin/games` ohne `team_ids` oder mit leerem Array aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: `event_type`-Feld

Jedes Event SHALL einen `event_type` haben: `heim`, `auswärts` oder `generisch`.

#### Scenario: Standard-Typ bei fehlendem Feld
- **WHEN** `POST /api/admin/games` ohne `event_type` aufgerufen wird
- **THEN** wird `event_type = 'heim'` gesetzt

#### Scenario: Ungültiger Typ abgelehnt
- **WHEN** `POST /api/admin/games` mit einem ungültigen `event_type` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: Explizite Vorlage bei Erstellung

(Existing requirements below this line preserved as-is)

## MODIFIED Requirements

### Requirement: PUT /api/admin/games/{id} erreichbar für trainer und vorstand

`PUT /api/admin/games/{id}` SHALL für die Rollen admin, trainer und vorstand zugänglich sein (bisher nur admin). Dies ermöglicht dem `GameEditModal` im Kalender das direkte Bearbeiten von Spieltagen durch Trainer.

#### Scenario: Trainer bearbeitet Spieltag via PUT

- **WHEN** ein User mit Rolle trainer `PUT /api/admin/games/{id}` mit gültigen Feldern aufruft
- **THEN** antwortet der Server mit HTTP 200 und den aktualisierten Spieltag-Daten

#### Scenario: Spieler kann Spieltag nicht bearbeiten

- **WHEN** ein User mit Rolle spieler `PUT /api/admin/games/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403
