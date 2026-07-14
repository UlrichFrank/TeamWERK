## Context

Der `AuthImage`-Layout-Shift beim Bild-Load ist die letzte verbleibende
Ursache dafür, dass ein Klick in der Chat-Übersicht nicht zuverlässig am
Ende des Chatverlaufs landet. Der bereits deployte Client-Fix (Aspect-
Ratio-Probe im Browser nach Blob-Download, Commit `b01a657`) verhindert
das Nach-Ruckeln beim erneuten Rendern, aber der *erste* Übergang vom
6-rem-Placeholder auf die echte Bildhöhe passiert weiterhin.

Der `media`-Store aus `chat-broadcast-bilder` speichert bisher
`disk_name`, `mime_type`, `size`, `uploaded_by` — die Pixel-Dimensionen
kennt nur der Client, und auch erst *nach* dem Blob-Download. Der Server
sitzt bei jedem Upload sowieso mit den Bytes im Speicher: eine Probe kostet
dort quasi nichts. Diesen Zeitpunkt nutzen wir.

Randbedingungen aus `docs/agent/*.md`:
- **Kein ORM** (`03-go.md`): direktes `database/sql`, Handler-Struct-
  Pattern; das ist Zeile-für-Zeile mit dem Bestandscode kompatibel.
- **VPS-Budget** (1 GB RAM, `04-api-db.md`): nur `image.DecodeConfig`,
  kein Full-Decode; Backfill sequentiell, keine Parallelisierung.
- **Migrations-Nummerierung** (`02-workflow.md`): nächste freie ist 030
  (029 = `unified-user-photo`).
- **SSE** (`06-gotchas.md`): keine neue Mutationsroute, also kein neuer
  `Broadcast`-Aufruf nötig — Änderung ist rein an Read-Pfaden und einem
  bestehenden Write-Pfad (Upload).
- **Tests** (`07-testing.md`): Upload-Route existiert; wir ergänzen
  Tests für „Response enthält Width/Height nach Upload" und „Backfill
  füllt NULL-Zeilen".

Reihenfolge: `chat-broadcast-bilder` steht auf `complete` und wartet auf
Archivierung. Migration 028 dort legt die `media`-Tabelle an. Unsere 030
setzt darauf auf. Wir dokumentieren die Abhängigkeit im Proposal; ein
Blocker ist es nicht (der Merge kann parallel laufen, die Archivierung
klärt die Delta-Reihenfolge).

## Goals / Non-Goals

**Goals:**
- Chat-Bilder haben ab dem ersten React-Render die korrekte
  `aspect-ratio` → kein Layout-Shift mehr beim Blob-Load.
- Klick auf eine Konversation in der Chat-Übersicht landet
  reproduzierbar am Ende des Verlaufs, unabhängig davon wie viele
  Bild-Nachrichten die letzte Seite enthält.
- Bestandsbilder (aktuell ~einige Dutzend) werden per einmaligem
  Backfill nachgezogen; kein manueller Eingriff nötig.
- Additive Response-Erweiterung, keine Breaking Changes für alte
  Frontend-Bundles (die Felder ignorieren sie stillschweigend).

**Non-Goals:**
- **Kein BlurHash / kein Thumbnail-Preview**. Das wäre die Kür (WhatsApp
  liefert einen ~30-Byte-Blur mit); für unseren Fall reicht die
  Dimensions-Reservation. Kann später als eigener Change nachgereicht
  werden.
- **Kein Bild-Resizing serverseitig**. Der Client verkleinert bereits
  vor dem Upload auf ≤ 1 MB (`docs/agent`-Konvention). Wir speichern
  weiterhin nur das Original.
- **Kein Umbau des Client-seitigen Probe-Codes**. Der bleibt als
  Fallback für alte Bilder ohne Dims und für andere `AuthImage`-Aufrufer
  (falls die je auftauchen, aktuell nur Chat/Broadcast).
- **Kein Retouch am Match-Report-Bild-Pfad**. Der nutzt eine eigene
  Tabelle/Storage; Layout-Shift ist dort kein Problem (Formular-Kontext).

## Decisions

