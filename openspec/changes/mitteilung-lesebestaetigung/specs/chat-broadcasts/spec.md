## MODIFIED Requirements

### Requirement: Empfangene Broadcasts abrufen

Das System SHALL die sichtbaren Broadcasts eines Users zurückgeben. Zu jedem Broadcast werden geliefert: `id`, `senderName`, `body`, `media` (Liste der Bilder in Album-Reihenfolge, leer ohne Bild; je Eintrag `id`, `url = "/media/<id>"`, `width`/`height` nur bei bekannter Dimension, sonst weggelassen), `mediaId`/`mediaUrl`/`mediaWidth`/`mediaHeight` (Angaben zum **ersten** Bild, Kompatibilität für ältere Clients; null bzw. weggelassen ohne Bild), `sentAt`, `isRead`, `isSent`, `editedAt`.

Für Broadcasts, die der Aufrufer **selbst gesendet** hat (`isSent = true`), SHALL die Antwort zusätzlich `readCount` und `readTotal` tragen — die Anzahl der Empfänger, die den Broadcast gelesen haben, und die beim Fan-out festgeschriebene Empfängermenge, beide **ohne** den Absender. Für fremde Broadcasts SHALL keines der beiden Felder im JSON-Objekt erscheinen, damit der Lese-Zustand Dritter für Empfänger unsichtbar bleibt.

#### Scenario: Broadcast mit mehreren Bildern

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und ein Broadcast drei Bilder A, B, C trägt
- **THEN** enthält `media` genau A, B, C in dieser Reihenfolge, und `mediaId`/`mediaUrl` bezeichnen A

#### Scenario: Broadcast mit Bild und bekannten Dimensionen

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und ein Broadcast ein Bild trägt, dessen `media`-Zeile `width=800`, `height=600` hat
- **THEN** enthält der `media`-Eintrag `width=800`, `height=600`, und das Broadcast-Objekt `mediaId`, `mediaUrl = "/media/<mediaId>"`, `mediaWidth=800`, `mediaHeight=600`

#### Scenario: Broadcast mit Bild ohne bekannte Dimensionen

- **WHEN** ein Broadcast mit einem Bild abgerufen wird, dessen `media`-Zeile `width IS NULL` hat
- **THEN** fehlen `width`/`height` im `media`-Eintrag sowie `mediaWidth`/`mediaHeight` im Broadcast-Objekt; `mediaId` und `mediaUrl` sind gesetzt

#### Scenario: Bestands-Broadcast mit Einzelbild

- **WHEN** ein vor Einführung der Alben gesendeter Broadcast mit einem Bild abgerufen wird
- **THEN** enthält `media` genau diesen einen Eintrag

#### Scenario: Broadcast ohne Bild abrufen

- **WHEN** ein Broadcast ohne Bild abgerufen wird
- **THEN** ist `media` eine leere Liste, `mediaId` und `mediaUrl` sind null; `mediaWidth`/`mediaHeight` fehlen

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
