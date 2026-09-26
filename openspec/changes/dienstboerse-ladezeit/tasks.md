# Tasks

## 1. Team-Prädikat umformulieren (Backend)

- [x] 1.1 Vorher-Messung festhalten: SQL-Text der Assignee-Query aus `duties.Board` in eine unexportierte Funktion `boardAssigneesSQL(n int) string` ziehen (reines Refactoring, Form unverändert); verifiziert durch unverändert grünes `go test ./internal/duties/...`
- [ ] 1.2 `duties.slotInTeamsSQL` und `dashboard.slotInTeams` auf `EXISTS (SELECT 1 FROM game_teams gt_s WHERE gt_s.game_id = ds.game_id AND gt_s.team_id IN (…))` umstellen, den Zweig `ds.game_id IS NULL AND ds.team_id IN (…)` unverändert lassen und den Doc-Kommentar um den Grund ergänzen (Korrelation über den PK, planerunabhängig); verifiziert durch grüne `go test ./internal/duties/... ./internal/dashboard/... ./internal/dutyfairness/...`, insbesondere `TestAushilfePraedikat_BoardUndBilanzDeckungsgleich` und die Tests in `board_aushilfe_test.go`
- [ ] 1.3 (optional) Beide Hilfsfunktionen als `appdb.SlotInTeamsSQL` nach `internal/db/user_teams.go` zusammenlegen und die Kopien entfernen; nur umsetzen, wenn der Arch-Test (`go test ./internal/arch/...`) grün bleibt, sonst Task als bewusst verworfen markieren

## 2. Regressionsschutz (Tests)

- [ ] 2.1 `TestBoardAssigneesPlan_KeinScanAufGameTeams` in `internal/duties`: kleine Saison mit Spiel, `game_teams`, Slot und Zusage über `testutil` anlegen, `EXPLAIN QUERY PLAN` auf `boardAssigneesSQL` ausführen und verlangen, dass keine Planzeile `SCAN gt_s` oder `SCAN game_teams` enthält; verifiziert dadurch, dass der Test gegen die alte Form (1.1-Stand, lokal kurz zurückgestellt) rot und gegen die neue grün ist
- [ ] 2.2 Denselben Test nach `ANALYZE` wiederholen (Szenario „Plan bleibt nach Statistik-Erhebung gleich"); verifiziert durch grünen Testlauf
- [ ] 2.3 Ergebnis-Gleichheit über Personas: in `board_aushilfe_test.go` einen Test ergänzen, der für Admin, Trainer, Stamm-Elternteil und Nur-Erweitert-Elternteil die komplette Board-Response (Gruppen, Slots, Eingetragene, beide `aushilfe`-Kennzeichen) gegen erwartete Werte prüft, falls die bestehenden Tests das Eingetragenen-Kennzeichen für den Nur-Erweitert-Fall noch nicht abdecken; verifiziert durch grünen Testlauf

## 3. Messung und Abschluss

- [ ] 3.1 Nachher-Messung gegen eine Kopie der lokalen DB (lokaler Server auf eigenem Port, Tokens für Admin/Trainer/Eltern/Spieler): `GET /api/duty-board` mit und ohne `from` sowie die Slot-Query allein für Nicht-Vorstand-Personas; Ziel: Board < 100 ms für alle Personas, Slot-Query nicht langsamer als vorher (Toleranz 20 ms, sonst Fallback aus design.md „Risks" umsetzen); Zahlen in den Commit-Text übernehmen
- [ ] 3.2 `/dienste` einmal im Browser (Chrome DevTools, Performance-Trace) als Admin laden und prüfen, dass nach dem Backend-Fix kein Rendering-Engpass > 500 ms bleibt; Befund notieren und bei Bedarf als eigenen Folge-Change vorschlagen, nicht in diesem Change umsetzen
- [ ] 3.3 Gotcha-Absatz in `docs/agent/06-gotchas.md` ergänzen: korrelierte Team-Prädikate über `game_teams` immer über `game_id` (PK) formulieren, nicht über `team_id IN`; ein Index `game_teams(team_id)` hilft nur ohne `ANALYZE`; verifiziert durch Review des Absatzes
- [ ] 3.4 `/verify-change` ausführen (Build, `go test ./...`, Lint, `pnpm -C web build/test/lint`, `openspec validate dienstboerse-ladezeit`); verifiziert durch grünen Lauf
