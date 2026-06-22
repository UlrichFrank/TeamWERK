## 1. Backend — Migration

- [x] 1.1 `internal/db/migrations/006_template_id_backfill.up.sql` schreiben: `UPDATE games SET template_id = (SELECT id FROM game_templates WHERE template_type = games.event_type ORDER BY id LIMIT 1) WHERE template_id IS NULL AND event_type IN ('heim','auswärts')`
- [x] 1.2 `internal/db/migrations/006_template_id_backfill.down.sql` schreiben (best-effort Reset; im SQL-Kommentar Limitation dokumentieren)
- [x] 1.3 `make migrate-up` lokal testen; mit Test-DB verifizieren *(über `testutil.NewDB` + `TestMigration006_BackfillsHeimAuswaertsNull` abgedeckt)*

## 2. Backend — `runAutoRegen` / `regen.go`

- [x] 2.1 In `regenSingleDay` den Zweig „`g.TemplateID.Valid == false`" auf `continue` umstellen — kein Fallback mehr
- [x] 2.2 `findTemplateForGameTx` löschen (vorher prüfen, ob noch andere Aufrufer existieren: `grep -rn findTemplateForGameTx internal/`)
- [x] 2.3 `effectiveEventDurationTx` Aufruf-Pfad prüfen: bei NULL-Template wird er nicht mehr erreicht — sicherstellen, dass kein toter Code zurückbleibt

## 3. Backend — `CreateGame`

- [x] 3.1 Validierung ergänzen: wenn `event_type='generisch'` UND `template_id != nil` → HTTP 400 mit Body `{"error":"template_id muss bei event_type=generisch null sein"}`
- [x] 3.2 Sicherstellen, dass `template_id=null` korrekt als SQL NULL persistiert wird (heute schon korrekt — Regression-Test ergänzen)

## 4. Backend — `UpdateGame`

- [x] 4.1 Request-Struct um `TemplateID` mit Tri-State-Semantik erweitern. *(via `json.RawMessage` — Feld fehlt → unverändert, `"null"` → NULL, Zahl → Wert)*
- [x] 4.2 `UPDATE games SET ...` um `template_id=?` ergänzen, **nur wenn `Set==true`**
- [x] 4.3 Wenn `Set==true && Valid==false` → `NULL`; wenn `Set==true && Valid==true` → Wert
- [x] 4.4 Validierung: wenn `event_type='generisch'` UND `template_id != null` → HTTP 400 (gleiche Regel wie Create)
- [x] 4.5 `runAutoRegen` läuft unverändert nach dem Update — Auto-Regen reagiert auf den neuen `template_id`-Wert

## 5. Tests Backend

- [x] 5.1 `TestCreateGame_HomeWithoutTemplate` (201, NULL, keine Auto-Slots)
- [x] 5.2 `TestCreateGame_GenericWithCustomSlots` (201, `is_custom=1`-Slots persistiert)
- [x] 5.3 `TestCreateGame_GenericWithTemplateRejected` (400)
- [x] 5.4 `TestUpdateGame_TemplateIDFieldOmitted_Preserves` (200, Wert unverändert)
- [x] 5.5 `TestUpdateGame_TemplateIDExplicitNull_SetsNull` (200, NULL, Slots weg)
- [x] 5.6 `TestUpdateGame_TemplateIDChange_RegeneratesSlots` (200, neue Slots aus neuem Template)
- [x] 5.7 `TestAutoRegen_NullTemplate_NoSlotsGenerated` (regen.go-Verhalten direkt)
- [x] 5.8 `TestMigration006_BackfillsHeimAuswaertsNull` (Migration up: NULL-Backfill korrekt)

## 6. Frontend — Anlege-Form (Game/Event im Kalender)

- [x] 6.1 Vorlage-Dropdown: erste Option `{ value: null, label: '— Keine Vorlage (keine Auto-Dienste) —' }`
- [x] 6.2 Optionen filtern: `templates.filter(t => t.template_type === eventType)`
- [x] 6.3 Default-Selektion: erste passende Vorlage falls vorhanden, sonst die Null-Option *(initialer State ist `null`; User wählt aktiv — bewusst keine Vorauswahl)*
- [x] 6.4 Bei `event_type=generisch`: Vorlage-Dropdown ausblenden ODER fix auf „Keine Vorlage" gesperrt *(filteredTemplates für `generisch` → `[]`, nur Null-Option sichtbar)*
- [x] 6.5 „Ohne Dienste"-Toggle aus der UI entfernen
- [x] 6.6 Submit: `template_id: null` (nicht `undefined`) senden, wenn Null-Option gewählt — damit Backend NULL setzt

## 7. Frontend — Edit-Form

- [x] 7.1 Vorlage-Dropdown im Edit-Modal/-Form (heute fehlt es) hinzufügen, gleiche Optionen wie Anlege-Form
- [x] 7.2 Aktuellen `template_id`-Wert aus Game-Detail-Response vorbelegen *(List-Endpoint um `template_id` erweitert)*
- [x] 7.3 Submit sendet `template_id` immer mit (Zahl oder explizit `null`)
- [x] 7.4 SSE: nach Erfolg `useLiveUpdates`-Reload greift bereits (kein Extra-Aufwand)

## 8. Frontend — `/dienstplan-vorlagen`

- [x] 8.1 Hinweistext „Vorlagen werden nur initial verwendet" / „nicht für Regenerierung persistiert" entfernen *(„Achtung: niedrigste ID wird verwendet"-Warnung und Pro-Row-Badge entfernt)*
- [ ] 8.2 Optional: pro Template Anzeige „X aktive Events nutzen diese Vorlage" (kann auch ein Folge-Change sein) — *als optional zurückgestellt*

## 9. Dokumentation

- [x] 9.1 `CLAUDE.md` — Abschnitt „Bekannte Gotchas" um Hinweis ergänzen, dass `template_id IS NULL` = keine Auto-Dienste bedeutet
- [ ] 9.2 OpenSpec-Proposal nach Apply archivieren *(separater Schritt durch User/`openspec archive`)*
