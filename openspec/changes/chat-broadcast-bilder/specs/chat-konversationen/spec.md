## MODIFIED Requirements

### Requirement: Nachrichten einer Konversation abrufen

Das System SHALL die letzten 100 Nachrichten einer Konversation zurückgeben (absteigend nach `sent_at`, im Frontend umgekehrt angezeigt). Zu jeder Nachricht werden geliefert: `id`, `senderId`, `senderName`, `body`/`preview` (leer wenn gelöscht oder reine Bildnachricht), `mediaId` (null wenn kein Bild), `mediaUrl` (null wenn kein Bild; sonst `"/media/<mediaId>"` ohne `/api`-Prefix), `sentAt`, `replyToId`, `replyToBody`, `replyToSenderName`, `editedAt`, `deletedAt`, `isSystem`, `reactions`.

#### Scenario: Mitglied ruft Nachrichten ab

- **WHEN** ein Mitglied `GET /api/chat/conversations/{id}/messages` aufruft
- **THEN** gibt der Server bis zu 100 Nachrichten zurück, jeweils inkl. `mediaId` und `mediaUrl`

#### Scenario: Nachricht mit Bild

- **WHEN** eine Nachricht mit gesetztem `media_id` abgerufen wird
- **THEN** enthält das Nachrichtenobjekt `mediaId` mit der ID und `mediaUrl` = `"/media/<mediaId>"`

#### Scenario: Nachricht ohne Bild

- **WHEN** eine Nachricht ohne `media_id` abgerufen wird
- **THEN** sind `mediaId` und `mediaUrl` beide null

#### Scenario: Nicht-Mitglied wird abgewiesen

- **WHEN** ein User der nicht Mitglied der Konversation ist die Nachrichten abruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Nachricht senden

Das System SHALL das Senden einer Nachricht erlauben. Der Request kann optional `replyToId` und/oder `mediaId` enthalten. Mindestens `body` (nicht leer) **oder** `mediaId` MUSS vorhanden sein. Ein angegebenes `mediaId` MUSS auf eine existierende `media`-Zeile verweisen. Die referenzierte Nachricht bei `replyToId` MUSS zur selben Konversation gehören. Nach erfolgreichem Speichern SHALL der Server via SSE alle aktiven Mitglieder benachrichtigen und Push an Offline-Mitglieder senden.

#### Scenario: Textnachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ "body": "Hallo!" }` aufruft
- **THEN** wird die Nachricht gespeichert, HTTP 201 zurückgegeben und ein SSE-Event `chat:new-message:<id>` verteilt

#### Scenario: Reine Bildnachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ "body": "", "mediaId": <id> }` aufruft
- **THEN** wird die Nachricht mit `media_id` und leerem Body gespeichert und HTTP 201 zurückgegeben

#### Scenario: Bild mit Text kombiniert

- **WHEN** ein Mitglied `{ "body": "Schaut mal", "mediaId": <id> }` sendet
- **THEN** wird eine Nachricht mit `body` und `media_id` gespeichert

#### Scenario: Leere Nachricht ohne Bild wird abgelehnt

- **WHEN** ein User eine Nachricht mit leerem `body` und ohne `mediaId` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Ausgetretenes Mitglied kann nicht senden

- **WHEN** ein User der die Gruppe verlassen hat eine Nachricht sendet
- **THEN** antwortet der Server mit HTTP 403
