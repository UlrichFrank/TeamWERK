## Why

Beim Generieren von Dienst-Slots aus einem Spielplan-Template werden heute für jedes Heimspiel blind alle Template-Einträge erzeugt. Das führt dazu, dass Aufbau und Abbau für jedes Spiel des Tages generiert werden — obwohl nur das erste Spiel Aufbau und nur das letzte Spiel Abbau benötigt. An Wochenenden mit Spielen an aufeinanderfolgenden Tagen soll außerdem ein reduzierter Auf-/Abbau oder gar keiner generiert werden.

## What Changes

- `duty_types` bekommt zwei neue Felder: `consecutive_behavior` (Verhalten bei aufeinanderfolgenden Spieltagen) und optionales `consecutive_variant_id` (Verweis auf reduzierten Ersatz-Diensttyp)
- Die Slot-Generierung (Template-Anwendung via `RegenerateSlots`) **berechnet** `applies_when` beim Erzeugen: berücksichtigt die Position des Spiels im Tagesablauf und prüft ob Vor- oder Folgetag ebenfalls Heimspiele hat
- Die Admin-Oberfläche für Diensttypen (`/admin/dienste`) zeigt und pflegt die neuen Felder (ohne `applies_when` — wird berechnet)

## Capabilities

### New Capabilities

- `duty-type-schedule-role`: Konfiguration wann ein Diensttyp im Spieltagskontext aktiv ist (`applies_when`) und wie er sich bei aufeinanderfolgenden Spieltagen verhält (`consecutive_behavior` + `consecutive_variant_id`)

### Modified Capabilities

## Impact

- DB-Migration: zwei neue Spalten auf `duty_types` (`consecutive_behavior`, `consecutive_variant_id` FK)
- Backend: `internal/duties/handler.go` (CRUD für Diensttypen), `internal/games/handler.go` (berechnet `applies_when` bei Slot-Generierung)
- Frontend: `AdminDutyTypesPage.tsx` (zeigt/bearbeitet `consecutive_behavior` und `consecutive_variant_id`), `AdminSpielplanPage.tsx` (Regenerate-Flow nutzt neue Logik automatisch)
