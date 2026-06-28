## ADDED Requirements

### Requirement: Stream-Token-Ausgabe

Nutzer mit Sicht-Berechtigung für ein Video SHALL via `GET /api/videos/{id}/play` einen kurzlebigen Stream-Token erhalten. Der Token MUST HMAC-SHA256 mit `VIDEO_STREAM_SECRET` signiert sein, die Claims `vid`, `uid`, `exp` enthalten und 1 Stunde gültig sein.

#### Scenario: Berechtigter Spieler ruft play auf
- **WHEN** ein Spieler eines Teams `GET /api/videos/{id}/play` für ein Video seines Teams aufruft
- **THEN** liefert der Server `{ token, master_url }` mit HTTP 200; der Token enthält `vid={id}` und `uid={user_id}`

#### Scenario: Eltern eines Spielers
- **WHEN** ein Elternteil eines aktiven Team-Spielers `GET /api/videos/{id}/play` für ein Video dieses Teams aufruft
- **THEN** liefert der Server HTTP 200 mit gültigem Token

#### Scenario: Nicht-berechtigter Nutzer
- **WHEN** ein Nutzer ohne Bezug zum Team `GET /api/videos/{id}/play` aufruft
- **THEN** antwortet der Server mit HTTP 403 oder HTTP 404 (Existenz nicht enthüllen)

#### Scenario: Video noch nicht ready
- **WHEN** der Status nicht `ready` ist
- **THEN** antwortet der Server mit HTTP 409 ohne Token auszugeben

### Requirement: Token-validierte HLS-Auslieferung

Der Server SHALL HLS-Master-Playlist, Rendition-Manifeste und `.ts`-Segmente unter `/api/videos/{id}/hls/...` ausliefern. Jeder Request MUST einen Stream-Token im Query-Parameter `?st=` mitbringen. Die Middleware MUST Signatur prüfen, `exp` prüfen und `vid` aus dem Token gegen den Pfad-Parameter `{id}` vergleichen.

#### Scenario: Gültiger Token, Segment-Auslieferung
- **WHEN** ein Client `GET /api/videos/42/hls/720p/seg_001.ts?st=<gültig>` aufruft
- **THEN** liefert der Server das Segment mit HTTP 200 (oder 206 bei Range-Request)

#### Scenario: Abgelaufener Token
- **WHEN** der Token-`exp` in der Vergangenheit liegt
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Token für anderes Video
- **WHEN** der Token `vid=10` enthält, aber der Pfad `/api/videos/42/hls/...` lautet
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Manipulierte Signatur
- **WHEN** die HMAC-Signatur des Tokens ungültig ist
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Range-Request-Support

Der Server SHALL für Segmente und Manifeste `Range`-Anfragen mit HTTP 206 Partial Content beantworten und `Accept-Ranges: bytes` setzen.

#### Scenario: Range-Request auf Segment
- **WHEN** ein Client `Range: bytes=0-499999` für ein Segment sendet
- **THEN** antwortet der Server mit HTTP 206 und den ersten 500 000 Bytes

### Requirement: Master-Playlist mit Token-Pass-Through

Der Server SHALL bei der Auslieferung der Master-Playlist die enthaltenen Rendition-Pfade so umschreiben, dass der aktuelle Stream-Token als Query-Parameter mitgegeben wird.

#### Scenario: Token wird an Rendition-URLs angehängt
- **WHEN** ein Client die Master-Playlist mit `?st=<token>` abruft
- **THEN** enthalten die zurückgelieferten Rendition-Pfade `?st=<token>`, sodass `hls.js` die Token automatisch mitschickt
