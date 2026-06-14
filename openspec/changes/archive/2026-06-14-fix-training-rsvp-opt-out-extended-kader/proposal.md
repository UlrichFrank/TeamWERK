## Why

`GetAttendances` wendet `rsvp_opt_out` (Auto-Bestätigung für alle Mitglieder ohne explizite Absage) auf primären und erweiterten Kader gleichermassen an. Für den primären Kader ist das korrekt — es ist ihr Training. Für den erweiterten Kader ist es falsch: diese Spieler sind situativ verfügbar, nicht automatisch eingeplant. Ihr RSVP-Status soll ausschliesslich explizite Rückmeldungen spiegeln.

## What Changes

- **Backend**: `rsvp_opt_out`-Auto-Confirm in `GetAttendances` greift nur noch für primäre Kader-Mitglieder (`is_extended = false`). Erweiterte Kader-Mitglieder zeigen immer ihren tatsächlichen Response-Status — `null` wenn keine Rückmeldung vorliegt.

## Capabilities

### New Capabilities

*(keine)*

### Modified Capabilities

- `training-attendance`: Auto-Confirm via `rsvp_opt_out` gilt nur für primären Kader, nie für erweiterten Kader.

## Impact

- `internal/trainings/handler.go` — eine Zeile in der Scan-Schleife von `GetAttendances`
- Keine DB-Migration, keine Frontend-Änderung, kein neuer Endpoint
