## Context

Spielvideos zeigen Minderjährige aus Junioren-Teams. Eine YouTube-basierte Lösung mit "nicht gelisteten" Videos würde bedeuten: ein einziges Weiterleiten des Links macht das Video weltweit zugänglich, dauerhaft auf Google-Servern (USA). Das ist für Aufnahmen von Minderjährigen (DSGVO Art. 8, Schutz besonders schutzbedürftiger Personen) nicht akzeptabel.

TeamWERK hostet die Videos daher selbst auf dem IONOS-VPS. Der Speicher ist begrenzt, daher müssen Upload-Größe, Transcode-Output und Aufbewahrungsdauer aktiv kontrolliert werden. Der Storage wird vor produktiver Nutzung manuell erweitert; die Software setzt klare Disk-Guards, damit der Server nie volläuft.

Die Architektur folgt vier Bausteinen: **Upload** (resumable via tus), **Transcode** (Hintergrund-Worker mit nice -n 19), **Streaming** (HLS mit Stream-Token-Auth) und **Management** (CRUD, Berechtigungen, Retention).

## Goals / Non-Goals

**Goals:**
- Videos werden ausschließlich an berechtigte Nutzer (Team-Mitglieder, deren Trainer, deren Eltern) ausgeliefert
- Resumable Upload bis 2,5 GB pro Datei
- Hintergrund-Transcode ohne erkennbare Beeinträchtigung des Webserver-Betriebs
- Adaptive Bitrate-Streaming (HLS 720p + 360p), funktioniert auf Desktop und Mobile
- Datensparsamkeit durch automatische Retention 90 Tage nach Saisonende
- Voraussichtbare Disk-Auslastung — kein "Datei abgewiesen, weil Server voll"

**Non-Goals:**
- Kein YouTube, kein Drittanbieter-Hosting
- Kein eigenes CDN, kein Object Storage in Phase 1
- Kein Backup der Video-Dateien in Phase 1 (Trainer behalten Originale)
- Kein Download-Endpoint (Streaming reicht; Original wird nach Transcode gelöscht)
- Keine Kommentare, Reaktionen, Bookmarks
- Keine Live-Streaming-Funktion
- Keine vereinsweite Sichtbarkeit (jedes Video gehört zu genau einem Team)

## Decisions

### Self-hosted Storage, kein externer Anbieter

**Entscheidung:** Videos werden auf der VPS-Disk unter `/storage/videos/` abgelegt. Der Storage wird bei Bedarf erweitert; die Software prüft vor jedem Upload und vor jedem Transcode den freien Speicher.

**Warum:** Echte Zugangskontrolle erfordert, dass der Server die Auslieferung selbst kontrolliert. Cloud-Storage (z.B. Hetzner Object Storage) wäre möglich, fügt aber Komplexität und ein zweites System hinzu; in Phase 1 nicht nötig.

**Trade-off:** Skalierungsgrenze ist die VPS-Disk. Sobald mehr als ein Team produktiv hochlädt, kann das eng werden — dann wechsel auf Object Storage (Phase 2).

### tus-Protokoll für Upload

**Entscheidung:** Upload erfolgt über das tus-Protokoll (`github.com/tus/tusd` als Go-Library, `tus-js-client` im Browser). Max 2 GB pro Datei.

**Warum:** Trainer laden oft vom Spielfeldrand mit mobilen Daten hoch. Ein 1-GB-Upload kann mehrere Minuten dauern; Mobilfunk-Wechsel, Bildschirm-Sleep, kurzer WLAN-Drop führen sonst zu Komplettverlust. tus erlaubt echte Session-übergreifende Wiederaufnahme.

**Alternative geprüft:** Single-Shot multipart (zu fragil), eigenes Chunked-Protokoll (200+ Zeilen Eigencode, dasselbe Problem). tus ist Industriestandard, klein in der Abhängigkeit.

### HLS mit zwei Renditions (720p + 360p)

**Entscheidung:** FFmpeg transcodiert in HLS mit Master-Playlist und zwei Bitrates. Format: H.264, AAC, CRF 26, preset medium.

**Warum:** Trainer-Analyse braucht 720p, 4G-Mobile braucht 360p Fallback. Adaptive Bitrate macht Streaming auf wechselnden Verbindungen robust. HLS ist Standard und vom `<video>`-Element mit `hls.js` überall lauffähig.

**Performance-Erwartung:** preset medium ≈ 0.4× Echtzeit pro Rendition auf 1 vCPU mit `nice -n 19`. Eine 60-min-Quelle dauert ca. 2.5–3 h pro Rendition, also ~5–6 h für beide Renditions seriell. Akzeptabel für ein Nachmittag-Upload mit Push-Notification am Abend.

**Bewusst kein 1080p:** verdoppelt Disk-Verbrauch und Transcode-Zeit ohne Mehrwert für taktische Analyse.

**Bewusst kein Audio-Re-Encode wenn AAC:** `-c:a copy` wenn Quelle bereits AAC ist; sonst `-c:a aac -b:a 128k`.

### Serielle Transcode-Queue, DB als Wahrheit

