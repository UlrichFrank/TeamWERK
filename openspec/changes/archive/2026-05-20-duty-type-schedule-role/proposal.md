## Why

Beim Generieren von Dienst-Slots aus einem Spielplan-Template werden heute für jedes Heimspiel blind alle Template-Einträge erzeugt. Das führt dazu, dass Aufbau und Abbau für jedes Spiel des Tages generiert werden — obwohl nur das erste Spiel Aufbau und nur das letzte Spiel Abbau benötigt. An Wochenenden mit Spielen an aufeinanderfolgenden Tagen soll außerdem ein reduzierter Auf-/Abbau oder gar keiner generiert werden.

## What Changes

- `duty_types` bekommt vier neue Felder:
  - `same_day_behavior`: Verhalten wenn mehrere Heimspiele am **gleichen Tag** existieren (`normal`, `skip`, `reduced`)
  - `same_day_variant_id`: optionaler Verweis auf Ersatz-Diensttyp bei `same_day_behavior='reduced'`
  - `adjacent_day_behavior`: Verhalten wenn Heimspiele am **Vortag/Folgetag** existieren (`normal`, `skip`, `reduced`)
  - `adjacent_day_variant_id`: optionaler Verweis auf Ersatz-Diensttyp bei `adjacent_day_behavior='reduced'`
- Die Slot-Generierung (Template-Anwendung via `RegenerateSlots`) **berechnet** `applies_when` beim Erzeugen: berücksichtigt die Position des Spiels im Tagesablauf
- Beide Verhaltensweisen sind orthogonal und können kombiniert werden
- Die Admin-Oberfläche für Diensttypen (`/admin/dienste`) zeigt und pflegt die neuen Felder (ohne `applies_when` — wird berechnet)

## Capabilities

### New Capabilities

- `duty-type-schedule-role`: Konfiguration wann ein Diensttyp im Spieltagskontext aktiv ist (`applies_when` — berechnet) und wie er sich bei mehreren Spielen am gleichen Tag oder an aufeinanderfolgenden Tagen verhält (separate Konfigurationen für beide Szenarien)

### Modified Capabilities

## Impact

- DB-Migration: vier neue Spalten auf `duty_types` (`same_day_behavior`, `same_day_variant_id`, `adjacent_day_behavior`, `adjacent_day_variant_id`)
- Backend: `internal/duties/handler.go` (CRUD für Diensttypen), `internal/games/handler.go` (berechnet `applies_when` und wendet beide Verhaltensweisen an)
- Frontend: `AdminDutyTypesPage.tsx` (zeigt/bearbeitet vier neue Felder), `AdminSpielplanPage.tsx` (Regenerate-Flow nutzt neue Logik automatisch)
