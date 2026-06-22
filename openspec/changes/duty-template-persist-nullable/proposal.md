## Why

Die Wahl der Dienstvorlage beim Event-Anlegen wird heute zwar in `games.template_id` geschrieben, aber das Verhalten ist auf mehreren Ebenen kaputt oder unklar:

1. **`PUT /api/games/{id}` lässt `template_id` aus** — einmal angelegt, kann die Vorlage nicht mehr geändert werden, ohne das Event zu löschen und neu anzulegen.
2. **Fallback bei `template_id IS NULL` ist „Vorlage mit der kleinsten ID"** (`findTemplateForGameTx`: `ORDER BY id LIMIT 1`). Das ist nicht deterministisch sinnvoll und macht die explizite Wahl unzuverlässig wahrnehmbar.
3. **„Kein Template" ist keine echte Option** beim Anlegen. Stattdessen muss man `event_type='generisch'` („Ohne Dienste") wählen, was den Event-Typ mit der Slot-Quelle vermischt: ein Heimspiel ohne Auto-Dienste ist heute nicht ausdrückbar.
4. Die UI `/dienstplan-vorlagen` suggeriert, dass Vorlagen „nur initial verwendet" werden — was funktional stimmt, weil der Edit-Pfad die Wahl nicht erhalten kann.

## What Changes

- `games.template_id` wird **persistente, frei änderbare Slot-Quelle** pro Event (Wert oder `NULL`).
- `PUT /api/admin/games/{id}` akzeptiert `template_id` (Zahl oder `null`) und schreibt den Wert; bei fehlendem Feld wird der bestehende Wert beibehalten (Partial-Update für dieses eine Feld, damit Bestands-Clients nicht versehentlich NULL setzen).
- `runAutoRegen` interpretiert `template_id IS NULL` als **„keine Auto-Dienste für dieses Event"** für alle `event_type`-Werte. Der ID-basierte Fallback `findTemplateForGameTx` entfällt; die Funktion selbst kann gelöscht werden, sofern keine anderen Aufrufer existieren.
- Frontend-Anlege- und Edit-Form (`/kalender`): Vorlage-Dropdown filtert nach `event_type` (`heim`/`auswärts`/`generisch`) und enthält die explizite Option **„— Keine Vorlage (keine Auto-Dienste) —"** (Wert `null`).
- Das separate „Ohne Dienste"-Toggle (heute via `event_type='generisch'` erzwungen) wird **abgeschafft**. Wer keine Auto-Dienste will, wählt „Keine Vorlage". `event_type='generisch'` bleibt erhalten, bedeutet aber wieder nur den Event-**Typ** (Turnier, Sondertermin) — nicht den Slot-Modus.
- Frontend `/dienstplan-vorlagen`: Hinweistext „nur initial verwendet" entfernen; optional Anzeige „X Events nutzen diese Vorlage".

**Bestehende Events:**
- Events mit `event_type='generisch'` UND vorhandenen `is_custom=1`-Slots bleiben unverändert; sie hatten ohnehin nie eine Template-Quelle.
- Events mit `template_id IS NULL` UND `event_type IN ('heim','auswärts')` haben heute Auto-Slots aus dem niedrigsten-ID-Fallback. Nach dem Change verlieren sie diese Slots bei nächster Regeneration. Migration setzt `template_id` einmalig auf das Ergebnis des heutigen Fallbacks, um den Status quo zu konservieren.

## Capabilities

### Modified Capabilities

- `games`: Vorlage-Auswahl pro Event ist explizit, optional (`NULL` = keine Auto-Dienste), beim Edit änderbar; ID-basierter Fallback entfällt.

## Impact

- `internal/games/handler.go` — `CreateGame` (Validierung), `UpdateGame` (Feld `template_id` aufnehmen, Partial-Update)
- `internal/games/regen.go` — `regenSingleDay` Zweig „NULL = nichts tun", `findTemplateForGameTx` entfernen
- `internal/db/migrations/006_template_id_backfill.{up,down}.sql` — Bestands-Events einmalig auf den heutigen Fallback-Wert setzen, damit das Verhalten nicht still kippt
- `web/src/pages/KalenderPage.tsx` (oder das verwendete Game-Form-Modal) — Dropdown, „Keine Vorlage"-Option, „Ohne Dienste"-Toggle entfernen
- `web/src/pages/AdminDutyTemplatesPage.tsx` — Hinweistext entfernen
- Tests: `internal/games/handler_test.go`, `internal/games/regen_test.go` falls vorhanden — siehe Test-Anforderungen in `design.md`
