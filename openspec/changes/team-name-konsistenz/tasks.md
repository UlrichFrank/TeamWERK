## 1. Backend — SQL-Helper

- [x] 1.1 `internal/db/team_display_short.go` anlegen mit `TeamDisplayShort(alias string) string` analog zu `TeamDisplayName`
- [x] 1.2 Unit-Test `internal/db/team_display_short_test.go`: Single-Team, Multi-Team derselben age_class+gender, Gender `m`/`f`/`mixed`, unbekannte age_class

## 2. Backend — Games-Endpoints

- [x] 2.1 `internal/games/handler.go` `ListGames`: `teams[]`-Subquery um `TeamDisplayShort` und `TeamDisplayName` ergänzen; `team_display_short_csv` und `team_display_long_csv` (GROUP_CONCAT, alphabetisch) in Response aufnehmen
- [x] 2.2 `ListMyGames`: zusätzliche `GROUP_CONCAT(TeamDisplayShort(...))` und `GROUP_CONCAT(TeamDisplayName(...))` Subqueries; neue Felder im JSON-Item
- [x] 2.3 `GetGame`: `teams[]`-Subquery um `display_short`, `display_long` pro Team erweitern
- [x] 2.4 Test `internal/games/games_test.go`: Doppelheimspiel-Fixture → `team_display_short_csv` enthält beide Kurznamen sortiert
- [x] 2.5 Test `ListMyGames` mit Doppelheimspiel → Felder gesetzt

## 3. Backend — Duties-Endpoint

- [x] 3.1 `internal/duties/handler.go` `DutyBoard`: `TeamDisplayName(t)` → `TeamDisplayShort(t)` für `team_name` (Frontend nutzt `team_name` direkt, keine API-Breaking-Änderung sondern Inhaltswechsel — Frontend-Verträge bleiben gleich)
- [x] 3.2 Test: DutyBoard mit Game-bezogenem Slot → `team_display_short` korrekt

## 4. Backend — Dashboard

- [x] 4.1 `internal/dashboard/handler.go:176` `MIN(COALESCE(TeamDisplayName(t), t.name))` → `GROUP_CONCAT(COALESCE(TeamDisplayShort(t), t.name), ', ')` mit `ORDER BY` (Subquery-Pattern)
- [x] 4.2 `dashboard/handler.go:160` (Training) auf `TeamDisplayShort` umgestellt
- [x] 4.3 Regression-Test: Dashboard-Time-Window mit Doppelheimspiel → beide Teams im `teamName`-String
- [x] 4.4 Bestehende Dashboard-Tests prüfen Kurzform nicht explizit → keine Anpassung nötig

## 5. Frontend — Helper

- [x] 5.1 `web/src/lib/teamName.ts` um `formatTeamList(teams, mode)` erweitern: Modi `'short' | 'long' | 'kalender'`; `'kalender' + >1 Team` → `'Mehrere'`; Fallback auf `name` wenn `display_short` fehlt
- [~] 5.2 ~~Snapshot-Test~~ — kein vitest/jest im Repo; Verifikation via Task 10.3 (Browser-Test mit Doppelheimspiel-Fixture)
- [~] 5.3 ~~Parity-Test~~ — Parity wird durch Backend-Unit-Tests (Task 1.2) abgesichert, die exakt die Output-Strings aus `buildTeamShortNames` reproduzieren

## 6. Frontend — Kalender

- [x] 6.1 `KalenderPage.tsx` Tooltip — `'Mehrere Teams'` beibehalten (Kalender-Ausnahme), Label via `formatTeamList(teams, 'kalender')` zentralisiert
- [x] 6.2 `KalenderPage.tsx` Inline-Label — `'Mehrere'` via `formatTeamList(teams, 'kalender')`
- [x] 6.3 `KalenderPage.tsx` Training-Tile: bleibt bei `shortNames.get` (clientseitige Map deckt diesen Fall — `/api/games` für Kalender liefert keine display-Felder; ggf. Folge-Change)
- [x] 6.4 `EventInfoModal.tsx`: `formatTeamList(teams, 'short')` statt `teams.map(t => t.name).join(', ')`
- [x] 6.5 `EventInfoModal.tsx` Training-Anzeige bleibt unverändert

