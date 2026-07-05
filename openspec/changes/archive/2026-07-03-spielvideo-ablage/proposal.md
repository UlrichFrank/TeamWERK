## Why

Spielvideos (Highlight-Clips, Spielhälften) sollen zentral im System verwaltet werden. Die Videos zeigen **Minderjährige** (Junioren-Teams), daher ist **echte Zugangskontrolle** erforderlich — kein "Security by Obscurity"-Modell und kein Hosting bei US-Anbietern (DSGVO Art. 8, Schutz Minderjähriger).

Konsequenz: TeamWERK hostet die Videos selbst, transcodiert sie ressourcenschonend im Hintergrund und liefert sie ausschließlich an berechtigte Nutzer aus. Der VPS-Speicher wird bewusst beachtet (Disk-Guard, Auto-Retention).

## What Changes

- Neue Seite `/videos` im Frontend: Videoliste pro Team mit eingebettetem HLS-Player
- Upload direkt im Browser über **tus-Protokoll** (resumable, max 2,5 GB) — kein Drittanbieter
- Hintergrund-**Transcode** mit `nice -n 19` (FFmpeg → HLS, 720p + 360p), seriell über Worker-Goroutine
- **Stream-Token** (HMAC-signiert, 1 h) schützt jede Segment-Auslieferung
- **Disk-Guard** prüft vor Upload und vor Transcode den freien Speicher; Worker pausiert bei kritisch wenig Platz
- **Retention**: Videos werden 90 Tage nach Saisonende automatisch gelöscht
- **Push-Notification** an Hochladenden + alle Team-Mitglieder/-Eltern, sobald Video fertig ist
- Optionaler Spiel-Link (`game_id`): Video erscheint in der Spiel-Detailansicht
- Keine YouTube-Integration, kein Cloud-Storage, kein Backup in Phase 1

## Capabilities

### New Capabilities

- `video-upload`: Resumable Video-Upload via tus, mit Vorab-Disk-Check und Größen-Limit
- `video-transcode`: Hintergrund-Transcode in HLS-Renditions (720p/360p) über serielle Worker-Queue
- `video-stream`: HLS-Auslieferung mit Stream-Token-Authentifizierung und Range-Support
- `video-management`: CRUD für Video-Metadaten, Berechtigungen, Saison-Retention

### Modified Capabilities

*(keine)*

## Impact

- Neues Package `internal/videos/` (Handler, DB-Zugriff, Transcode-Worker, Stream-Token)
- Neue DB-Migration `013_videos.up.sql/.down.sql`: Tabelle `videos`
- Neue API-Routen unter `/api/videos/` (Upload, Liste, Player-Token, HLS, CRUD)
- Neue Frontend-Seite `web/src/pages/VideosPage.tsx` + Detail-Seite mit `hls.js`
- Neuer Nav-Eintrag im `AppShell` (für alle Nutzer mit Team-Zugehörigkeit oder Trainer-Funktion)
- Neuer Scheduler-Job: Saison-Retention (90 Tage nach `saisons.end_date`)
- Neue Storage-Pfade: `/storage/videos/{uploads,raw,processed}/`
- Externe Abhängigkeiten: `ffmpeg` muss auf dem VPS installiert sein (`apt install ffmpeg`); `tusd` als Go-Library (`github.com/tus/tusd`); `hls.js` als Frontend-Dependency
- Sicherheits-Modell: echte Auth (kein Obscurity); Streaming-Endpoints validieren HMAC-Token
- Kostenrahmen: keine zusätzlichen laufenden Kosten in Phase 1 (Storage wird auf VPS erweitert)
