## Why

Spieler im erweiterten Kader sehen aktuell keine Trainingseinheiten ihres Teams, obwohl sie bei Spielen bereits korrekt eingeschlossen sind. Die Sichtbarkeit, Benachrichtigungen und die Anwesenheitsliste berücksichtigen nur `player_memberships` (= Hauptkader), nicht `kader_extended_members`.

## What Changes

- Erweiterter-Kader-Spieler können Trainings ihres Teams in der Liste sehen
- Erweiterter-Kader-Spieler können ihr RSVP für Trainings abgeben (der Respond-Handler hat keine Kader-Prüfung, daher reicht Sichtbarkeit)
- Erweiterter-Kader-Spieler erscheinen in der Anwesenheitsliste (`GetAttendances`)
- Erweiterter-Kader-Spieler erhalten Push-/E-Mail-Benachrichtigungen bei neu angelegten Trainings

## Capabilities

### New Capabilities
- `erweiterter-kader-trainings-access`: Spieler im erweiterten Kader erhalten denselben Zugang zu Trainingseinheiten wie Hauptkader-Spieler (Sichtbarkeit, RSVP, Anwesenheitsliste, Benachrichtigungen bei neuen Terminen)

### Modified Capabilities

Keine — das ist eine Bugfix-artige Erweiterung; die bestehende Kader-Semantik ändert sich nicht.

## Impact

- `internal/trainings/handler.go`: drei Stellen
  - `teamMembersAndParents()`: UNION auf `kader_extended_members` für Benachrichtigungen
  - `ListTrainingSessions` Team-Filter: UNION auf `kader_extended_members` für Spieler-Sichtbarkeit
  - `GetAttendances`: UNION auf `kader_extended_members` damit Erw.-Kader in der Liste erscheint
- Keine API-Änderungen, keine neuen Routen, kein Frontend-Änderungsbedarf
- Keine DB-Migration erforderlich (`kader_extended_members` existiert seit Migration 021)
