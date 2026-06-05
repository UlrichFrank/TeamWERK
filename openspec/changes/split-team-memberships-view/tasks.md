## 1. DB-Migration anlegen

- [x] 1.1 Nächste freie Migrationsnummer ermitteln (aktuell: 016 → neue Nummer: 017)
- [x] 1.2 `internal/db/migrations/017_split_membership_views.up.sql` anlegen mit `CREATE VIEW player_memberships` (nur `kader_members`) und `CREATE VIEW trainer_memberships` (nur `kader_trainers`)
- [x] 1.3 `internal/db/migrations/017_split_membership_views.down.sql` anlegen mit `DROP VIEW IF EXISTS player_memberships; DROP VIEW IF EXISTS trainer_memberships;`

## 2. Go-Queries umstellen: trainings/handler.go

- [x] 2.1 `GetAttendances` (Zeile 945): `team_memberships` → `player_memberships` (Bug-Fix)
- [x] 2.2 `ListSessions` rsvp_opt_out-Zählung (Zeile 621): `team_memberships tm2` → `player_memberships tm2`
- [x] 2.3 `GetSession` rsvp_opt_out-Zählung (Zeile 722): `team_memberships tm2` → `player_memberships tm2`
- [x] 2.4 `ListSessions` Team-Filter für Spieler (Zeile 601): `team_memberships tm` → `player_memberships tm`
- [x] 2.5 `ListSessions` Team-Filter für Eltern (Zeile 598): `team_memberships tm` → `player_memberships tm`

## 3. Go-Queries umstellen: duties/handler.go

- [x] 3.1 Zeile 286: `team_memberships tm` → `player_memberships tm`
- [x] 3.2 Zeile 298: `team_memberships tm2` → `player_memberships tm2`

## 4. Go-Queries umstellen: members/handler.go

- [x] 4.1 Zeile 139: `team_memberships tm` → `player_memberships tm`
- [x] 4.2 Zeile 165: `team_memberships tm` → `player_memberships tm`

## 5. Go-Queries umstellen: scheduler/scheduler.go

- [x] 5.1 Zeile 148: `team_memberships tm` → `player_memberships tm`
- [x] 5.2 Zeile 172: `team_memberships tm` → `player_memberships tm`

## 6. Verifikation

- [x] 6.1 `make migrate-up` lokal ausführen — keine Fehler
- [x] 6.2 `go build ./...` — keine Compile-Fehler
- [ ] 6.3 Manuell prüfen: `/termine/training/242` zeigt keine Trainer mehr in der Teilnahmeliste
- [x] 6.4 `grep -rn "team_memberships" internal/ cmd/` — nur noch `games/handler.go` und die Migration selbst sollten treffen
