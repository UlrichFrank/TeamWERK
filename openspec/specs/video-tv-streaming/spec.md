# video-tv-streaming Specification

## Purpose
HLS-Ausspielung von Spielvideos auf TV-Geräte: CODECS-Signalisierung für AirPlay/tvOS, dauerabhängige Stream-Token-TTL, CORS für Chromecast-Receiver, Chromecast-Sender und `x-webkit-airplay`-Attribut im Frontend sowie ein idempotenter Backfill für Bestandsvideos.

## Requirements

### Requirement: HLS-Master-Manifest mit CODECS-Signalisierung

Der Server SHALL bei jedem erfolgreichen Transcode die Codec-Attribute des produzierten Video- und Audio-Streams per `ffprobe` ermitteln und in `videos.codecs` speichern. `master.m3u8` MUST für jede STREAM-INF-Zeile ein `CODECS="<video>,<audio>"`-Attribut enthalten (Format `avc1.PPCCLL,mp4a.40.X`). `master.m3u8` MUST zusätzlich die Zeile `#EXT-X-INDEPENDENT-SEGMENTS` enthalten.

#### Scenario: Neu-Transcode schreibt CODECS
- **WHEN** ein Video erfolgreich transcodiert wurde
- **THEN** ist `videos.codecs` auf `"avc1.PPCCLL,mp4a.40.X"` gesetzt und `processed/{id}/master.m3u8` enthält beide neuen Zeilen

#### Scenario: AirPlay auf AppleTV
- **WHEN** ein Nutzer über iPhone Safari ein Video per AirPlay auf einen AppleTV wirft
- **THEN** startet die Video-Wiedergabe (nicht nur Audio) — verifiziert durch Rauchtest, nicht durch Unit-Test

### Requirement: Nur eine 720p-Rendition

Der Server MUST NUR die 720p-Rendition transkodieren. `workerRenditions` MUST NOT die 360p-Rendition enthalten. `master.m3u8` MUST NUR eine STREAM-INF-Zeile referenzieren (`720p/index.m3u8`).

#### Scenario: Neu-Transcode erzeugt nur 720p
- **WHEN** ein Video transcodiert wurde
- **THEN** existiert `processed/{id}/720p/index.m3u8`, aber `processed/{id}/360p/` existiert NICHT

### Requirement: pix_fmt yuv420p explizit

ffmpeg MUST beim Transcode `-pix_fmt yuv420p` als expliziten Ausgabe-Pixelformat-Parameter erhalten (vor `-c:v libx264`), damit 10-bit-Quellen (moderne iPhone-Aufnahmen) auf 8-bit umkodiert werden und tvOS/AppleTV sie dekodieren können.

#### Scenario: ffmpeg-Args enthalten pix_fmt
- **WHEN** der Transcode-Worker ffmpeg für eine Rendition aufruft
- **THEN** enthält die Arg-Liste die aufeinanderfolgenden Werte `"-pix_fmt", "yuv420p"` vor `"-c:v", "libx264"`

### Requirement: Dauerabhängige Stream-Token-TTL

`GET /api/videos/{id}/play` MUST einen Stream-Token mit einer TTL ausstellen, die sich an der Video-Dauer bemisst:

- Wenn `videos.duration_sec` bekannt und > 0: `TTL = clamp(duration_sec + 1800, 3600, 14400)` Sekunden (also mindestens 1 h, höchstens 4 h).
- Wenn `duration_sec` NULL oder 0: `TTL = 3600` Sekunden (Legacy-Verhalten).

#### Scenario: Kurzvideo bekommt 1h-Token
- **WHEN** ein Nutzer `GET /api/videos/{id}/play` für ein Video mit `duration_sec = 1200` aufruft
- **THEN** liegt `exp - now` im Bereich `[3599, 3601]` Sekunden

#### Scenario: Mittleres Video bekommt Dauer + 30min
- **WHEN** ein Nutzer `GET /api/videos/{id}/play` für ein Video mit `duration_sec = 5400` (90 min) aufruft
- **THEN** liegt `exp - now` im Bereich `[7199, 7201]` Sekunden

#### Scenario: Sehr langes Video wird auf 4h gedeckelt
- **WHEN** ein Nutzer `GET /api/videos/{id}/play` für ein Video mit `duration_sec = 100000` aufruft
- **THEN** liegt `exp - now` im Bereich `[14399, 14401]` Sekunden

#### Scenario: Video ohne Dauer bekommt Legacy-1h
- **WHEN** ein Nutzer `GET /api/videos/{id}/play` für ein Video mit `duration_sec IS NULL` aufruft
- **THEN** liegt `exp - now` im Bereich `[3599, 3601]` Sekunden

### Requirement: CORS für HLS-Routen (Chromecast-Kompatibilität)

