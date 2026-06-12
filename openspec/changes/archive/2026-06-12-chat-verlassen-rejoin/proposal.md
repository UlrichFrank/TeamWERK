## Why

Wenn ein User einen Direkt-Chat löscht, sieht das Gegenüber keine Rückmeldung — der Chat verschwindet still. Außerdem entstehen Duplikate wenn jemand nach dem Verlassen wieder einen Direkt-Chat mit derselben Person eröffnet, weil `createDirect` keine partiell verlassenen Threads erkennt.

## What Changes

- `DeleteConversation` (Direkt-Chat): schreibt Systemnachricht „hat diesen Chat verlassen" und broadcastet SSE an noch aktive Mitglieder; löscht die Conversation dauerhaft nur wenn beide Seiten sie verlassen haben
- `createDirect`: findet bestehende Direkt-Konversationen auch wenn A bereits `left_at` gesetzt hat (B ist noch aktiv) — re-joined A statt Duplikat anzulegen; broadcastet SSE an B
- `SendMessage` (Direkt-Chat): wenn B schreibt und A hat `left_at` gesetzt, wird A wiederhergestellt (left_at = NULL) und erhält das `chat:new-message`-SSE
- Frontend: System-Nachrichten-Rendering wird generisch (`{senderName} {msg.body}` statt hardcoded „hat die Gruppe verlassen")

## Capabilities

### New Capabilities

*(keine)*

### Modified Capabilities

- `chat-leave-notification`: Bisherige Spec deckt nur Gruppen ab. Wird erweitert um Direkt-Chat-Verlassen (Systemnachricht + SSE + dauerhafte Löschung wenn beide weg) und generisches Frontend-Rendering.
- `chat-konversationen`: Re-join-Verhalten für `createDirect` (partiell verlassene Threads werden fortgesetzt statt dupliziert) und automatischer Re-join beim Eingang einer Nachricht.

## Impact

- **Backend**: `internal/chat/handler.go` — `DeleteConversation`, `createDirect`, `SendMessage`
- **Frontend**: `web/src/pages/ChatPage.tsx` — System-Nachrichten-Rendering
- **Keine neue Migration** — `is_system`, `left_at` existieren bereits
- **Keine neuen Abhängigkeiten**
