## Why

Ein Dienst-Slot, der über das Spieltag-Detail-Modal an einem Termin mit **mehreren**
Teams angelegt wird, bekommt im Frontend hart das **erste** Team des Termins
(`SpieltagDetailModal.tsx`: `team_id: game.teams?.[0]?.id`). Damit greift in der
Dienstbörse nur noch der Zweig `ds.team_id IN (meine Teams)` — die übrigen Teams des
Termins und deren Eltern sehen den Dienst nie, obwohl `duties/handler.go` für genau
diesen Fall bereits einen `team_id IS NULL`-Fallback über `game_teams` besitzt.

Beobachtet an einem realen Termin (generisches Event mit drei Kadern): alle elf Slots
tragen `team_id` der A-Jugend; ein Spieler der B-Jugend sieht 0 von 11, über den
Fallback wären es 11 von 11.

`team_id = NULL` ist an einem game-gebundenen Slot **nicht** „vereinsweit", sondern
„die Teams des Termins" — für die Sichtbarkeit (Dienstbörse, Dashboard) und für das
SSE-Publikum (`hub.Audience.TeamIDsForDutySlot`) ist das bereits korrekt umgesetzt.
Zwei Benachrichtigungs-Pfade fehlen jedoch: `duties.eligibleDutyUsers` und
`scheduler.eligibleUsers` schauen ausschließlich auf `team_id` und fallen bei NULL auf
den **ganzen Verein** zurück. Ohne diese beiden Stellen würde der Sichtbarkeits-Fix die
Push-/Mail-Zustellung vereinsweit streuen.

## What Changes

- `SpieltagDetailModal` sendet bei einem Termin mit mehr als einem Team `team_id: null`
  (bei genau einem Team unverändert dessen ID) — der bestehende `game_id`-Fallback löst
  die Sichtbarkeit dann über `game_teams` auf.
- `duties.eligibleDutyUsers` nimmt eine Team-Menge statt eines einzelnen Teams; bei
  `team_id = NULL` und gesetztem `game_id` wird sie aus `game_teams` abgeleitet.
- `scheduler.eligibleUsers` (Dienst-Erinnerung, T-2) leitet die Empfänger bei
  `team_id = NULL` und gesetztem `game_id` ebenfalls aus `game_teams` ab. Nur Slots
  ohne Team **und** ohne Spiel bleiben vereinsweit.
- Datenkorrektur für Bestands-Slots (`team_id` → NULL bei game-gebundenen Slots an
  Mehr-Team-Terminen) — separat, nach Bestätigung.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `duties`: manuell angelegte Slots an Mehr-Team-Terminen tragen kein Team
- `duty-reminder-emails`: Empfängerbestimmung bei `team_id IS NULL` berücksichtigt `game_id`
- `push-duties`: Empfängermenge bei `POST /api/duty-slots` ohne `team_id`

## Impact

- `web/src/components/SpieltagDetailModal.tsx` — `team_id` nur bei genau einem Team
- `internal/duties/handler.go` — `eligibleDutyUsers` auf Team-Menge, Auflösung über `game_teams`
- `internal/scheduler/scheduler.go` — `openSlot.gameID`, Team-Scope in `eligibleUsers`
- Tests: `internal/duties/handler_test.go`, `internal/scheduler/*_test.go`,
  `web/src/components/__tests__/SpieltagDetailModal.test.tsx`