### Decision 1: `image.DecodeConfig` statt Full-Decode oder externer Tools

**Was**: In `internal/media/handler.go` nach der MIME-Prüfung ein
`decodeDimensions(data, mimeType) (w, h int, ok bool)` aufrufen, das
`bytes.NewReader(data)` an `image.DecodeConfig` (JPEG/PNG/GIF stdlib) bzw.
`webp.DecodeConfig` reicht.

**Warum**: `DecodeConfig` liest nur den Header (typisch ≤ 1 KB) statt das
ganze Bild in Pixel-Buffer zu materialisieren. Für ein 1-MB-JPEG spart
das ~50 ms und ~4 MB Peak-Speicher gegenüber `image.Decode`. Das RAM-
Budget des VPS bleibt unangetastet.

**Alternativen**:
- `ffprobe` per exec: schon vorhanden für Videos, aber externer Prozess-
  Fork für ein paar Header-Bytes ist unverhältnismäßig.
- `bimg`/`libvips` bindings: cgo, verletzt „kein cgo"-Konvention.
- Nur JPEG/PNG unterstützen und WEBP ignorieren: bricht die
  Upload-Whitelist (`extByMime` erlaubt WEBP explizit).

### Decision 2: `golang.org/x/image/webp` als neue Dependency

**Was**: Ein einziger Import (~500 KB im Binary, keine transitiven
Non-x-Abhängigkeiten). Registriert sich per `init()` beim `image`-Package,
so dass `image.DecodeConfig` auch WEBP versteht — oder direkter Aufruf
von `webp.DecodeConfig`, je nachdem was cleaner ist.

**Warum**: WEBP-Uploads sind heute möglich (Client kann sie erzeugen,
insbesondere von iOS-Shares). Ohne den Decoder würde jedes WEBP-Bild in
der DB als `width IS NULL` landen und dauerhaft auf Client-Probe angewiesen
bleiben.

**Alternativen**:
- WEBP als `width=NULL` akzeptieren und Client-Probe machen lassen: geht,
  aber lässt eine Zwei-Klassen-Bild-Behandlung im Code stehen.
- Nur JPEG/PNG in der Upload-Whitelist erlauben: nutzerfeindlich (iOS-
  Screenshots kommen oft als WEBP).

### Decision 3: Backfill als Goroutine bei `serve()`, kein CLI-Subcommand

**Was**: Neue Datei `internal/media/backfill.go` nach exakt dem Muster
von `internal/videos/backfill.go`: idempotenter `Backfill(ctx, db,
mediaDir)`, gestartet in `cmd/teamwerk/main.go serve()` als
`go func() { media.Backfill(...) }()`. Loggt Start, Anzahl bearbeitete,
Ende. Bei Fehlern pro Datei: loggen, weiter.

**Warum**: Der Video-Codec-Backfill funktioniert seit Monaten unauffällig
nach dem Muster; Kopieren minimiert Bug-Oberfläche. Ein separater
`teamwerk backfill-media` würde manuellen Deploy-Schritt bedeuten — der
Punkt der Idempotenz ist genau, das zu vermeiden.

**Alternativen**:
- Migration 030 macht den Backfill in SQL: geht nicht, SQLite kann keine
  Bilder decoden.
- Migration ruft Go-Code: mischt die Migrations-Ebene (SQL-first)
  unnötig.
- Cronjob via Scheduler: overengineered für einen Einmal-Job.

### Decision 4: Response-Felder `mediaWidth`/`mediaHeight` als
`*int`/`omitempty`

**Was**: Beide Felder als Pointer auf int, JSON-Tag `omitempty`. Für alte
Bilder ohne Dims (Bestand bis Backfill, plus WEBP falls Decode
scheitert) werden die Felder weggelassen.

