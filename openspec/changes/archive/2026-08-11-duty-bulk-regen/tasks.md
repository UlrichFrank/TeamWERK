## 1. Engine — Ausnahme-Menge durchreichen

- [x] 1.1 `internal/games/regen.go`: `runAutoRegen` und `regenSingleDay` um einen Parameter `skip map[int]bool` (Spiel-IDs) erweitern; bestehende Aufrufer (`handler.go`, `h4aimport_handler.go`) mit `nil` versorgen
- [x] 1.2 `loadDayGames` filtert die übersprungenen Spiele aus der Mutations-Menge heraus; `loadSameDayContextTx` bleibt **unverändert** (ausgenommene Termine bleiben Kontext für `allGameTimes`/`hasPrevDay`/`hasNextDay` — siehe design.md §2)
- [x] 1.3 Test `TestRegen_AusgenommenesSpielBleibtKontext`: zwei Spiele am selben Tag, eines ausgenommen → Slot-IDs des ausgenommenen unverändert **und** das einbezogene erhält dieselbe `same_day_behavior`-Reduktion wie ohne Ausnahme
- [x] 1.4 Test `TestRegen_HeimspielKettenKontextUnabhaengigVomZustand`: (a) zwei aufeinanderfolgende Heimspiel-Tage im selben Lauf, Tag 2 bekommt `none` → Tag 1 erhält trotzdem seine `adjacent_day_behavior`-Reduktion, weil `hasNextDay` nur die Existenz des Heimspiels prüft, nicht seinen Dienst-Zustand (design.md §2); (b) Heimspiel am Vortag von `from` (außerhalb des gewählten Zeitraums, unangetastet) → erster Termin im Zeitraum erhält dieselbe `adjacent_day_behavior`-Reduktion wie beim Einzelspiel-Regen

## 2. Engine — Aufschlüsselung pro Spiel

- [x] 2.1 `RegenSummary` um `PerGame []GameDelta` erweitern (`game_id`, `created`, `deleted_auto`, `assignments_kept`, `assignments_lost`, `conflicts`); `regenGameItems` berechnet die Werte bereits, `regen.go:186-189` mergt sie heute nur sofort weg. `deleted_custom` bewusst **nicht** Teil von `GameDelta` — die Engine löscht nie `is_custom=1`-Slots, das übernimmt ausschließlich der Bulk-Regen-Handler (Section 5) beim `purge`-Zustand; er ergänzt `deleted_custom` beim Zusammensetzen der `rows`-Antwort.
- [x] 2.2 `capSummary` anpassen: `PerGame` bleibt **ungecappt** (die Zeilenliste ist das Produkt der Vorschau), die bestehenden Listen behalten `summaryCap = 20`; Verhalten im Kommentar begründet
- [x] 2.3 Test `TestRegen_PerGameZweiSpieleGetrenntGezaehlt`: Regen über zwei Spiele an einem Tag liefert zwei `PerGame`-Einträge mit korrekt getrennten Zahlen

## 3. Engine — Zuweisungs-Restore

