## Why

`GET /api/training-sessions/{id}/attendances` unterscheidet nicht zwischen primärem Kader und erweitertem Kader — alle Mitglieder landen in einer Gruppe ohne `is_extended`-Flag. Die Trainings-Detailseite kann den erweiterten Kader daher nicht separat anzeigen, obwohl `ResponseTable` bereits korrekt nach `is_extended` filtert.

## What Changes

- **Backend**: `attendanceItem`-Struct erhält `IsExtended bool`; SQL in `GetAttendances` wird auf UNION-Muster umgebaut (primärer Kader → `is_extended=0`, erweiterter Kader → `is_extended=1`, Duplikate via `NOT EXISTS`-Guard verhindert).
- **Frontend**: `AttendanceItem`-Interface erhält `is_extended?: boolean`; `tableRows`-Mapping für Trainings reicht das Feld durch.

## Capabilities

### New Capabilities

*(keine)*

### Modified Capabilities

- `training-attendances`: API-Response enthält neu `is_extended` — erweiterter Kader wird separat ausgewiesen.

## Impact

- `internal/trainings/handler.go` — `attendanceItem`, `GetAttendances`
- `web/src/pages/TermineDetailPage.tsx` — `AttendanceItem`, `tableRows`-Mapping
- Kein neuer Endpoint, keine DB-Migration, keine API-Breaking-Change (additives Feld)
