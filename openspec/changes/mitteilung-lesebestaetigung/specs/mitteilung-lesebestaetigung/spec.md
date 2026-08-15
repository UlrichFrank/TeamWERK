## ADDED Requirements

### Requirement: Absender sieht den Lese-Zustand seiner Mitteilung

Das System SHALL dem Absender einer Mitteilung ein Aggregat `readCount` / `readTotal`
anzeigen, dargestellt als `N / M gelesen`. Beide Zahlen SHALL den Absender selbst
ausschließen, obwohl `SendBroadcast` ihm eine `broadcast_reads`-Zeile mit gesetztem
`read_at` schreibt.

`readTotal` SHALL die beim Fan-out festgeschriebene Empfängermenge sein (Anzahl der
`broadcast_reads`-Zeilen ohne den Absender) und SHALL sich nachträglich **nicht** ändern,
wenn dem Verein weitere Nutzer beitreten. Damit unterscheidet er sich bewusst vom Nenner
der Chat-Lesebestätigungen, der aktive Mitglieder live zählt.

Es SHALL **kein** zwei-Zustands-Häkchen wie im Direktchat geben — bei einer Mitteilung an
viele Empfänger wäre „gelesen, weil mindestens einer gelesen hat" irreführend.

Fremde Mitteilungen SHALL weder `readCount` noch `readTotal` tragen.

#### Scenario: Absender sieht Aggregat

- **WHEN** ein Vorstand eine Mitteilung an 10 Empfänger gesendet hat, 3 davon haben sie geöffnet, und er `GET /api/chat/broadcasts` aufruft
- **THEN** trägt diese Mitteilung `readCount = 3` und `readTotal = 10`

#### Scenario: Absender ist in keiner der beiden Zahlen

- **WHEN** ein Vorstand eine Mitteilung an eine Zielgruppe sendet, die ihn selbst einschließt, und niemand sonst sie öffnet
- **THEN** trägt die Mitteilung `readCount = 0`
- **AND** `readTotal` entspricht der Empfängerzahl ohne ihn

#### Scenario: Nenner bleibt eingefroren

- **WHEN** eine Mitteilung an 10 Empfänger gesendet wurde und anschließend 5 weitere Nutzer angelegt werden
- **THEN** bleibt `readTotal` bei 10

#### Scenario: Empfänger sieht keinen Lese-Zustand

- **WHEN** ein Empfänger (nicht der Absender) `GET /api/chat/broadcasts` aufruft
- **THEN** fehlen `readCount` und `readTotal` im JSON-Objekt dieser Mitteilung

### Requirement: Leserliste einer Mitteilung ist absender-only

Das System SHALL unter `GET /api/chat/broadcasts/{id}/reads` die Leser einer Mitteilung
als `[{userId, name, readAt}]` liefern, aufsteigend nach `readAt` sortiert und **ohne**
den Absender. Nur der Absender (`broadcasts.sender_id = claims.UserID`) SHALL die Route
aufrufen dürfen; alle anderen — auch Empfänger derselben Mitteilung — SHALL HTTP 403
erhalten. Für eine nicht existierende Mitteilung SHALL die Route HTTP 404 liefern.

#### Scenario: Absender ruft die Leserliste ab

- **WHEN** der Absender `GET /api/chat/broadcasts/{id}/reads` für seine Mitteilung aufruft
- **THEN** antwortet der Server mit HTTP 200 und der nach `readAt` aufsteigend sortierten Leserliste ohne ihn selbst

#### Scenario: Empfänger darf die Leserliste nicht sehen

- **WHEN** ein Empfänger derselben Mitteilung `GET /api/chat/broadcasts/{id}/reads` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Unbekannte Mitteilung

- **WHEN** ein User die Route für eine nicht existierende Broadcast-ID aufruft
- **THEN** antwortet der Server mit HTTP 404

### Requirement: Ausgeblendete Mitteilungen bleiben im Nenner

Das System SHALL Empfänger, die eine Mitteilung ausgeblendet haben (`hidden_at` gesetzt),
weiterhin in `readTotal` zählen und sie nicht als gelesen werten, solange `read_at` NULL
ist. Ausblenden ist kein Lesen; der Nenner SHALL nicht nachträglich schrumpfen, weil er
sonst kein Snapshot der Empfängermenge mehr wäre.

Als bewusste Folge SHALL `readCount == readTotal` in diesen Fällen nie erreicht werden.

#### Scenario: Wegwischen ohne Öffnen

- **WHEN** ein Empfänger eine Mitteilung über `DELETE /api/chat/broadcasts/{id}` ausblendet, ohne sie zuvor als gelesen zu markieren
- **THEN** bleibt er Teil von `readTotal`
- **AND** `readCount` erhöht sich durch diese Aktion nicht

### Requirement: Live-Erhöhung genau beim ersten Lesevorgang

Das System SHALL beim Markieren einer Mitteilung als gelesen dem **Absender** das SSE-Event
`chat:broadcast-read:<broadcastId>` über den Chat-Kanal senden — und zwar genau dann, wenn
das `UPDATE` auf `broadcast_reads` tatsächlich eine Zeile verändert hat (`read_at IS NULL`
vor dem Schreiben). Wiederholtes Markieren derselben Mitteilung durch denselben Nutzer
SHALL **kein** weiteres Event auslösen, damit der clientseitig hochgezählte `readCount`
nicht über `readTotal` hinauswächst.

Markiert der Absender seine eigene Mitteilung, SHALL kein Event an ihn gehen.

Das Event SHALL keine Angaben zum Leser tragen; es ist ein plain colon-string wie die
übrigen Chat-Events. Das Frontend SHALL `readCount` der betroffenen Mitteilung lokal um
eins erhöhen.

#### Scenario: Erster Lesevorgang eines Empfängers

- **WHEN** ein Empfänger `POST /api/chat/broadcasts/{id}/read` zum ersten Mal aufruft
- **THEN** antwortet der Server mit HTTP 204
- **AND** der Absender erhält genau ein SSE-Event `chat:broadcast-read:<id>`

#### Scenario: Wiederholtes Markieren erzeugt kein zweites Event

- **WHEN** derselbe Empfänger `POST /api/chat/broadcasts/{id}/read` ein zweites Mal aufruft
- **THEN** antwortet der Server mit HTTP 204
- **AND** es geht **kein** weiteres SSE-Event an den Absender
- **AND** `readCount` bleibt unverändert

#### Scenario: Absender markiert die eigene Mitteilung

- **WHEN** der Absender seine eigene Mitteilung als gelesen markiert
- **THEN** erhält er kein `chat:broadcast-read`-Event
