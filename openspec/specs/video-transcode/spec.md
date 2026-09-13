# video-transcode Specification

## Purpose
TBD - created by archiving change spielvideo-ablage. Update Purpose after archive.
## Requirements
### Requirement: Serielle Transcode-Verarbeitung

Der Server SHALL genau eine Worker-Goroutine betreiben, die Videos mit `status='queued'` in FIFO-Reihenfolge nacheinander transcodiert. Parallele FFmpeg-Aufrufe MUST NOT vorkommen.

#### Scenario: Zwei Videos in Queue
- **WHEN** zwei Videos gleichzeitig `status='queued'` haben
- **THEN** wird das ältere zuerst auf `status='processing'` gesetzt, vollständig verarbeitet und erst danach das jüngere gestartet

#### Scenario: Crash-Recovery
- **WHEN** der Server beim Start ein Video mit `status='processing'` vorfindet
- **THEN** wird der Status auf `queued` zurückgesetzt, bevor der Worker startet

### Requirement: Pre-Transcode-Disk-Check

Der Worker SHALL vor jedem FFmpeg-Aufruf prüfen, dass im Storage-Verzeichnis mindestens das 1,5-Fache der geschätzten Output-Größe frei ist. Bei Unterschreitung MUST der Status auf `queued` bleiben und der Worker für eine Stunde schlafen.

#### Scenario: Speicher reicht nicht
- **WHEN** der Pre-Transcode-Check fehlschlägt
- **THEN** bleibt `status='queued'`, der FFmpeg-Aufruf unterbleibt, und der Worker schläft 1 h vor dem nächsten Versuch

### Requirement: HLS-Transcode-Format

FFmpeg MUST mit `nice -n 19` aufgerufen werden. Die eingehende Datei ist bereits vom Offline-Encoding-Tool auf das Zielformat normalisiert (H.264, `yuv420p`, festes Profil/Level, 720p, AAC-Audio) — der Server MUST daher KEINEN erneuten Video-/Audio-Encode durchführen, sondern die Datei per Stream-Copy (`-c:v copy -c:a copy`) in eine einzelne HLS-Rendition (720p) umpacken. Die Ausgabe MUST eine Master-Playlist mit genau einer `#EXT-X-STREAM-INF`-Zeile (`720p/index.m3u8`) sein. Segmentlänge MUST 4 Sekunden betragen (passend zu den vom Tool bereits alle 4 Sekunden erzwungenen Keyframes).

#### Scenario: Output-Struktur nach erfolgreichem Transcode
- **WHEN** eine bereits normalisierte Videodatei erfolgreich umgepackt wurde
- **THEN** existieren `processed/{id}/master.m3u8`, `processed/{id}/720p/index.m3u8` und die zugehörigen `.ts`-Segmente; `processed/{id}/360p/` existiert NICHT (nur eine Rendition, siehe `video-tv-streaming`)

#### Scenario: Master-Playlist verweist auf beide Renditions
- **WHEN** die Master-Playlist gelesen wird
- **THEN** enthält sie genau einen `#EXT-X-STREAM-INF`-Eintrag für `720p/index.m3u8` (nicht mehr zwei Renditions — dieses Szenario hieß historisch anders, als es noch 720p+360p gab; die Anforderung „Nur eine 720p-Rendition" aus `video-tv-streaming` hat das bereits vor diesem Change abgelöst)

#### Scenario: Kein Re-Encode findet statt
- **WHEN** der Worker eine Rendition erzeugt
- **THEN** enthält der ffmpeg-Aufruf `-c:v copy` und `-c:a copy`, NICHT `-c:v libx264`

### Requirement: Eingangsformat-Validierung vor dem Remux

Bevor der Worker eine eingehende Datei per Stream-Copy umpackt, MUST er per `ffprobe` (ohne Decode) prüfen, dass der erste Video-Stream den Codec H.264 und das Pixelformat `yuv420p` verwendet. Erfüllt die Datei diese Bedingung nicht, MUST der Worker den Remux-Versuch unterlassen und das Video mit `status='failed'` und `failure_reason='unsupported_input_format'` markieren. Die Rohdatei MUST wie bei jedem anderen Fehlerfall für 7 Tage zur Analyse erhalten bleiben.

#### Scenario: Eingehende Datei entspricht dem erwarteten Format
- **WHEN** eine Datei mit H.264/`yuv420p`-Videostream zum Remux ansteht
- **THEN** wird der Remux normal durchgeführt

#### Scenario: Eingehende Datei entspricht nicht dem erwarteten Format
- **WHEN** eine Datei mit einem anderen Video-Codec oder Pixelformat (z. B. weil sie nicht über das Offline-Encoding-Tool erzeugt wurde) zum Remux ansteht
- **THEN** wird `status='failed'` mit `failure_reason='unsupported_input_format'` gesetzt, kein `-c copy`-ffmpeg-Aufruf erfolgt, und die Rohdatei bleibt 7 Tage erhalten

### Requirement: Erfolgs- und Fehlerpfad

Bei erfolgreichem Transcode SHALL der Worker `status='ready'` und `ready_at=now()` setzen, die raw-Datei löschen und `video-ready` broadcasten. Bei Fehler MUST `status='failed'` und `failure_reason` gesetzt werden; die raw-Datei MUST für 7 Tage zur Fehleranalyse erhalten bleiben.

#### Scenario: Erfolgreicher Transcode
- **WHEN** FFmpeg ohne Fehler beendet wird und die Master-Playlist existiert
- **THEN** ist `status='ready'`, `ready_at` gesetzt, `raw/{id}.mp4` nicht mehr vorhanden, und `video-ready` wurde broadcastet

#### Scenario: Fehlgeschlagener Transcode
- **WHEN** FFmpeg mit einem Fehlercode beendet wird
- **THEN** ist `status='failed'` mit aussagekräftigem `failure_reason`, raw-Datei bleibt erhalten

### Requirement: Push-Notification bei Fertigstellung

Bei `status='ready'` SHALL der Worker — nicht-blockierend in einer Goroutine — Push-Notifications an folgende Empfänger senden: Hochladenden (`created_by`), alle aktiven Spieler des Teams (`team_memberships`), alle Eltern dieser Spieler (`family_links`), alle Trainer des Teams (`team_trainers`) sowie die **Mitglieder des erweiterten Kaders des Teams und deren Elternteile**. Inhalt: Titel `"Neues Video: {team_name}"`, Body `"{title}"`, Ziel-URL `/videos/{id}`.

Der Empfängerkreis SHALL deckungsgleich mit der Sicht-Berechtigung aus `video-management` sein: es SHALL niemand über ein Video benachrichtigt werden, das er anschließend nicht öffnen kann.

#### Scenario: Empfängerkreis
- **WHEN** ein Video für Team `U17` fertig wird
- **THEN** erhalten Hochladender, alle aktiven U17-Spieler, deren Eltern und alle U17-Trainer eine Push-Notification

#### Scenario: Erweiterter Kader im Empfängerkreis
- **WHEN** ein Video für Team `U17` fertig wird und ein Mitglied steht nur im erweiterten Kader dieser Mannschaft
- **THEN** erhalten dieses Mitglied und seine über `family_links` verknüpften Elternteile dieselbe Push-Notification

#### Scenario: Push schlägt einzeln fehl
- **WHEN** Push-Versand an einen Empfänger fehlschlägt
- **THEN** wird der Transcode-Erfolg nicht rückgängig gemacht, der Fehler wird geloggt

