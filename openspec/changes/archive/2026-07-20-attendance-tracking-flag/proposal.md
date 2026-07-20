## Why

Wenn ein Trainer die Anwesenheit im `/termine`-Detail zum ersten Mal öffnet und **ein** Kind ankreuzt, sendet das Frontend ein Bulk-Paket, das für **alle** anderen Kinder `present=false` mitschreibt. Ergebnis: die anderen Kinder erscheinen auf `/profil/kind/{id}` sofort als „abwesend" statt als „—" (nicht erfasst). Auch das Weg-Klicken einer einmal gesetzten Anwesenheit lässt eine `present=0`-Row zurück und zählt in der Statistik als Fehlen. Es fehlt ein Weg, eine Anwesenheits-Erfassung eines Termins **explizit als „noch nicht bewertet"** zu markieren bzw. wieder darauf zurückzusetzen.

## What Changes

- Neue Spalte `attendance_tracked INTEGER NOT NULL DEFAULT 0` an `training_sessions` und `games`.
- `SaveAttendances` (Trainings + Spiele) setzt `attendance_tracked = 1` als Teil derselben Transaktion, sobald mindestens ein Spieler-Eintrag persistiert wird.
- **Neue Routen** `DELETE /api/training-sessions/{id}/attendance-tracking` und `DELETE /api/games/{id}/attendance-tracking` setzen `attendance_tracked = 0`. Vorhandene `attendance`-Rows bleiben unverändert liegen (Undo durch erneutes Klicken möglich).
- **Aggregations-Änderung** (`internal/attendance/handler.go`): Für Sessions/Spiele mit `attendance_tracked = 0` werden `training_attendances`/`game_attendances`-Rows in der Statistik ignoriert — die Termine erscheinen dann wieder auf der „offen zu erfassen"-Liste und die Kategorien pro Mitglied zeigen `unknown` (Anzeige „—").
- **UI** (`web/src/pages/TermineDetailPage.tsx`): neuer „Erfassung zurücksetzen"-Button im Trainer-View, sichtbar wenn `attendance_tracked=true`. Session/Game-Response um `attendance_tracked: boolean` erweitert. Broadcast bleibt `attendance-changed`.
- **Backfill (Migration)**: Alle Sessions/Spiele, für die bereits ≥ 1 `attendance`-Row existiert, bekommen `attendance_tracked=1` — damit ändert sich das Verhalten historischer Daten nicht.

## Capabilities

### New Capabilities
_Keine — die bestehenden Attendance-Capabilities werden erweitert._

### Modified Capabilities
- `training-attendance`: neue Route `DELETE /api/training-sessions/{id}/attendance-tracking` + Auto-Set-Semantik beim Erst-Save.
- `game-attendance`: neue Route `DELETE /api/games/{id}/attendance-tracking` + Auto-Set-Semantik beim Erst-Save.
- `attendance-statistics`: Klassifikation & „offene Erfassung"-Liste respektieren `attendance_tracked=0` (Rows werden ignoriert bzw. Termin gilt als offen).

## Impact

- **Migration**: `internal/db/migrations/N_attendance_tracked.up.sql` + `.down.sql` (Spalte + Backfill).
- **Go**: `internal/trainings/handler.go` (SaveAttendances, GetSession, neuer Reset-Handler), `internal/games/handler.go` (SaveAttendances, GetGame, neuer Reset-Handler), `internal/app/router.go` (zwei neue Routen unter Trainer-/Vorstand-Tier), `internal/attendance/handler.go` (SQL-Filter in `loadCounts`, `loadMemberEvents`, `GetTeamOpen`).
- **Frontend**: `web/src/pages/TermineDetailPage.tsx` (Reset-Button + Response-Feld), Types-Datei falls vorhanden.
- **Tests**: neue Route (Happy + 403 + 404), Aggregation ignoriert Rows bei tracked=0, Auto-Set beim Save, Backfill-Assertion.
- **Broadcast-Gate**: Reset-Handler ruft `Broadcast("attendance-changed")` — keine Allowlist-Änderung nötig.
- **Kein neues externes System, kein RAM-Impact.**
