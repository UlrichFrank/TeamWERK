## ADDED Requirements

### Requirement: User-spezifischer SSE-Endpoint für Chat

Das System SHALL einen zweiten SSE-Endpoint `GET /api/chat/events` bereitstellen der nur Events für den jeweils authentifizierten User sendet. Der Endpoint MUSS im authenticated-Block registriert sein. Auth erfolgt wie beim bestehenden `/api/events` via `?token=<jwt>` Query-Parameter. Der bestehende globale `/api/events` Endpoint bleibt unverändert.

#### Scenario: Neue Chat-Nachricht wird nur an Konversations-Mitglieder gesendet

- **WHEN** User A eine Nachricht in einer Konversation mit User B sendet
- **THEN** erhält User B ein SSE-Event `chat:new-message:<conversationId>` auf `/api/chat/events`
- **THEN** erhält User C der NICHT Mitglied der Konversation ist KEIN Event

#### Scenario: Neuer Broadcast wird an alle matching Empfänger gesendet

- **WHEN** ein Admin einen Broadcast mit `target_type: all` sendet
- **THEN** erhalten alle User die gerade `/api/chat/events` verbunden haben ein `chat:new-broadcast` Event

#### Scenario: Keepalive auf Chat-SSE-Endpoint

- **WHEN** 30 Sekunden kein Chat-Event stattgefunden hat
- **THEN** sendet der Server einen SSE-Kommentar (`: ping`) auf der `/api/chat/events` Verbindung

#### Scenario: Verbindungsaufbau ohne Token schlägt fehl

- **WHEN** ein nicht-authentifizierter Request `/api/chat/events` aufruft
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Hub unterstützt User-aware Delivery

Der `EventHub` SHALL neben dem globalen Broadcast auch User-spezifische Channels verwalten. Mehrere gleichzeitige Verbindungen desselben Users (mehrere Browser-Tabs) SHALL alle ein Event erhalten.

#### Scenario: Mehrere Tabs desselben Users erhalten alle das Event

- **WHEN** User A in zwei Browser-Tabs `/api/chat/events` verbunden hat und eine Nachricht für User A eintrifft
- **THEN** erhalten beide Tabs das SSE-Event

#### Scenario: Unsubscribe räumt Channel auf

- **WHEN** ein User die SSE-Verbindung schließt (Tab zu, Seite verlassen)
- **THEN** wird der Channel aus der Hub-Map entfernt und keine weiteren Events werden gesendet
