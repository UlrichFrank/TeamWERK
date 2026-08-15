## MODIFIED Requirements

### Requirement: Empfangene Broadcasts abrufen

Das System SHALL die sichtbaren Broadcasts eines Users zurückgeben. Zu jedem Broadcast werden geliefert: `id`, `senderName`, `body`, `mediaId` (null wenn kein Bild), `mediaUrl` (null wenn kein Bild; sonst `"/media/<mediaId>"`), `mediaWidth` (nur bei Bild-Broadcasts mit bekannter Dimension; sonst weggelassen), `mediaHeight` (nur bei Bild-Broadcasts mit bekannter Dimension; sonst weggelassen), `sentAt`, `isRead`, `isSent`, `editedAt`.

Für Broadcasts, die der Aufrufer **selbst gesendet** hat (`isSent = true`), SHALL die Antwort zusätzlich `readCount` und `readTotal` tragen — die Anzahl der Empfänger, die den Broadcast gelesen haben, und die beim Fan-out festgeschriebene Empfängermenge, beide **ohne** den Absender. Für fremde Broadcasts SHALL keines der beiden Felder im JSON-Objekt erscheinen, damit der Lese-Zustand Dritter für Empfänger unsichtbar bleibt.

#### Scenario: Broadcast mit Bild und bekannten Dimensionen

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und ein Broadcast `media_id` gesetzt hat, dessen `media`-Zeile `width=800`, `height=600` hat
- **THEN** enthält das Broadcast-Objekt `mediaId`, `mediaUrl = "/media/<mediaId>"`, `mediaWidth=800`, `mediaHeight=600`

#### Scenario: Broadcast mit Bild ohne bekannte Dimensionen

- **WHEN** ein Broadcast mit `media_id` abgerufen wird, dessen `media`-Zeile `width IS NULL` hat
- **THEN** enthält das Broadcast-Objekt `mediaId`, `mediaUrl`; `mediaWidth` und `mediaHeight` fehlen im JSON-Objekt

#### Scenario: Broadcast ohne Bild abrufen

- **WHEN** ein Broadcast ohne `media_id` abgerufen wird
- **THEN** sind `mediaId` und `mediaUrl` beide null; `mediaWidth`/`mediaHeight` fehlen

#### Scenario: Eigener Broadcast trägt das Lese-Aggregat

- **WHEN** der Absender eines an 10 Empfänger gesendeten Broadcasts, den 3 davon gelesen haben, `GET /api/chat/broadcasts` aufruft
- **THEN** trägt das Objekt `isSent = true`, `readCount = 3` und `readTotal = 10`

#### Scenario: Fremder Broadcast trägt kein Lese-Aggregat

- **WHEN** ein Empfänger (nicht der Absender) `GET /api/chat/broadcasts` aufruft
- **THEN** fehlen `readCount` und `readTotal` im JSON-Objekt dieses Broadcasts

### Requirement: Broadcast als gelesen markieren

Das System SHALL es Empfängern erlauben einen Broadcast als gelesen zu markieren. Dies beeinflusst den Ungelesen-Badge im Nav.

Das Markieren SHALL idempotent sein: das `UPDATE` greift nur, solange `read_at` NULL ist. Ausschließlich bei einer tatsächlich veränderten Zeile SHALL zusätzlich das SSE-Event `chat:broadcast-read:<broadcastId>` an den **Absender** gehen; wiederholtes Markieren SHALL kein weiteres Event auslösen.

#### Scenario: Broadcast öffnen markiert als gelesen

- **WHEN** ein User einen Broadcast öffnet und `POST /api/chat/broadcasts/{id}/read` aufruft
- **THEN** wird `broadcast_reads.read_at` für diesen User gesetzt
- **THEN** erscheint der Broadcast als gelesen in der Liste

#### Scenario: Erstes Markieren benachrichtigt den Absender

- **WHEN** ein Empfänger einen Broadcast zum ersten Mal als gelesen markiert
- **THEN** erhält der Absender das SSE-Event `chat:broadcast-read:<broadcastId>`

#### Scenario: Zweites Markieren bleibt still

- **WHEN** derselbe Empfänger denselben Broadcast erneut als gelesen markiert
- **THEN** antwortet der Server mit HTTP 204 und es geht kein weiteres SSE-Event an den Absender
