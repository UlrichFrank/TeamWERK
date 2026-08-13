## 1. Rotationsplan auf Kuchen umstellen

- [x] 1.1 `rotationAssignment` zu `{TeamID int, Cakes int}` ändern und `rotationPlan` als „Anker-Spiel → Zuteilung" dokumentieren (`internal/games/regen.go`)
- [x] 1.2 `rotationGroup` um `anchorByTeam map[int]int` erweitern; beim Aufbau der Warteschlange die `game_id` des Eintritts mitschreiben
- [x] 1.3 `buildRotationPlan`: Bedarf auf `ceil(games × verhaeltnis)` ohne Deckelung umstellen; greedy `min(cap, rest)` Kuchen je Team; Plan-Eintrag nur für Teams mit Zuteilung, geschlüsselt auf ihr Anker-Spiel
- [x] 1.4 Restbedarf als Zahl je (Tag, Duty-Type) zurückgeben statt team-lose Slots zu erzeugen
- [x] 1.5 Prüfen, dass das Ausrichter-Gate weiterhin vor Warteschlange und Bedarfsrechnung greift (`itemPassesAusrichterGate` in derselben Sammel-Schleife)

## 2. Slot-Erzeugung und Zusammenfassung

- [x] 2.1 Rotations-Zweig in `regenGameItems`: `slots_total` aus `Cakes` statt `it.SlotsCount`, `team_id` aus `TeamID`, kein `Unassigned`-Eintrag mehr an dieser Stelle
- [x] 2.2 `UnassignedEntry` von `{Date, DutyType, GameID}` auf `{Date, DutyType, Count}` umstellen und in `regenSingleDay` mit dem dort bekannten `date` befüllen
- [x] 2.3 `Created`-Zählung prüfen: `Count` muss die Kuchen zählen (`Cakes`), nicht `matchedTeams × slots_count`
- [x] 2.4 Frontend-Anzeige der `unassigned`-Liste prüfen — entfällt: kein Frontend-Code liest das Feld (Grep über `web/src`), es ist bislang rein serverseitig

## 3. Match-Key vereinheitlichen

- [x] 3.1 `rotationTypes`-Parameter aus `makeCustomKey`, `snapshotCustomSlots` und `restoreAssignments` entfernen
- [x] 3.2 Aufbau der `rotationTypes`-Menge in `regenSingleDay` und `itemRotationTypes` in `regenGameItems` entfernen
- [x] 3.3 `restoreAssignments` gegen den Dreier-Match `(duty_type_id, event_time, team_id)` verifizieren, inkl. Rückschreiben bis `slots_total` in aufsteigender `duty_assignments.id`-Reihenfolge

## 4. Vorlagen-Editor

- [x] 4.1 `AdminDutyTemplateDetailPage.tsx`: Feld „Anzahl" deaktivieren, wenn `rotation_enabled` gesetzt ist, mit Hinweis auf die Zuteilung
- [x] 4.2 Gleiches im Modal auf `AdminDutyTemplatesPage.tsx`
- [x] 4.3 Vitest für beide Editoren: gesetzte Checkbox ⇒ „Anzahl" disabled, Hinweis sichtbar

## 5. Tests umschreiben und ergänzen

- [x] 5.1 `rotation_regen_test.go`: „Fünf Spiele, drei Teams, Cap zwei" auf die neue Erwartung umschreiben (Slots je Mannschaft mit `slots_total`, Anker-Spiel prüfen)
- [x] 5.2 Test „Verhältnis kleiner eins" auf Kuchen-Semantik umstellen
- [x] 5.3 Test „Verhältnis größer eins" von „gedeckelt" auf „schlägt durch" umschreiben
- [x] 5.4 Test „Cap-Überlauf" auf „Restbedarf verfällt" umschreiben: kein `team_id IS NULL`-Slot, `unassigned` mit korrektem `Count`
- [x] 5.5 Neuer Test: Slot hängt am eigenen Termin der Mannschaft (`game_id` und `event_time` gegen das Anker-Spiel prüfen)
- [x] 5.6 Neuer Test: `slots_count` der Vorlage bleibt für Rotations-Items ohne Wirkung
- [x] 5.7 Neuer Test: zwei gleichzeitige Heimspiele verschiedener Mannschaften ⇒ zwei getrennte Slots, keine Verwechslung im Restore
- [x] 5.8 Test „Zusage überlebt verschobene Team-Zuordnung" durch die drei Restore-Szenarien aus der Spec ersetzen (gleichbleibende Zuteilung / gesunkene Kuchenzahl / Mannschaft fällt raus)
- [x] 5.9 `ausrichter_regen_test.go` auf die neue Bedarfs-/Slot-Semantik nachziehen
- [x] 5.10 Realdaten-Regressionstest 27.09.: fünf Heimspiele (gD 10:00, wB 11:30, mA2 13:15, mA1 15:15, mA1 15:15), Verhältnis 1, Cap 2 ⇒ drei Slots 2 / 2 / 1 an den Terminen von gD, wB, mA2

## 6. Abschluss

- [x] 6.1 `make test` und `pnpm -C web test` grün
- [x] 6.2 Gate: `go vet`, `go test ./...`, `golangci-lint` (0 issues), `pnpm -C web test` (801) + `lint`, `openspec validate --all --strict` (257/257)
- [ ] 6.3 Deploy und Massenlauf „Dienste aktualisieren" über den betroffenen Zeitraum (angekündigt — Zusagen auf wegfallenden Slots werden benachrichtigt)
