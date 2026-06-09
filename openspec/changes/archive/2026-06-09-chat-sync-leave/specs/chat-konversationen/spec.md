## MODIFIED Requirements

### Requirement: Nachricht senden

Das System SHALL das Senden einer Nachricht in einer Konversation erlauben. Der Request kann optional `replyToId` enthalten. Die referenzierte Nachricht MUSS zur selben Konversation gehören. Nach erfolgreichem Speichern SHALL der Server via SSE **alle** aktiven Mitglieder einschließlich des Senders selbst benachrichtigen (damit andere Geräte/Tabs des Senders die Nachricht erhalten) und Push Notifications an Mitglieder senden die gerade offline sind.

#### Scenario: Nachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ body: "Hallo!" }` aufruft
- **THEN** wird die Nachricht gespeichert und HTTP 201 zurückgegeben
- **THEN** erhalten alle aktiven Mitglieder (einschließlich Sender) ein SSE-Event `chat:new-message:<conversationId>`

#### Scenario: Sender-Gerät B erhält Echtzeit-Update

- **WHEN** der Sender auf Gerät A eine Nachricht sendet und gleichzeitig auf Gerät B eingeloggt ist
- **THEN** empfängt Gerät B das SSE-Event `chat:new-message:{convId}` und zeigt die neue Nachricht an

#### Scenario: Nachricht mit Reply senden

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit gültigem `replyToId` aufruft
- **THEN** wird `messages.reply_to_id` auf den angegebenen Wert gesetzt

#### Scenario: Ungültige Reply-Referenz

- **WHEN** `replyToId` auf eine Nachricht in einer anderen Konversation zeigt
- **THEN** antwortet das Backend mit HTTP 400

#### Scenario: Ausgetretenes Mitglied kann nicht senden

- **WHEN** ein User der die Gruppe verlassen hat eine Nachricht sendet
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Leere Nachricht wird abgelehnt

- **WHEN** ein User eine Nachricht mit leerem `body` sendet
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Nachrichten einer Konversation abrufen

Das System SHALL die letzten 100 Nachrichten einer Konversation zurückgeben (absteigend nach `sent_at`, dann im Frontend umgekehrt anzeigen). Zu jeder Nachricht werden folgende Felder geliefert: `id`, `senderId`, `senderName`, `body` (leer wenn gelöscht), `sentAt`, `replyToId` (null wenn kein Reply), `replyToBody` (null oder „[Nachricht gelöscht]"), `replyToSenderName`, `editedAt` (null wenn nicht bearbeitet), `deletedAt` (null wenn nicht gelöscht), **`isSystem`** (true wenn System-Nachricht).

#### Scenario: Mitglied ruft Nachrichten ab

- **WHEN** ein Mitglied `GET /api/chat/conversations/{id}/messages` aufruft
- **THEN** gibt der Server bis zu 100 Nachrichten zurück mit allen o.g. Feldern inklusive `isSystem`

#### Scenario: Normale Nachricht ohne Reply

- **WHEN** eine Nachricht ohne Reply abgerufen wird
- **THEN** sind `replyToId`, `replyToBody`, `replyToSenderName`, `editedAt`, `deletedAt` alle null und `isSystem` ist false

#### Scenario: System-Nachricht in der Liste

- **WHEN** eine Nachricht mit `is_system = 1` abgerufen wird
- **THEN** ist `isSystem: true` im Response-Objekt gesetzt

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
