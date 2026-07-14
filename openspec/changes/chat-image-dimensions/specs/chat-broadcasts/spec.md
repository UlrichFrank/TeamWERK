## MODIFIED Requirements

### Requirement: Broadcasts abrufen

Das System SHALL die sichtbaren Broadcasts eines Users zurückgeben. Zu jedem Broadcast werden geliefert: `id`, `senderName`, `body`, `mediaId` (null wenn kein Bild), `mediaUrl` (null wenn kein Bild; sonst `"/media/<mediaId>"`), `mediaWidth` (nur bei Bild-Broadcasts mit bekannter Dimension; sonst weggelassen), `mediaHeight` (nur bei Bild-Broadcasts mit bekannter Dimension; sonst weggelassen), `sentAt`, `isRead`, `isSent`, `editedAt`.

#### Scenario: Broadcast mit Bild und bekannten Dimensionen

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und ein Broadcast `media_id` gesetzt hat, dessen `media`-Zeile `width=800`, `height=600` hat
- **THEN** enthält das Broadcast-Objekt `mediaId`, `mediaUrl = "/media/<mediaId>"`, `mediaWidth=800`, `mediaHeight=600`

#### Scenario: Broadcast mit Bild ohne bekannte Dimensionen

- **WHEN** ein Broadcast mit `media_id` abgerufen wird, dessen `media`-Zeile `width IS NULL` hat
- **THEN** enthält das Broadcast-Objekt `mediaId`, `mediaUrl`; `mediaWidth` und `mediaHeight` fehlen im JSON-Objekt

#### Scenario: Broadcast ohne Bild abrufen

- **WHEN** ein Broadcast ohne `media_id` abgerufen wird
- **THEN** sind `mediaId` und `mediaUrl` beide null; `mediaWidth`/`mediaHeight` fehlen
