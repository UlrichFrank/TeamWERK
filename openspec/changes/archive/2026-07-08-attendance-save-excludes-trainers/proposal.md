## Why

Trainer können auf `/termine` die Anwesenheit weder bei Trainings noch bei Spielen speichern: Setzt man bei einem Spieler den Haken, erscheint er kurz und verschwindet sofort wieder.

Ursache ist eine Regression aus dem Trainer-RSVP-Feature (eigene Trainer-Sektion in der Teilnahme-/Anwesenheitsliste): Die Roster-Antworten (`GET /training-sessions/{id}/attendances`, `GET /games/{id}/participants`) enthalten seither auch **Trainer-Zeilen** (`is_trainer=1`). Die Anzeige rendert für Trainer bewusst **keine** Checkbox (`TermineDetailPage.tsx:550`), aber `toggleAttendance` (`TermineDetailPage.tsx:265-267`) baut das Speicher-Paket aus dem **kompletten** Roster — inklusive Trainer. Beide Speicher-Endpoints lehnen jedoch ein Paket, das einen Trainer-only-Member enthält, komplett mit **HTTP 400** ab (`trainings/handler.go:1697-1715`, `games/handler.go:2884-2909`). Der Fehler lässt die optimistische Checkbox zurückspringen.

Der Trainer-Claim, `manage_trainings` und die Nav sind intakt — das Problem liegt ausschließlich im Speicher-Pfad, nicht in der Berechtigung.

## What Changes

- **Frontend:** `toggleAttendance` filtert Trainer-Zeilen (`is_trainer`) aus dem Speicher-Paket heraus — analog zum bereits vorhandenen Anzeige-Filter. Gilt für Trainings (`attendances`) und Spiele (`participants`).
- **Backend (Härtung, Defense-in-Depth):** `SaveAttendances` (Trainings und Spiele) **überspringt** Trainer-only-Einträge still, statt das gesamte Paket mit 400 abzulehnen. Ein versehentlich mitgeschickter Trainer darf nie wieder ein ganzes Speichern blockieren. Der 400-Pfad für den Fall „*nur* Trainer-Einträge, sonst nichts" bleibt erhalten (siehe design.md).
- **Test:** Regressionstest je Domäne — Team mit Trainer **und** Spieler, Toggle eines Spielers → 204 und persistierter Spieler-Eintrag; Trainer-Eintrag im selben Paket wird ignoriert, nicht gespeichert.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `training-attendance`: Speichern ignoriert Trainer-Roster-Einträge statt das Paket abzulehnen
- `game-attendance`: Speichern ignoriert Trainer-Roster-Einträge statt das Paket abzulehnen

## Impact

- `web/src/pages/TermineDetailPage.tsx` — `toggleAttendance`: `is_trainer`-Filter beim Aufbau von `ids`
- `internal/trainings/handler.go` — `SaveAttendances`: Trainer-only-Eintrag `continue` statt `http.Error(400)`
- `internal/games/handler.go` — `SaveAttendances`: dito
- Tests: `internal/trainings/handler_test.go` (bzw. `trainer_rsvp_test.go`), `internal/games/attendance_test.go`
- Kein Schema-, kein Migrations-, kein Routen-Change.
