## ADDED Requirements

### Requirement: Broadcast senden

Das System SHALL es Usern mit der Rolle admin, vorstand oder trainer erlauben eine einweg Mitteilung an eine Zielgruppe zu senden. Trainer dürfen nur an ihr eigenes Team senden (`target_type: team` mit ihrer `target_id`). Admin und Vorstand dürfen `target_type: all`, `team` oder `role` verwenden. Nach dem Senden werden alle matching User benachrichtigt (SSE + Push).

#### Scenario: Admin sendet Broadcast an alle

- **WHEN** ein Admin `POST /api/chat/broadcasts` mit `{ body: "Wichtige Info", target_type: "all" }` aufruft
- **THEN** wird der Broadcast gespeichert und alle aktiven User erhalten ein SSE-Event `chat:new-broadcast`
- **THEN** gibt der Server HTTP 201 zurück

#### Scenario: Trainer sendet Broadcast an eigenes Team

- **WHEN** ein Trainer `POST /api/chat/broadcasts` mit `{ body: "Training morgen abgesagt", target_type: "team", target_id: 3 }` aufruft und Team 3 sein Team ist
- **THEN** wird der Broadcast gespeichert und alle Mitglieder von Team 3 erhalten eine Benachrichtigung

#### Scenario: Trainer kann nicht an fremdes Team senden

- **WHEN** ein Trainer versucht einen Broadcast mit `target_id` eines fremden Teams zu senden
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Spieler kann keinen Broadcast senden

- **WHEN** ein User mit Rolle spieler oder elternteil `POST /api/chat/broadcasts` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Leerer Broadcast wird abgelehnt

- **WHEN** ein berechtigter User einen Broadcast mit leerem `body` sendet
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Empfangene Broadcasts abrufen

Das System SHALL für jeden User eine Liste der für ihn bestimmten Broadcasts zurückgeben, sortiert nach `sent_at` absteigend. Der Sender ist namentlich sichtbar. Andere Empfänger sind NICHT sichtbar (anonym). Jeder Broadcast zeigt ob er gelesen wurde.

#### Scenario: User ruft empfangene Broadcasts ab

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft
- **THEN** gibt der Server alle Broadcasts zurück die für diesen User bestimmt waren
- **THEN** jeder Broadcast enthält `senderName`, `body`, `sentAt`, `isRead`
- **THEN** andere Empfänger sind NICHT im Response enthalten

#### Scenario: Gesendete Broadcasts für Sender

- **WHEN** ein Admin `GET /api/chat/broadcasts` aufruft
- **THEN** erscheinen auch selbst gesendete Broadcasts in der Liste (mit Markierung `isSent: true`)

### Requirement: Broadcast als gelesen markieren

Das System SHALL es Empfängern erlauben einen Broadcast als gelesen zu markieren. Dies beeinflusst den Ungelesen-Badge im Nav.

#### Scenario: Broadcast öffnen markiert als gelesen

- **WHEN** ein User einen Broadcast öffnet und `POST /api/chat/broadcasts/{id}/read` aufruft
- **THEN** wird `broadcast_reads.read_at` für diesen User gesetzt
- **THEN** erscheint der Broadcast als gelesen in der Liste

### Requirement: Kein Rückkanal bei Broadcasts

Das System SHALL keinerlei Reply-Funktionalität für Broadcasts bereitstellen. Der Endpoint zur Konversationserstellung (`POST /api/chat/conversations`) darf nicht über einen Broadcast-Kontext erreichbar sein. Im Frontend wird kein Reply-Eingabefeld angezeigt.

#### Scenario: Kein Reply-Endpoint für Broadcasts

- **WHEN** ein User versucht auf einen Broadcast zu antworten
- **THEN** existiert kein API-Endpoint für diese Aktion (HTTP 404 oder nicht vorhanden)
