## MODIFIED Requirements

### Requirement: Nachrichten einer Konversation abrufen

Das System SHALL die letzten 100 Nachrichten einer Konversation zurückgeben (absteigend nach `sent_at`, dann im Frontend umgekehrt anzeigen). Zu jeder Nachricht werden folgende Felder geliefert: `id`, `senderId`, `senderName`, `body` (leer wenn gelöscht oder reine Bildnachricht), `imageUrl` (null wenn kein Bild), `sentAt`, `replyToId` (null wenn kein Reply), `replyToBody` (null oder „[Nachricht gelöscht]"), `replyToSenderName`, `editedAt` (null wenn nicht bearbeitet), `deletedAt` (null wenn nicht gelöscht).

#### Scenario: Mitglied ruft Nachrichten ab

- **WHEN** ein Mitglied `GET /api/chat/conversations/{id}/messages` aufruft
- **THEN** gibt der Server bis zu 100 Nachrichten zurück mit allen o.g. Feldern inkl. `imageUrl`

#### Scenario: Normale Nachricht ohne Reply

- **WHEN** eine Nachricht ohne Reply abgerufen wird
- **THEN** sind `replyToId`, `replyToBody`, `replyToSenderName`, `editedAt`, `deletedAt`, `imageUrl` alle null

#### Scenario: Nachricht mit Bild

- **WHEN** eine Nachricht mit gesetztem `image_url` in der DB abgerufen wird
- **THEN** enthält das Nachrichtenobjekt `imageUrl` mit dem Pfad zum Bild

#### Scenario: Nachricht mit Reply-Referenz

- **WHEN** eine Nachricht mit `reply_to_id` abgerufen wird
- **THEN** sind `replyToBody` und `replyToSenderName` mit den Werten der Ursprungsnachricht befüllt

#### Scenario: Reply auf gelöschte Ursprungsnachricht

- **WHEN** eine Nachricht mit `reply_to_id` abgerufen wird, die Ursprungsnachricht aber gelöscht ist
- **THEN** ist `replyToBody = "[Nachricht gelöscht]"` und `replyToSenderName` bleibt erhalten

#### Scenario: Gelöschte Nachricht in der Liste

- **WHEN** eine Nachricht mit `deleted_at IS NOT NULL` abgerufen wird
- **THEN** ist `body` ein leerer String und `deletedAt` enthält den Lösch-Timestamp

#### Scenario: Nicht-Mitglied wird abgewiesen

- **WHEN** ein User der nicht Mitglied der Konversation ist die Nachrichten abruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Nachricht senden

Das System SHALL das Senden einer Nachricht in einer Konversation erlauben. Der Request kann optional `replyToId` und/oder `imageUrl` enthalten. Mindestens `body` (nicht leer) oder `imageUrl` MUSS vorhanden sein. Die referenzierte Nachricht bei `replyToId` MUSS zur selben Konversation gehören. Nach erfolgreichem Speichern SHALL der Server via SSE alle aktiven Mitglieder (left_at IS NULL) benachrichtigen und Push Notifications an Mitglieder senden die gerade offline sind.

#### Scenario: Textnachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ body: "Hallo!" }` aufruft
- **THEN** wird die Nachricht gespeichert und HTTP 201 zurückgegeben
- **THEN** erhalten alle anderen aktiven Mitglieder ein SSE-Event `chat:new-message:<conversationId>`

#### Scenario: Bildnachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ body: "", imageUrl: "/api/chat/images/<uuid>.jpg" }` aufruft
- **THEN** wird die Nachricht mit `image_url` gespeichert und HTTP 201 zurückgegeben

#### Scenario: Nachricht mit Reply senden

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit gültigem `replyToId` aufruft
- **THEN** wird `messages.reply_to_id` auf den angegebenen Wert gesetzt

#### Scenario: Ungültige Reply-Referenz

- **WHEN** `replyToId` auf eine Nachricht in einer anderen Konversation zeigt
- **THEN** antwortet das Backend mit HTTP 400

#### Scenario: Leere Nachricht ohne Bild wird abgelehnt

- **WHEN** ein User eine Nachricht mit leerem `body` und ohne `imageUrl` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Ausgetretenes Mitglied kann nicht senden

- **WHEN** ein User der die Gruppe verlassen hat eine Nachricht sendet
- **THEN** antwortet der Server mit HTTP 403