`GET /api/videos/{id}/hls/master.m3u8` und `GET /api/videos/{id}/hls/{rendition}/{segment}` MUST den Response-Header `Access-Control-Allow-Origin: *` setzen. Ein `OPTIONS`-Preflight auf denselben Pfaden MUST mit HTTP 204 und den gleichen Headern beantwortet werden. Die Authentifizierung bleibt ausschließlich der `?st=`-Token; es werden KEINE Credentials/Cookies gesendet oder akzeptiert.

#### Scenario: Master-Playlist trägt CORS-Header
- **WHEN** ein Chromecast-Receiver `GET /api/videos/{id}/hls/master.m3u8?st=<gültig>` aufruft
- **THEN** enthält der Response den Header `Access-Control-Allow-Origin: *`

#### Scenario: CORS-Preflight liefert 204
- **WHEN** ein Chromecast-Receiver `OPTIONS /api/videos/{id}/hls/master.m3u8` sendet
- **THEN** antwortet der Server mit HTTP 204 und `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Methods: GET`

#### Scenario: Fehlender Token bleibt 403
- **WHEN** ein Aufruf `GET /api/videos/{id}/hls/master.m3u8` ohne `?st=` erfolgt
- **THEN** antwortet der Server mit HTTP 403 (CORS-Header ändern nichts am Auth-Verhalten)

### Requirement: Backfill der Bestandsvideos beim Scheduler-Start

Beim Start des `scheduler:run`-Prozesses MUST einmalig ein idempotenter Backfill-Lauf für alle Videos mit `status='ready' AND codecs IS NULL` ausgeführt werden: Codec-Strings via `ffprobe` auf `720p/seg_001.ts` ermitteln (Fallback `360p/seg_001.ts`), in DB schreiben, `master.m3u8` mit CODECS und ohne 360p-Referenz neu schreiben, und ein vorhandenes `360p/`-Verzeichnis löschen. Einzelne fehlgeschlagene Videos MUST geloggt und übersprungen werden — der Gesamtlauf MUST NOT abbrechen.

#### Scenario: Erster Lauf migriert Alt-Videos
- **WHEN** ein Video mit `status='ready', codecs IS NULL` existiert und `processed/{id}/720p/seg_001.ts` vorhanden ist
- **THEN** ist nach dem Backfill `videos.codecs` gesetzt, `master.m3u8` enthält CODECS + INDEPENDENT-SEGMENTS und referenziert nur `720p/index.m3u8`, und `processed/{id}/360p/` existiert nicht mehr

#### Scenario: Zweiter Lauf ist No-Op
- **WHEN** derselbe Backfill zum zweiten Mal ausgeführt wird
- **THEN** werden keine Videos in der Ergebnis-Query zurückgeliefert und keine Datei-Änderungen vorgenommen

#### Scenario: Fehlgeschlagenes Video blockiert nicht
- **WHEN** eines der zu migrierenden Videos keine `seg_001.ts` mehr besitzt
- **THEN** wird der Fehler geloggt und andere Videos werden trotzdem erfolgreich migriert

### Requirement: Frontend AirPlay-Attribut

Das `<video>`-Element in `VideoDetailPage` MUST das Attribut `x-webkit-airplay="allow"` tragen, um AirPlay auf Safari explizit zu erlauben (belt-and-suspenders gegen künftige Safari-Default-Verschärfungen).

#### Scenario: Rendered Video hat AirPlay-Attribut
- **WHEN** `VideoDetailPage` gerendert wird
- **THEN** trägt das `<video>`-Element das Attribut `x-webkit-airplay="allow"`

### Requirement: Chromecast-Sender im Frontend

`VideoDetailPage` MUST eine Cast-Schaltfläche rendern, sobald die Google-Cast-Framework-API im Browser verfügbar ist. Das Cast-SDK (`https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1`) MUST erst nach dem ersten Klick auf die Schaltfläche geladen werden (kein passives Google-Ping beim Seitenaufruf). Der Script-Tag MUST `crossorigin="anonymous"` tragen; ein `integrity=`-Attribut MUST NICHT gesetzt werden, da Google das Script serverseitig ohne stabile Hashes versioniert. Eine Cast-Session MUST dem Default-Media-Receiver die Master-URL mit `application/vnd.apple.mpegurl` und Stream-Type `BUFFERED` übergeben.

#### Scenario: Cast-Button erscheint mit verfügbarer API
- **WHEN** `VideoDetailPage` rendert und `window.chrome.cast` verfügbar ist
- **THEN** ist der Cast-Button sichtbar

#### Scenario: Cast-Button erscheint nicht ohne API
- **WHEN** `VideoDetailPage` rendert und `window.chrome.cast` NICHT verfügbar ist (z. B. Safari, Firefox)
- **THEN** ist der Cast-Button nicht sichtbar

#### Scenario: Cast-Session lädt HLS-Media
- **WHEN** der User auf den Cast-Button klickt und einen Receiver auswählt
- **THEN** wird `loadCastSDK()` einmalig ausgeführt, `chrome.cast.media.MediaInfo` mit `masterURL` (inkl. `?st=`-Token) und `application/vnd.apple.mpegurl` erzeugt und via `loadRequest` an den Default-Receiver übergeben
