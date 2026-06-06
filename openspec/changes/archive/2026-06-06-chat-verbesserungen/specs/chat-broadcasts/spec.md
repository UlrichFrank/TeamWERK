## MODIFIED Requirements

### Requirement: Broadcast-Liste zeigt nur sichtbare Mitteilungen
Die Broadcast-Liste eines Users SHALL nur Mitteilungen anzeigen, die er noch nicht ausgeblendet hat (`hidden_at IS NULL` in `broadcast_reads`). Neu: `broadcast_reads` erhält die Spalte `hidden_at DATETIME`.

#### Scenario: Ausgeblendeter Broadcast nicht mehr sichtbar
- **WHEN** ein User eine Broadcast-Mitteilung ausgeblendet hat (`hidden_at` gesetzt)
- **THEN** erscheint diese Mitteilung nicht mehr in `GET /api/chat/broadcasts`

#### Scenario: Neue Broadcasts weiterhin sichtbar
- **WHEN** ein neuer Broadcast eintrifft
- **THEN** ist `hidden_at` für alle Empfänger NULL und die Mitteilung erscheint in ihren Listen

## ADDED Requirements

### Requirement: Mobile Broadcast-Detailansicht
Beim Antippen einer Broadcast-Mitteilung auf Mobile SHALL die Detailansicht (rechtes Panel) korrekt geöffnet werden.

#### Scenario: Broadcast-Detail auf Mobile öffnen
- **WHEN** ein User auf einem Mobilgerät eine Broadcast-Mitteilung antippt
- **THEN** wird das rechte Panel mit der vollständigen Mitteilung angezeigt (mobileShowChat = true)
- **THEN** ist der gesamte Nachrichtentext lesbar (whitespace-pre-wrap, kein Truncate)

#### Scenario: Zurück zur Liste auf Mobile
- **WHEN** der User im Broadcast-Detail auf "Zurück" (X-Button) tippt
- **THEN** kehrt die Ansicht zur Broadcast-Liste zurück (mobileShowChat = false, activeBroadcast = null)
