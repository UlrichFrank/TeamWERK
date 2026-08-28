## 1. Schema

- [x] 1.1 `internal/db/migrations/052_dienst_dauer.up.sql`: `hours_value REAL NOT NULL DEFAULT 1.0` auf `game_template_items` und `duty_slots`; Backfill je Tabelle per korreliertem Subselect aus `duty_types` (Copy-on-pick rückwirkend)
- [x] 1.2 `052_dienst_dauer.down.sql`: beide Spalten droppen (`ALTER TABLE … DROP COLUMN`, wie `047_member_chat_visible.down.sql`)
- [x] 1.3 `make migrate-up` lokal, danach stichprobenartig prüfen, dass keine Zeile auf dem `1.0`-Default steht, die einen abweichenden Typ-Wert hat

## 2. Regen-Engine (`internal/games/regen.go`)

- [x] 2.1 `templateItemRow` um `HoursValue float64` erweitern; `loadTemplateItemsTx` (Z. 1169) selektiert und scannt `gti.hours_value`
- [x] 2.2 `lookupDutyTypeNameTx` (Z. 906) zu einem Helfer erweitern, der Name **und** `hours_value` der Variante in einem Query liefert
- [x] 2.3 In `regenGameItems`: bei `isReduce` die Dauer des Varianten-Typs verwenden, sonst `it.HoursValue` (Decision 3)
- [x] 2.4 `insertOne` (Z. 936) schreibt `hours_value` in den `INSERT INTO duty_slots`
- [x] 2.5 Go-Test `TestRegen_SlotErbtHoursValueAusVorlage`: Vorlagen-Zeile mit abweichender Dauer → Slot trägt den Vorlagen-Wert, nicht den inzwischen geänderten Typ-Wert
- [x] 2.6 Go-Test `TestRegen_ReducedVarianteBestimmtDauer`: Mehrfachspieltag mit `same_day_behavior='reduced'` → Slot trägt die Dauer des Varianten-Typs; Uhrzeit und `slots_total` stammen weiter aus der Vorlagen-Zeile
- [x] 2.7 Go-Test `TestRegen_DauerAenderungErhaeltZusagen`: Zusage anlegen, Vorlagen-Dauer ändern, Regen → Zusage wiederhergestellt, kein „entfernt"-Intent in `buildNotificationIntents`

## 3. Vorlagen-CRUD (`internal/games/handler.go`)

- [x] 3.1 Template-Item-Insert (Z. 2020) trägt `hours_value`; Request-Struct des Vorlagen-`PUT` um das Feld erweitern
- [x] 3.2 Lesende Vorlagen-Queries (Z. 476, Z. 1793, Z. 1821) liefern `hours_value` je Item mit
- [x] 3.3 Go-Test `TestUpdateTemplate_SpeichertHoursValueJeItem`: Zeile behält ihre eigene Dauer, unabhängig vom aktuellen Typ-Wert
- [x] 3.4 Go-Test (Fehlerfall): Vorlagen-`PUT` mit `hours_value <= 0` in einer Zeile → HTTP 400, keine Teil-Persistenz (Delete-then-Insert läuft in einer Tx — vor dem ersten Schreibvorgang validieren)

## 4. Dienst-Konto (`internal/games/handler.go`)

- [x] 4.1 `ist`-Neuberechnung (Z. 1532): `SUM(dt.hours_value)` → `SUM(ds.hours_value)`, JOIN auf `duty_types` entfällt
- [x] 4.2 Go-Test `TestDeleteGame_IstNutztSlotDauer`: Slot mit überschriebener Dauer + `fulfilled`-Zuweisung, Termin löschen → `ist` sinkt um die **Slot**-Dauer, nicht um die des Typs

## 5. Dienst-Routen (`internal/duties/handler.go`)

