## 1. Datenbank

- [x] 1.1 Migration `012_videos.up.sql` anlegen: Tabelle `videos` (Schema siehe `design.md`), Indizes auf `(team_id, status)`, `(season_id)`, `(status, created_at)`
- [x] 1.2 Migration `012_videos.down.sql` anlegen
- [x] 1.3 `make migrate-up` lokal ausführen und Schema prüfen
- [x] 1.4 Testfixture `internal/testutil/CreateVideo(...)` ergänzen

## 2. Backend — Package-Grundlagen

- [x] 2.1 Package `internal/videos/` anlegen mit `Handler struct{ db *sql.DB; hub *hub.EventHub; cfg *config.Config }`
- [x] 2.2 `NewHandler(db, hub, cfg)` implementieren
- [x] 2.3 Pfad-Helper `paths.go`: `RawPath(id)`, `ProcessedDir(id)`, `MasterManifestPath(id)`, `RenditionDir(id, rendition)`
- [x] 2.4 Disk-Helper `disk.go`: `FreeBytes(dir)`, `RequireFreeBytes(dir, needed, reserved)`; Tests mit fake `/tmp`
- [x] 2.5 Berechtigungs-Helper `access.go`: `CanUploadToTeam(claims, teamID)`, `CanViewVideo(claims, video)`, `CanManageTeamVideos(claims, teamID)` — Tests aus Matrix in `design.md`

## 3. Backend — Upload (Capability video-upload)

- [x] 3.1 Dependency `github.com/tus/tusd/v2` einbinden, in Tests pinnen
- [x] 3.2 `POST /api/videos` (Pre-Upload): validiert `title`, `team_id`, `season_id`, optional `description`, `game_id`; ruft `CanUploadToTeam`; ruft `RequireFreeBytes(/storage/videos, size*2.5, 2GiB)`; legt DB-Zeile `status='uploading'` an; gibt `upload_id` + tus-URL zurück
- [x] 3.3 tus-Handler unter `/api/videos/upload/*` mountieren; `OnUploadFinish`-Hook verschiebt Datei nach `raw/{id}.mp4`, ruft `ffprobe` für `duration_sec`, setzt `status='queued'`, broadcastet `video-queued`
- [x] 3.4 2-GB-Hard-Limit in tus-Konfiguration setzen
- [x] 3.5 Cleanup-Job: tus-Sessions in `uploads/` älter als 24 h löschen (in Scheduler einhängen)
- [x] 3.6 Tests: Happy-Path Init + Upload-Fertig, 403 bei fehlender Berechtigung, 507 bei Disk-Voll, 400 bei ungültigem `team_id`

## 4. Backend — Transcode-Worker (Capability video-transcode)

- [x] 4.1 `internal/videos/worker.go`: Worker-Loop mit `pickNextQueued()` (`SELECT … WHERE status='queued' ORDER BY created_at LIMIT 1`); idle-Sleep 30 s
- [x] 4.2 Crash-Recovery beim Start: `UPDATE videos SET status='queued' WHERE status='processing'`
- [x] 4.3 Pre-Transcode-Disk-Check; bei Mangel zurück auf `queued` und 1-h-Sleep
- [x] 4.4 FFmpeg-Aufruf: `nice -n 19 ffmpeg -i raw/{id}.mp4 …` → 720p+360p HLS in `processed/{id}/`; `-c:a copy` wenn AAC, sonst `-c:a aac -b:a 128k`; `-hls_time 10`, `-hls_list_size 0`
- [x] 4.5 Master-Playlist `master.m3u8` erzeugen, die beide Renditions referenziert
- [x] 4.6 Bei Erfolg: `status='ready'`, `ready_at=now()`; raw-Datei löschen; Broadcast `video-ready`
- [x] 4.7 Bei Fehler: `status='failed'`, `failure_reason` setzen; raw-Datei für Debug behalten (löscht Cleanup nach 7 Tagen)
- [x] 4.8 Push-Notification an Hochladenden + alle Team-Spieler + Eltern + Trainer (Goroutine, nicht-blockierend)
- [x] 4.9 Worker in `cmd/teamwerk/main.go` starten (eine Goroutine); sauberes Beenden bei SIGTERM
- [x] 4.10 Tests: serielle Verarbeitung (zwei queued → nacheinander), Failure-Pfad mit kaputter Quelle (FFprobe-Reject), Disk-Mangel-Pfad

## 5. Backend — Streaming (Capability video-stream)

- [x] 5.1 `internal/videos/stream_token.go`: HMAC-Signing mit `VIDEO_STREAM_SECRET`; Claims `{vid, uid, exp}`; `Sign(vid, uid)` und `Verify(token, vid) → uid, error`
- [x] 5.2 `.env.example` um `VIDEO_STREAM_SECRET` ergänzen; Config-Loader ergänzen; bei fehlendem Secret im Production-Modus Fail-Fast
- [x] 5.3 `GET /api/videos/{id}/play`: Auth-Check via `CanViewVideo`; signiert Token und gibt `{ token, master_url }` zurück
- [x] 5.4 Stream-Routen unter `/api/videos/{id}/hls/*`: Middleware verifiziert `?st=…`-Token gegen `vid` aus Pfad; bei Fehler 403
- [x] 5.5 `master.m3u8`-Auslieferung: liest Datei, ersetzt Rendition-URLs sodass `?st=…` mitgegeben wird
- [x] 5.6 Segment- und Rendition-Manifest-Auslieferung über `http.ServeContent` (Range-Support, ETag)
- [x] 5.7 Tests: 200 für gültigen Token, 403 für fehlenden/abgelaufenen/falscher-vid-Token, Range-Request liefert 206