- [x] 3.1 `snapshotDeletedSlots` (`regen.go`): Query um `da.id`, `da.status`, `da.cash_amount`, `da.fulfilled_at` erweitert; `deletedSlot` trägt statt `UserIDs []int` eine nach `da.id` sortierte Liste vollständiger Zuweisungen (`deletedAssignment`)
- [x] 3.2 Restore nach den Inserts in `regenSingleDay` (`restoreAssignments`): neue Slots des Spiels laden, Match auf `(duty_type_id, event_time, team_id)` — derselbe `customKey` wie die Konflikterkennung —, Zuweisungen bis `slots_total` in ID-Reihenfolge wieder eingefügt
- [x] 3.3 `duty_slots.slots_filled` auf die Zahl der wiederhergestellten Zuweisungen gesetzt (denormalisiert, kein Trigger — vgl. `duties/handler.go:967`)
- [x] 3.4 `buildNotificationIntents` erhält die Menge der **nicht** wiederhergestellten Zuweisungen (`deletedAssignment.Restored`) statt aller; wiederhergestellte lösen keine Benachrichtigung aus
- [x] 3.5 Bestehende Regen-Charakterisierungstests geprüft — keine Anpassung nötig: die volle `internal/games`-Suite (inkl. `TestRegen_NotifiesRemovedAssignee`) ist ohne Änderung grün geblieben, weil kein Bestandstest den jetzt neuen Pfad „identische Regeneration mit Zuweisung" abdeckte (design.md §5 hatte das als Risiko benannt, nicht als Gewissheit)
- [x] 3.6 Tests: `TestRegen_ZuweisungUeberlebtIdentischeRegeneration` (inkl. `slots_filled` und ohne Benachrichtigung) · `TestRegen_CashSubstituteBleibtErhalten` (`status`/`cash_amount`/`fulfilled_at`) · `TestRegen_SchrumpfungBehaeltAelteste` (3→2, zwei älteste überleben, dritte benachrichtigt) · `TestRegen_VerschobeneUhrzeitLoestZuweisung` · `TestRegen_ReduzierteVarianteKeinTreffer`

## 4. Berechtigung

- [x] 4.1 `internal/policy/rules.go`: `CapBulkRegenDuties = "bulk_regen_duties"`, vergeben an `IsVorstandLike` (bewusst enger als `CapManageDuties`, das auch Trainer haben — design.md §10)
- [x] 4.2 Test `TestCapabilities_BulkRegenDuties` in `internal/policy`: Vorstand und Admin haben die Capability, Trainer/sportliche Leitung/Kassierer/Spieler/Standard-Nutzer nicht

## 5. Handler — Preview/Apply

- [x] 5.1 `internal/games/bulkregen_handler.go` (neu, im `games`-Package weil `runAutoRegen` unexportiert ist — gleiche Begründung wie `h4aimport_handler.go`): Request-/Response-Typen nach design.md §9
- [x] 5.2 Zeitraum auflösen: `from`/`to` optional, Default `[morgen, MAX(games.date) der aktiven Saison]`, `from` ≤ heute → HTTP 400 `range_in_past` (kein stilles Clamping)
- [x] 5.3 Plan aufbauen: Termine der aktiven Saison im Zeitraum laden, je Termin den effektiven Zustand aus `overrides` → `defaults` → gespeicherter `games.template_id` auflösen, Ausnahmen markieren; jede `template_id` gegen `game_templates` re-validiert (400 `invalid_template`); zusätzlich unbekannte `action`-Werte mit 400 `invalid_action` abgewiesen (Härtung, nicht in spec.md verlangt, aber konsistent mit `invalid_template`)
- [x] 5.4 Transaktion: `UPDATE games SET template_id` für `template`/`none`/`purge`, `DELETE FROM duty_slots WHERE game_id=?` (auch `is_custom=1`) für `purge`-Termine, dann **ein** `runAutoRegen` über die Vereinigungsmenge der Datumsfenster mit der Ausnahme-Menge als `skip`
- [x] 5.5 `PreviewBulkRegen`: identischer Pfad (`runBulkRegen(apply=false)`), Abschluss über `defer tx.Rollback()` (kein `Commit`); kein Broadcast, keine Benachrichtigung
- [x] 5.6 `ApplyBulkRegen`: `Commit`, danach genau ein `Broadcast("duties")` + ein `Broadcast("games")`, danach `dispatchRegenNotifications` — nur wenn `notify` nicht auf `false` steht (Default `true`)
- [x] 5.7 Antwort zusammensetzen: `range`, `rows` (aus `RegenSummary.PerGame` + Bestandszählung vor/nach, getrennt nach `auto`/`custom`), `totals`, `warnings` (aktuell immer `[]` — die `purge`+`notify:false`-Warnung ist laut design.md §7 ein Frontend-Konzern, siehe Task 8.4)

## 6. Routen & Gates

