## Why

Spielvideos (Highlight-Clips, Spielhälften) werden aktuell nicht zentral im System verwaltet. YouTube bietet kostenloses Hosting mit adaptivem Streaming — TeamWERK übernimmt die Zugangskontrolle, damit Videos nicht öffentlich auffindbar sind.

## What Changes

- Neue Seite `/videos` im Frontend: Videoliste mit eingebettetem YouTube-Player
- Admin/Trainer können YouTube-Links (nicht gelistet) mit Metadaten erfassen
- Berechtigungsprüfung im Go-Backend: Link wird nur für berechtigte Nutzer ausgeliefert
- Kein eigener Video-Storage — YouTube ist der Byte-Store

## Capabilities

### New Capabilities

- `video-management`: YouTube-Video-Links erfassen, bearbeiten und löschen (Admin/Trainer)
- `video-access`: Videoliste und Embed-Links für berechtigte Nutzer abrufen

### Modified Capabilities

*(keine)*

## Impact

- Neues Package `internal/videos/` (Handler, DB-Zugriff)
- Neue DB-Migration: Tabelle `videos`
- Neue API-Routen unter `/api/videos/`
- Neue Frontend-Seite `web/src/pages/VideosPage.tsx` + Nav-Eintrag
- Kein externer Dienst außer YouTube (kein API-Key nötig für nicht gelistete Videos)
- Sicherheitshinweis: YouTube-Link ist "Security by Obscurity" — geeignet für Vereinsvideos, nicht für vertrauliche Inhalte
