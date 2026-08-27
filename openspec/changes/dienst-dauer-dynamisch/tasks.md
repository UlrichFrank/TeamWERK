## 1. Schema

- [x] 1.1 `internal/db/migrations/053_dienst_dauer_dynamisch.up.sql`: `duration_mode` (CHECK `absolut|dynamisch`, Default `absolut`), `end_anchor` (CHECK `start|end`, Default `end`), `end_offset_minutes` (Default 0) — je auf `duty_types` und `game_template_items`
- [x] 1.2 `.down.sql`: alle sechs Spalten droppen
- [x] 1.3 `make migrate-up`; prüfen, dass alle Bestandszeilen auf `absolut` stehen (kein Verhaltenswechsel durch die Migration)

## 2. Regen-Engine (`internal/games/regen.go`)

- [x] 2.1 Anker-Auflösung aus `regenGameItems` (Z. 883) in `resolveAnchorTime(anchor string, offsetMinutes int, g dayGame, durationMins int) string` ziehen; Start-Berechnung darauf umstellen (verhaltensneutral)
- [x] 2.2 `templateItemRow` um `DurationMode`, `EndAnchor`, `EndOffsetMinutes`; beide Loader (`loadTemplateItems` in handler.go, `loadTemplateItemsTx` in regen.go) erweitern
- [x] 2.3 In `regenGameItems`: bei `DurationMode=='dynamisch'` Endzeit über `resolveAnchorTime` bestimmen, Dauer als Differenz in Stunden; bei Differenz ≤ 0 auf `HoursValue` zurückfallen
- [x] 2.4 Zusammenspiel mit der Varianten-Auflösung klären: bei `reduced` gewinnt weiterhin der Varianten-Typ — auch dessen Modus und End-Felder, nicht nur `hours_value` (`dienst-dauer` Decision 3 konsequent weiterführen)
- [x] 2.5 Go-Test `TestRegen_AbsoluterModusUnveraendert` — Regressionsschutz für den Refactor aus 2.1
- [x] 2.6 Go-Test `TestRegen_DynamischeDauerFolgtSpieldauer` — zwei Altersklassen, eine Vorlage, unterschiedliche Slot-Dauern
- [x] 2.7 Go-Test `TestRegen_DynamischeDauerNutztEndTime` — gepflegte `games.end_time` hat Vorrang
- [x] 2.8 Go-Test `TestRegen_DynamischeDauerAnkerStart` — Halbzeit-Dienst, Dauer unabhängig von der Spieldauer
- [x] 2.9 Go-Test `TestRegen_DynamischeDauerFaelltAufAbsoluteZurueck` — Ende vor Start → Slot entsteht mit `hours_value`

## 3. Diensttyp-Routen (`internal/duties/handler.go`)

- [x] 3.1 `ListTypes` liefert `duration_mode`, `end_anchor`, `end_offset_minutes`
- [x] 3.2 `CreateType`/`UpdateType` nehmen die drei Felder entgegen; fehlende Felder behalten die Defaults bzw. den Bestand
- [x] 3.3 Validierung: `duration_mode` ∈ {`absolut`,`dynamisch`}, `end_anchor` ∈ {`start`,`end`} → sonst HTTP 400 **vor** dem Schreiben
- [x] 3.4 Go-Test `TestUpdateType_SpeichertDauerModus` (Happy-Path)
- [x] 3.5 Go-Test `TestUpdateType_UngueltigerDauerModus_400` (Fehlerfall, nichts persistiert)

## 4. Vorlagen-CRUD (`internal/games/handler.go`)

- [x] 4.1 `templateItem` (Request/Response) um die drei Felder; `scanTemplateItems` liefert sie mit
- [x] 4.2 `UpdateTemplate`: Felder schreiben; fehlende Felder erben vom Diensttyp (dasselbe Pointer-Muster wie `hours_value`)
- [x] 4.3 Validierung analog 3.3, ebenfalls vor `tx.BeginTx`
- [x] 4.4 Go-Test `TestUpdateTemplate_SpeichertDauerModusJeItem`
- [x] 4.5 Go-Test: ungültiger Modus in einer Zeile → 400, keine Teil-Persistenz

## 5. Diensttyp-Maske (`web/src/pages/AdminDutyTypesPage.tsx`)

- [ ] 5.1 Radio-Umschalter „Dauer: absolut / dynamisch"; im dynamischen Modus End-Anker-Dropdown + Versatz-Feld (`OffsetInput`) zusätzlich zum bestehenden Dauer-Feld
- [ ] 5.2 Das absolute Dauer-Feld bleibt im dynamischen Modus sichtbar und beschriftet als Rückfall (design.md Decision 3)
- [ ] 5.3 Beispielrechnung unter der Maske („bei 60 min Spieldauer: 09:30–11:45"), damit ein negativer Versatz auffällt
- [ ] 5.4 Vitest: Umschalten zeigt/verbirgt die End-Felder; gespeicherte Payload trägt alle drei Felder

## 6. Vorlagen-Editor (`web/src/pages/AdminDutyTemplatesPage.tsx`)

- [ ] 6.1 Dieselbe Modus-Maske je Vorlagen-Zeile
- [ ] 6.2 Copy-on-pick beim Diensttyp-Auswählen um die drei Felder erweitern
- [ ] 6.3 `newItem()` um Defaults ergänzen
- [ ] 6.4 Vitest: Diensttyp-Auswahl überträgt auch den Modus

## 7. Auffrischen mitziehen (`web/src/lib/dutyTemplateItems.ts`)

- [ ] 7.1 `refreshItemsFromDutyTypes` um `duration_mode`, `end_anchor`, `end_offset_minutes` erweitern — sonst frischt „Aus Diensttypen auffrischen" den Modus nicht mit auf und hinterlässt eine Zeile aus zwei Ständen (design.md, Risks)
- [ ] 7.2 Vitest: ein Typ, der von `absolut` auf `dynamisch` gewechselt ist, wird beim Auffrischen vollständig übernommen

## 8. Anlege-Modal (`web/src/components/SpieltagDetailModal.tsx`)

- [ ] 8.1 Vorbelegung der Dauer aus einer dynamischen Typ-Definition gegen den konkreten Termin berechnen (Anker + Versätze gegen `game.time` / `game.end_time`)
- [ ] 8.2 Kein Modus-Umschalter im Modal — der Slot ist `is_custom=1` und bleibt absolut (spec.md, Requirement 3)
- [ ] 8.3 Vitest: dynamischer Typ ergibt eine ausgerechnete Dauer, die danach frei editierbar ist

## 9. Abschluss

- [ ] 9.1 `make test`, `make lint`, `pnpm -C web build/test/lint`
- [ ] 9.2 `/verify-change`: Route→Tests, Broadcast/useLiveUpdates, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`
- [ ] 9.3 Erwägen: `CHECK(hours_value > 0)` auf `duty_slots`/`game_template_items` nachrüsten (offener Punkt aus `dienst-dauer`, design.md-Korrektur)
