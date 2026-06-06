# chat-push-notifications Specification

## Purpose
TBD - created by archiving change chat-feature. Update Purpose after archive.
## Requirements
### Requirement: Push Notification bei neuer Chat-Nachricht

Das System SHALL nach dem Speichern einer neuen Chat-Nachricht Push Notifications an alle aktiven Mitglieder der Konversation senden, die nicht der Sender sind. Der Aufruf von `notifications.SendToUsers` MUSS als Goroutine erfolgen und darf die HTTP-Response nicht blockieren.

#### Scenario: Empfänger ist offline

- **WHEN** ein Mitglied eine Nachricht sendet und ein anderes Mitglied hat keine aktive SSE-Verbindung
- **THEN** sendet der Server eine Push Notification an das offline Mitglied mit Titel (Name des Senders) und einem gekürzten Nachrichtentext

#### Scenario: Sender erhält keine eigene Push Notification

- **WHEN** ein Mitglied eine Nachricht sendet
- **THEN** erhält der Sender selbst keine Push Notification

#### Scenario: Ausgetretene Mitglieder erhalten keine Push Notifications

- **WHEN** ein User eine Gruppe verlassen hat (left_at NOT NULL) und eine neue Nachricht gesendet wird
- **THEN** erhält der ausgetretene User KEINE Push Notification für diese Nachricht

### Requirement: Push Notification bei neuem Broadcast

Das System SHALL nach dem Speichern eines neuen Broadcasts Push Notifications an alle matching Empfänger senden. Die Zielgruppe wird zur Sendzeit aus der DB aufgelöst. Der Push-Aufruf erfolgt als Goroutine.

#### Scenario: Broadcast an alle löst Push Notifications aus

- **WHEN** ein Admin einen Broadcast mit `target_type: all` sendet
- **THEN** sendet der Server Push Notifications an alle User mit aktiver Push-Subscription
- **THEN** enthält die Notification den Namen des Senders und einen gekürzten Text

#### Scenario: Broadcast an Team löst Push Notifications aus

- **WHEN** ein Trainer einen Broadcast mit `target_type: team` an sein Team sendet
- **THEN** sendet der Server Push Notifications nur an User die Mitglied in diesem Team sind

#### Scenario: Push schlägt fehl ohne Blockierung

- **WHEN** der Push-Service für einen Empfänger HTTP 410 (Subscription abgelaufen) zurückgibt
- **THEN** wird die Subscription aus `push_subscriptions` entfernt und der Fehler still geloggt
- **THEN** der HTTP-Response des Senders ist davon unberührt

### Requirement: Nav-Badge zeigt Gesamtzahl ungelesener Chat-Items

Das System SHALL im Frontend einen Badge an der Chat-Nav-Entry anzeigen der die Summe aus ungelesenen Nachrichten (aus Konversationen) und ungelesenen Broadcasts anzeigt. Der Badge wird aktualisiert wenn ein SSE-Event `chat:new-message:<id>` oder `chat:new-broadcast` empfangen wird.

#### Scenario: Badge erscheint bei neuer Nachricht

- **WHEN** ein User eine neue Nachricht in einer Konversation erhält
- **THEN** zeigt der Nav-Badge eine Zahl ≥ 1

#### Scenario: Badge verschwindet nach Lesen

- **WHEN** ein User alle Konversationen und Broadcasts als gelesen markiert hat
- **THEN** zeigt der Nav-Badge keine Zahl mehr (Badge ausgeblendet)

#### Scenario: Badge-Count wird via SSE aktualisiert

- **WHEN** ein SSE-Event `chat:new-message` oder `chat:new-broadcast` eintrifft
- **THEN** ruft das Frontend `GET /api/chat/conversations` und `GET /api/chat/broadcasts` neu ab um den Badge-Count zu aktualisieren

