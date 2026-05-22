## 1. Datenbank-Migration

- [x] 1.1 Migration `0NN_duty_type_schedule_role.up.sql` anlegen: `ALTER TABLE duty_types ADD COLUMN same_day_behavior TEXT NOT NULL DEFAULT 'normal' CHECK (same_day_behavior IN ('normal','skip','reduced'))`, `ADD COLUMN same_day_variant_id INTEGER REFERENCES duty_types(id)`, `ADD COLUMN adjacent_day_behavior TEXT NOT NULL DEFAULT 'normal' CHECK (adjacent_day_behavior IN ('normal','skip','reduced'))`, `ADD COLUMN adjacent_day_variant_id INTEGER REFERENCES duty_types(id)`
- [x] 1.2 Passende `.down.sql` anlegen (Spalten via Tabellen-Rebuild entfernen)

## 2. Backend — Diensttypen CRUD

- [x] 2.1 `ListTypes`-Handler in `internal/duties/handler.go` um vier neue Felder erweitern (SELECT + JSON-Response)
- [x] 2.2 `CreateType`- und `UpdateType`-Handler um die vier neuen Felder erweitern; Validierung: wenn `*_behavior='reduced'` muss entsprechende `*_variant_id` gesetzt sein, sonst HTTP 400

## 3. Backend — Slot-Generierungslogik

- [x] 3.1 Hilfsfunktion `sameDayGameCount(db, date, seasonID) int` anlegen: zählt alle Heimspiele am angegebenen Tag
- [x] 3.2 Hilfsfunktion `adjacentDayHasGames(db, date, seasonID, direction) bool` anlegen (`direction`: -1 für Vortag, +1 für Folgetag)
- [x] 3.3 Hilfsfunktion `effectiveDutyType(dt DutyType, isFirst, isLast bool, sameDayCount int, hasPrevDay, hasNextDay bool) (dutyTypeID int, applies_when string, skip bool)` anlegen: **berechnet** `applies_when`, wendet dann beide Verhaltensweisen orthogonal an
- [x] 3.4 `RegenerateSlots`-Handler anpassen: wendet `effectiveDutyType` pro Item an bevor Slot inseriert wird
- [x] 3.5 `PreviewSlots`-Handler um optionalen Query-Parameter `game_id` erweitern: wenn übergeben, wird Spieltag-Kontext berechnet und beide Verhaltensweisen angewendet

## 4. Frontend — AdminDutyTypesPage

- [x] 4.1 API-Typ `DutyType` um vier neue Felder erweitern
- [x] 4.2 Formular (Neu anlegen + Inline-Edit) um vier neue Felder ergänzen: zwei Dropdowns für `same_day_behavior`/`adjacent_day_behavior`, zwei Dropdowns für `same_day_variant_id`/`adjacent_day_variant_id` (nur sichtbar wenn `*_behavior == 'reduced'`)
- [x] 4.3 Diensttyp-Liste zeigt `same_day_behavior` und `adjacent_day_behavior` an wenn nicht 'normal' (als Badge)

## 5. Frontend — Spielplan Slot-Regenerierung

- [x] 5.1 `PreviewSlots`-Aufruf in `SpieltagDetailPage` übergibt `game_id` damit der Backend-Preview den korrekten Kontext hat
- [x] 5.2 Preview-Ansicht vor Regenerate zeigt korrekte gefilterte Slots basierend auf beiden Verhaltensweisen
