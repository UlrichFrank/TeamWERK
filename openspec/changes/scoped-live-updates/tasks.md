# Tasks — scoped-live-updates

> Umbau des SSE-Fan-outs auf adressierten Versand. Topic-weise, jede Phase testbar. Backend-only (Frontend-Vertrag unverändert). Nicht migrierte Topics bleiben global lauffähig.

## 1. Hub-Infrastruktur

- [ ] 1.1 `internal/hub/hub.go`: `BroadcastToUsers(userIDs []int, event string)` ergänzen (dedupliziert, nicht-blockierend wie `BroadcastToUser`).
- [ ] 1.2 `internal/hub/handler.go`: `Events` auf `SubscribeUser(claims.UserID)`/`UnsubscribeUser` umstellen; Claims aus dem authentifizierten Tier lesen.
- [ ] 1.3 Tests: `TestHub_BroadcastToUsers_OnlyTargets`, `TestEvents_SubscribesPerUser`.

  _Commit:_ `feat(hub): adressierter Domänen-Event-Versand (BroadcastToUsers, per-user /api/events)`

## 2. Audience-Resolver

- [ ] 2.1 Entscheidung A vs. B (siehe `design.md`) treffen; Resolver-Helfer anlegen, ohne `internal/arch/arch_test.go` zu verletzen.
- [ ] 2.2 Generische Auflösungen: Finance-Gruppe (`vorstand`/`vorstand_beisitzer`/`kassierer` + admin); Team-Audience (Team-Mitglieder + Trainer/sL + Vorstand) aus bekannten Team-IDs.
- [ ] 2.3 Tests des Resolvers (Finance-Gruppe, Team-Audience, leere Menge).

  _Commit:_ `feat(hub): Audience-Resolver für rollen-/team-basiertes Event-Scoping`

## 3. Topic `members`/`users`

- [ ] 3.1 Alle `Broadcast("members")`/`Broadcast("users")`-Aufrufe (`internal/members`, `internal/auth`, …) auf Finance-Gruppen-Audience umstellen (+ betroffener Nutzer bei Eigenprofil).
- [ ] 3.2 Test: `TestMembersMutation_ScopedToVorstand` (Spieler-Stream erhält KEIN `members`-Event).

  _Commit:_ `feat(members): members/users-Events nur an Vorstand/Kassierer/Admin`

## 4. Topic `games`/`trainings`

- [ ] 4.1 `Broadcast("games")`/`Broadcast("trainings")` (+ `attendance-changed`/`event-note`, wo game/training-gebunden) auf Team-Audience umstellen.
- [ ] 4.2 Test: `TestGamesMutation_ScopedToTeamAndStaff` (teamfremder Spieler erhält kein Event).

  _Commit:_ `feat(games,trainings): Spiel-/Trainings-Events nur an betroffene Teams + Staff`

## 5. Topic `kader`/`duties`/`absences`

- [ ] 5.1 `Broadcast("kader")`/`Broadcast("duties")`/`Broadcast("absences")` auf Team-Audience umstellen.
- [ ] 5.2 Tests je Topic (Happy + teamfremder Nutzer bekommt nichts).

  _Commit:_ `feat(kader,duties): kader-/duties-/absences-Events team-gescopet`

## 6. Global bleibende Topics absichern

- [ ] 6.1 `venues`/`settings`/`beitragssatz-changed`/`stammvereine` bewusst bei `Broadcast` (global) belassen; im Code kurz kommentieren.
- [ ] 6.2 Test: `TestSettingsMutation_StaysGlobal` (Event erreicht weiterhin alle Streams).

  _Commit:_ `test(hub): vereinsweite Topics bleiben global`

## 7. Abschluss

- [ ] 7.1 `make test` (inkl. Architektur-Test) + `/verify-change`.
- [ ] 7.2 `openspec validate scoped-live-updates --strict`.
- [ ] 7.3 Proposal archivieren.

  _Commit:_ `chore(hub): archiviere scoped-live-updates`
