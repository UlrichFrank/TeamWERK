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

`POST /api/admin/games` SHALL ein optionales Feld `template_id` akzeptieren. Wenn angegeben, wird diese Vorlage direkt für die Slot-Generierung verwendet.

#### Scenario: Vorlage explizit übergeben
- **WHEN** `POST /api/admin/games` mit `template_id: 5` aufgerufen wird
- **THEN** werden Slots aus Vorlage 5 generiert, ohne nach `template_type` zu suchen

#### Scenario: Kein template_id — Fallback
- **WHEN** `POST /api/admin/games` ohne `template_id` aufgerufen wird
- **THEN** wählt das Backend automatisch anhand `event_type` eine passende Vorlage (bisheriges Verhalten)

---

### Requirement: Game-Response enthält Teams

`GET /api/games` und `GET /api/games/{id}` SHALL pro Event eine Liste der zugeordneten Teams zurückgeben.

#### Scenario: Teams in der Antwort
- **WHEN** `GET /api/games` aufgerufen wird
- **THEN** enthält jedes Game-Objekt ein Array `teams: [{id, name}]`
