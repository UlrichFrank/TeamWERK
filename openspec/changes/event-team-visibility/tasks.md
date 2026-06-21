# Tasks — event-team-visibility

> Baut auf `profile-cross-team-visibility` auf. Ein Commit pro Task. Scope = führendes Package.

## 1. Visibility-Helper

- [ ] 1.1 `internal/auth/event_visibility.go`: `UserCanSeeGame(ctx, db, userID, gameID) (bool, error)` und `GameIDsVisibleToUser(ctx, db, userID, seasonID) ([]int, unrestricted bool, error)`. Funktionsträger-Bypass (admin/trainer/sportliche_leitung/vorstand) → `unrestricted=true`. Sonst: Schnittmenge `game_teams` mit Teams aus `kader_members`/`kader_extended_members` des Users + seiner Kinder via `family_links`.
- [ ] 1.2 `internal/auth/event_visibility_test.go`: Unit-Tests für alle Pfade (eigenes Team, Kind-Team, Funktionsträger, keine Zugehörigkeit, mehrere Teams im Event).
- [ ] 1.3 `go vet` + `gofmt`.

  _Commit:_ `feat(auth): zentraler Helper für Event-Sichtbarkeit pro User`

## 2. Games-Routen filtern

- [ ] 2.1 `internal/games/handler.go` (`ListGames`, `GetGame`, `GetParticipants`, `GetLineup`, `GetDutySlots` etc.): `GameIDsVisibleToUser` für Listen, `UserCanSeeGame` für Detail-Routen. Bei fehlender Sichtbarkeit: 404.
- [ ] 2.2 `go vet` + `gofmt`.

  _Commit:_ `feat(games): /games-Routen filtern nach Team-Sichtbarkeit`

## 3. Dashboard und Kalender

- [ ] 3.1 `internal/dashboard/handler.go`: Alle Game-bezogenen Sub-Queries (Nächste Termine, Offene Gesuche, etc.) per `GameIDsVisibleToUser` whitelisten.
- [ ] 3.2 `internal/calendar/handler.go` (oder Calendar-Endpoint, ggf. in `games/calendar.go`): Analog.

  _Commit:_ `feat(dashboard,calendar): Game-Sichtbarkeit zentral filtern`

## 4. Carpooling / Mitfahrgelegenheiten

- [ ] 4.1 `internal/carpooling/handler.go` (bzw. `mitfahrten/`): `POST` und `GET` prüfen `UserCanSeeGame` für das referenzierte Game → 404 bei fehlender Sichtbarkeit.

  _Commit:_ `feat(carpooling): Mitfahr-Routen prüfen Event-Sichtbarkeit`

## 5. Push-Notifications synchronisieren

- [ ] 5.1 Inventur: `grep` über Push-Caller (`SendToUsers`, `SendToUsersForGame`, …) sammeln, Empfängerlogik pro Stelle dokumentieren.
- [ ] 5.2 Empfänger-Sets so anpassen, dass das Ergebnis stets ⊆ `usersWithAccessToGame(gameID)` ist. Funktionsträger bleiben unverändert in inhaltlich gerichteten Pushes; allgemeine "Event geändert"-Pushes nur an sichtbarkeitsberechtigte Nutzer.
- [ ] 5.3 Tests: `TestPush_FremdEventKeinEmpfaenger`, `TestPush_TrainerImmerEmpfaenger`.

  _Commit:_ `feat(notifications): Push-Empfänger an Event-Sichtbarkeit ankoppeln`

## 6. Backend-Tests (Routen)

- [ ] 6.1 `TestListGames_FilterEigeneTeams`, `TestListGames_TrainerSiehtAlle`, `TestListGames_ElternSiehtTeamsDerKinder`.
- [ ] 6.2 `TestGetGame_FremdEvent_404`, `TestGetGame_EigenesEvent_200`.
- [ ] 6.3 `TestGetParticipants_FremdEvent_404`.
- [ ] 6.4 `TestDashboard_NaechsteTermine_Filter`, `TestCalendar_Filter`.
- [ ] 6.5 `TestCarpooling_FremdGame_404`.

  _Commit:_ `test: Event-Sichtbarkeit – Listen, Detail, Dashboard, Carpooling`

## 7. Bestandstests fitten

- [ ] 7.1 Bestehende Game-Tests durchgehen; wo der Test-User keine Team-Mitgliedschaft im Test-Game hat: `CreateKader` + `CreateKaderMember` ergänzen, sonst werden bisher grüne Tests 404.

  _Commit:_ `test: Bestandstests an Event-Sichtbarkeits-Regel anpassen`

## 8. Verifikation & Archive

- [ ] 8.1 `/verify-change` (Build/Test/Lint + Invariants).
- [ ] 8.2 OpenSpec-Proposal archivieren.

  _Commit:_ `chore(openspec): event-team-visibility archivieren`
