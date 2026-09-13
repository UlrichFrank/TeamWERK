## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Eingangsformat-Validierung vor dem Remux

Bevor der Worker eine eingehende Datei per Stream-Copy umpackt, MUST er per `ffprobe` (ohne Decode) prüfen, dass der erste Video-Stream den Codec H.264 und das Pixelformat `yuv420p` verwendet. Erfüllt die Datei diese Bedingung nicht, MUST der Worker den Remux-Versuch unterlassen und das Video mit `status='failed'` und `failure_reason='unsupported_input_format'` markieren. Die Rohdatei MUST wie bei jedem anderen Fehlerfall für 7 Tage zur Analyse erhalten bleiben.

#### Scenario: Eingehende Datei entspricht dem erwarteten Format
- **WHEN** eine Datei mit H.264/`yuv420p`-Videostream zum Remux ansteht
- **THEN** wird der Remux normal durchgeführt

#### Scenario: Eingehende Datei entspricht nicht dem erwarteten Format
- **WHEN** eine Datei mit einem anderen Video-Codec oder Pixelformat (z. B. weil sie nicht über das Offline-Encoding-Tool erzeugt wurde) zum Remux ansteht
- **THEN** wird `status='failed'` mit `failure_reason='unsupported_input_format'` gesetzt, kein `-c copy`-ffmpeg-Aufruf erfolgt, und die Rohdatei bleibt 7 Tage erhalten
