# chat-konversationen Specification

## Purpose
TBD - created by archiving change chat-feature. Update Purpose after archive.
## Requirements
### Requirement: Konversationsliste abrufen

Das System SHALL für jeden authentifizierten User eine Liste seiner Konversationen zurückgeben. Jede Konversation SHALL die Felder `id`, `type` (direct|group), `name`, `lastMessage` (body + sent_at), `unreadCount` und die Liste der Teilnehmer enthalten. Ausgetretene Mitglieder (left_at NOT NULL) werden NICHT in der aktiven Teilnehmerliste zurückgegeben.

#### Scenario: Spieler sieht nur eigene Konversationen

- **WHEN** ein Spieler `GET /api/chat/conversations` aufruft
- **THEN** gibt der Server nur Konversationen zurück in denen der Spieler Mitglied ist (left_at IS NULL)

#### Scenario: Konversation mit ungelesenen Nachrichten

- **WHEN** in einer Konversation Nachrichten existieren die der User noch nicht gelesen hat
- **THEN** enthält das Konversations-Objekt `unreadCount > 0`

#### Scenario: Leere Konversationsliste

- **WHEN** ein User noch keine Konversation hat
- **THEN** gibt der Server ein leeres Array zurück (HTTP 200)

### Requirement: Direct-Konversation erstellen oder öffnen

Das System SHALL beim Erstellen einer Direct-Konversation prüfen ob bereits eine Konversation zwischen den beiden Usern existiert. Falls ja, SHALL der bestehende Datensatz zurückgegeben werden (idempotent). Falls nein, wird eine neue Konversation angelegt.

#### Scenario: Erste Direct-Konversation zwischen zwei Usern

- **WHEN** User A `POST /api/chat/conversations` mit `{ type: "direct", userId: B }` aufruft
- **THEN** wird eine neue Konversation angelegt und beide User als Mitglieder eingetragen
- **THEN** gibt der Server HTTP 201 mit dem Konversations-Objekt zurück

#### Scenario: Bestehende Direct-Konversation erneut öffnen

- **WHEN** User A `POST /api/chat/conversations` mit `{ type: "direct", userId: B }` aufruft und bereits eine Direct-Konversation existiert
- **THEN** gibt der Server HTTP 200 mit der bestehenden Konversation zurück (kein Duplikat)

#### Scenario: Spieler kann nur User aus eigenem Team anschreiben

- **WHEN** ein Spieler eine Direct-Konversation mit einem User aus einem anderen Team versucht
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Gruppen-Konversation erstellen

Das System SHALL die Erstellung von Gruppen-Konversationen mit einem frei wählbaren Namen und einer initialen Teilnehmerliste erlauben. Der erstellende User wird automatisch als Mitglied hinzugefügt. Die Sichtbarkeitsregel (wer kann in die Teilnehmerliste aufgenommen werden) entspricht der User-Picker-Filterung.

#### Scenario: Trainer erstellt Gruppen-Chat

- **WHEN** ein Trainer `POST /api/chat/conversations` mit `{ type: "group", name: "Taktik-Runde", memberIds: [2, 3, 4] }` aufruft
- **THEN** wird eine Gruppen-Konversation angelegt mit dem Trainer und den genannten Usern als Mitglieder
- **THEN** gibt der Server HTTP 201 zurück

#### Scenario: Gruppe ohne Namen wird abgelehnt

- **WHEN** ein User eine Gruppen-Konversation ohne `name`-Feld erstellt
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Nachrichten einer Konversation abrufen

Das System SHALL die letzten 100 Nachrichten einer Konversation zurückgeben (absteigend nach `sent_at`, dann im Frontend umgekehrt anzeigen). Nur aktive Mitglieder (left_at IS NULL) oder Mitglieder die ausgetreten sind (left_at NOT NULL) mit Nachrichten die vor ihrem Austritt gesendet wurden können Nachrichten abrufen.

#### Scenario: Mitglied ruft Nachrichten ab

- **WHEN** ein Mitglied `GET /api/chat/conversations/{id}/messages` aufruft
- **THEN** gibt der Server bis zu 100 Nachrichten zurück mit `id`, `senderId`, `senderName`, `body`, `sentAt`

#### Scenario: Nicht-Mitglied wird abgewiesen

- **WHEN** ein User der nicht Mitglied der Konversation ist die Nachrichten abruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Nachricht senden

Das System SHALL das Senden einer Nachricht in einer Konversation erlauben. Nach erfolgreichem Speichern SHALL der Server via SSE alle aktiven Mitglieder (left_at IS NULL) benachrichtigen und Push Notifications an Mitglieder senden die gerade offline sind.

#### Scenario: Nachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ body: "Hallo!" }` aufruft
- **THEN** wird die Nachricht gespeichert und HTTP 201 zurückgegeben
- **THEN** erhalten alle anderen aktiven Mitglieder ein SSE-Event `chat:new-message:<conversationId>`

#### Scenario: Ausgetretenes Mitglied kann nicht senden

- **WHEN** ein User der die Gruppe verlassen hat eine Nachricht sendet
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Leere Nachricht wird abgelehnt

- **WHEN** ein User eine Nachricht mit leerem `body` sendet
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Konversation als gelesen markieren

Das System SHALL es erlauben alle Nachrichten einer Konversation als gelesen zu markieren. Dies aktualisiert den `unreadCount` auf 0 für den aufrufenden User.

#### Scenario: Konversation öffnen markiert als gelesen

- **WHEN** ein User `POST /api/chat/conversations/{id}/read` aufruft
- **THEN** werden alle Nachrichten der Konversation für diesen User als gelesen markiert
- **THEN** gibt `GET /api/chat/conversations` für diese Konversation `unreadCount: 0` zurück

### Requirement: Gruppe verlassen

Das System SHALL es Mitgliedern erlauben eine Gruppen-Konversation zu verlassen. Direct-Konversationen können nicht verlassen werden. Nach dem Verlassen wird `left_at` gesetzt, der User erhält keine weiteren SSE-Events oder Push Notifications für diese Konversation.

#### Scenario: Mitglied verlässt Gruppe

- **WHEN** ein Mitglied `DELETE /api/chat/conversations/{id}/members/me` aufruft
- **THEN** wird `left_at` auf die aktuelle Zeit gesetzt
- **THEN** erscheint die Konversation nicht mehr in der Konversationsliste des Users

#### Scenario: Direct-Konversation kann nicht verlassen werden

- **WHEN** ein User versucht eine Direct-Konversation zu verlassen
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Rollenbasierter User-Picker

Das System SHALL einen Endpoint bereitstellen der eine nach Suchbegriff gefilterte und rollenbasiert eingeschränkte User-Liste zurückgibt. Diese Liste wird beim Erstellen von Konversationen und beim Hinzufügen zu Gruppen genutzt.

#### Scenario: Spieler sieht nur Teammitglieder

- **WHEN** ein Spieler `GET /api/chat/users?q=Müller` aufruft
- **THEN** gibt der Server nur User zurück die im selben Team wie der Spieler sind

#### Scenario: Trainer sieht User aller seiner Teams

- **WHEN** ein Trainer `GET /api/chat/users?q=` aufruft
- **THEN** gibt der Server alle User zurück die in Teams sind wo der Trainer Mitglied ist

#### Scenario: Admin sieht alle User

- **WHEN** ein Admin `GET /api/chat/users?q=` aufruft
- **THEN** gibt der Server alle User des Systems zurück