**Entscheidung:** Genau **eine** Worker-Goroutine zieht Videos mit `status = 'queued'` aus der DB, transcodiert sie und markiert sie als `ready`. Bei Server-Restart werden hängende `processing`-Jobs wieder auf `queued` gesetzt.

**Warum:** Auf einem 1-vCPU-VPS würden parallele Transcodes alle Uploads langsam machen und RAM-Druck erzeugen. Serielle Verarbeitung ist vorhersehbar. DB als State-Maschine vermeidet ein zusätzliches Job-System.

**Status-Lebenszyklus:**
```
uploading → queued → processing → ready
                  ↘            ↘ failed
```

### Stream-Token statt Cookie-Auth

**Entscheidung:** Vor dem Abspielen ruft das Frontend `GET /api/videos/{id}/play` auf und erhält einen kurzlebigen, HMAC-signierten Token (HS256, exp 1 h, Claims: `vid`, `uid`, `exp`). Alle HLS-Requests führen den Token als `?st=…` mit. Jede Segment-Auslieferung validiert nur die Signatur und die Claims — keine DB-Abfrage.

**Warum:** HLS löst pro Video ~50–500 HTTP-Requests aus. JWT-Bearer-Header funktioniert mit `hls.js` nur über `xhrSetup` und ist fragil bei Range-Requests. Signed-Token im Query ist Stand der Technik (CloudFront, Mux, etc.), millisekundenschnell zu validieren, und der Token läuft ohnehin nach 1 h ab.

**Sicherheit:** Token wird im Webserver-Log mit `?st=...` sichtbar — Lifetime 1 h und Bindung an `vid`+`uid` machen ihn praktisch wertlos außerhalb der Session. Optionaler Hardening-Schritt (Phase 2): `?st=` aus Nginx-Access-Logs filtern.

### Strenge Berechtigung — kein "vereinsweit"

**Entscheidung:** Jedes Video gehört zu genau einem Team (`team_id NOT NULL`). Sichtbar sind Videos eines Teams nur für:

- aktive Spieler des Teams (`team_memberships`)
- Trainer des Teams (`team_trainers`)
- Eltern dieser Spieler (`family_links`)
- Vorstand und Admin (immer)

**Warum:** Minderjährige Spieler dürfen nicht für unbeteiligte Vereinsmitglieder sichtbar sein. Das Berechtigungsmodell ist enger als bei der Dateiablage (dort gibt es `vereinsweit`).

**Upload-Berechtigung** (orthogonal): Trainer-Funktion, Sportliche Leitung, Vorstand, Admin. Beim Upload muss ein `team_id` ausgewählt werden, in dem der Hochladende Trainer ist (oder admin/vorstand).

**Lösch-Berechtigung:** Jeder Trainer des Teams, Vorstand, Admin (nicht nur der ursprüngliche Hochlader — Team-Trainer können sich gegenseitig vertreten).

### Disk-Guard auf drei Ebenen

**Entscheidung:** Drei Prüfungen schützen vor Disk-Überlauf:

1. **Pre-Upload-Check** (vor tus-Init): Client meldet erwartete Dateigröße. Server prüft:
   `free(/storage) ≥ size × 2.5 + RESERVED_BYTES`
   `RESERVED_BYTES = 2 GiB` (für DB-Wachstum, Logs, parallele Uploads)
   Faktor 2.5: raw + processed-Peak (vor raw-Delete) + Sicherheit
   Bei Fehler: HTTP 507 Insufficient Storage

2. **Pre-Transcode-Check** (vor FFmpeg-Aufruf): Disk könnte zwischenzeitlich anders aussehen. Wenn frei < geschätzte Output-Größe × 1.5: Video bleibt `queued`, Worker schläft 1 h und versucht erneut.

3. **Periodischer Check** (Scheduler, stündlich): Wenn `free < 5 GiB`: Worker pausiert, Push an Admin "Disk niedrig".

**Implementierung:** `syscall.Statfs` auf `/storage`. Ein Helper in `internal/videos/disk.go`.

### Saison-basierte Retention (90 Tage Karenz)

**Entscheidung:** Täglicher Scheduler-Job löscht Videos, deren zugehörige Saison vor mehr als 90 Tagen geendet hat (`saisons.end_date < now() - 90 days`). Gelöscht werden: DB-Eintrag, raw-Datei (falls noch da), processed-HLS-Ordner.

**Warum:** Datensparsamkeit (DSGVO). Sommerpause wird abgedeckt. 90 Tage Karenz geben Trainern Zeit, eine Sicherung zu ziehen, falls sie das Video behalten wollen.

**Vorlaufzeit:** 7 Tage vor Löschung Push-Notification an alle Trainer des Teams: "Video XY wird am DD.MM. gelöscht."

### Push-Notifications

**Entscheidung:** Bei `status = ready` wird per Goroutine eine Push-Notification verschickt an:

- den Hochladenden
- alle aktiven Spieler des Teams
- alle Eltern dieser Spieler (über `family_links`)
- alle Trainer des Teams

Inhalt: `"Neues Video: {team_name} — {title}"`, Ziel-URL `/videos/{id}`.

