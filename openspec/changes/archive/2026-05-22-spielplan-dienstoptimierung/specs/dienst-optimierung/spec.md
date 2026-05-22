## ADDED Requirements

### Requirement: Dienstoptimierung bei Preview für neue Spiele

`GET /admin/duty-templates/{id}/preview` SHALL einen optionalen Query-Parameter `date` (ISO-Date, z.B. `2026-05-30`) akzeptieren. Wenn `date` angegeben wird (ohne `game_id`), lädt das Backend alle Heimspiele desselben Datums aus der aktiven Saison und fügt die Uhrzeit des neuen Spiels (`time`-Parameter) in die Same-Day-Liste ein. `applyBehavior` und `loadSameDayContext` werden aufgerufen.

#### Scenario: Preview für neues Heimspiel am selben Tag wie ein anderes Heimspiel

- **WHEN** `GET /admin/duty-templates/1/preview?time=14:00&date=2026-05-30` aufgerufen wird
- **AND** an `2026-05-30` existiert bereits ein Heimspiel um 11:00 Uhr
- **THEN** enthält `allGameTimes` beide Uhrzeiten (`["11:00", "14:00"]`)
- **THEN** werden Duty-Slots mit `same_day_behavior != "normal"` entsprechend reduziert oder übersprungen

#### Scenario: Preview ohne same-day-Kontext (erstes Spiel an dem Tag)

- **WHEN** `GET /admin/duty-templates/1/preview?time=14:00&date=2026-05-30` aufgerufen wird
- **AND** kein weiteres Heimspiel existiert an `2026-05-30`
- **THEN** enthält `allGameTimes` nur `["14:00"]`
- **THEN** werden alle Duty-Slots unverändert zurückgegeben (keine Optimierung)

#### Scenario: Preview ohne `date`-Parameter (altes Verhalten)

- **WHEN** `GET /admin/duty-templates/1/preview?time=14:00` ohne `date` und ohne `game_id` aufgerufen wird
- **THEN** wird `applyBehavior` nicht aufgerufen (Verhalten wie bisher)
- **THEN** gibt der Endpunkt HTTP 200 mit allen Slots zurück

---

### Requirement: Dienstoptimierung bei Regenerierung

`POST /admin/games/{id}/regenerate` SHALL `template_id` aus dem Request-Body verwenden, Template-Items aus der DB laden, `loadSameDayContext` aufrufen und `applyBehavior` anwenden.

#### Scenario: Regenerierung mit same-day-Kontext

- **WHEN** `POST /admin/games/42/regenerate` mit `{"template_id": 1}` aufgerufen wird
- **AND** das Spiel ist ein Heimspiel am selben Datum wie ein weiteres Heimspiel
- **THEN** werden Duty-Slots gemäß `same_day_behavior` reduziert oder übersprungen
- **THEN** wird `template_id` in `games.template_id` gespeichert

#### Scenario: Regenerierung ohne gespeichertes und ohne übergebenes Template

- **WHEN** `POST /admin/games/42/regenerate` mit leerem Body oder ohne `template_id` aufgerufen wird
- **AND** `games.template_id` ist NULL
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: Nur Heimspiele zählen als same-day-Kontext

`loadSameDayContext` SHALL nur Spiele mit `is_home=1` als Same-Day-Kontext berücksichtigen.

#### Scenario: Auswärtsspiel am selben Tag

- **WHEN** ein Heimspiel für `2026-05-30` erstellt wird
- **AND** an `2026-05-30` existiert ein Auswärtsspiel
- **THEN** gilt der Tag als Einzelspiel-Tag (kein same-day-Kontext ausgelöst)
