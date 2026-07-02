## 1. Backend — Board-Gruppen tragen Termin-Teams

- [x] 1.1 `boardGroup`-Struct in `internal/duties/handler.go` umstellen: `TeamID *int`/`TeamName string` → `TeamIDs []int` (kein `omitempty`, immer Array) + `TeamNames []string`.
- [x] 1.2 Im Scan-Loop game-lose Gruppen initialisieren: `TeamIDs = [team_id]` falls `team_id > 0`, sonst `[]`; Kurzname aus der gejointen `teams`-Zeile in `TeamNames`.
- [x] 1.3 Nach dem Scan-Loop `game_id`s aller Gruppen sammeln und per Zusatz-Query `SELECT gt.game_id, gt.team_id, TeamDisplayShort … FROM game_teams gt JOIN teams t … WHERE gt.game_id IN (…) ORDER BY gt.game_id, t.age_class, t.gender` die Termin-Teams laden und je Gruppe (positionsgleich `TeamIDs`/`TeamNames`) anhängen — Muster wie Assignee-Nachladen.
- [x] 1.4 `go build ./...` + `go vet ./...` grün.

## 2. Backend — Tests

- [x] 2.1 Bestehenden Kontrakttest (`handler_test.go`, bisher numerisches `team_id`) auf `team_ids: []float64` umstellen.
- [x] 2.2 Test: Mehr-Team-Spiel → Gruppe trägt beide `team_ids` + `team_names` (positionsgleich).
- [x] 2.3 Test: generisches Event mit `game_teams` → Gruppe trägt Termin-Team trotz `ds.team_id = NULL`.
- [x] 2.4 Test: game-loser Handslot ohne `team_id` → leeres `team_ids`; mit `team_id` → `[team_id]`.
- [x] 2.5 `go test ./internal/duties/...` grün.

## 3. Frontend — Filter & Anzeige

- [x] 3.1 `BoardGroup`-Interface in `web/src/pages/DutyPage.tsx`: `team_id: number | null`/`team_name: string` → `team_ids: number[]`/`team_names: string[]`.
- [x] 3.2 `visibleGroups`-Filter: `filterTeamId !== null && !g.team_ids.includes(filterTeamId)` → ausblenden.
- [x] 3.3 Gruppen-Header (Zeile ~265): `team_names` komma-separiert anzeigen statt `team_name` (leer → nichts).
- [x] 3.4 Weitere Konsumenten des alten `team_id`/`team_name`-Feldes prüfen (v.a. Kalender-Dienste-Panel) und ggf. anpassen.
- [x] 3.5 `pnpm -C web build` + `pnpm -C web lint` grün.

## 4. Verifikation

- [x] 4.1 `/verify-change` bzw. volles Gate (`go test ./...`, `golangci-lint`, `pnpm -C web build/test/lint`, `openspec validate`) grün.
- [ ] 4.2 Manuell auf `/dienste`: Team-Filter zeigt Dienste eines Mehr-Team-Spiels bei Auswahl beider Teams; Header listet alle Teams.