## 6. Backend — CRUD und Liste (Capability video-management)

- [x] 6.1 `GET /api/videos`: liefert berechtigte Videos (JOIN über `team_memberships` / `team_trainers` / `family_links` / Rolle), Filter `?team_id=`, `?status=`, Paginierung `limit/offset`; Format `{ items, total }`
- [x] 6.2 `GET /api/videos/{id}`: Details inkl. Status; 404 wenn nicht sichtbar
- [x] 6.3 `PATCH /api/videos/{id}`: Titel/Beschreibung/`game_id` ändern; Auth via `CanManageTeamVideos`
- [x] 6.4 `DELETE /api/videos/{id}`: Auth-Check; DB-Zeile löschen; alle Dateien (`raw/{id}.mp4`, `processed/{id}/`) entfernen; Broadcast `video-deleted`
- [x] 6.5 Routen in `internal/app/router.go` registrieren (`/api/videos/...`) — Auth-Tier "Authenticated", Upload-Routen "Vorstand+Trainer+sportl. Leitung+Admin"
- [x] 6.6 Tests: Liste filtert korrekt nach Berechtigung (Spieler, Elternteil, Trainer, Vorstand), 403/404 bei Zugriffsversuch auf fremdes Team, 200 + Datei-Löschung bei DELETE

## 7. Backend — Retention (Scheduler)

- [ ] 7.1 Scheduler-Job `videos_retention` (täglich 03:00 in `internal/scheduler/`): findet Videos mit `season.end_date < now() - 90 days`, löscht Eintrag + Dateien
- [ ] 7.2 Vorlauf-Push (T-7): Job sendet 7 Tage vor geplanter Löschung an alle Team-Trainer Push "Video XY wird am DD.MM. gelöscht"
- [ ] 7.3 Idempotenz via `notification_log`-Eintrag pro `(video_id, retention_warning)`
- [ ] 7.4 Tests: Stichtag-Logik (88, 90, 92 Tage), Push-Idempotenz, kein Löschen wenn `end_date` NULL

## 8. Frontend — Liste und Detail

- [ ] 8.1 `web/src/pages/VideosPage.tsx`: serverseitige Suche/Filter (`team_id`), "Mehr laden"-Button; Card-Layout mobil, Tabelle desktop
- [ ] 8.2 Status-Pill anzeigen (queued/processing/ready/failed) mit `brand-*`-Tokens und lucide-Icons
- [ ] 8.3 `web/src/pages/VideoDetailPage.tsx`: ruft `/play`, lädt `hls.js`, mountet `<video>` mit `master_url`; Fallback-Hinweis bei Browser ohne HLS-Support
- [ ] 8.4 Lösch-Modal mit Bestätigung; PATCH-Inline-Form für Titel/Beschreibung
- [ ] 8.5 Live-Updates via `useLiveUpdates`: `video-queued`/`video-ready`/`video-deleted` → Liste neu laden
- [ ] 8.6 Routen `/videos` und `/videos/:id` in `App.tsx`; Nav-Eintrag in `AppShell.tsx`

## 9. Frontend — Upload-Form

- [ ] 9.1 `pnpm add hls.js tus-js-client` in `web/`
- [ ] 9.2 `web/src/pages/VideoUploadPage.tsx`: Formular mit Titel/Beschreibung/Team-Select/Spiel-Select/Datei
- [ ] 9.3 Datei-Check vor Upload: `file.size > 2 GB` → Fehler; ohne erwartete Größe POST `/api/videos`
- [ ] 9.4 tus-Client startet Upload an `/api/videos/upload/`; Progress-Bar (Prozent + Restzeit)
- [ ] 9.5 Bei Fehler 507: klare Meldung "Server-Speicher voll, bitte später erneut versuchen oder Admin informieren"
- [ ] 9.6 Bei Abbruch/Reload: tus speichert Position in localStorage; Re-Open der Seite zeigt "Upload fortsetzen?"-Button

## 10. Deployment-Vorbereitung

- [x] 10.1 `deploy/setup-vps.sh` um `apt install -y ffmpeg` ergänzen; Disk-Layout `/storage/videos/{uploads,raw,processed}` mit korrektem Owner anlegen
- [x] 10.2 `deploy/vps-setup-runbook.md` ergänzen: Storage-Erweiterung, ffmpeg-Version, neuer Env-Eintrag
- [x] 10.3 `.env.example` aktualisieren: `VIDEO_STREAM_SECRET`, optional `VIDEO_STORAGE_DIR` (default `/storage/videos`), `VIDEO_RESERVED_BYTES` (default 2 GiB)

## 11. Validierung

- [ ] 11.1 `make test` grün (inkl. neuer Architektur-Test-Klassifizierung für `internal/videos`)
- [ ] 11.2 `make lint` grün
- [ ] 11.3 `openspec validate spielvideo-ablage --strict` grün
- [ ] 11.4 Manueller End-to-End-Test: Upload → Transcode-Wartezeit → Push erhalten → Abspielen funktioniert → Löschen entfernt Dateien
- [ ] 11.5 `/verify-change` ausführen
