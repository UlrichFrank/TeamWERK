## ADDED Requirements

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

FFmpeg MUST mit `nice -n 19` aufgerufen werden. Die Ausgabe MUST HLS mit Master-Playlist und zwei Renditions (720p und 360p) sein. Video-Codec MUST H.264 sein (CRF 26, preset medium). Audio MUST `aac 128k` sein, außer der Input ist bereits AAC — dann `copy`. Segmentlänge MUST 10 Sekunden sein.

#### Scenario: Output-Struktur nach erfolgreichem Transcode
- **WHEN** ein Video erfolgreich transcodiert wurde
- **THEN** existieren `processed/{id}/master.m3u8`, `processed/{id}/720p/index.m3u8`, `processed/{id}/360p/index.m3u8` und alle `.ts`-Segmente

#### Scenario: Master-Playlist verweist auf beide Renditions
- **WHEN** die Master-Playlist gelesen wird
- **THEN** enthält sie zwei `#EXT-X-STREAM-INF`-Einträge für 720p und 360p mit korrekten `BANDWIDTH`-Angaben

### Requirement: Erfolgs- und Fehlerpfad

Bei erfolgreichem Transcode SHALL der Worker `status='ready'` und `ready_at=now()` setzen, die raw-Datei löschen und `video-ready` broadcasten. Bei Fehler MUST `status='failed'` und `failure_reason` gesetzt werden; die raw-Datei MUST für 7 Tage zur Fehleranalyse erhalten bleiben.

#### Scenario: Erfolgreicher Transcode
- **WHEN** FFmpeg ohne Fehler beendet wird und die Master-Playlist existiert
- **THEN** ist `status='ready'`, `ready_at` gesetzt, `raw/{id}.mp4` nicht mehr vorhanden, und `video-ready` wurde broadcastet

#### Scenario: Fehlgeschlagener Transcode
- **WHEN** FFmpeg mit einem Fehlercode beendet wird
- **THEN** ist `status='failed'` mit aussagekräftigem `failure_reason`, raw-Datei bleibt erhalten

### Requirement: Push-Notification bei Fertigstellung

Bei `status='ready'` SHALL der Worker — nicht-blockierend in einer Goroutine — Push-Notifications an folgende Empfänger senden: Hochladenden (`created_by`), alle aktiven Spieler des Teams (`team_memberships`), alle Eltern dieser Spieler (`family_links`) und alle Trainer des Teams (`team_trainers`). Inhalt: Titel `"Neues Video: {team_name}"`, Body `"{title}"`, Ziel-URL `/videos/{id}`.

#### Scenario: Empfängerkreis
- **WHEN** ein Video für Team `U17` fertig wird
- **THEN** erhalten Hochladender, alle aktiven U17-Spieler, deren Eltern und alle U17-Trainer eine Push-Notification

#### Scenario: Push schlägt einzeln fehl
- **WHEN** Push-Versand an einen Empfänger fehlschlägt
- **THEN** wird der Transcode-Erfolg nicht rückgängig gemacht, der Fehler wird geloggt