Bei Retention-Warnung (T-7) Push an Trainer des Teams.

**Warum:** Pattern ist konsistent mit Chat/Games/Duties. Nicht-blockierend via Goroutine. Badge wird nicht gesetzt (kein Unread-Counter wie Chat).

### Storage-Layout

```
/storage/videos/
  uploads/           tus-Sessions (Chunks während Upload)
  raw/{id}.mp4       fertiger Upload, wird nach Transcode gelöscht
  processed/{id}/
    master.m3u8      multi-variant Manifest
    720p/
      index.m3u8
      seg_001.ts … seg_NNN.ts
    360p/
      index.m3u8
      seg_001.ts … seg_NNN.ts
```

`{id}` ist die `videos.id`. Pfad-Konstruktion in einem Helper zentralisiert.

### Datenmodell

```sql
CREATE TABLE videos (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  title           TEXT NOT NULL,
  description     TEXT,
  team_id         INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  season_id       INTEGER NOT NULL REFERENCES saisons(id),
  game_id         INTEGER REFERENCES games(id) ON DELETE SET NULL,
  status          TEXT NOT NULL CHECK (status IN
                    ('uploading','queued','processing','ready','failed')),
  upload_id       TEXT,                    -- tus session
  size_bytes      INTEGER,                 -- finale Originalgröße
  duration_sec    INTEGER,                 -- aus ffprobe nach Upload
  failure_reason  TEXT,
  created_by      INTEGER NOT NULL REFERENCES users(id),
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ready_at        DATETIME
);
CREATE INDEX idx_videos_team_status ON videos(team_id, status);
CREATE INDEX idx_videos_season ON videos(season_id);
CREATE INDEX idx_videos_status_created ON videos(status, created_at);
```

### Routen-Übersicht

```
Upload (Auth: trainer / sportl. Leitung / vorstand / admin)
  POST   /api/videos                    Metadata + tus-Session init (mit Disk-Check)
  *      /api/videos/upload/*           tus-Endpoints (tusd Handler)

Lesen (Auth: berechtigte Nutzer, Filter via JOIN)
  GET    /api/videos                    Liste der berechtigten Videos
  GET    /api/videos/{id}               Details
  GET    /api/videos/{id}/play          Liefert Stream-Token aus

Streaming (Auth: Stream-Token in ?st=)
  GET    /api/videos/{id}/hls/master.m3u8
  GET    /api/videos/{id}/hls/{rendition}/index.m3u8
  GET    /api/videos/{id}/hls/{rendition}/{segment}

CRUD (Auth: Team-Trainer / Vorstand / Admin)
  PATCH  /api/videos/{id}               Titel/Beschreibung/game_id ändern
  DELETE /api/videos/{id}               Video + Dateien löschen
```

## Risks / Trade-offs

- **Disk-Überlauf trotz Guard** → Mitigation: drei Prüfebenen, Admin-Push, Worker pausiert; im Worst-Case wird Upload abgelehnt (kein silent fail).
- **Phase-1 ohne Backup** → Ein Disk-Crash bedeutet Verlust aller Videos. Mitigation: klare Trainer-Kommunikation ("Originale behalten"), Backup als Phase-2-Aufgabe vorgemerkt.
- **Transcode-Zeit lang** → 6 h für 60-min-Quelle. Mitigation: klare UX ("Du bekommst eine Benachrichtigung"), Worker läuft auch nachts.
- **Stream-Token-Leak in Logs** → 1 h Lifetime begrenzt Schaden; Phase-2-Hardening: Nginx-Log-Filter.
- **Aktiv bösartiger Server** kann Auth umgehen → außerhalb des Bedrohungsmodells (gilt für jeden Server, der Code ausliefert).
- **Quelle ohne brauchbares Video** (z.B. .mp4 mit Audio-only) → FFprobe-Check vor Transcode-Start; bei Fehler `status=failed` mit Begründung.

## Migration Plan

1. **DB-Migration `013_videos.up.sql`**: Tabelle `videos` mit Indizes und Constraints
2. **Storage-Verzeichnisse**: `/storage/videos/{uploads,raw,processed}` auf VPS anlegen (Setup-Script-Update)
3. **VPS-Setup**: `apt install ffmpeg` (Setup-Runbook ergänzen); `ffmpeg -version` ≥ 4.x sicherstellen
4. **Stream-Token-Secret**: neuer `.env`-Eintrag `VIDEO_STREAM_SECRET` (separates HMAC-Secret von `JWT_SECRET`, damit Token-Kompromiss nicht JWTs betrifft)
5. **Frontend-Dep**: `pnpm add hls.js tus-js-client` in `web/`
6. **Scheduler-Job**: Retention in `internal/scheduler/` einhängen (täglich um 03:00)
7. **Roll-out**: zuerst für ein Test-Team, dann freigeben
8. **Rollback**: Migration `013.down.sql`, Storage-Verzeichnisse manuell aufräumen, Frontend-Route hinter Feature-Flag falls nötig

## Open Questions

*(keine — Architektur ist mit dem Auftraggeber abgestimmt)*
