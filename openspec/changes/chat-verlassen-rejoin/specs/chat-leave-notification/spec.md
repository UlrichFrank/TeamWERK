## ADDED Requirements

### Requirement: System-Nachricht beim Verlassen eines Direkt-Chats

Wenn ein User einen Direkt-Chat löscht (`DELETE /api/chat/conversations/{id}`), SHALL das System eine System-Nachricht mit `body = "hat diesen Chat verlassen"` und `is_system = 1` in der Conversation anlegen. Ist der andere User noch aktiv (left_at IS NULL), SHALL er ein SSE-Event `chat:member-left:{convId}` erhalten. Sind nach dem Verlassen 0 aktive Mitglieder übrig, SHALL die Conversation dauerhaft gelöscht werden (physisches DELETE).

#### Scenario: A löscht Direkt-Chat, B ist noch aktiv

- **WHEN** User A `DELETE /api/chat/conversations/{id}` aufruft und B's `left_at IS NULL`
- **THEN** wird A's `left_at` auf die aktuelle Zeit gesetzt
- **THEN** wird in `messages` ein Eintrag mit `sender_id = A`, `body = "hat diesen Chat verlassen"` und `is_system = 1` angelegt
- **THEN** erhält B ein SSE-Event `chat:member-left:{convId}` und sieht die System-Nachricht im Verlauf
- **THEN** ist der Chat für A aus der Konversationsliste verschwunden, für B aber weiterhin sichtbar

#### Scenario: Beide Seiten haben gelöscht — Conversation wird entfernt

- **WHEN** User A `DELETE /api/chat/conversations/{id}` aufruft und B ebenfalls bereits `left_at IS NOT NULL` hat
- **THEN** wird die Conversation dauerhaft aus der Datenbank gelöscht (inklusive aller Nachrichten via CASCADE)

## MODIFIED Requirements

### Requirement: Darstellung von System-Nachrichten im Frontend

Das Frontend SHALL `is_system = true`-Nachrichten nicht als Chat-Bubble sondern als zentriertes, grau gefärbtes Inline-Label rendern. Der angezeigte Text SHALL `{senderName} {msg.body}` sein — der Body kommt direkt aus der Datenbank, womit unterschiedliche System-Nachrichten (Gruppe verlassen, Direkt-Chat verlassen) ohne Änderungen am Frontend-Code dargestellt werden können.

#### Scenario: Rendering einer Gruppen-System-Nachricht

- **WHEN** `GET /api/chat/conversations/{id}/messages` eine Nachricht mit `isSystem: true` und `body = "hat die Gruppe verlassen"` zurückgibt
- **THEN** wird diese Nachricht zentriert als `<senderName> hat die Gruppe verlassen` gerendert (kein Bubble, kein Avatar)

#### Scenario: Rendering einer Direkt-Chat-System-Nachricht

- **WHEN** `GET /api/chat/conversations/{id}/messages` eine Nachricht mit `isSystem: true` und `body = "hat diesen Chat verlassen"` zurückgibt
- **THEN** wird diese Nachricht zentriert als `<senderName> hat diesen Chat verlassen` gerendert (kein Bubble, kein Avatar)
