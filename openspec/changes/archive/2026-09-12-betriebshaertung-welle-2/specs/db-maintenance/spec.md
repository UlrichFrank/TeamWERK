## Purpose

Die SQLite-Datenbank bleibt im Betrieb kompakt.

## ADDED Requirements

### Requirement: Wöchentlicher WAL-Checkpoint

Der Scheduler MUST einmal pro Woche `PRAGMA wal_checkpoint(TRUNCATE)` ausführen und Dauer sowie Ergebnis protokollieren. Der Job MUST idempotent sein (mehrfacher Aufruf im Zeitfenster führt ihn einmal aus).

#### Scenario: Checkpoint läuft
- **WHEN** der Scheduler im Wartungsfenster läuft
- **THEN** wird der Checkpoint ausgeführt und eine Log-Zeile mit Dauer geschrieben