**Warum**: Ein `int`-Feld mit Nullwert 0 wäre für den Client
mehrdeutig („echt 0 px"? Nein — aber Client-Code muss dann prüfen).
Fehlen ist klarer als 0.

**Alternativen**:
- Ein zusammengesetztes Objekt `media: {url, width, height}`:
  Response-Struktur würde umgemodelt, alte Clients verwirrt (aktuell
  liegt `mediaUrl` flach). Additiv bleibt kompatibel.
- Aspect-Ratio als Dezimalzahl statt zwei Ints: verliert Information;
  Ints sind offensichtlicher zu lesen.

### Decision 5: `AuthImage` behält Client-Probe als Fallback

**Was**: `AuthImage`-Props werden erweitert um `naturalWidth?: number`,
`naturalHeight?: number`. Wenn beide gesetzt: `aspectRatio` sofort
anwenden, `Image()`-Probe überspringen. Sonst: bisheriger Probe-Weg.

**Warum**: Backfill kann irgendwo scheitern (korrupte Datei, gelöschtes
File aber vorhandene DB-Zeile). Falls Server keine Dims liefert, muss
das UI trotzdem funktionieren — mit dem existierenden Client-Probe geht
das.

### Decision 6: `openUser`-Deep-Link konsolidieren (Bonus-Task)

**Was**: In `ChatPage.tsx` die Zeilen ~367-381 (`?openUser=<id>`-Handler)
so umbauen, dass sie nach `POST /api/chat/conversations` ein
`openConversation(conv)` aufrufen statt inline `setActiveConv` +
`loadMessages`. Damit erbt der Pfad automatisch das `forceScrollToEndRef
= true` aus `openConversation`.

**Warum**: Duplikation, die schon einmal den identischen Bug reproduziert
hat (Klick von PersonChip → landet nicht am Ende). Konsolidieren
verhindert Regressionen bei künftigen Änderungen an `openConversation`.

## Risks / Trade-offs

- **WEBP-Decode-Fehler** (seltene, halbkorrupte Dateien): Datei wird
  akzeptiert, aber `mediaWidth`/`mediaHeight` fehlen. **Mitigation**:
  Client-Probe-Fallback in `AuthImage` bleibt bestehen; UI degradiert
  auf das Vor-Fix-Verhalten (kleiner Shift beim Load), bricht nicht.

- **Backfill-Laufzeit auf großen Media-Beständen**: aktuell ~einige
  Dutzend Bilder — trivial (<1 s). Bei 10 000 Bildern wäre es ~1 min
  Header-Reads. **Mitigation**: Backfill läuft im Hintergrund; blockt
  nichts. Falls je nötig: `LIMIT N per Iteration + sleep` einbauen (wie
  im Video-Backfill vorbereitet).

- **`golang.org/x/image/webp` als neue Dep**: klein und stabil, aber
  eine Dep mehr. **Mitigation**: Alternativen (kein WEBP-Support) sind
  schlechter; die x/image-Familie zählt effektiv zur Go-Standardbibliothek.

- **Race zwischen Upload und Backfill**: Backfill nimmt nur
  `width IS NULL`-Zeilen; ein frisch uploadedes Bild hat width bereits
  gesetzt → wird übersprungen. **Kein Datenrisiko**.

- **Response-Größe wächst leicht**: 2 zusätzliche Integer-Felder pro
  Bild-Nachricht. Marginal (~30 Byte JSON), vertretbar.

- **`AuthImage` an dritten Stellen** (wenn's die je gibt) bekommt keine
  Dims → Fallback greift. **Kein Regressionsrisiko**.

## Migration Plan

1. **Deploy dieser Change** (nach `chat-broadcast-bilder` archiviert
   wurde):
   - `migrate up` läuft Migration 030 → zwei NULL-Spalten dazu, Server
     startet weiter (ohne Downtime).
   - Backfill-Goroutine startet, iteriert Bestandsbilder, füllt Dims.
2. **Kein Rollback nötig, aber möglich**: `.down.sql` droppt beide
   Spalten. Frontend-Code toleriert fehlende Felder (Fallback greift),
   also auch ohne DB-Rollback kompatibel.
3. **Verifikation**: neuer Upload sieht in DB `width`/`height` gesetzt.
   Nach Backfill-Ende (Log) sind keine `width IS NULL`-Zeilen mehr da.
   Manueller UI-Check: Chat mit Bild-Nachricht öffnen → keine
   sichtbaren Layout-Shifts, Scroll bleibt am Ende.

## Open Questions

- Keine.
