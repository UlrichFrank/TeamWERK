## MODIFIED Requirements

### Requirement: Nachrichten einer Konversation abrufen
Das System SHALL beim Abruf von Nachrichten einer Konversation zu jeder Nachricht folgende Felder zurückgeben: `id`, `senderId`, `senderName`, `body` (leer wenn gelöscht), `sentAt`, `replyToId` (null wenn kein Reply), `replyToBody` (null oder "[Nachricht gelöscht]"), `replyToSenderName` (null wenn kein Reply), `editedAt` (null wenn nicht bearbeitet), `deletedAt` (null wenn nicht gelöscht).

#### Scenario: Normale Nachricht ohne Reply
- **WHEN** GET `/api/chat/conversations/{id}/messages` aufgerufen wird
- **THEN** enthält jede Nachricht ohne Reply die Felder `replyToId: null`, `replyToBody: null`, `replyToSenderName: null`, `editedAt: null`, `deletedAt: null`

#### Scenario: Nachricht mit Reply-Referenz
- **WHEN** eine Nachricht mit `reply_to_id` abgerufen wird
- **THEN** sind `replyToBody` und `replyToSenderName` mit den Werten der Ursprungsnachricht befüllt

#### Scenario: Reply auf gelöschte Nachricht
- **WHEN** eine Nachricht mit `reply_to_id` abgerufen wird, die Ursprungsnachricht aber `deleted_at IS NOT NULL`
- **THEN** ist `replyToBody = "[Nachricht gelöscht]"` und `replyToSenderName` bleibt erhalten

#### Scenario: Gelöschte Nachricht in der Liste
- **WHEN** eine Nachricht mit `deleted_at IS NOT NULL` abgerufen wird
- **THEN** ist `body` ein leerer String und `deletedAt` enthält den Lösch-Timestamp

### Requirement: Nachricht in Konversation senden
Das System SHALL beim Senden einer Nachricht ein optionales Feld `replyToId` akzeptieren. Ist `replyToId` angegeben, MUSS die referenzierte Nachricht zur selben Konversation gehören.

#### Scenario: Nachricht ohne Reply senden
- **WHEN** POST `/api/chat/conversations/{id}/messages` ohne `replyToId` aufgerufen wird
- **THEN** wird die Nachricht ohne Reply-Referenz gespeichert

#### Scenario: Nachricht mit Reply senden
- **WHEN** POST `/api/chat/conversations/{id}/messages` mit gültigem `replyToId` aufgerufen wird
- **THEN** wird `messages.reply_to_id` auf den angegebenen Wert gesetzt

#### Scenario: Ungültige Reply-Referenz
- **WHEN** `replyToId` auf eine Nachricht in einer anderen Konversation zeigt
- **THEN** antwortet das Backend mit HTTP 400
