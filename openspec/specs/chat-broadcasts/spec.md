# chat-broadcasts Specification

## Purpose
Einweg-Mitteilungen an Zielgruppen (alle, Team, Rolle). Sender kann Broadcasts bearbeiten und löschen.
## Requirements
### Requirement: Broadcast senden

Das System SHALL es Usern mit Rolle admin, vorstand oder trainer erlauben, eine Mitteilung an eine Zielgruppe zu senden. Trainer dürfen nur an ihr eigenes Team senden (`target_type: team`). Admin und Vorstand dürfen `target_type: all`, `team` oder `role`. Der Request kann optional `mediaId` enthalten; mindestens `body` (nicht leer) **oder** `mediaId` MUSS vorhanden sein. Ein angegebenes `mediaId` MUSS auf eine existierende `media`-Zeile verweisen. Nach dem Senden werden alle matching User benachrichtigt (SSE `chat:new-broadcast` + Push).

#### Scenario: Admin sendet Text-Broadcast an alle

- **WHEN** ein Admin `POST /api/chat/broadcasts` mit `{ "body": "Wichtige Info", "targetType": "all" }` aufruft
- **THEN** wird der Broadcast gespeichert, alle aktiven User erhalten ein SSE-Event `chat:new-broadcast`, HTTP 201

#### Scenario: Reine Bild-Mitteilung senden

- **WHEN** ein Vorstand `POST /api/chat/broadcasts` mit `{ "body": "", "mediaId": <id>, "targetType": "all" }` aufruft
- **THEN** wird der Broadcast mit `media_id` und leerem Body gespeichert, HTTP 201

#### Scenario: Leere Mitteilung ohne Bild wird abgelehnt

- **WHEN** ein berechtigter User einen Broadcast mit leerem `body` und ohne `mediaId` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Unberechtigter User

- **WHEN** ein User ohne admin/vorstand/trainer einen Broadcast sendet
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Empfangene Broadcasts abrufen

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

### Requirement: Broadcast als gelesen markieren

Das System SHALL es Empfängern erlauben einen Broadcast als gelesen zu markieren. Dies beeinflusst den Ungelesen-Badge im Nav.

#### Scenario: Broadcast öffnen markiert als gelesen

- **WHEN** ein User einen Broadcast öffnet und `POST /api/chat/broadcasts/{id}/read` aufruft
- **THEN** wird `broadcast_reads.read_at` für diesen User gesetzt
- **THEN** erscheint der Broadcast als gelesen in der Liste

### Requirement: Kein Rückkanal bei Broadcasts

Das System SHALL keinerlei Reply-Funktionalität für Broadcasts bereitstellen. Der Endpoint zur Konversationserstellung (`POST /api/chat/conversations`) darf nicht über einen Broadcast-Kontext erreichbar sein. Im Frontend wird kein Reply-Eingabefeld angezeigt.

#### Scenario: Kein Reply-Endpoint für Broadcasts

- **WHEN** ein User versucht auf einen Broadcast zu antworten
- **THEN** existiert kein API-Endpoint für diese Aktion (HTTP 404 oder nicht vorhanden)