- [x] 5.1 `Board` (Z. 802) und `ListSlots` (Z. 524) liefern `hours_value` je Slot im Response-Struct
- [x] 5.2 `CreateSlot` (Z. 556) nimmt `hours_value` entgegen; fehlt es, Dauer aus `duty_types` des `duty_type_id` einsetzen
- [x] 5.3 `UpdateSlot` (Z. 600) nimmt `hours_value` entgegen und schreibt es mit (`is_custom=1` bleibt unverändert bedingungslos)
- [x] 5.4 Beide Routen lehnen `hours_value <= 0` mit HTTP 400 ab, **bevor** geschrieben wird
- [x] 5.5 Go-Test `TestBoard_LiefertHoursValueJeSlot` (Happy-Path)
- [x] 5.6 Go-Tests `TestCreateSlot_UebernimmtHoursValue`, `TestCreateSlot_OhneHoursValueErbtVomTyp`, `TestUpdateSlot_AendertHoursValue`
- [x] 5.7 Go-Test `TestUpdateSlot_HoursValueNullOderNegativ_400` (Fehlerfall, beide Routen)

## 6. Frontend — geteilte Helfer

- [x] 6.1 `hoursToDisplay` und `parseHoursInput` aus `web/src/pages/AdminDutyTypesPage.tsx` (Z. 40–54) nach `web/src/lib/duration.ts` verschieben, Aufrufstelle umstellen
- [x] 6.2 `formatTimeSpan(eventTime: string | null, hours: number): string` ergänzen — Halbgeviertstrich, kein „Uhr", Mitternachtsüberlauf per Modulo 24 h, `null`-Startzeit → Platzhalter
- [x] 6.3 Vitest für `formatTimeSpan`: Normalfall, Mitternachtsüberlauf (`23:30` + 1 h), fehlende Startzeit, krumme Dauer (`0.3333…` → `20min`)

## 7. Frontend — Anzeige

- [x] 7.1 `web/src/components/DutySlotList.tsx` (Z. 164): `{s.event_time || '—'}` → `formatTimeSpan(...)`; Slot-Typ um `hours_value: number` erweitern
- [x] 7.2 Vitest: Slot mit Dauer rendert die Spanne, Slot ohne `event_time` rendert weiter den Platzhalter
- [x] 7.3 Prüfen, dass `web/src/pages/DutyPage.tsx` (Z. 382) unverändert bleibt — das ist die Anstoßzeit des Spiels, kein Slot

## 8. Frontend — Bearbeiten und Anlegen

- [x] 8.1 `web/src/components/SpieltagDetailModal.tsx`: Dauer-Feld im „Dienst bearbeiten"-Modal (Z. 340 ff.), vorbelegt aus `editSlot.hours_value`, mit `hours-presets`-Datalist wie auf `/diensttypen`
- [x] 8.2 Dauer-Feld im „Dienst hinzufügen"-Modal (Z. 284 ff.)
- [x] 8.3 Beim Auswählen des Diensttyps im Add-Modal (Z. 288) zusätzlich zu den Zielgruppen Dauer **und** Uhrzeit vorbelegen — Uhrzeit aus `default_anchor` + `default_offset_minutes` gegen die Zeit des Termins
- [x] 8.4 `hours_value` in beiden Requests mitsenden (`handleAddSlot` Z. 138 ff., `handleEditSlot`)
- [x] 8.5 Vitest: Diensttyp-Auswahl im Add-Modal füllt Dauer und Uhrzeit; manuelle Änderung überschreibt die Vorbelegung und überlebt einen erneuten Render

## 9. Frontend — Vorlagen-Editor

- [x] 9.1 `web/src/pages/AdminDutyTemplateDetailPage.tsx`: `hours_value` in den Copy-on-pick-Spread beider Auswahl-Handler (Z. 247 und Z. 340) aufnehmen
- [x] 9.2 Dauer-Feld je Zeile in Desktop- und Mobile-Variante der Item-Maske
- [x] 9.3 `emptyItem()` (Z. 61) um einen Default ergänzen
- [x] 9.4 Vitest: Diensttyp-Auswahl füllt die Dauer der Zeile; abweichend gesetzte Dauer überlebt die Auswahl eines *anderen* Feldes

## 10. Abschluss

- [x] 10.1 `/diensttypen` einmal durchsehen: sind die gepflegten `hours_value` als Uhrzeit-Spannen plausibel? (Risiko aus design.md — Werte, die bisher nie als Zeit gelesen wurden)
- [x] 10.2 `make test`, `make lint`, `pnpm -C web build/test/lint`
- [x] 10.3 `/verify-change`: Route→Tests, Mutation→`Broadcast`/`useLiveUpdates`, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`
- [x] 10.4 Folge-Change festhalten: `Fulfill` bucht `duty_accounts.ist` nicht (design.md, Decision 4)
