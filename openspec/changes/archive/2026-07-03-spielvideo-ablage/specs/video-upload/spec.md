## ADDED Requirements

### Requirement: Upload-Initialisierung mit Pre-Disk-Check

Nutzer mit Vereinsfunktion `trainer`, `sportliche_leitung`, `vorstand` oder Rolle `admin` SHALL via `POST /api/videos` einen Upload initialisieren können. Die Anfrage MUST `title`, `team_id`, `season_id` und die erwartete `size_bytes` enthalten; optional `description`, `game_id`. Der Server MUST vor der Annahme prüfen, dass im Storage-Verzeichnis mindestens `size_bytes × 2.5 + 2 GiB` frei sind.

#### Scenario: Erfolgreiche Initialisierung
- **WHEN** ein Trainer mit ausreichender Berechtigung für `team_id` und ausreichend freiem Speicher `POST /api/videos` aufruft
- **THEN** legt der Server eine DB-Zeile mit `status='uploading'` an und liefert `{ video_id, upload_url }` mit HTTP 201

#### Scenario: Unzureichender Speicher
- **WHEN** die geforderte Größe das freie Speicherbudget übersteigt
- **THEN** antwortet der Server mit HTTP 507 ohne DB-Eintrag anzulegen

#### Scenario: Fehlende Upload-Berechtigung
- **WHEN** ein Nutzer ohne Trainer-/Vorstand-/Admin-Rolle `POST /api/videos` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Trainer fremdes Team
- **WHEN** ein Trainer für ein `team_id` hochlädt, in dem er nicht Trainer ist und nicht admin/vorstand ist
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Resumable Upload via tus

Der Server SHALL Uploads über das tus-Protokoll (Version 1.0) unter `/api/videos/upload/` annehmen. Sessions MUST über Browser-Restarts und Verbindungsabbrüche fortsetzbar sein. Die Maximalgröße pro Upload MUST 2,5 GiB betragen.

#### Scenario: Wiederaufnahme nach Verbindungsabbruch
- **WHEN** ein Upload bei 600 MB unterbrochen wird und der Client mit derselben tus-Session erneut verbindet
- **THEN** wird der Upload ab Byte 600 000 000 fortgesetzt

#### Scenario: Überschreitung der Maximalgröße
- **WHEN** ein Client eine `Upload-Length` > 2,5 GiB ankündigt
- **THEN** lehnt der Server die Session mit HTTP 413 ab

### Requirement: Abschluss-Verarbeitung

Bei erfolgreichem tus-Upload SHALL der Server die Datei aus dem `uploads/`-Verzeichnis nach `raw/{video_id}.mp4` verschieben, mit `ffprobe` die Dauer ermitteln, `videos.duration_sec` und `videos.size_bytes` setzen, `status` auf `queued` setzen und `video-queued` broadcasten.

#### Scenario: Upload abgeschlossen
- **WHEN** der tus-Upload den letzten Chunk empfängt
- **THEN** ist die Datei unter `raw/{video_id}.mp4` vorhanden, `videos.status = 'queued'`, `duration_sec` ist gesetzt, und `video-queued` wurde gesendet

#### Scenario: Hochgeladene Datei ist kein Video
- **WHEN** `ffprobe` keine Video-Streams in der hochgeladenen Datei findet
- **THEN** wird `status='failed'` mit `failure_reason='invalid_media'` gesetzt und die raw-Datei gelöscht

### Requirement: Verwaiste tus-Sessions

Der Scheduler SHALL einmal täglich tus-Sessions im `uploads/`-Verzeichnis löschen, die älter als 24 Stunden sind und keinem DB-Eintrag im Status `uploading` mehr entsprechen.

#### Scenario: Alter abgebrochener Upload
- **WHEN** eine tus-Session-Datei älter als 24 h ist und im DB-Eintrag der Status nicht `uploading` ist
- **THEN** wird die Session-Datei beim nächsten Cleanup-Lauf gelöscht