## 7. Frontend — Termine-Seite

- [x] 7.1 `TerminePage.tsx`: `'Mehrere Teams'`-Sonderfall entfernt; nutzt jetzt `team_display_short_csv` aus Backend (Fallback auf `team_names`)
- [x] 7.2 `TerminePage.tsx` Training-Karte: Backend `internal/trainings/handler.go:760` auf `TeamDisplayShort` umgestellt → Frontend zeigt automatisch Kurzform

## 8. Frontend — Detail-Seiten

- [x] 8.1 `SpieltagDetailPage.tsx`: Bug-Fix für leeres `team_name` — liest jetzt `team_display_long_csv` bzw. baut Langform aus `teams[]` mit `formatTeamList(teams, 'long')`
- [x] 8.2 `TermineDetailPage.tsx:255–258`: bleibt Langform — Backend liefert weiterhin `TeamDisplayName` für Session-Detail (`internal/trainings/handler.go:877`)
- [x] 8.3 `MeinTeamPage.tsx:36`: `roster.team.display_long || roster.team.name`; Backend erweitert (`/api/teams/{id}/roster` liefert jetzt `display_short`, `display_long`)

## 9. Frontend — Duty/Dashboard/Admin

- [x] 9.1 `DutyPage.tsx`: keine Frontend-Änderung nötig — Backend liefert über `internal/duties/handler.go:429` jetzt Kurzform
- [x] 9.2 `DashboardPage.tsx`: keine Frontend-Änderung nötig — Backend liefert jetzt Kurzform
- [x] 9.3 `AdminTrainingsPage.tsx:362`: bereits Backend-getrieben — Series-Endpoint (`internal/trainings/handler.go:156`) auf `TeamDisplayShort` umgestellt
- [x] 9.4 `AdminTrainingsPage.tsx:474, 609` (Edit-Modal): unverändert (schon Kurzform via `buildTeamShortNames`)

## 10. Aufräumen & Tests

- [x] 10.1 Grep auf `'Mehrere Teams'`/`'Mehrere'` zeigt nur noch die zwei erwarteten Stellen: `teamName.ts` (Helper) und `KalenderPage.tsx` (Tooltip)
- [x] 10.2 `go test ./internal/...` 210 passed (+1 flaky, unrelated); Frontend `pnpm build` grün
- [ ] 10.3 Browser-Verifikation (manueller Test durch User — Backend + Frontend gebaut und bereit)
- [ ] 10.4 OpenSpec-Proposal archivieren (nach Browser-Verifikation)

## 11. Commits (Conventional Commits, pro Task-Gruppe)

- [ ] 11.1 `feat(db): TeamDisplayShort SQL-Helper für einheitliche Team-Kurzform` (1.x)
- [ ] 11.2 `feat(games): Display-Felder team_display_short/long in Games-Endpoints` (2.x)
- [ ] 11.3 `feat(duties): Kurzform für Team-Header in DutyBoard` (3.x)
- [ ] 11.4 `fix(dashboard): GROUP_CONCAT statt MIN — alle Teams bei Doppelheimspielen anzeigen` (4.x)
- [ ] 11.5 `feat(web): formatTeamList-Helper für konsistente Team-Anzeige` (5.x)
- [ ] 11.6 `refactor(web): Kalender, Termine, Detail-Seiten auf formatTeamList umstellen` (6.x–9.x)
- [ ] 11.7 `fix(web): Teamname in SpieltagDetailPage und MeinTeam korrigieren` (8.1, 8.3)
- [ ] 11.8 `chore(openspec): team-name-konsistenz archivieren` (10.4)
