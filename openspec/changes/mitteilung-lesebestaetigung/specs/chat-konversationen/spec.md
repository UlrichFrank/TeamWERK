## ADDED Requirements

### Requirement: Tab-Leiste trennt ungelesene Chats von ungelesenen Mitteilungen

Das System SHALL in der Tab-Leiste der Chat-Seite je einen Ungelesen-Badge an beiden Tabs
anzeigen: am Tab „Chats" die Summe der `unreadCount`-Werte aller Konversationen, am Tab
„Mitteilungen" die Anzahl der ungelesenen, nicht selbst gesendeten Mitteilungen. Ein Badge
mit dem Wert 0 SHALL nicht dargestellt werden.

Beide Zahlen SHALL aus den ohnehin geladenen Listen abgeleitet werden; es SHALL keine
zusätzliche Anfrage entstehen. Die Summe beider Zahlen SHALL weiterhin dem Nav-Badge und
dem App-Icon-Badge entsprechen, deren Darstellung als **eine** Zahl unverändert bleibt —
die Web Badging API kann keine Aufteilung abbilden.

#### Scenario: Beide Tabs tragen ihren Anteil

- **WHEN** ein User 3 ungelesene Konversations-Nachrichten und 2 ungelesene Mitteilungen hat und die Chat-Seite öffnet
- **THEN** trägt der Tab „Chats" den Badge `3` und der Tab „Mitteilungen" den Badge `2`
- **AND** der App-Icon-Badge steht weiterhin auf `5`

#### Scenario: Leerer Anteil zeigt keinen Badge

- **WHEN** ein User 4 ungelesene Konversations-Nachrichten und keine ungelesenen Mitteilungen hat
- **THEN** trägt der Tab „Chats" den Badge `4`
- **AND** am Tab „Mitteilungen" wird kein Badge dargestellt
