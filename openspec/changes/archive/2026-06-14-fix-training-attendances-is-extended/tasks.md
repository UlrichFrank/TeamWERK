## 1. Backend — attendanceItem und SQL

- [x] 1.1 `IsExtended bool` zu `attendanceItem`-Struct in `internal/trainings/handler.go` hinzufügen (json-Tag: `is_extended`)
- [x] 1.2 SQL in `GetAttendances` auf UNION-Muster umbauen: erster Zweig `player_memberships` → `is_extended=0`, zweiter Zweig `kader_extended_members` → `is_extended=1` mit `NOT EXISTS`-Guard gegen primären Kader
- [x] 1.3 `rows.Scan` in `GetAttendances` um `isExtended int` erweitern und `item.IsExtended = isExtended == 1` setzen

## 2. Frontend — Interface und Mapping

- [x] 2.1 `is_extended?: boolean` zu `AttendanceItem`-Interface in `TermineDetailPage.tsx` hinzufügen
- [x] 2.2 `is_extended: a.is_extended` in der `tableRows`-Map für Trainings (Zeile ~208) hinzufügen

## 3. Test

- [x] 3.1 Test `TestGetAttendances_IsExtended`: Mitglied im primären Kader → `is_extended=false`; Mitglied nur im erweiterten Kader → `is_extended=true`; Mitglied in beiden → erscheint einmal mit `is_extended=false`
