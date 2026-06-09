## 1. DB-Migration

- [x] 1.1 Migration `032_chat_system_messages.up.sql` anlegen: `ALTER TABLE messages ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT 0;`
- [x] 1.2 Migration `032_chat_system_messages.down.sql` anlegen mit Kommentar `-- no down migration (ALTER ADD COLUMN not reversible in SQLite)`

## 2. Backend — Cross-Device SSE

- [x] 2.1 In `SendMessage` den Sender aus der Ausschluss-Liste entfernen: `activeMembers(r, convID, claims.UserID)` → `activeMembers(r, convID, 0)` (alle aktiven Mitglieder inkl. Sender)
- [x] 2.2 In `EditMessage` dieselbe Änderung wie 2.1 vornehmen (SSE-Event an alle inkl. Sender)
- [x] 2.3 In `DeleteMessage` dieselbe Änderung wie 2.1 vornehmen (SSE-Event an alle inkl. Sender)

## 3. Backend — Leave-Notification

- [x] 3.1 In `LeaveConversation` nach dem UPDATE eine System-Nachricht einfügen: `INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, 'hat die Gruppe verlassen', 1)`
- [x] 3.2 In `LeaveConversation` nach der System-Nachricht alle verbleibenden aktiven Mitglieder per SSE benachrichtigen: `BroadcastToUser(uid, fmt.Sprintf("chat:member-left:%d", convID))`

## 4. Backend — Message-Serialisierung

- [x] 4.1 Im `Message`-Struct in `ListMessages` das Feld `IsSystem bool` ergänzen (`json:"isSystem"`)
- [x] 4.2 In der `ListMessages`-Query `is_system` in der SELECT-Liste und im `Scan`-Aufruf ergänzen

## 5. Frontend — Cross-Device Event-Handling

- [x] 5.1 In `useChatEvents` das Event `chat:member-left:{convId}` behandeln: Konversationsliste neu laden und — wenn die betroffene Konversation gerade aktiv ist — Nachrichten neu laden

## 6. Frontend — System-Nachrichten rendern

- [x] 6.1 In `ChatPage.tsx` beim Rendern der Nachrichten-Liste prüfen ob `msg.isSystem === true` und in diesem Fall ein zentriertes graues Label `{msg.senderName} hat die Gruppe verlassen` anzeigen statt einer Chat-Bubble
