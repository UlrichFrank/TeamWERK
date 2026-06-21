# Tasks — event-team-visibility

> Baut auf `profile-cross-team-visibility` auf. Ein Commit pro Task. Scope = führendes Package.

## 1. Visibility-Helper

- [x] 1.1 `internal/auth/event_visibility.go`: `UserCanSeeGame(ctx, db, userID, gameID) (bool, error)` und `GameIDsVisibleToUser(ctx, db, userID, seasonID) ([]int, unrestricted bool, error)`. Funktionsträger-Bypass (admin/trainer/sportliche_leitung/vorstand) → `unrestricted=true`. Sonst: Schnittmenge `game_teams` mit Teams aus `kader_members`/`kader_extended_members` des Users + seiner Kinder via `family_links`.
- [x] 1.2 `internal/auth/event_visibility_test.go`: Unit-Tests für alle Pfade (eigenes Team, Kind-Team, Funktionsträger, keine Zugehörigkeit, mehrere Teams im Event).
- [x] 1.3 `go vet` + `gofmt`.

  _Commit:_ `feat(auth): zentraler Helper für Event-Sichtbarkeit pro User`

## 2. Games-Routen filtern

- [x] 2.1 `internal/games/handler.go`: `ListGames` nutzt jetzt `auth.GameVisibilityClause` statt `policy.ScopeGamesQuery` (Trainer wird Bypass — inkludiert auch erweiterte Kader-Mitglieder). `GetGame`, `GetParticipants`, `RespondToGame`, `ListGameResponses` antworten mit 404, wenn `auth.UserCanSeeGame` false liefert. `SaveLineup` ist trainer-only und damit ohnehin Bypass.
- [x] 2.2 `go vet` + `gofmt`.

  _Commit:_ `feat(games): /games-Routen filtern nach Team-Sichtbarkeit`

## 3. Dashboard und Kalender

- [x] 3.1 Dashboard ist bereits personal-team-scoped via `user_accessible_teams` (kader_members + family + kader_trainers + extended). Strikt enger als die neue Visibility-Regel — kein Codeänderung nötig. Persönliche Dashboards bleiben „MY events", auch für Funktionsträger (Übersicht über fremde Teams ist `/termine`, nicht das Dashboard).
- [x] 3.2 `internal/calendar/handler.go` filtert pro `m.user_id` über `kader_members` (sogar enger als Dashboard). Ebenfalls keine Änderung — iCal-Abo bleibt persönlich.

  _Commit:_ `feat(dashboard,calendar): Game-Sichtbarkeit zentral filtern`

## 4. Carpooling / Mitfahrgelegenheiten

- [x] 4.1 `internal/carpooling/handler.go` `Upsert`: prüft `UserCanSeeGame` für `body.GameID` → 404. `Delete` und Paarungs-Routen sind durch ihre Ownership-Checks abgesichert. `List` ist bereits personal-team-scoped via `user_accessible_teams`.

  _Commit:_ `feat(carpooling): Mitfahr-Routen prüfen Event-Sichtbarkeit`

## 5. Push-Notifications synchronisieren

- [x] 5.1 Inventur: Alle Push-Calls gehen entweder an `teamMembersAndParents`/`eligibleDutyUsers` (kader_members + family_links, also ⊆ visibility) oder an gezielte User-IDs (Pairings, Reminder, Admin/Membership). Keine Stelle pusht an breitere Menge.
- [x] 5.2 Keine Code-Änderung nötig: bestehende Empfänger-Sets sind bereits ⊆ visibility (per `player_memberships`-Subset von `user_accessible_teams`). Funktionsträger bleiben über ihre Inhalts-Filter und über `auth.UserCanSeeGame` (Bypass) abgedeckt.
- [x] 5.3 Tests: `TestPush_FremdEventKeinEmpfaenger` (Außenstehender hat keine Sichtbarkeit) und `TestPush_TrainerImmerEmpfaenger` (Bypass) in `internal/games/push_visibility_test.go`.

  _Commit:_ `feat(notifications): Push-Empfänger an Event-Sichtbarkeit ankoppeln`

## 6. Backend-Tests (Routen)

- [x] 6.1 `TestListGames_FilterEigeneTeams`, `TestListGames_TrainerSeesAllGames` (Rename), `TestListGames_ElternSiehtTeamsDerKinder` — in `internal/games/event_visibility_routes_test.go` und `handler_test.go`.
- [x] 6.2 `TestGetGame_FremdEvent_404`, `TestGetGame_EigenesEvent_200`.
- [x] 6.3 `TestGetParticipants_FremdEvent_404`.
- [x] 6.4 `TestDashboard_NaechsteTermine_Filter` in `internal/dashboard/event_visibility_test.go`, `TestCalendar_Filter` in `internal/calendar/event_visibility_test.go`.
- [x] 6.5 `TestCarpooling_FremdGame_404` in `internal/carpooling/event_visibility_test.go`.

  _Commit:_ `test: Event-Sichtbarkeit – Listen, Detail, Dashboard, Carpooling`

## 7. Bestandstests fitten

- [x] 7.1 Bestehende Tests: nur 4 Test-Fixtures mussten ergänzt werden — `TestListGames_TrainerSeesOnlyOwnTeamGames` (rewritten), die drei cross-team-Tests in `participants_crossteam_test.go` (DB-Funktion gesetzt, single-team-Außenstehender erwartet jetzt 404) sowie 2 Elternteil-Carpooling-Tests (Team-Membership ergänzt). Die übrigen Bestandstests verwenden admin-Tokens (Bypass) oder hatten ohnehin Team-Membership.

  _Commit:_ `test: Bestandstests an Event-Sichtbarkeits-Regel anpassen`

## 8. Verifikation & Archive

- [ ] 8.1 `/verify-change` (Build/Test/Lint + Invariants).
- [ ] 8.2 OpenSpec-Proposal archivieren.

  _Commit:_ `chore(openspec): event-team-visibility archivieren`