- [x] 6.1 `internal/app/router.go`: `POST /api/duty-slots/bulk-regen/preview` und `.../apply` im Vorstand-Tier registriert
- [x] 6.2 `internal/arch/broadcast_test.go`: Allowlist-Eintrag `"Games.PreviewBulkRegen": "Dry-Run mit Rollback, kein DB-Write; ApplyBulkRegen broadcastet 'duties'+'games'"`
- [x] 6.3 (nicht ursprünglich geplant, aber vom Gate verlangt) `internal/permissions/matrix_test.go`: beide Routen mit `exVorstand` in die Persona-Matrix aufgenommen (`TestArch_AuthzGatesMatchMatrix` verlangt das für jede neue `RequireClubFunction`-Route)

## 7. Backend-Tests (Routen + Invarianten)

- [x] 7.1 Routen-Matrix je Endpoint (`TestBulkRegen{Preview,Apply}_*`): 200 Happy-Path · 403 ohne Capability · 400 ohne aktive Saison · 400 `range_in_past` · 400 `invalid_template` · Preview ohne `from`/`to` liefert Default-Range
- [x] 7.2 **Preview schreibt nicht** (`TestPreviewBulkRegen_SchreibtNicht`): vollständiger DB-Snapshot vor/nach einem Preview mit `purge` über den ganzen Zeitraum ist identisch, inkl. Poison-Sanity für den Vergleicher (`dbFingerprint`)
- [x] 7.3 **Preview sagt die Wahrheit** (`TestBulkRegen_PreviewSagtDieWahrheit`): gleicher Body an `preview` und `apply` → identische `totals`, und die tatsächlichen DB-Zählungen nach dem Apply stimmen mit den Preview-Werten
- [x] 7.4 **`purge` vs. `none`** (`TestBulkRegen_PurgeVsNone`): gleicher Ausgangszustand mit einem `is_custom=1`-Slot → nach `purge` weg, nach `none` unverändert vorhanden
- [x] 7.5 **`notify: false`** (`TestBulkRegen_NotifyFalse_SendetNichts`) sendet nichts (Notification-Spy zählt 0), `totals.notified_users` ist trotzdem befüllt
- [x] 7.6 **Ein Broadcast pro Lauf** (`TestBulkRegenApply_EinBroadcastProLauf`): Apply über 40 Termine an 25 Tagen → genau ein `Broadcast("duties")` und ein `Broadcast("games")` (Hub-Spy via `prodserver.NewWithHub` + `SubscribeUser`, quiet-window-Drain statt Einzel-Empfang, da `Subscribe()` nur Puffer 1 hat und einen doppelten Broadcast stillschweigend verschlucken könnte)
- [x] 7.7 **Vergangenheit unerreichbar** (`TestBulkRegen_VergangenheitUnerreichbar`): ein Slot mit `event_date` in der Vergangenheit bleibt bei einem zukunftsseitigen Lauf unangetastet, und ein Request mit `from` in der Vergangenheit wird direkt abgelehnt

**Gefundener Bug (beim Testen von 7.1 aufgedeckt):** `loadBulkRangeGames` scannte `games.date` roh und benutzte den Wert direkt als Schlüssel für `dateSet`/`runAutoRegen` — die SQLite-DATE-Gotcha (`docs/agent/06-gotchas.md`) liefert dort einen ISO-Timestamp (`"2026-08-21T00:00:00Z"`) statt der reinen Datumszeichenkette, wodurch `regenSingleDay`'s `WHERE date=?` nie traf und der gesamte Massenlauf lautlos nichts erzeugte. Fix: Normalisierung auf die ersten 10 Zeichen direkt nach dem Scan (`bulkregen_handler.go`, `loadBulkRangeGames`).

## 8. Frontend — Modal

