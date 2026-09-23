## Why

Zu einem Spiel gibt es in der Praxis typischerweise **mehrere Videos** (1. Halbzeit, 2. Halbzeit, einzelne Szenen, Auswärtsfahrt-Clips). Das Datenmodell erlaubt das bereits (`videos.game_id` ist `NULL`-fähig und ohne `UNIQUE`-Constraint), aber Liste, Detailseite und Upload-Flow präsentieren Videos heute flach und stiften beim Hochladen eines zweiten Clips zum selben Spiel Verwirrung. Diese Änderung bündelt Videos sichtbar pro Spiel bzw. pro Titel, ohne das Schema anzufassen.

## What Changes

- **Frontend Video-Liste** (`VideosPage.tsx`): aus flacher Liste wird ein Gruppen-Layout — eine Karte pro Spiel-bzw.-Titel-Gruppe; Karten mit >1 Video sind aufklappbar (Default: eingeklappt, zeigen Anzahl + Vorschau des ersten Videos).
- **Video-Detailseite** (`VideoDetailPage.tsx`): unter dem Player erscheint eine „Weitere Videos zu …"-Liste mit Direktsprung, sortiert nach `created_at` aufsteigend.
- **Video-Upload** (`VideoUploadPage.tsx`): sobald Spiel **oder** Titel gewählt sind und zur gleichen Gruppe schon Videos existieren, erscheint ein Hinweis „Es gibt bereits N Video(s) zu diesem Spiel/Titel — dies wird Video Nr. N+1" mit Titel-Vorschlag (z. B. „2. Halbzeit", „Szene 2").
- **Backend** (optional, nur falls Filter-Variante über `/api/videos?game_id=` nicht ausreicht): neuer Endpoint `GET /api/games/{id}/videos` als bequeme Geschwister-Abfrage.
- **Tests**: Liste & Detail-Endpoint geben bei mehreren Videos pro Spiel **alle** zurück; Upload-Happy-Path mit zweitem Video zum gleichen Spiel.

**Kein Schema-Change.** Gruppierungs-Logik:
- Videos mit `game_id IS NOT NULL` → Gruppen-Schlüssel = `game_id`.
- Videos mit `game_id IS NULL` → Gruppen-Schlüssel = exakter `title` (case-sensitive, trim).
- Innerhalb einer Gruppe sortiert nach `created_at ASC` (erster Upload = erste Halbzeit).

## Capabilities

### New Capabilities

- `video-grouping`: Logik und UI-Anforderungen für die Bündelung mehrerer Videos pro Spiel oder Titel.

### Modified Capabilities

_keine — `video-management`/`video-upload` aus `spielvideo-ablage` sind noch nicht archiviert; die hier neu definierten Anforderungen leben eigenständig in `video-grouping`._

## Impact

- **Code**: `web/src/pages/VideosPage.tsx`, `web/src/pages/VideoDetailPage.tsx`, `web/src/pages/VideoUploadPage.tsx`, evtl. neuer Helper `web/src/lib/videoGroups.ts`.
- **API**: bestehender `GET /api/videos` reicht clientseitig; optional ergänzend `GET /api/games/{id}/videos` (Auth wie `GET /api/videos`).
- **DB**: keine Migration nötig — `videos.game_id` ist bereits `NULL`-fähig und nicht unique.
- **Tests**: ergänzende Backend-Tests in `internal/videos/`, Frontend-Verhalten manuell verifizieren (Gruppen aufklappen, Upload-Hinweis).
- **CHANGELOG**: ein `[feat] videos: …`-Eintrag.
- **Keine RAM-, Deploy- oder Berechtigungs-Auswirkungen** — Gruppierung ist rein darstellend, Lese-Berechtigungen bleiben pro Video unverändert.
