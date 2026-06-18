## Why

Heute existiert die skip/reduce-Logik für Dienste (`applyBehavior` in `internal/games/handler.go:248`) nur im manuellen Pfad „Dienste generieren". Die zwei Folgen:

1. **Beim Anlegen eines Heim-/Auswärts- oder generischen Events übernimmt `CreateGame` das vom Frontend gelieferte `slots[]`-Array 1:1.** Keine Nachbarschaftslogik, kein skip/reduce. Wer nach dem Speichern nicht zusätzlich auf „Dienste generieren" klickt, bekommt einen falschen Dienstplan — mehr Helferslots als nötig, oder die falsche Variante (z.B. „Kassendienst Voll" statt „Kassendienst Reduziert").
2. **Skip/reduce sind nachbarschaftsabhängig.** Ein neues Heimspiel an Tag N verändert nicht nur die eigenen Dienste, sondern auch die von N-1 und N+1 (über `hasPrevDay`/`hasNextDay` in `loadSameDayContext`). Heute werden diese Nachbar-Slots erst aktualisiert, wenn jemand für N-1 oder N+1 erneut „Dienste generieren" anstößt — sie veralten still.

Der Vorstand erlebt das als versteckte Pflichtaktion: Spielplan ändern → daran denken, dass man jetzt für drei Tage manuell regenerieren muss. Diese kognitive Last gehört ins Backend.

## What Changes

- **CreateGame / UpdateGame / DeleteGame im `games`-Package leiten die Dienst-Slots eines Heim- oder Auswärtsspiels autoritativ aus dem Template + Nachbarschaftskontext ab.** Das vom Frontend gelieferte `slots[]`-Array entfällt für `event_type ∈ {heim, auswärts}` — der Wizard schickt nur noch Event-Metadaten und `template_id`.
- **Nach jeder Mutation eines Heim-/Auswärtsspiels regeneriert das Backend implizit das Drei-Tage-Fenster** (Event-Datum ± 1 Tag) für alle Spiele und generischen Events, deren Slots aus Templates stammen. Bei Update mit Datums-/Zeit-/Heim-Auswärts-Wechsel wird das Drei-Tage-Fenster sowohl am alten als auch am neuen Datum verarbeitet.
- **Befüllte Slots verlieren ihren Sonderschutz.** Wenn skip/reduce einen befüllten Slot trifft, wird er gelöscht und der eingetragene Helfer per `notify.Send(..., "duties", ...)` informiert („Dein Dienst zum {Event} am {Datum} wurde aufgrund einer Spielplanänderung angepasst"). Bei `reduced`-Variante wird der Slot ebenfalls neu angelegt (anderer `duty_type_id`), der Helfer aber nicht automatisch übernommen — er muss neu eintragen.
- **Inline-bearbeitete Slots werden geschont.** Neue Spalte `duty_slots.is_custom INTEGER DEFAULT 0`. Wenn ein Vorstand/Trainer einen Slot manuell über `POST /api/duty-slots` oder `PUT /api/duty-slots/{id}` anlegt/ändert, wird `is_custom=1` gesetzt. Auto-Regen löscht nur Slots mit `is_custom=0`.
- **Generische Events bleiben Helfer-manuell.** Sie haben kein Template, der Wizard akzeptiert weiterhin `slots[]` und persistiert sie mit `is_custom=1`. Der Auto-Regen läuft trotzdem im Drei-Tage-Fenster — er kann generische Events „beerben", wenn Adjacency-Effekte greifen (z.B. wenn ein generic Event den einzigen Slot an Tag N-1 stellt), aber er schont sie qua `is_custom=1`.
- **Der Knopf „Dienste generieren" entfällt** auf `/kalender` (`KalenderPage.tsx:893,909`) und `/kalender/{id}` (`SpieltagDetailPage.tsx:267`). Die Backend-Endpunkte `POST /api/kalender/regenerate-day` und `POST /api/kalender/{id}/regenerate` bleiben bestehen (als interner Reuse für die Auto-Regen-Logik und für künftige „Optimieren"-Repair-Funktion).
- **Mutation-Response liefert Änderungsbericht.** `POST /api/admin/games` und `PUT /api/admin/games/{id}` antworten mit einem `regen_summary`-Objekt:
  ```json
  {
    "id": 42,
    "regen_summary": {
      "created": [{"date":"2026-06-13","duty_type":"Kassendienst","count":2}],
      "reduced": [{"date":"2026-06-12","from":"Kassendienst Voll","to":"Kassendienst Reduziert","count":1}],
      "skipped": [{"date":"2026-06-14","duty_type":"Hallenaufsicht"}],
      "notified_users": [17, 23],
      "conflicts": []
    }
  }
  ```
  Das Frontend rendert die Liste als Toast/Card im Event-Detail nach Save: „Folgendes hat sich geändert: …".