- [x] 8.1 `web/src/components/DutyBulkRegenModal.tsx`: Zeitraum-Felder (vorbelegt aus `range` der ersten Preview-Antwort, nur solange das Feld noch leer ist — spätere Nutzereingaben werden nie überschrieben), Pauschalwahl je Terminart, Zeilenliste mit Ausnahme-Checkbox und Zustands-Dropdown
- [x] 8.2 Dropdown-Einträge: Templates (gefiltert wie in `H4AImportModal.tsx:334`) plus „keine Dienste anlegen" und „alle Dienste löschen"; letzterer über den `BTN_DANGER`-Aktionsbutton (sobald irgendeine Zeile effektiv `purge` ist) als destruktiv markiert
- [x] 8.3 Zeilendarstellung mit Bestand und Wirkung: `N Slots · M handgemacht` → `+created / −deleted_auto · … Zuweisungen erhalten · … Konflikt(e)`; `assignments_lost > 0` mit `brand-danger` hervorgehoben
- [x] 8.4 Summenzeile über dem Aktionsbutton, inkl. Warnhinweis bei `purge` in Kombination mit abgeschalteter Benachrichtigung
- [x] 8.5 Checkbox „Betroffene nicht benachrichtigen" (Default aus = es wird benachrichtigt)
- [x] 8.6 brand-Tokens, lucide-Icons, verbindliche Klassen-Strings (Modal/Button/Input/Alert) nach `docs/agent/05-frontend.md`. **Abweichung:** kein separates Mobile-Kartenlayout — die Zeilen sind schon auf Desktop ein flex-wrap-Card-Layout (keine `<table>`), das auf Mobile identisch umbricht; eine zweite Darstellung hätte hier keinen Mehrwert gebracht.

## 9. Frontend — Live-Vorschau

