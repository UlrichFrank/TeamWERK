## MODIFIED Requirements

### Requirement: Nachrichten einer Konversation abrufen

Das System SHALL die letzten 100 Nachrichten einer Konversation zurückgeben (absteigend nach `sent_at`, im Frontend umgekehrt angezeigt). Zu jeder Nachricht werden geliefert: `id`, `senderId`, `senderName`, `body`/`preview` (leer wenn gelöscht oder reine Bildnachricht), `mediaId` (null wenn kein Bild), `mediaUrl` (null wenn kein Bild; sonst `"/media/<mediaId>"` ohne `/api`-Prefix), `mediaWidth` (nur bei Bild-Nachrichten mit bekannter Dimension; sonst weggelassen), `mediaHeight` (nur bei Bild-Nachrichten mit bekannter Dimension; sonst weggelassen), `sentAt`, `replyToId`, `replyToBody`, `replyToSenderName`, `editedAt`, `deletedAt`, `isSystem`, `reactions`.

#### Scenario: Mitglied ruft Nachrichten ab

- **WHEN** ein Mitglied `GET /api/chat/conversations/{id}/messages` aufruft
- **THEN** gibt der Server bis zu 100 Nachrichten zurück, jeweils inkl. `mediaId` und `mediaUrl`

#### Scenario: Nachricht mit Bild und bekannten Dimensionen

- **WHEN** eine Nachricht mit `media_id` abgerufen wird, deren `media`-Zeile `width=1200`, `height=800` hat
- **THEN** enthält das Nachrichtenobjekt `mediaId`, `mediaUrl = "/media/<mediaId>"`, `mediaWidth=1200`, `mediaHeight=800`

#### Scenario: Nachricht mit Bild ohne bekannte Dimensionen (Bestand vor Backfill oder unlesbarer Header)

- **WHEN** eine Nachricht mit `media_id` abgerufen wird, deren `media`-Zeile `width IS NULL` hat
- **THEN** enthält das Nachrichtenobjekt `mediaId`, `mediaUrl`; `mediaWidth` und `mediaHeight` fehlen im JSON-Objekt

#### Scenario: Nachricht ohne Bild

- **WHEN** eine Nachricht ohne `media_id` abgerufen wird
- **THEN** sind `mediaId` und `mediaUrl` beide null; `mediaWidth`/`mediaHeight` fehlen

#### Scenario: Nicht-Mitglied wird abgewiesen

- **WHEN** ein User der nicht Mitglied der Konversation ist die Nachrichten abruft
- **THEN** antwortet der Server mit HTTP 403
