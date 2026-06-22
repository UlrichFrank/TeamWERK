## MODIFIED Requirements

### Requirement: CreateGame-Request für Heim/Auswärts ohne `slots[]`

Das System SHALL für `POST /api/admin/games` mit `event_type ∈ {heim, auswärts}` das Feld `slots[]` aus dem Request-Body ignorieren. Slots werden ausschließlich aus dem persistierten `games.template_id` generiert. Ist `template_id` `null`, werden keine Auto-Dienste erzeugt.

Für `event_type=generisch` MUSS `template_id` `null` sein; ein gesetzter Wert führt zu HTTP 400. `slots[]` bleibt erhalten und wird mit `is_custom=1` persistiert.

#### Scenario: Heimspiel mit `slots[]` im Request — `slots[]` wird ignoriert

- **WHEN** `POST /api/admin/games` mit `event_type=heim`, `template_id=7` und nicht-leerem `slots[]`-Array aufgerufen wird
- **THEN** ignoriert das Backend `slots[]` und erzeugt die Slots ausschließlich per Auto-Regen aus Template 7
- **AND** die Response liefert HTTP 201 mit `id` und `regen_summary`

#### Scenario: Heimspiel ohne Vorlage erzeugt keine Auto-Slots

- **WHEN** `POST /api/admin/games` mit `event_type=heim` und `template_id=null` aufgerufen wird
- **THEN** wird das Game persistiert mit `template_id=NULL`
- **AND** der Auto-Regen erzeugt keine `is_custom=0`-Slots für dieses Event
- **AND** die Response liefert HTTP 201 mit `id` und `regen_summary`

#### Scenario: Generisches Event persistiert `slots[]` mit `is_custom=1`

- **WHEN** `POST /api/admin/games` mit `event_type=generisch`, `template_id=null` und `slots[]` aufgerufen wird
- **THEN** werden alle Slots aus `slots[]` mit `is_custom=1` in `duty_slots` persistiert
- **AND** Auto-Regen für `{D-1, D, D+1}` läuft anschließend, betrifft aber nur eventuelle template-basierte Slots benachbarter Spiele

#### Scenario: Generisches Event mit Template wird abgewiesen

- **WHEN** `POST /api/admin/games` mit `event_type=generisch` und `template_id != null` aufgerufen wird
- **THEN** antwortet die API mit HTTP 400 und Body `{"error":"template_id muss bei event_type=generisch null sein"}`

## ADDED Requirements

### Requirement: Auflösung von `template_id` ohne Fallback

Das System SHALL beim Auto-Regen ausschließlich den persistierten Wert von `games.template_id` als Slot-Quelle verwenden. Ist der Wert `NULL`, werden keine `is_custom=0`-Slots für dieses Event erzeugt — unabhängig vom `event_type`. Der frühere ID-basierte Fallback („kleinste passende Template-ID") entfällt ersatzlos.

#### Scenario: Auto-Regen für Event mit `template_id=NULL` erzeugt keine Slots

- **WHEN** `runAutoRegen` für ein Event mit `template_id IS NULL` läuft
- **THEN** werden alle vorhandenen `is_custom=0`-Slots des Events gelöscht und nicht ersetzt
- **AND** `is_custom=1`-Slots des Events bleiben unverändert

#### Scenario: Kein impliziter Default bei NULL

- **GIVEN** mehrere `game_templates` mit `template_type='heim'` existieren
- **WHEN** ein Event mit `event_type='heim'` und `template_id IS NULL` regeneriert wird
- **THEN** wird KEINE Vorlage automatisch ausgewählt; das Event bleibt ohne Auto-Slots

### Requirement: `template_id` per `PUT /api/admin/games/{id}` änderbar

Das System SHALL im `PUT /api/admin/games/{id}`-Endpoint das Feld `template_id` mit Tri-State-Semantik akzeptieren:

| Body-Inhalt              | Verhalten                                  |
|--------------------------|--------------------------------------------|
| Feld nicht vorhanden     | `template_id` bleibt unverändert           |
| `"template_id": null`    | `template_id` wird auf NULL gesetzt        |
| `"template_id": <int>`   | `template_id` wird auf den Wert gesetzt    |

Nach der Persistierung läuft `runAutoRegen` für das Datum-Fenster (`oldDate ± 1` ∪ `newDate ± 1`) wie bisher; bei `NULL` werden bestehende `is_custom=0`-Slots des Events gelöscht und nicht ersetzt.

Wenn `event_type='generisch'` UND das Feld `template_id` mit Wert ≠ `null` gesendet wird, antwortet die API mit HTTP 400.

#### Scenario: Feld fehlt im Body — Wert bleibt unverändert

- **GIVEN** ein Game mit `template_id=5`
- **WHEN** `PUT /api/admin/games/{id}` ohne `template_id`-Feld im Body aufgerufen wird
- **THEN** bleibt `games.template_id=5` erhalten
- **AND** Auto-Regen verwendet weiterhin Template 5

#### Scenario: Explizites `null` setzt auf NULL

- **GIVEN** ein Game mit `template_id=5` und mehreren `is_custom=0`-Slots
- **WHEN** `PUT /api/admin/games/{id}` mit `"template_id": null` aufgerufen wird
- **THEN** wird `games.template_id=NULL` gesetzt
- **AND** die `is_custom=0`-Slots des Events werden im Auto-Regen entfernt
- **AND** `is_custom=1`-Slots des Events bleiben unverändert
- **AND** die Response liefert HTTP 200 mit `regen_summary`

#### Scenario: Wechsel der Vorlage regeneriert Slots

- **GIVEN** ein Game mit `template_id=5`
- **WHEN** `PUT /api/admin/games/{id}` mit `"template_id": 7` aufgerufen wird
- **THEN** wird `games.template_id=7` gesetzt
- **AND** die Slots werden aus Template 7 neu erzeugt
- **AND** die Response liefert HTTP 200 mit `regen_summary`

#### Scenario: Generisches Event mit Template-Wechsel abgewiesen

- **GIVEN** ein Game mit `event_type='generisch'` und `template_id=NULL`
- **WHEN** `PUT /api/admin/games/{id}` mit `"template_id": 7` aufgerufen wird
- **THEN** antwortet die API mit HTTP 400 und der bestehende NULL-Wert bleibt erhalten
