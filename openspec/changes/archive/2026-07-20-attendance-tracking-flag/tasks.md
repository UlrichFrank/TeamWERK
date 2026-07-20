## 1. Migration + Schema

- [x] 1.1 `internal/db/migrations/036_attendance_tracked.up.sql`: `ALTER TABLE training_sessions ADD COLUMN attendance_tracked INTEGER NOT NULL DEFAULT 0`, dasselbe für `games`, plus Backfill `UPDATE … SET attendance_tracked=1 WHERE EXISTS (…attendance-Row…)` für beide Tabellen
- [x] 1.2 `036_attendance_tracked.down.sql`: `ALTER TABLE … DROP COLUMN attendance_tracked` für beide Tabellen
- [x] 1.3 `make migrate-up` lokal ausführen und via `sqlite3` prüfen, dass Bestandsdaten korrekt backfilled sind

## 2. Backend — Trainings

- [x] 2.1 `internal/trainings/handler.go` `SaveAttendances`: Zähler `wroteAny` einführen (increment nach jedem erfolgreichen Upsert, der weder Trainer-only noch unavailable ist); nach dem Loop `if wroteAny { UPDATE training_sessions SET attendance_tracked=1 WHERE id=? }` in derselben Tx
- [x] 2.2 Neuer Handler `ResetAttendanceTracking` in `internal/trainings/handler.go`: Authz-Check via `hasTeamAccess`, `UPDATE training_sessions SET attendance_tracked=0 WHERE id=?`, Broadcast `attendance-changed`, HTTP 204
- [x] 2.3 Session-GET-Response (`GetSession` bzw. Detail-Handler) um `attendance_tracked` erweitern
- [x] 2.4 `internal/app/router.go`: `r.Delete("/api/training-sessions/{id}/attendance-tracking", trainH.ResetAttendanceTracking)` im Trainer-/Vorstand-Tier
- [x] 2.5 Tests in `internal/trainings/`:
  - Happy: Reset → 204, DB-Flag=0, Rows bleiben
  - Reset idempotent (zweimal)
  - 403 für fremdes Team
  - 404 für nicht-existierende Session
  - Erst-Save setzt Flag; No-op-Save (nur Trainer/unavailable) setzt NICHT

## 3. Backend — Games

- [x] 3.1 `internal/games/handler.go` `SaveAttendances`: analoges `wroteAny`-Muster + `UPDATE games SET attendance_tracked=1 WHERE id=?`
- [x] 3.2 Neuer Handler `ResetAttendanceTracking` in `internal/games/handler.go`: Authz-Check wie SaveAttendances, `UPDATE games SET attendance_tracked=0`, Broadcast `attendance-changed`, HTTP 204
- [x] 3.3 Game-GET-Response um `attendance_tracked` erweitern
- [x] 3.4 `internal/app/router.go`: `r.Delete("/api/games/{id}/attendance-tracking", gameH.ResetAttendanceTracking)` im entsprechenden Tier
- [x] 3.5 Tests in `internal/games/`: analoges Set (Happy/Idempotent/403/404/401/Save-Set)

## 4. Backend — Aggregation

- [x] 4.1 `internal/attendance/handler.go` `loadCounts` (Trainings-SQL + Games-SQL): `AND ts.attendance_tracked=1` bzw. `AND g.attendance_tracked=1` in der LEFT-JOIN-Bedingung der `attendances`, sodass `ta.present`/`ga.present` als NULL erscheinen wenn Flag=0 (bestehende Classify-Logik greift dann)
- [x] 4.2 `loadMemberEvents`: derselbe Filter im Trainings- und Games-Query
- [x] 4.3 `GetTeamOpen`: `NOT EXISTS`-Prädikat durch `attendance_tracked=0` ersetzt (Trainings + Games)
- [x] 4.4 Tests in `internal/attendance/`:
  - Session mit `attendance_tracked=0` + `present=0`-Row → IGNORIERT (kein missed)
  - `GetTeamOpen` liefert eine reset-e Session (tracked=0)

## 5. Backend — Scheduler

- [x] 5.1 `internal/scheduler/attendance_reminders.go`: SQL-Muster angleichen (`attendance_tracked=0` statt `NOT EXISTS (attendance)`)
- [x] 5.2 Test in `internal/scheduler/attendance_reminders_test.go`: nach Reset erscheint die Session wieder in den Kandidaten (bestehender Test blieb grün nach Fixture-Anpassung — RecordTrainingAttendance setzt jetzt automatisch tracked=1)

## 6. Frontend

- [x] 6.1 `web/src/pages/TermineDetailPage.tsx`: `SessionDetail`/`GameDetail`-Types um `attendance_tracked: boolean` erweitert
- [x] 6.2 „Erfassung zurücksetzen"-Button (nur `isTrainer && isPast && attendance_tracked`); ruft `DELETE /training-sessions/{id}/attendance-tracking` bzw. `/games/{id}/attendance-tracking`, bei Erfolg `load(true)` + `loadAttendances`
- [x] 6.3 Confirm-Dialog vor Reset

## 7. Verification

- [x] 7.1 `openspec validate attendance-tracking-flag --strict` → grün
- [x] 7.2 `go test ./...` → 1706 passed
- [x] 7.3 `pnpm -C web test` → 617 passed
- [x] 7.4 `pnpm -C web build` → grün
- [x] 7.5 `pnpm -C web lint` → keine neuen Warnings (bestehende Warnings unberührt)
