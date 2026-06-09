# chat-leave-notification Specification

## Purpose
System-Nachrichten und SSE-Events wenn ein Mitglied eine Gruppen-Konversation verlässt.
## Requirements
### Requirement: System-Nachricht beim Gruppenaustritt

Wenn ein Mitglied eine Gruppe verlässt, SHALL das System automatisch eine System-Nachricht in der Konversation speichern. Die System-Nachricht SHALL `is_system = 1` haben, `sender_id` des austretenden Users und einen festen Body "hat die Gruppe verlassen". Alle verbleibenden Mitglieder (left_at IS NULL nach dem Austritt) SHALL ein SSE-Event `chat:member-left:{convId}` empfangen.

#### Scenario: Mitglied verlässt Gruppe — System-Nachricht erscheint

- **WHEN** ein Mitglied `DELETE /api/chat/conversations/{id}/members/me` aufruft
- **THEN** wird in `messages` ein Eintrag mit `sender_id = <ausgetretener User>`, `body = "hat die Gruppe verlassen"` und `is_system = 1` angelegt
- **THEN** erhalten alle verbleibenden aktiven Mitglieder ein SSE-Event `chat:member-left:{convId}`
- **THEN** erscheint im Chatverlauf der verbleibenden Mitglieder die System-Nachricht als zentriertes Label (kein Bubble)

#### Scenario: System-Nachricht wird bei anderen Clients sofort sichtbar

- **WHEN** Mitglied A die Gruppe verlässt während Mitglied B aktiv die Konversation geöffnet hat
- **THEN** empfängt Mitglied B das SSE-Event `chat:member-left:{convId}` und lädt Nachrichten + Teilnehmerliste neu
- **THEN** sieht Mitglied B im Chatverlauf "A hat die Gruppe verlassen"

#### Scenario: Austretender User sieht keine weiteren Nachrichten

- **WHEN** ein Mitglied die Gruppe verlässt
- **THEN** erscheint die Konversation nicht mehr in seiner Konversationsliste
- **THEN** erhält er keine SSE-Events für diese Konversation mehr

### Requirement: Darstellung von System-Nachrichten im Frontend

Das Frontend SHALL `is_system = true`-Nachrichten nicht als Chat-Bubble sondern als zentriertes, grau gefärbtes Inline-Label rendern. Der angezeigte Text SHALL `{senderName} hat die Gruppe verlassen` sein.

#### Scenario: Rendering einer System-Nachricht

- **WHEN** `GET /api/chat/conversations/{id}/messages` eine Nachricht mit `isSystem: true` zurückgibt
- **THEN** wird diese Nachricht ohne Absender-Avatar, ohne Bubble und zentriert als `<senderName> hat die Gruppe verlassen` gerendert
