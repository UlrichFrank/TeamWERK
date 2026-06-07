## MODIFIED Requirements

### Requirement: Broadcasts abrufen
Das System SHALL beim Abruf von Broadcast-Mitteilungen zu jeder Mitteilung das Feld `editedAt` (null wenn nicht bearbeitet) zurückgeben.

#### Scenario: Unbearbeitete Mitteilung
- **WHEN** GET `/api/chat/broadcasts` aufgerufen wird und eine Mitteilung nie bearbeitet wurde
- **THEN** ist `editedAt: null` in der Antwort

#### Scenario: Bearbeitete Mitteilung
- **WHEN** GET `/api/chat/broadcasts` aufgerufen wird und eine Mitteilung bearbeitet wurde
- **THEN** enthält `editedAt` den Timestamp der letzten Bearbeitung
