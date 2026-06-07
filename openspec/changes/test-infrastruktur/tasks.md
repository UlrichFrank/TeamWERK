## 1. Migration-Embed in db-Package verschieben

- [ ] 1.1 `internal/db/migrations.go` anlegen mit `//go:embed migrations/*.sql` und `var FS embed.FS`
- [ ] 1.2 `cmd/teamwerk/main.go`: inline-embed entfernen, stattdessen `db.FS` an `db.Migrate()` übergeben
- [ ] 1.3 `go build ./cmd/teamwerk` muss ohne Fehler durchlaufen

## 2. testutil-Package aufbauen

- [ ] 2.1 `internal/testutil/db.go`: `NewDB(t *testing.T) *sql.DB` — öffnet `:memory:` SQLite, führt `db.Migrate()` mit `db.FS` aus, registriert `t.Cleanup(db.Close)`
- [ ] 2.2 `internal/testutil/server.go`: `NewServer(t, db, jwtSecret, routeFn)` — erstellt Chi-Router mit `auth.Middleware`, ruft `routeFn(r)` für Routen-Registrierung auf, startet `httptest.NewServer`, registriert `t.Cleanup(srv.Close)`
- [ ] 2.3 `internal/testutil/server.go`: `Token(t, secret, userID int, role string, clubFunctions []string) string` — ruft `auth.IssueAccessToken` auf, gibt Bearer-String zurück
- [ ] 2.4 `internal/testutil/fixtures.go`: `CreateUser(t, db, role string, teamID int) (userID int)` — legt User mit bcrypt-Hash an
- [ ] 2.5 `internal/testutil/fixtures.go`: `CreateTeam(t, db, name string) (teamID int)` — legt Team an
- [ ] 2.6 `internal/testutil/fixtures.go`: `CreateSeason(t, db, name string) (seasonID int)` — legt aktive Saison an (`is_active=1`)
- [ ] 2.7 `internal/testutil/fixtures.go`: `CreateTrainingSeries(t, db, teamID, seasonID int) (seriesID int)` — legt minimale Serie an
- [ ] 2.8 `internal/testutil/fixtures.go`: `CreateTrainingSession(t, db, seriesID, teamID, seasonID int, date string) (sessionID int)` — legt einzelne Session an
- [ ] 2.9 `go test ./internal/testutil/...` muss grün sein (Smoke-Test der Helpers selbst)

## 3. Integration-Tests: internal/trainings

- [ ] 3.1 `internal/trainings/handler_test.go` anlegen; `testServer`-Hilfsfunktion im `_test`-Package schreibt Trainings-Routen in den Chi-Router
- [ ] 3.2 Test `TestListSessions_FilterByTeam`: Trainer sieht nur Sessions seines Teams, nicht fremde Teams
- [ ] 3.3 Test `TestListSessions_AdminSeesAll`: Admin erhält Sessions aller Teams
- [ ] 3.4 Test `TestListSessions_Unauthenticated`: Request ohne Token → HTTP 401
- [ ] 3.5 Test `TestCreateSeries_GeneratesSessions`: Serie für 4 Dienstage → genau 4 Sessions in DB
- [ ] 3.6 Test `TestCreateSeries_WrongTeam_Forbidden`: Trainer ohne Teamzugriff → HTTP 403
- [ ] 3.7 Test `TestRespond_SavesRSVP`: Spieler gibt RSVP ab → Status in DB gespeichert
- [ ] 3.8 Test `TestRespond_UpdatesExistingRSVP`: Zweites Respond überschreibt erstes (kein Duplicate)
- [ ] 3.9 Test `TestSaveAttendances_TrainerOK`: Trainer speichert Anwesenheit → HTTP 200
- [ ] 3.10 Test `TestSaveAttendances_PlayerForbidden`: Spieler → HTTP 403
- [ ] 3.11 `go test ./internal/trainings/...` muss grün sein

## 4. Integration-Tests: internal/games

- [ ] 4.1 `internal/games/handler_test.go` anlegen; `testServer`-Hilfsfunktion registriert Kalender-Routen
- [ ] 4.2 Test `TestListGames_ReturnsGamesInRange`: Spiele im Zeitraum werden zurückgegeben
- [ ] 4.3 Test `TestListGames_EmptyRange`: Kein Spiel im Zeitraum → HTTP 200 + leeres Array
- [ ] 4.4 Test `TestCreateGame_AdminOK`: Admin legt Heimspiel an → HTTP 201
- [ ] 4.5 Test `TestCreateGame_TrainerForbidden`: Trainer → HTTP 403
- [ ] 4.6 `go test ./internal/games/...` muss grün sein

## 5. Statische Analyse & Makefile

- [ ] 5.1 `.golangci.yml` im Repo-Root anlegen mit Linter-Set: `govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`
- [ ] 5.2 `make lint` Target im Makefile: prüft ob `golangci-lint` im PATH, gibt Installationshinweis wenn nicht, sonst `golangci-lint run ./...`
- [ ] 5.3 `make test` Target im Makefile: `go test -race ./...`
- [ ] 5.4 `make lint` ausführen und alle Findings entweder beheben oder mit `//nolint:<linter> // Reason: ...` annotieren
- [ ] 5.5 `make test` und `make lint` laufen beide sauber durch
