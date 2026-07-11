## MODIFIED Requirements

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

### Requirement: Broadcasts abrufen

Das System SHALL die sichtbaren Broadcasts eines Users zurückgeben. Zu jedem Broadcast werden geliefert: `id`, `senderName`, `body`, `mediaId` (null wenn kein Bild), `mediaUrl` (null wenn kein Bild; sonst `"/media/<mediaId>"`), `sentAt`, `isRead`, `isSent`, `editedAt`.

#### Scenario: Broadcast mit Bild abrufen

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und ein Broadcast `media_id` gesetzt hat
- **THEN** enthält das Broadcast-Objekt `mediaId` und `mediaUrl` = `"/media/<mediaId>"`

#### Scenario: Broadcast ohne Bild abrufen

- **WHEN** ein Broadcast ohne `media_id` abgerufen wird
- **THEN** sind `mediaId` und `mediaUrl` beide null
