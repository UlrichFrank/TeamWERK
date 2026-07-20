## Why

Auf `/termine` sieht ein Spieler die RSVP-Buttons (Zusagen/Vielleicht/Absagen) für einen Termin nur dann, wenn `my_rsvp !== null` — also nur bei bereits gegebener Antwort oder existierendem Default (`rsvp_default_players != 'none'`). Bei einem Termin mit „Keine automatische Rückmeldung" (`rsvp_default_players='none'`) und noch fehlender Antwort verschwinden die Buttons komplett.

Der Spec verlangt aber (`game-rsvp` → „Kein Default-Status bei Spielen"): „Wenn ein User die `/termine`-Seite aufruft und ein Spiel noch keine Rückmeldung hat, sind alle drei RSVP-Buttons **inaktiv** (kein Button ist hervorgehoben)". „Inaktiv" heißt hier **nicht hervorgehoben** — nicht **unsichtbar**. Der Spieler kann so gar nicht antworten, der Elternteil desselben Kindes hingegen schon (die Kind-Zeilen werden über einen anderen Codepfad — Kader-Zugehörigkeit statt RSVP-Wert — gerendert). Sichtbarkeits-Asymmetrie zwischen Selbst- und Kind-Ansicht.

Zusätzlich: Der Cutoff für Spiel-RSVP liegt aktuell bei **18 Stunden** vor Anpfiff. Das ist unnötig früh — ein Spieler, der um 21:00 am Vorabend krank wird, kann sich nicht mehr für ein 19:00-Spiel des Folgetags abmelden. Die 2h-Regel für Trainings hat sich als praxistauglich erwiesen; für Spiele gilt sie ab jetzt analog.

## What Changes

- **Spiel-RSVP-Cutoff: 18h → 2h** (analog Trainings). Trainer/Vorstand/Admin/sportliche_leitung können den Cutoff weiterhin überschreiben.
- Neues Response-Feld **`am_i_participant: bool`** in `gameListItem`/`gameDetail` und `sessionListItem`/`sessionDetail`. `true`, wenn der aufrufende User selbst im regulären, erweiterten oder Trainer-Kader eines beteiligten Teams des Termins ist.
- Frontend `TerminePage` und `TermineDetailPage` gaten die Sichtbarkeit der eigenen RSVP-Buttons an `am_i_participant` statt an `my_rsvp !== null`. Bei erreichtem Cutoff bleiben die Buttons sichtbar, sind aber `disabled` mit `RsvpLockNotice`.
- Kind-Zeilen (Eltern-Ansicht) bleiben unverändert, weil sie schon heute korrekt kader-basiert gerendert werden.

## Capabilities

### Modified Capabilities

- **`game-rsvp`** — Cutoff-Requirement von 18h auf 2h, neues Requirement „Teilnehmer sehen RSVP-Buttons unabhängig von Response" mit API-Feld `am_i_participant`.
- **`training-rsvp`** — neues Requirement für `am_i_participant` in Session-Response (Cutoff bleibt 2h).

## Impact

- `internal/games/handler.go` — Konstante `GameRSVPCutoff = 2 * time.Hour`, Response-Structs `gameListItem`/`gameDetail` um `AmIParticipant`, Fehlermeldung in `writeRSVPLocked` für Games.
- `internal/trainings/handler.go` — Response-Structs `sessionListItem`/`sessionDetail` um `AmIParticipant`; SQL um drei `EXISTS`-Subqueries erweitert (Kader-Mitgliedschaft des Aufrufers).
- `internal/games/handler_test.go`, `internal/games/cutoff_test.go` — bestehende 18h/12h-Testzeiten auf 2h/30min umziehen; neue Tests für `am_i_participant`.
- `internal/trainings/handler_test.go` — neue Tests für `am_i_participant`.
- `web/src/pages/TerminePage.tsx` — Typen `TrainingSession`/`GameSummary` um `am_i_participant`, `showOwn`-Berechnung + outer condition (2 Stellen: Training/Game), Cutoff-Text/RsvpLockNotice-Text für Games.
- `web/src/pages/TermineDetailPage.tsx` — analog.
- `openspec/specs/game-rsvp/spec.md`, `openspec/specs/training-rsvp/spec.md` — MODIFIED/ADDED per Delta.
