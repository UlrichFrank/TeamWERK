## Context

YouTube bietet kostenloses, adaptives Video-Streaming mit nativer Mobile-Unterstützung. "Nicht gelistete" Videos sind nur für Nutzer mit dem direkten Link sichtbar — kein Login bei YouTube nötig. TeamWERK speichert die YouTube-Video-IDs in SQLite und gibt sie nur nach Berechtigungsprüfung an das Frontend weiter. Nutzer interagieren nie direkt mit YouTube; der Link wird ausschließlich über die API geliefert.

## Goals / Non-Goals

**Goals:**
- YouTube-Links zentral erfassen und mit Metadaten (Titel, Datum, Team, Beschreibung) versehen
- Berechtigungskontrolle: Rolle und (spätere Stufe) Team-Zugehörigkeit
- Frontend: Videoliste mit eingebettetem YouTube-Player (`<iframe>`)
- Kein YouTube-API-Key erforderlich (öffentlicher Embed reicht für nicht gelistete Videos)

**Non-Goals:**
- Eigener Video-Storage oder Video-Transcoding
- YouTube-API-Integration (Uploads direkt aus TeamWERK)
- Download-Funktion für Videos
- Kommentare oder Reaktionen

## Decisions

### YouTube als Storage, TeamWERK als Gate

**Entscheidung:** Videos werden manuell auf YouTube als "nicht gelistet" hochgeladen. TeamWERK speichert nur die YouTube-Video-ID (11 Zeichen, z.B. `dQw4w9WgXcQ`). Der Embed-URL (`https://www.youtube.com/embed/<id>`) wird nur nach Berechtigungsprüfung zurückgegeben.

**Warum:** Kein eigener Video-Storage, kein Streaming-Proxy, kein RAM-Problem auf dem VPS. YouTube übernimmt Transcoding, CDN und adaptive Bitrate.

**Risiko:** Jemand teilt den YouTube-Link weiter → Video ist für jeden aufrufbar. Für Vereinsvideos akzeptabel (kein sensitiver Inhalt erwartet).

### Kein YouTube-API-Key

**Entscheidung:** Kein Zugriff auf die YouTube Data API. Admin/Trainer fügen Video-IDs manuell ein.

**Warum:** Kein OAuth-Flow, kein API-Key-Management. YouTube-Embed funktioniert ohne API-Key.

### Berechtigungsmodell analog zu Dateiablage

**Entscheidung:** `visibility` CHECK-Constraint: `vereinsweit | team`. Stufe 1: Rolle. Stufe 2 (später): Team-Zugehörigkeit.

| Visibility   | Lesen (Stufe 1)          | Schreiben          |
|--------------|--------------------------|--------------------|
| vereinsweit  | alle eingeloggten Nutzer | admin, trainer     |
| team         | alle eingeloggten Nutzer | admin, trainer     |

Stufe 2: team-Videos nur für Mitglieder des referenzierten Teams.

## Risks / Trade-offs

- **Link-Leak** → Mitigation: Dokumentation für Admins, nur Vereins-Videos (kein sensitiver Inhalt)
- **YouTube löscht Video** → Mitigation: DB-Eintrag bleibt, Frontend zeigt Fehlermeldung im Embed; Admin kann Eintrag löschen
- **YouTube ändert Embed-Policy** → Mitigation: Embed-URL ist standard (`/embed/<id>`), seit Jahren stabil

## Migration Plan

1. Migration `006_spielvideos.up.sql`: Tabelle `videos`
2. Keine Filesystem-Änderungen auf VPS nötig
3. Rollback: `.down.sql` + keine weiteren Abhängigkeiten

## Open Questions

- Sollen Videos nach Team filterbar sein, oder reicht eine chronologische Liste?
- Soll ein Vorschaubild (YouTube Thumbnail) angezeigt werden? (`https://img.youtube.com/vi/<id>/hqdefault.jpg` — kein API-Key nötig)
