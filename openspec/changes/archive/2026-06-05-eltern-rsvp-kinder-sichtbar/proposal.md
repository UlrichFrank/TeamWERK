## Why

Eltern können auf `/termine` nicht sehen, ob ihre Kinder Trainings oder Spiele zu- oder abgesagt haben — und können es auch nicht stellvertretend tun, weil der Submit für die Rolle `elternteil` still fehlschlägt. Kinder, die aktive Unterstützung der Eltern brauchen, fallen durch dieses Netz komplett.

## What Changes

- **Neu:** Eltern sehen auf der Terminliste pro Kind einen eigenen RSVP-Bereich (Name + Status + Zu-/Absagen-Buttons)
- **Fix:** RSVP-Submit für `elternteil` sendet `member_id` mit und hat sichtbares Fehler-Handling
- **Fix:** Eingegebene Kommentare (Absage-/Vielleicht-Grund) werden korrekt gespeichert und sind für den Eintragsersteller sichtbar
- **Neu:** Detail-Seite (`/termine/training/:id`, `/termine/spiel/:id`) zeigt die vollständige Kaderliste auch für Spieler und Eltern — nicht nur für Trainer
- **Regel:** Kommentare sind rollenabhängig sichtbar: Trainer → alle, Spieler → eigener, Elternteil → Kinder

## Capabilities

### New Capabilities

- `eltern-rsvp`: Per-Kind-RSVP auf der Terminliste — Sichtbarkeit und Bearbeitung für Eltern

### Modified Capabilities

- `games`: Detail-Ansicht eines Spiels gibt vollständige Kaderliste zurück (bisher nur Responder); reason-Filterung nach Rolle
- `termine-detail`: Detail-Ansicht eines Trainings gibt vollständige Kaderliste für alle authentifizierten User zurück; reason-Filterung nach Rolle

## Impact

- `internal/trainings/handler.go`: `ListSessions` (+ `children_rsvp`), `GetAttendances` (Zugangsprüfung + `reason`-Feld)
- `internal/games/handler.go`: `ListMyGames` (+ `children_rsvp`), `ListGameResponses` (vollständige Kaderliste statt nur Responder)
- `web/src/pages/TerminePage.tsx`: per-Kind-RSVP-UI für Eltern, `member_id` im Submit, Fehler-Handling
- `web/src/pages/TermineDetailPage.tsx`: Attendances für alle User laden, `reason` aus Attendances-Response
- Keine neuen Abhängigkeiten, keine DB-Migrationen erforderlich
