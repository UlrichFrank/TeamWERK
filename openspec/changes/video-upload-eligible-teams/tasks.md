## 1. Backend — Endpoint erweitern

- [ ] 1.1 `internal/videos/eligible_games.go`: Response um `teams` (`id`, `name`, `upload_without_game`) und `games` (`id`, `date`, `opponent`, `event_type`, `season_id`, `team_ids`) erweitern; Rollen-Teams gemäß D2 (admin/vorstand/sportliche_leitung → alle Teams mit Kader in aktiver Saison; trainer → `kader_trainers` aktive Saison), Dienst-Teams aus `game_teams` der Dienst-Spiele, `upload_without_game` nur für Rollen-Teams; `game_ids` unverändert. Verifikation: bestehende drei Tests in `eligible_games_test.go` bleiben grün.
- [ ] 1.2 Tests in `internal/videos/eligible_games_test.go`: Dienst-only-Nutzer erhält Team A mit `upload_without_game=false` und den `games`-Datensatz mit `team_ids=[A]`, obwohl er keinerlei Kader-Verbindung zu A hat; Trainer erhält Team A mit `true`; Trainer + Dienst bei Team B erhält beide Teams mit `true`/`false`; Vorstand erhält alle Teams mit Kader in der aktiven Saison, keins ohne Kader; Nutzer ohne Berechtigung erhält drei leere Listen mit 200. Verifikation: `go test ./internal/videos/ -run TestEligibleGames`.

## 2. Backend — Härtung Dienst-Pfad

- [ ] 2.1 `internal/videos/upload.go`: wenn die Berechtigung ausschließlich über `CanUploadForGameViaDuty` kommt, `team_id` gegen `game_teams` des Spiels prüfen und bei Fehltreffer HTTP 400 (`team_not_in_game`) antworten — nach der Berechtigungsprüfung, ohne DB-Eintrag. Verifikation: bestehende Upload-Tests grün.
- [ ] 2.2 Tests in `internal/videos/`: Dienst-Nutzer mit passendem `team_id` → 201; mit fremdem `team_id` → 400 und keine `videos`-Zeile; Trainer/Vorstand mit fremdem `team_id` weiterhin wie bisher (Rollen-Pfad unberührt). Verifikation: `go test ./internal/videos/ -run TestCreateUpload`.
- [ ] 2.3 `make test` und `make lint` für das Backend grün (inkl. Broadcast-Gate: Route bleibt Lese-Route in der Allowlist, Objektrechte-Matrix unverändert).

## 3. Desktop-Tool — Client und Auswahl-Logik

- [ ] 3.1 `tools/video-encoder/internal/client`: `Eligible`-Typ (`GameIDs`, `Teams`, `Games`) und `Eligible(ctx)` als Ersatz für `EligibleGameIDs`; `Teams()`, `Games()`, `Team`, `Game` des alten Vertrags entfernen, `Label()`/`dateOnly` auf den neuen `Game`-Typ übernehmen. Tests `TestEligibleGameIDs` → `TestEligible` (neues Parsing inkl. `upload_without_game` und `team_ids`), `TestAPI_CreateVideoSeasonsGames` ohne `/api/games`-Teil. Verifikation: `cd tools/video-encoder && go test ./internal/client/`.
- [ ] 3.2 Neues Paket `tools/video-encoder/internal/pick` mit `Teams`, `Games` (Filter Team + aktive Saison + Datum ≤ heute, jüngstes zuerst) und `AllowFreeTitle`; Tests decken die vier Spec-Szenarien ab (Dienst-only-Team ohne Freier Titel, Trainer-Team mit Freier Titel, keine Berechtigung → leer, Dienst mit nur Zukunftsspiel → Team wählbar, Spiele leer) sowie Dedup eines Teams, das Rollen- und Dienst-Team zugleich ist. Verifikation: `go test ./internal/pick/`.

## 4. Desktop-Tool — Oberfläche

- [ ] 4.1 `tools/video-encoder/ui.go`: Login lädt nur noch `ActiveSeasonID` + `Eligible`; `setTeams` aus `pick.Teams`, `onTeamChanged` aus `pick.Games` (kein `/api/games`-Aufruf mehr), „Freier Titel" nur bei `pick.AllowFreeTitle`, andernfalls erstes Spiel vorausgewählt; `eligibleGames`-Map durch die `Eligible`-Antwort ersetzen. Verifikation: `cd tools/video-encoder && go vet ./... && go build ./...` (CGo/Fyne lokal vorhanden), manueller Durchlauf gegen lokalen Server mit Dienst-only-Nutzer zeigt das Dienst-Team.
- [ ] 4.2 Hinweistexte ersetzen: leere Mannschaftsliste → „weder Trainer-/Vorstands-Berechtigung noch Video-Dienst hinterlegt"; leere Spielliste bei Dienst-only-Team → „Upload erst nach dem Spiel möglich"; Kommentar zu `/api/teams` im Tool entfernen. Verifikation: Texte in `ui.go` vorhanden, `go build` grün.

## 5. Doku und Abschluss

- [ ] 5.1 `docs/agent/06-gotchas.md`, Absatz „Video-Upload-Berechtigung über den Dienst ‚Video'": Ergänzen, dass `upload-eligible-games` `teams`/`games` liefert und das Tool keine Sichtbarkeits-Endpunkte mehr nutzt; Hinweis auf die `team_id`-Härtung im Dienst-Pfad. Verifikation: Abschnitt aktualisiert, `CHANGELOG.md` (falls gepflegt) ergänzt.
- [ ] 5.2 `openspec validate --all` grün; Change archivieren, Delta-Specs synchronisieren. Verifikation: `openspec validate --all`, Archiv-Verzeichnis vorhanden.