## Capabilities

### New Capabilities

Keine neuen Capabilities. Die Logik bleibt im bestehenden `games`- und `duties`-Capability-Scope.

### Modified Capabilities

- `games` — `CreateGame`/`UpdateGame`/`DeleteGame` triggern implizite Drei-Tage-Fenster-Regeneration; Request-Schema von `POST` und `PUT` ändert sich (Slot-Array entfällt für Heim/Auswärts); Response liefert `regen_summary`.
- `duties` — Auto-Regen darf befüllte Slots löschen, wenn skip/reduce greift; betroffene Helfer werden notifiziert; neuer Slot-Marker `is_custom` schont manuelle Slot-Edits.

## Impact

- **Migration 037** (`internal/db/migrations/037_duty_slots_is_custom.up.sql` + `.down.sql`): `ALTER TABLE duty_slots ADD COLUMN is_custom INTEGER NOT NULL DEFAULT 0`. Bestand wird mit Default-0 migriert (alle existierenden Slots stammen aus dem alten Pfad, der ggf. manuell editiert wurde — siehe design.md zur Bestandsbehandlung).
- **`internal/games/handler.go`** (~150 Zeilen Netto-Änderung):
  - `CreateGame`: Slot-Array für Heim/Auswärts ignorieren, stattdessen `runAutoRegen(ctx, tx, gameDate, seasonID)` aufrufen.
  - `UpdateGame`: bei Date/Time/IsHome/TemplateID-Änderung `runAutoRegen` für altes + neues Datum aufrufen.
  - `DeleteGame`: nach Cascade-Delete `runAutoRegen` für N-1 und N+1 aufrufen (das Event-Datum selbst ist leer ohne Slots).
  - Neue Helper `runAutoRegen(ctx, tx, date, seasonID) (RegenSummary, error)` und `regenSingleDay(ctx, tx, date, seasonID) (DaySummary, error)` — extrahieren die Logik aus heutigem `RegenerateDaySlots`.
  - `RegenSummary` als API-Returntyp + Aufruf von `notify.Send` für `notified_users`.
- **`internal/duties/handler.go`** (~20 Zeilen): bei `CreateSlot` und `UpdateSlot` (falls existiert) `is_custom=1` setzen.
- **`internal/games/handler.go` `RegenerateDaySlots`/`RegenerateSlots`** bleiben — die HTTP-Endpunkte werden zu dünnen Wrappern um `runAutoRegen`. Das Frontend nutzt sie nicht mehr, ein späterer „Optimieren"-Bulk kann darauf bauen.
- **Frontend:**
  - `web/src/pages/KalenderPage.tsx`: „Dienste generieren"-Card entfernen (Zeilen ~893, 909, plus `dayRegenDate`-State).
  - `web/src/pages/SpieltagDetailPage.tsx`: „Dienste generieren"-Modal entfernen (Zeilen ~267, plus `regenTemplateID`-State).
  - `web/src/pages/KalenderPage.tsx` Event-Anlage-Wizard: `slots[]` aus dem POST-Body entfernen für Heim/Auswärts; nach Save `regen_summary` rendern.
  - Neue Toast-Komponente oder Inline-Banner `<RegenSummaryCard summary={…} />` im Event-Detail.
- **Keine Änderung** an `duty_assignments`, `duty_accounts`, `game_template_items` oder `duty_types`.
