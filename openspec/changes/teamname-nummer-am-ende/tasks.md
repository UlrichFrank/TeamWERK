# Tasks

## Test-Anforderungen

| Route / Oberfläche | Test | Erwartung |
|---|---|---|
| `GET /api/games` (Doppelheimspiel) | `internal/games/handler_test.go` (bestehende `team_display_long_csv`-Assertions) | 200, `"B-Jugend männlich 1, B-Jugend männlich 2"` |
| `GET /api/teams/my` | `TestListMyTeams_DisplayLongNummerAmEnde` | 200, `display_long = "C-Jugend männlich 2"`, `display_short = "mC2"` |
| `GET /api/teams/my` | `TestListMyTeams_EinzigeMannschaftOhneNummer` | 200, `display_long = "A-Jugend weiblich"` |
| `GET /api/teams/my` | `TestListMyTeams_OhneToken` | 401 |
| `GET /api/teams/{id}/roster` | `TestGetRoster_DisplayLongNummerAmEnde` | 200, `team.display_long = "C-Jugend männlich 1"` |
| MeinTeamPage | `MeinTeamPage.teamname.test.tsx` | eingeklappt und aufgeklappt identischer Titel |
| `buildTeamLongName` | `teamName.test.ts` | Nummer am Ende bei `groupCount > 1`, sonst ohne |
| Kader-Verwaltung | `AdminKaderPage.teamname.test.tsx` | Titel und Lösch-Bestätigung „C-Jugend männlich 1“ |
| Automatische Zuordnung | `AutoAssignModal.teamname.test.tsx` | „… 1“ / „… 2“ unterscheidbar |
| Kader kopieren | `CopyKaderModal.teamname.test.tsx` | eine Zeile je Kombination + Hinweis |

**Invariante:** Eine Schreibweise „<Altersklasse> <n> <Geschlecht>" wird von keinem Endpoint und keiner Oberfläche mehr erzeugt; `teams.name` bleibt unverändert.

## 1. Server: Langform mit Nummer am Ende

