## 1. Auth-Helper

- [x] 1.1 In `internal/auth/claims.go` Methode `func (c *Claims) HasAnyFunction(fns ...string) bool` ergänzen, die true zurückgibt, wenn mindestens eine der übergebenen Funktionen in `c.ClubFunctions` enthalten ist.
- [x] 1.2 Unit-Test `TestClaims_HasAnyFunction` in `internal/auth/claims_test.go` für leere Liste, kein Match, ein Match, mehrere Funktionen.

## 2. Backend: Team-Quelle erweitern

- [x] 2.1 In `internal/duties/handler.go` `Board()` die `whereParts`-SQL (Zeile ~355-380) so anpassen, dass die team-`IN`-Subquery zusätzlich `trainer_memberships` per UNION enthält (Spieler-/Familien-/Trainer-Teams kombiniert). Argumente entsprechend ergänzen.
- [x] 2.2 Die game-lose Sub-Query (`ds.team_id IS NULL AND ds.game_id IN (...)`) ebenfalls um `trainer_memberships` erweitern, damit Trainer auch game-lose Slots zu Spielen ihrer Teams sehen.
- [x] 2.3 commit `feat(duties): Dienstbörse zeigt Trainern Dienste ihrer trainierten Teams`

## 3. Backend: Audience-Filter umbauen

- [x] 3.1 In `Board()` den `audienceBypass`-Block (Zeile ~339-347 + 384-403) ersetzen: Bypass nur für `claims.Role == "admin"` oder wenn `claims.HasAnyFunction("vorstand","vorstand_beisitzer","trainer","sportliche_leitung") && r.URL.Query().Get("audience") == "all"`.
- [x] 3.2 Bestehende Audience-Filter-WHERE-Klausel bleibt unverändert, wird aber jetzt auch für privilegierte Funktionen angehängt, solange `?audience=all` nicht gesetzt ist.
- [x] 3.3 commit `feat(duties): Audience-Filter ist für Trainer/Vorstand standardmäßig aktiv und über ?audience=all deaktivierbar`

## 4. Backend-Tests

- [x] 4.1 Test `TestDutyBoard_TrainerSeesOwnTeam` in `internal/duties/handler_test.go`: Trainer ohne Spieler-Status sieht Slots seines Teams.
- [x] 4.2 Test `TestDutyBoard_TrainerDoesNotSeeOtherTeams`: Trainer sieht keine Slots von Teams, die er nicht trainiert und in denen er nicht spielt.
- [x] 4.3 Test `TestDutyBoard_TrainerAudienceFilterDefault`: Trainer ohne `?audience` sieht nur Slots mit Audience-Match oder NULL, **nicht** Slots mit `audiences=["spieler"]`.
- [x] 4.4 Test `TestDutyBoard_TrainerAudienceAll`: Trainer mit `?audience=all` sieht alle Audiences seiner Teams.
- [x] 4.5 Test `TestDutyBoard_VorstandAudienceFilterDefault`: Vorstand ohne `?audience` sieht nur Audience-Matches; mit `?audience=all` alle.
- [x] 4.6 Test `TestDutyBoard_SpielerAudienceAllIgnored`: Spieler mit `?audience=all` sieht trotzdem nur Audience-gefilterte Slots (Parameter ignoriert).
- [x] 4.7 Test `TestDutyBoard_AdminAudienceBypass`: Admin sieht alle Audiences unabhängig vom Parameter.
- [x] 4.8 commit `test(duties): Audience-Filter und Trainer-Sicht auf Dienstbörse`

## 5. Frontend: Pille hinzufügen

- [x] 5.1 In `web/src/pages/DutyPage.tsx`: `parseFilters` um `audience: 'mine' | 'all'` (default `'mine'`) erweitern, `updateFilter` analog. Default-Wert `'mine'` schreibt keinen Param; `'all'` schreibt `?audience=all`.
- [x] 5.2 In `load()` die URL bauen: `audience=all` an `/duty-board` anhängen, wenn Filter inaktiv ist (kombinierbar mit bestehendem `?view=mine`).
- [x] 5.3 Neue Pille mit `Filter`-Icon (Lucide, `import { Filter }`) in der Toggle-Gruppe rechts neben „Meine"/„Vergangene" rendern. Pille nur rendern, wenn `hasFunction(user, fn)` für eine der vier privilegierten Funktionen true ist (Helper `hasAnyFunction` ggf. ergänzen).
- [x] 5.4 Label „Nur Audience" im Vollformat, im Compact-Modus nur Icon. Aktiv-Stil identisch zu Meine/Vergangene.
- [x] 5.5 commit `feat(duties): Audience-Filter-Pille auf Dienste-Seite für Trainer und Vorstand`

## 6. Verifikation

- [x] 6.1 `go test ./internal/duties/... ./internal/auth/...` lokal grün (modulo bestehender Pre-Fail `TestClaimDutySlot_NoConcurrentOverclaim`).
- [x] 6.2 `pnpm --filter web build` grün (keine TypeScript-Fehler).
- [ ] 6.3 Manuell mit Trainer-Account auf `/dienste` prüfen: eigenes Team-Spiel + Slots sichtbar; Pille umschalten zeigt mehr/weniger Slots wie erwartet.
- [ ] 6.4 Manuell mit Spieler-Account prüfen: Pille **nicht** sichtbar; Verhalten unverändert.
- [ ] 6.5 `openspec archive duty-board-trainer-audience` nach Merge zum Archivieren der Spec-Deltas in `openspec/specs/duties/spec.md`.
