## 1. Backend — Team-Scope-Helper

- [x] 1.1 In `internal/games/handler.go` einen Helper `trainerTeamScope(ctx, claims, teamIDs []int) (own int, err error)` anlegen, der zählt, wie viele der übergebenen `team_ids` über `kader_trainers` × `kader` (aktive Saison) zum `claims.UserID` gehören. Muster von `canRecordGameAttendance` übernehmen (Fehler nicht schlucken, `error` zurückgeben).
- [x] 1.2 Helper `isScopedTrainer(claims) bool` ergänzen: `true` nur für Vereinsfunktion `trainer` **ohne** `sportliche_leitung`/`vorstand` und ohne System-Rolle `admin`. Die heutige Inline-Bedingung in `CreateGame` dagegen austauschen (`vorstand` fehlt dort bislang in der Ausnahme — mit übernehmen).
- [x] 1.3 Helper `canMutateGame(ctx, claims, gameID int) (bool, error)`: für `isScopedTrainer` prüfen, ob mindestens eine der aktuell in `game_teams` eingetragenen Mannschaften eine eigene ist; sonst `true`.

## 2. Backend — Routen anpassen

- [x] 2.1 `CreateGame`: bestehende Scope-Prüfung typabhängig machen — `heim`/`auswärts` verlangt `own == len(team_ids)`, `generisch` verlangt `own >= 1`. Bei Verletzung HTTP 403, kein Insert.
- [x] 2.2 `UpdateGame`: vor dem `BeginTx` `canMutateGame` prüfen → 403. Existenzprüfung (404) muss weiterhin vor der 403-Antwort greifen, damit fremde Event-IDs nicht per Statuscode enumeriert werden.
- [x] 2.3 `UpdateGame`: wenn `req.TeamIDs` gesetzt ist, die neuen `team_ids` gegen den Ziel-Event-Typ prüfen (Ziel-Typ = `req.EventType`, falls gesetzt, sonst der gespeicherte `event_type`) — `heim`/`auswärts`: alle eigen; `generisch`: mindestens eine eigen. Bei Verletzung 403 vor jedem Schreibzugriff.
- [x] 2.4 `DeleteGame`: `canMutateGame` prüfen → 403, nach der bestehenden 404-Prüfung und vor dem Cascade-Löschen.
- [x] 2.5 `go build ./...` + `make lint` grün.

## 3. Backend — Tests

- [x] 3.1 Testutil-Fixture prüfen: `makeTrainer` (in `internal/games/attendance_test.go`) für Multi-Team-Setups nutzbar machen bzw. lokalen Helper für „Trainer von Team A, fremdes Team B" ergänzen.
- [x] 3.2 `TestCreateGame_TrainerGenericForeignTeamsAllowed` → 201, `game_teams` enthält eigenes + fremde Teams.
- [x] 3.3 `TestCreateGame_TrainerGenericWithoutOwnTeam` → 403, kein Insert in `games`.
- [x] 3.4 `TestCreateGame_TrainerHomeGameForeignTeam` → 403 (Bestandsverhalten abgesichert).
- [x] 3.5 `TestUpdateGame_TrainerNotOnEvent` → 403, Event unverändert.
- [x] 3.6 `TestUpdateGame_TrainerGenericAddsForeignTeam` → 200, fremdes Team in `game_teams`.
- [x] 3.7 `TestUpdateGame_TrainerRemovesOwnLastTeam` → 403, `game_teams` unverändert.
- [x] 3.8 `TestUpdateGame_SportlicheLeitungUnrestricted` → 200 auf fremdem Event.
- [x] 3.9 `TestDeleteGame_TrainerNotOnEvent` → 403, Event existiert weiterhin.
- [x] 3.10 `TestUpdateGame_UnknownIDReturns404` → 404 (nicht 403) für nicht existierende Event-ID.

## 4. Frontend — GameEditModal

- [x] 4.1 `web/src/components/GameEditModal.tsx`: zusätzlich `GET /teams/names` laden (`clubTeams`). Response-Typ nach `TeamForName` (`id`, `age_class`, `gender`, `team_number`, `group_count`).
- [x] 4.2 Picker-Quelle nach `isGeneric` verzweigen: generisch → `clubTeams`, sonst → `availableTeams` (unverändert gefiltert).
- [x] 4.3 `teamShortNames` aus derselben Liste bauen, aus der gerendert wird, damit fremde Teams ein Label bekommen (`buildTeamShortNames`). Kein Fallback auf `t.name` im generischen Zweig.
- [x] 4.4 Ladezustand: „Lädt…" erst ausblenden, wenn die für den Typ relevante Liste da ist (heute an `availableTeams.length === 0` gekoppelt).
- [x] 4.5 403 beim Speichern von der generischen Fehlermeldung unterscheiden: „Mindestens eine deiner Mannschaften muss am Event beteiligt bleiben." im Alert-Fehler-Stil.

## 5. Frontend — Kalender-Wizard

- [x] 5.1 `web/src/pages/KalenderPage.tsx`: im generischen Checkbox-Zweig (Wizard-Schritt 2) über `allTeamNames` statt `teams` iterieren; Heim-/Auswärts-Single-Select bleibt auf `teams`.
- [x] 5.2 Filter-Dropdown oben auf der Seite bleibt unverändert auf `teams` (nutzergefiltert) — beim Umbau nicht versehentlich mitziehen.
- [x] 5.3 Sortierung der Checkboxen prüfen: `/teams/names` liefert bereits nach `AgeClassSortKey`, `gender`, `team_number` sortiert — keine zusätzliche Client-Sortierung nötig.

## 6. Frontend — Tests

- [x] 6.1 `web/src/components/__tests__/GameEditModal.genericTeams.test.tsx`: bei `event_type='generisch'` erscheinen alle Mannschaften aus `/teams/names` als Checkbox, die beteiligten sind vorausgewählt.
- [x] 6.2 Gegenprobe im selben File: bei `event_type='heim'` enthält das Single-Select nur die Einträge aus `/teams`.
- [x] 6.3 Test für die 403-Fehlermeldung beim Speichern (Modal bleibt offen, spezifischer Text).
- [x] 6.4 `pnpm -C web test` + `pnpm -C web build` + `pnpm -C web lint` grün.

## 7. Abschluss

- [x] 7.1 `make test` (inkl. Architektur- und Broadcast-Gate) grün — `UpdateGame`/`DeleteGame` broadcasten bereits, das Gate darf nicht kippen.
- [x] 7.2 `openspec validate generic-event-all-teams --strict` grün.
- [x] 7.3 `/verify-change` durchlaufen (Route→Tests, brand-Tokens, lucide-Icons, keine neue Migration).
- [ ] 7.4 Ein Commit pro Task-Gruppe, Conventional Commits mit Scope `games` (Backend) bzw. `kalender`/`games` (Frontend).
