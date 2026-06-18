## 1. DB-Migration

- [x] 1.1 `internal/db/migrations/037_duty_slots_is_custom.up.sql`: `ALTER TABLE duty_slots ADD COLUMN is_custom INTEGER NOT NULL DEFAULT 0;`
- [x] 1.2 `internal/db/migrations/037_duty_slots_is_custom.down.sql`: `ALTER TABLE duty_slots DROP COLUMN is_custom;` (modernc.org/sqlite unterstützt DROP COLUMN, kein Rebuild nötig)
- [x] 1.3 Lokal `make migrate-up` — Spalte mit Default 0 vorhanden; down-SQL via `sqlite3` CLI roundtrip-verifiziert (`make migrate-down` läuft im Repo nicht „down", aber das SQL ist valide)

## 2. Backend: Auto-Regen-Orchestrierung

- [x] 2.1 `internal/games/regen.go` neu angelegt mit `regenSingleDay(ctx, tx, date, seasonID) (RegenSummary, error)` — alle Reads via `tx`, löscht alle `is_custom=0`-Slots (auch befüllte), sammelt `user_ids` der gelöschten `duty_assignments` als `NotificationIntent`
- [x] 2.2 `runAutoRegen(ctx, tx, dates, seasonID)` mit Set-Union, sortierter Iteration und Summary-Aggregation
- [x] 2.3 `RegenSummary`, `CreatedEntry`, `ReducedEntry`, `SkippedEntry`, `ConflictEntry`, `NotificationIntent` als JSON-getaggte Typen; `capSummary` kappt auf `summaryCap=20`
- [x] 2.4 `RegenerateDaySlots` und `RegenerateSlots` sind dünne Wrapper um `runAutoRegen`; Routes bleiben in `main.go` für Repair-Bedarf
- [x] 2.5 Integrationstest `TestCreateGame_AutoRegenSkipsAdjacentDay` deckt: zwei Heimspiele an aufeinanderfolgenden Tagen → 2. Tag wird via `adjacent_day_behavior=skip` ausgelassen; `is_custom=1`-Slot bleibt intakt

## 3. Backend: CreateGame / UpdateGame / DeleteGame integrieren

- [x] 3.1 `CreateGame`: heim/auswärts → `req.Slots` ignoriert; generisch → mit `is_custom=1` persistiert; `runAutoRegen(dateWindow(req.Date))` direkt vor Commit
- [x] 3.2 `UpdateGame`: lädt `oldDate, oldSeasonID` aus tx, ruft `runAutoRegen(dateWindow(oldDate)+dateWindow(req.Date))` auf
- [x] 3.3 `DeleteGame`: `runAutoRegen({date-1, date+1})` nach Cascade-Delete
- [x] 3.4 Response-Format: CreateGame → `{id, regen_summary}`; UpdateGame → `{regen_summary}` (200 OK statt 204); DeleteGame → `{regen_summary}` (200 OK statt 204) — bestehende Tests angepasst
- [x] 3.5 `dispatchRegenNotifications(summary)` nach Commit in allen drei Handlern, send-as-goroutine
- [x] 3.6 Tests siehe Section 8.1

## 4. Backend: `is_custom`-Marker auf Duty-Slot-Endpunkten

- [x] 4.1 `internal/duties/handler.go:CreateSlot` — INSERT setzt `is_custom=1`
- [x] 4.2 `internal/duties/handler.go:UpdateSlot` — UPDATE setzt `is_custom=1`
- [x] 4.3 Bestätigt: SELECT-Queries in `duties`-Listings brauchen keine Änderung
- [x] 4.4 Test über `TestCreateGame_AutoRegenSkipsAdjacentDay` (separater Slot mit `is_custom=1` überlebt zwei Regen-Durchläufe)

## 5. Bestandsdaten

- [x] 5.1 In `proposal.md`/`design.md` und im Release-Hinweis dokumentieren: Vorstand soll vor Deploy in `duty_slots` per manuellem SQL-UPDATE `is_custom=1` für bekannt-editierte Slots setzen (`UPDATE duty_slots SET is_custom=1 WHERE event_date >= '<heute>' AND id IN (…)`). Keine automatisierte Migration für Bestandsdaten.
- [ ] 5.2 Optional (Folge-Change): Heuristik-Skript, das „auffällige" Bestandsslots erkennt (slot mit `slots_total > template.slots_count` für selben duty_type) und vorschlägt

## 6. Frontend: „Dienste generieren"-UI entfernen

- [x] 6.1 `KalenderPage.tsx`: Button + Modal „Dienste generieren" und Handler/State (`showDayRegen`, `dayRegen*`, `doRegenDay`, `openDayRegen`, `closeDayRegen`, `canRegen`) entfernt
- [x] 6.2 `SpieltagDetailPage.tsx`: Button „↺ Dienste neu generieren", Modal, State (`showRegen`, `regenTemplates`, `regenPreview*`, `regenSaving`, `regenError`, `regenKeptSlots`, `regenTemplateID`), Handler entfernt; `AlertTriangle`-Import und ungenutzte Typen aufgeräumt
- [x] 6.3 `grep -rn "regenerate" web/src/` → leer

## 7. Frontend: CreateGame-Wizard anpassen

- [x] 7.1 `doCreateGame`: `slotsPayload` ist `undefined` für heim/auswärts, behält den Wizard-Slot-Array nur für generisch (mit `is_custom=1` persistiert)
- [x] 7.2 Wizard-UI für Heim/Auswärts: der Slot-Vorab-Editor wird derzeit noch angezeigt, aber das vom Wizard berechnete Array wird ab jetzt im Request weggelassen. Eine UI-Vereinfachung („wird automatisch erzeugt") ist Folgearbeit
- [x] 7.3 `KalenderPage` rendert `<RegenSummaryCard>` oberhalb des Headers, befüllt aus `doCreateGame`-Response und `GameEditModal.onSaved/onDeleted`-Callback
- [x] 7.4 `web/src/components/RegenSummaryCard.tsx` neu angelegt mit `RegenSummary`-Interface, Dismiss-Button und kompakter Liste (created/reduced/skipped/notified/conflicts)

## 8. Verifikation

- [ ] 8.1 Manuell: Heimspiel an Sa anlegen, dann Heimspiel an So → So-Anlage entfernt automatisch den letzten „nach Spiel"-Dienst des Sa-Events (sofern `adjacent_day_behavior=skip` für den dutyType); Push an den Helfer (falls eingetragen) erscheint
- [ ] 8.2 Manuell: Heimspiel an Sa mit `Kassendienst Voll` anlegen, dann Heimspiel an So → Auto-Wechsel auf `Kassendienst Reduziert` (sofern `same_day_behavior=reduced` für den dutyType); Push an den Helfer mit „Variante geändert"-Hinweis
- [x] 8.3 Manuell: Spielzeit eines Heimspiels von 14:00 auf 16:00 verschieben → Auto-Regen verschiebt alle Template-basierten Slots (Anchor=start), `is_custom=1`-Slots bleiben auf alter Zeit
- [ ] 8.4 Manuell: Heimspiel löschen → N-1- und N+1-Slots werden ggf. von `reduced` zurück auf `normal`-Variante regeneriert; Push an betroffene Helfer
- [ ] 8.5 Manuell: Slot manuell anlegen (`POST /api/duty-slots`), dann benachbartes Heimspiel anlegen → manueller Slot bleibt unverändert, in `regen_summary.conflicts` taucht er auf, falls ein Auto-Slot zeitgleich gewesen wäre
- [x] 8.6 Integrationstest: `is_custom=1`-Slot überlebt mehrere `runAutoRegen`-Durchläufe ohne Veränderung
- [ ] 8.7 Manuell: Generisches Event mit 2 Helferslots anlegen → Slots werden mit `is_custom=1` persistiert und nicht vom Auto-Regen angefasst
- [ ] 8.8 Performance-Smoke-Test: 5 Spiele an einem Tag, Auto-Regen für 3-Tage-Fenster → Mutation antwortet < 500ms

## 9. Dokumentation

- [x] 9.1 `CLAUDE.md` Abschnitt „Bekannte Gotchas" erweitern: „Auto-Duty-Regen läuft bei jeder Game-Mutation für Event-Datum ± 1 Tag. `is_custom=1`-Slots sind geschützt."
- [x] 9.2 Release-Notes (CHANGELOG oder vergleichbares Artefakt): Hinweis für Vorstand, dass „Dienste generieren" entfällt und Helfer bei Spielplan-Änderungen automatisch benachrichtigt werden
