## Status

**Umgesetzt** auf Branch `feat/mannschaft-aufgaben-strafen` (Worktree). Backend + Frontend grün:
`go build ./...`, `go test ./...`, golangci-lint, `pnpm build`, `pnpm lint`, Broadcast-/Arch-/Permission-Gate, `openspec validate` — alle bestanden.

**Bewusste Abweichungen vom ursprünglichen Plan:**
- Gates liegen **inline im `internal/teams`-Package** (`access.go`), nicht in `internal/policy/rules.go` — `policy` führt kein `*sql.DB`, und die Checks sind DB-Lookups (wie `GetRoster` seinen Zugriffscheck schon inline macht).
- **Kein neues Package**: Aufgaben/Strafen leben in `internal/teams` (`responsibilities.go`, `penalties.go`) → keine `arch_test`-Klassifizierung nötig.
- Roster liefert `responsibilities` als **`[{id,label}]`** (statt nur Label), damit der Trainer jede Zuweisung direkt vom Chip entfernen kann.
- Zusätzliches Harness-Gate gepflegt: `internal/permissions/matrix_test.go` (Permission-Matrix-Drift-Check).

## 1. Datenmodell & Migration

- [x] 1.1 Migration `031_mannschaft_aufgaben_strafen.up.sql` + `.down.sql`
- [x] 1.2 `responsibility_types(id, kader_id, label)` UNIQUE(kader_id,label)
- [x] 1.3 `penalty_types(id, kader_id, reason, default_amount_cent)`
- [x] 1.4 `kader_strafenwarte(kader_id, member_id)` PK
- [x] 1.5 `member_responsibilities(id, kader_id, member_id, label)` — Snapshot
- [x] 1.6 `team_penalties(id, kader_id, member_id, amount_cent, reason, created_by_member_id, created_at)` — Snapshot, kein Status
- [x] 1.7 `.down.sql` FK-sicher
- [x] 1.8 Migration up/down verifiziert (Teil von `go test ./internal/db/...`)

## 2. Backend — Gates (teams/access.go)

- [x] 2.1 `isTrainerOfTeam` (kader_trainers, admin-Bypass)
- [x] 2.2 `isStrafenwartOfTeam` (kader_strafenwarte)
- [x] 2.3 `canReadPenalties` (Spieler/Trainer/Erweitert; Eltern/Extern → false)
- [x] 2.4 `resolveKaderID` (aktive Saison)
- [x] 2.5 Gates über die Handler-Tests abgedeckt (403-Fälle je Route)

## 3. Backend — Aufgaben

- [x] 3.1 Catalog-CRUD `responsibility_types` (Trainer, Broadcast)
- [x] 3.2 Zuweisung/Entfernen `member_responsibilities` (Snapshot, Trainer, Broadcast)
- [x] 3.3 `GetRoster` liefert memberId + responsibilities[{id,label}] + canManage
- [x] 3.4 Routen in `BuildRouter`
- [x] 3.5 Tests (Trainer 200, Non-Trainer 403, unauth 401, Roster inkl. Eltern, Snapshot)

## 4. Backend — Strafen

- [x] 4.1 `GET /penalties` mit Read-Gate + Kassenstand je Spieler
- [x] 4.2 `POST /penalties` (Strafenwart, editierbarer Betrag, Broadcast)
- [x] 4.3 `DELETE /penalties/{pid}` Storno
- [x] 4.4 `DELETE /penalties?member=` Zurücksetzen je Spieler
- [x] 4.5 Strafen-Catalog-CRUD + Strafenwart-Ernennung (Trainer)
- [x] 4.6 Routen in `BuildRouter`
- [x] 4.7 Read-Gate-Tests (Player/Extended 200, Parent/Outsider 403, Roster ohne Strafen)
- [x] 4.8 Write-Gate-Tests (Strafenwart 200, Non-Strafenwart 403, Fremd-Team 403, Storno, Reset)
- [x] 4.9 Ernennung 200/403, Snapshot-Invariante, `TestClubFunctions_NoStrafenwartValue`

## 5. Test-Fixtures

- [x] 5.1 `internal/testutil/fixtures_teamextras.go` (AppointStrafenwart, CreatePenalty, AssignResponsibility, AddResponsibilityType, AddPenaltyType)
- [x] 5.2 Eltern (family_links) + Erweiterter Kader in den Negativ-/Positiv-Tests abgebildet

## 6. Frontend — MeinTeamPage

- [x] 6.1 Aufgaben-Chips je Spieler (Team-Tab, inkl. Erweiterter Kader)
- [x] 6.2 Strafen-Tab: 403 versteckt Sektion; sonst Kassenstand je Spieler + Team (Cent→€)
- [x] 6.3 Trainer-Verwaltung: Catalog pflegen, Aufgaben zuweisen/entfernen, Strafenwart ernennen
- [x] 6.4 Strafenwart-Aktionen: vergeben (Betrag editierbar), stornieren, je Spieler zurücksetzen (confirm)
- [x] 6.5 `useLiveUpdates` um `responsibilities`/`penalties` erweitert

## 7. Verifikation

- [x] 7.1 `go test ./...` (inkl. Architektur-, Broadcast- + Permission-Gate) grün
- [x] 7.2 golangci-lint grün (brand-Tokens, lucide-Icons, keine Raw-Farben/Emojis)
- [x] 7.3 `pnpm -C web build` + `pnpm -C web lint` grün (0 Errors)
- [x] 7.4 `openspec validate` grün
- [ ] 7.5 Merge nach `main` + Deploy (offen — siehe Zusammenfassung)