- [x] 1.1 `internal/db/team_display_name.go`: Mehrfall auf `age_class || ' ' || <Geschlecht> || ' ' || team_number` umstellen, Doc-Kommentar mit Beispiel „C-Jugend männlich 1" ergänzen; verifizieren mit `go build ./...`
- [x] 1.2 Erwartungswerte in `internal/games/handler_test.go` (Zeilen mit `team_display_long_csv`) auf „B-Jugend männlich 1, B-Jugend männlich 2" und den Kommentar in `internal/duties/handler_test.go` anpassen; verifizieren mit `go test ./internal/games/ ./internal/duties/`
- [x] 1.3 `TestGetRoster_DisplayLongNummerAmEnde` in `internal/teams/handler_test.go` (zwei Kader C-Jugend m in aktiver Saison, Roster von Nummer 1 → „C-Jugend männlich 1"); verifizieren mit `go test ./internal/teams/`
- [x] 1.4 `grep -rn "Jugend [0-9] \(männlich\|weiblich\|gemischt\)" internal web/src docs` liefert keine Treffer mehr außer Archiv; verifizieren durch leere Ausgabe

## 2. Server: `GET /api/teams/my` liefert Display-Strings

- [x] 2.1 `ListMyTeams` (`internal/teams/handler.go`): in beiden UNION-Zweigen `COALESCE(TeamDisplayShort("t"), t.name)` und `COALESCE(TeamDisplayName("t"), t.name)` selektieren und in `Team.DisplayShort`/`DisplayLong` scannen; `ORDER BY t.name` beibehalten; verifizieren mit den bestehenden `TestListMyTeams_*`
- [x] 2.2 Tests `TestListMyTeams_DisplayLongNummerAmEnde`, `TestListMyTeams_EinzigeMannschaftOhneNummer`, `TestListMyTeams_OhneToken` (401 über den Router) ergänzen; verifizieren mit `go test ./internal/teams/`

## 3. Frontend: „Mein Team" zeigt eingeklappt die Langform

- [x] 3.1 `MeinTeamPage.tsx`: `myTeams`-Typ um `display_short?`/`display_long?` erweitern, Kartentitel auf `roster?.team.display_long || team.display_long || team.name` umstellen; verifizieren mit `pnpm -C web build`
- [x] 3.2 `MeinTeamPage.teamname.test.tsx`: `/teams/my` liefert zwei C-Jugenden mit `display_long` „… männlich 1/2", `name` „C-Jugend männlich"/„C-Jugend männlich 2" → eingeklappte Titel sind „C-Jugend männlich 1/2"; nach Aufklappen (Roster mit gleichem `display_long`) bleibt der Titel gleich; verifizieren mit `pnpm -C web test MeinTeamPage.teamname`

## 4. Frontend: Kader-Oberflächen mit gemeinsamem Helfer

- [x] 4.1 `web/src/lib/teamName.ts`: `buildTeamLongName({age_class, gender, team_number}, groupCount)` exportieren (Nummer am Ende genau bei `groupCount > 1`), Tests in `teamName.test.ts` (1 von 2 → „C-Jugend männlich 1", einzige → „A-Jugend weiblich", gemischt); verifizieren mit `pnpm -C web test teamName`
- [x] 4.2 `AdminKaderPage.tsx`: Kartentitel und Lösch-Bestätigung über `buildTeamLongName` (groupCount aus der Gruppe derselben age_class+gender), lokales `GENDER_LABEL` durch den Export ersetzen, sofern nur für Namen genutzt; verifizieren mit `pnpm -C web build` und bestehenden AdminKader-Tests
- [x] 4.3 `AutoAssignModal.tsx`: `team_number` ins lokale Interface, Titel über `buildTeamLongName` mit groupCount aus der geladenen Kader-Liste; `CopyKaderModal.tsx`: eine Zeile je Kombination ohne Nummer, Hinweis bei mehreren Quell-Mannschaften (Entscheidung siehe design.md); verifizieren mit `AutoAssignModal.teamname.test.tsx` und `CopyKaderModal.teamname.test.tsx`

## 5. Nachgemeldet: Dashboard und Videos

- [x] 5.1 Dashboard-Kachel „Mein Team" rendert `display_long || name`; verifizieren mit `DashboardPage.teamname.test.tsx`
- [x] 5.2 Videos (`crud.go` Liste/Detail, `worker.go` Push, `eligible_games.go` Teamnamen) liefern `COALESCE(TeamDisplayName, t.name)`; verifizieren mit `TestListVideos_TeamNameIstLangformMitNummerAmEnde`

## 6. Nachgemeldet: alle Seiten konsistent

- [x] 6.1 `appdb.TeamLongName(alias)` = `COALESCE(TeamDisplayName, alias.name)`; alle Anzeige-Selects umgestellt: Anwesenheit (Stats, RSVP-Matrix), `GET /api/teams` (+`display_short`/`display_long`), Kalender-Feed (Spiele, Trainings), Scheduler-Erinnerungen (Spiele, Trainings, Anwesenheit), Dienst-Export, Spielbericht-Veröffentlichung, Spiel-Suche, Mitfahrten, Staffeln, Ordner-Rechte, H4A-Vorschau (Abgleich bleibt auf `teams.name`); verifizieren mit `TestGetTeamStats_TeamNameIstLangform`, `TestListTeamsForUser_DisplayLongNummerAmEnde` und angepassten Kalender-/Staffel-Tests
- [x] 6.2 H4A-Import-Modal: Mannschaftsauswahl zeigt `display_long || name`; verifizieren mit `pnpm -C web build`
- [x] 6.3 Gotcha „Teamname: Anzeige vs. Identität" in `docs/agent/06-gotchas.md`

## 7. Dokumentation und Integration

- [x] 7.1 Benutzerhandbuch/Schulungsfolien auf Beispiele der alten Reihenfolge prüfen (`grep -rn "Jugend [0-9] " web/public docs/schulung`) und ggf. anpassen; verifizieren durch leere Ausgabe
- [x] 7.2 Gesamt-Gate: `make test`, `pnpm -C web test`, `pnpm -C web lint`, `openspec validate teamname-nummer-am-ende --strict` grün
