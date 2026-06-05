## 1. Training Handler

- [x] 1.1 `hasTeamAccess` in `internal/trainings/handler.go` auf `kader_trainers`-Join umstellen
- [x] 1.2 `ListSeries`-WHERE-Clause (non-admin) auf `kader_trainers`-Subquery umstellen
- [x] 1.3 `ListSessions`-WHERE-Clause (trainer-branch) auf `kader_trainers`-Subquery umstellen

## 2. Members Handler

- [x] 2.1 Mitgliederlisten-Query (Count + SELECT) in `internal/members/handler.go` von `team_trainers`-Join auf `kader_trainers`-Subquery umstellen

## 3. Auth Handler

- [x] 3.1 `notifyTrainersOfRequest` in `internal/auth/handler.go` von `team_trainers`-Join auf `kader_trainers`-Join umstellen

## 4. Scheduler

- [x] 4.1 Duty-Reminder-Query in `internal/scheduler/scheduler.go` (case "trainer"): `team_trainers`-Join und `u.role = 'trainer'`-Filter durch `kader_trainers`- und `member_club_functions`-Joins ersetzen

## 5. AssignTrainer entfernen

- [x] 5.1 `AssignTrainer`-Handler aus `internal/config/handler.go` entfernen
- [x] 5.2 Route `POST /api/admin/teams/{id}/assign-trainer` aus `cmd/teamwerk/main.go` entfernen

## 6. Migration

- [x] 6.1 `internal/db/migrations/010_drop_team_trainers.up.sql` anlegen: `DROP TABLE team_trainers`
- [x] 6.2 `internal/db/migrations/010_drop_team_trainers.down.sql` anlegen: Tabelle leer wiederherstellen
