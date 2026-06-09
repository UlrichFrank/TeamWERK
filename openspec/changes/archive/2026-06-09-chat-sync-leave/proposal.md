## Why

Chats sind zwar bereits serverseitig gespeichert, aber Nachrichten, die auf Gerät A gesendet werden, erscheinen auf Gerät B desselben Nutzers nicht in Echtzeit — der Sender wird aktuell von SSE-Broadcasts ausgeschlossen. Zudem fehlt beim Verlassen einer Gruppe jede Rückmeldung für die verbleibenden Mitglieder.

## What Changes

- Beim Senden, Bearbeiten und Löschen einer Nachricht erhalten auch alle **eigenen Sessions** des Senders (andere Geräte/Tabs) das SSE-Event, sodass der Chat überall live aktualisiert wird.
- Wenn ein Mitglied eine Gruppe verlässt, wird eine **System-Nachricht** in der Konversation gespeichert und ein SSE-Event an alle verbleibenden Mitglieder gesendet, damit diese den Austritt sofort sehen.

## Capabilities

### New Capabilities

- `chat-leave-notification`: System-Nachricht + SSE-Event wenn ein Mitglied eine Gruppe verlässt, sichtbar für alle verbleibenden Teilnehmer.

### Modified Capabilities

- `chat-konversationen`: Nachrichtenversand, -bearbeitung und -löschung senden das SSE-Event nun auch an alle eigenen Sessions des Senders (cross-device sync). Das Leave-Requirement erhält zusätzlich eine SSE-Notification für verbleibende Mitglieder (via neuer Capability).

## Impact

- **Backend**: `internal/chat/handler.go` — `SendMessage`, `EditMessage`, `DeleteMessage`, `LeaveConversation` jeweils um Sender-SSE bzw. Member-SSE ergänzen; `LeaveConversation` legt zusätzlich eine System-Nachricht in `messages` an.
- **DB**: Kein Schema-Change nötig — System-Nachrichten nutzen einen neuen `sender_id = NULL`-Eintrag in der bestehenden `messages`-Tabelle (bereits nullable laut Schema).
- **Frontend**: `ChatPage.tsx` — `useChatEvents` reagiert auf `chat:member-left:{convId}`, aktualisiert Teilnehmerliste und zeigt System-Nachricht im Chatverlauf. Keine neuen Abhängigkeiten.