- [x] 9.1 Vorschau bei jeder Änderung an Zeitraum/Pauschalwahl/Override/Ausnahme neu anfordern, entprellt (~400 ms)
- [x] 9.2 Laufende Anfrage per `AbortController` abbrechen; veraltete Antworten zusätzlich über eine monoton steigende Sequenznummer (`requestSeq`) verworfen (kein Flackern bei schneller Eingabe)
- [x] 9.3 Lade- und Fehlerzustand der Vorschau anzeigen (auch vor der allerersten Antwort: „Vorschau wird geladen…"); Aktionsbutton bleibt gesperrt, solange keine gültige Vorschau vorliegt (`disabled={applying || previewLoading || !preview}`)

## 10. Frontend — Einstieg im Kalender

- [x] 10.1 `web/src/pages/KalenderPage.tsx`: Menüeintrag „Dienste aktualisieren" (`<RefreshCw>`) als zweite Zeile im bestehenden Dropdown
- [x] 10.2 Sichtbarkeits-Gate des Dropdowns von `canImportGames` auf `canImportGames || canBulkRegenDuties` erweitert (Split-Button, Chevron, Menü-Container)
- [x] 10.3 Modal nach Apply schließen, `useLiveUpdates('duties')` lädt den Kalender nach. **Abweichung von der Aufgabenbeschreibung:** kein `RegenSummaryCard` — dessen Props-Shape (`created`/`reduced`/`skipped`-Arrays des Einzelspiel-Regens) passt nicht zu den Massenlauf-`totals` (aggregierte Zahlen: `games`/`created`/`deleted`/`custom_kept`/…). Stattdessen ein eigener Banner im selben Stil wie der bestehende `importResult`-Banner (H4A-Import), der die Massenlauf-`totals` direkt zusammenfasst.

## 11. Frontend-Tests (Vitest)

`web/src/components/__tests__/DutyBulkRegenModal.test.tsx`:

- [x] 11.1 Pauschalwahl setzt `defaults.heim`; Zeilen-Override sticht die Pauschalwahl (`overrides` trägt den Eintrag, `defaults.heim` bleibt unverändert bestehen)
- [x] 11.2 Ausnahme-Checkbox nimmt die Zeile in `excluded_game_ids` auf und sperrt ihr eigenes Zustands-Dropdown
- [x] 11.3 „alle Dienste löschen" färbt den Aktionsbutton `brand-danger` (sobald eine Zeile effektiv `purge` ist) und erzeugt `defaults.heim = {action:"purge"}` im Request
- [x] 11.4 Drei Änderungen ohne Await dazwischen erzeugen nach dem Debounce-Fenster genau eine zusätzliche Vorschau-Anfrage mit dem zuletzt gesetzten Wert
- [x] (zusätzlich, nicht ursprünglich gelistet) Apply-Ergebnis wird an `onApplied` gemeldet; Modal rendert nichts bei `isOpen=false`

**Nebenbei behoben:** `web/src/test/renderAsPersona.tsx`s Capability-Mirror kannte `bulk_regen_duties` noch nicht (Kommentar „keep in sync with rules.go") — ergänzt, sonst hätte jeder persona-basierte Test mit `vorstand` die neue Capability nie gesehen.

## 12. Doku

- [x] 12.1 `docs/agent/06-gotchas.md`: Gotcha „Massen-Dienstregeneration" — Restore-Vertrag (Match-Schlüssel, `slots_filled`, ID-Reihenfolge), Ausnahme-vs-Kontext (verallgemeinert auf: Nachbar-Kontext hängt nur an `is_home`-Existenz, nicht am Massenlauf-Zustand des Nachbarn), `purge` als einziger unumkehrbarer Zustand, Preview = Rollback-Transaktion. Zusätzlich ergänzt: die SQLite-DATE-Gotcha trifft jetzt nachweislich auch Go-Code (Bug in `loadBulkRangeGames` beim Testen von Section 7 gefunden und dokumentiert)
- [x] 12.2 `docs/agent/06-gotchas.md`: bekannter Rest — `duty_accounts.ist` wird beim Regen nicht nachgerechnet (nur in `DeleteGame`); deshalb ist der Massenlauf auf ab morgen begrenzt
- [x] 12.3 `docs/agent/04-api-db.md`: die beiden Routen im Vorstand-Tier ergänzt

## 13. Verifikation

- [x] 13.1 Laufzeit des Dry-Runs gemessen (`TestBenchBulkRegenPreviewLatency`, danach entfernt — Einmalmessung, kein Dauertest): ~36 ms für 40 Termine über `httptest`. design.md §8 korrigiert (Schätzung war „einstellige ms", gemessen ~36 ms — Schlussfolgerung „vernachlässigbar gegen 400 ms Debounce" bleibt gültig).
- [x] 13.2 Automatisiert nachgezogen statt rein manuell: `TestBulkRegen_ZweiterIdentischerLaufIstIdempotent` — Lauf mit gemischten Zuständen (`template`/`none`/`purge`, eine zwischenzeitlich hinzugefügte Zuweisung), zweiter identischer Lauf liefert `created == deleted`, `assignments_lost == 0`, `assignments_kept == 1`, `purge`-Termin bleibt bei 0 Slots.
- [x] 13.3 Vollständig grün: `go build ./...`, `go vet ./...`, `go test ./...` (gesamtes Repo, nicht nur `internal/games`), `golangci-lint run ./...` (0 Findings), `pnpm build`, `pnpm exec tsc --noEmit`, `pnpm exec eslint .` (0 Fehler in den neuen/geänderten Dateien), `pnpm test` (747/747). Broadcast-Gate (`TestEveryMutationRouteBroadcasts`) und Architektur-Test grün. Zusätzlich vom `TestArch_AuthzGatesMatchMatrix`-Gate verlangt und ergänzt: `internal/permissions/matrix_test.go` (Task 6.3). brand-Tokens/lucide-Icons manuell gegrept — keine raw Tailwind-Farben, keine Unicode-Icons in den neuen Dateien.
- [x] 13.4 `openspec validate duty-bulk-regen --strict` → „Change 'duty-bulk-regen' is valid"

**Nicht durchgeführt:** ein Live-Browser-Check des Modals (CLAUDE.md empfiehlt das für UI-Änderungen). Abgedeckt stattdessen durch: TypeScript-Typprüfung, ESLint, 6 Vitest-Interaktionstests (Pauschalwahl/Override/Ausnahme/purge-Färbung/Debounce/Apply-Callback) und 17 Go-HTTP-Tests gegen den echten `app.BuildRouter`. Vor dem ersten produktiven Einsatz sollte trotzdem einmal manuell durchgeklickt werden (Empfehlung, keine Pflicht laut CLAUDE.md).
