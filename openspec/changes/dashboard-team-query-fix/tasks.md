## 1. Dashboard Handler — team_query ersetzen

- [ ] 1.1 `teamQueryForUser`-Methode aus `internal/dashboard/handler.go` entfernen
- [ ] 1.2 `vehicleAction`: `fmt.Sprintf`-Pattern durch statische Query mit `user_accessible_teams`-Subquery ersetzen; Early-Return für `admin`/`vorstand`
- [ ] 1.3 `queryNextGames`: analog zu 1.2
- [ ] 1.4 `queryCarpoolingHint`: analog zu 1.2

## 2. Prüfung memberDutyActions

- [ ] 2.1 `memberDutyActions` analysieren: wird nur für `elternteil`/`spieler` aufgerufen — prüfen ob `user_accessible_teams` hier ebenfalls verwendet werden soll oder ob der bestehende Filter ausreicht

## 3. Nachbereitung role-model-refactor

- [ ] 3.1 Task 3.1 in `openspec/changes/role-model-refactor/tasks.md` als obsolet/abgedeckt kommentieren

## 4. Verifikation

- [ ] 4.1 Go-Build erfolgreich (`go build ./...`)
- [ ] 4.2 Manueller Test auf Produktion: als Trainer mit family_links-Zugang einloggen → Fahrtgemeinschaft und Spielplan erscheinen korrekt
