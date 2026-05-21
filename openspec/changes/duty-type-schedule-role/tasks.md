## 1. Datenbank-Migration

- [x] 1.1 Migration `0NN_duty_type_schedule_role.up.sql` anlegen: `ALTER TABLE duty_types ADD COLUMN consecutive_behavior TEXT NOT NULL DEFAULT 'normal' CHECK (consecutive_behavior IN ('normal','skip','reduced'))`, `ADD COLUMN consecutive_variant_id INTEGER REFERENCES duty_types(id)` (NICHT `applies_when` — wird berechnet)
- [x] 1.2 Passende `.down.sql` anlegen (Spalten via Tabellen-Rebuild entfernen)

## 2. Backend — Diensttypen CRUD

- [x] 2.1 `ListTypes`-Handler in `internal/duties/handler.go` um `consecutive_behavior`, `consecutive_variant_id` erweitern (SELECT + JSON-Response)
- [x] 2.2 `CreateType`- und `UpdateType`-Handler um die zwei neuen Felder erweitern; Validierung: wenn `consecutive_behavior='reduced'` muss `consecutive_variant_id` gesetzt sein, sonst HTTP 400

## 3. Backend — Slot-Generierungslogik

- [x] 3.1 Hilfsfunktion `sameDayGames(db, date, seasonID) []time.Time` anlegen: liefert alle Anpfiffzeiten von Heimspielen am angegebenen Tag in der Saison
- [x] 3.2 Hilfsfunktion `adjacentDayHasGames(db, date, seasonID, direction) bool` anlegen (`direction`: -1 für Vortag, +1 für Folgetag)
- [x] 3.3 Hilfsfunktion `effectiveDutyType(dt DutyType, isFirst, isLast bool, prevDay, nextDay bool) (dutyTypeID int, applies_when string, skip bool)` anlegen: **berechnet** `applies_when` basierend auf isFirst/isLast/prevDay/nextDay, wendet dann `consecutive_behavior` an
- [x] 3.4 `RegenerateSlots`-Handler anpassen: lädt Template-Items selbst (statt vom Frontend übergeben) — oder nimmt `game_id` als Kontext und wendet `effectiveDutyType` pro Item an bevor Slot inseriert wird
- [x] 3.5 `PreviewSlots`-Handler um optionalen Query-Parameter `game_id` erweitern: wenn übergeben, wird derselbe Spieltag-Kontext berechnet und `applies_when`/`consecutive_behavior` angewendet

## 4. Frontend — AdminDutyTypesPage

- [x] 4.1 API-Typ `DutyType` um `consecutive_behavior`, `consecutive_variant_id` erweitern
- [x] 4.2 Formular (Neu anlegen + Inline-Edit) um zwei neue Felder ergänzen: `consecutive_behavior`-Dropdown, `consecutive_variant_id`-Dropdown (nur sichtbar wenn `consecutive_behavior == 'reduced'`)
- [x] 4.3 Diensttyp-Liste zeigt `consecutive_behavior` an wenn nicht 'normal' (als Badge oder Text)

## 5. Frontend — Spielplan Slot-Regenerierung

- [x] 5.1 `RegenerateSlots`-Aufruf in `AdminSpielplanPage` (o.ä.) übergibt `game_id` damit der Backend-Preview den korrekten Kontext hat
- [x] 5.2 Preview-Ansicht vor Regenerate zeigt korrekte gefilterte Slots (kein Aufbau beim zweiten Spiel des Tages sichtbar)
