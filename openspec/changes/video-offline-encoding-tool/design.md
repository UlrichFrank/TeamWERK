## Context

Siehe `proposal.md` – Why. Relevanter Bestand:

- `internal/videos/worker.go`: serielle Ein-Goroutine-Job-Queue, `transcodeFunc`-Naht (`realFFmpegTranscode` in Produktion, Fake in Tests), `buildFFmpegRenditionArgs` baut die ffmpeg-Argumentliste rein funktional (testbar ohne Prozessausführung), `probeSegmentCodecs`/`writeMasterManifest` bleiben die bewährte, produktiv laufende CODECS-Signalisierung für AirPlay/tvOS.
- `internal/videos/upload.go`: `CreateUpload` (Metadaten + Disk-Guard) → tusd-Handler unter `/api/videos/upload/` (15 GiB Hard-Limit, `PreUploadCreateCallback` bindet Session an Eigentümer) → `finishUpload` (verschiebt nach `raw/{id}.mp4`, ffprobed Dauer, setzt `status='queued'`).
- `web/src/pages/VideoDetailPage.tsx`: lädt `hls.js` dynamisch, prüft aktuell `Hls.isSupported()` VOR `video.canPlayType(...)` — das ist der Root Cause des AirPlay-Bugs (siehe proposal.md).
- `.github/workflows/release.yml`: taggt + released bei jedem relevanten Merge nach `main` automatisch (Conventional Commits), `gh release create` legt bereits ein GitHub-Release an. Repository ist public.

## Goals / Non-Goals

**Goals:**
- Server führt für neue Videos keinen CPU-intensiven Encode mehr aus, nur noch Stream-Copy-Remux.
- AirPlay spielt zuverlässig Bild **und** Ton (bestehendes, bisher unerfülltes Spec-Szenario wird tatsächlich erfüllt).
- Ein einziges, downloadbares Tool pro Plattform (Windows, macOS) deckt den gesamten Weg „Rohdatei → hochgeladenes, abspielbares Video" ab.
- Minimale Änderung an der Server-API-Fläche (kein neues Upload-Protokoll, keine neue Auth-Route).

**Non-Goals:**
- Kein vollständiger Entfall von `ffmpeg`/`ffprobe` auf dem VPS (bleibt für Remux, Formatprüfung und den bestehenden Codec-Backfill nötig).
- Keine Kapitel-/Marken-Funktion.
- Keine Änderung an der Chromecast-Anbindung.
- Keine automatische Migration/Deinstallation eines bereits ausgelieferten Tools — Versionierung läuft über den normalen Download-Link, der immer auf das neueste Release zeigt.

## Decisions

### 1. Encode-Vertrag zwischen Tool und Server: normalisierte MP4, kein HLS-Archiv

Erwogen wurde, dass das Tool bereits fertige HLS-Segmente hochlädt (Server macht dann gar kein ffmpeg mehr). Verworfen zugunsten einer normalisierten MP4-Datei über den **bestehenden** Upload-Weg, weil:
- der Server-seitige Remux-Schritt (`-c:v copy -c:a copy`) ist auf einem 1-GB-VPS bereits so günstig, dass der Aufwand für ein neues, sicherheitskritisches Archiv-Format (Zip-Slip-Schutz, Dateinamens-Allowlist, Größen-/Anzahl-Deckel) nicht gerechtfertigt ist;
- die bewährte `probeSegmentCodecs`-Logik (produktiv verifiziert für AirPlay) bleibt vollständig unverändert nutzbar, statt eine neue, ungetestete CODECS-Ermittlung im Tool nachzubauen;
- `CreateUpload` + tusd-Handler bleiben 1:1 bestehen — das Tool ist aus Server-Sicht „ein Browser, der schon vorencodiert hat", kein neuer Client-Typ mit eigenem Protokoll.

### 2. Server validiert das Eingangsformat vor dem Remux, statt dem Tool blind zu vertrauen

Mit dem Wegfall des Browser-Uploads gibt es keine serverseitige Instanz mehr, die eine falsch- oder unformatierte Datei ablehnt, bevor sie den Worker erreicht. Ein `-c copy`-Remux einer Datei, die nicht tatsächlich H.264/`yuv420p` ist, würde eine kaputte oder falsch signalisierte HLS-Ausgabe erzeugen (derselbe Symptomkreis wie der ursprüngliche 10-bit-Bug, nur unentdeckt bis zur Wiedergabe). Der Worker prüft deshalb per `ffprobe` (kein Decode, vernachlässigbare Last) Video-Codec und Pixelformat, bevor er den Remux startet, und schlägt mit einem eigenen `failure_reason='unsupported_input_format'` fehl, statt eine kaputte HLS-Ausgabe stillschweigend zu erzeugen.

### 3. GUI-Framework: Fyne — mit einer Korrektur zur ursprünglichen Annahme

Fyne wurde gewählt, weil es in reinem Go geschrieben ist und ohne Wails-artige Webview-Laufzeitabhängigkeit (WebView2 unter Windows) auskommt. **Korrektur gegenüber der ursprünglichen Einschätzung im Gespräch:** Fyne ist NICHT CGo-frei — es bindet über `go-gl`/`glfw` gegen die native Fenstersystem-API (Win32/Cocoa/X11) und braucht dafür einen C-Compiler beim Bauen. Das bedeutet in der Praxis: Cross-Compilation von einer einzigen Build-Maschine aus (z. B. macOS → Windows-`.exe`) ist unnötig fehleranfällig (mingw-w64-Toolchain nötig). Der Release-Workflow baut deshalb **nativ pro Zielplattform** über eine GitHub-Actions-Matrix (`windows-latest`, `macos-latest`) statt per Cross-Compile von einem Runner aus — jeder Runner erzeugt nur die zu seinem eigenen OS passende Binary.

### 4. ffmpeg-Bündelung: eingebettete, unveränderte Binary als Subprozess — keine Bibliotheks-Verlinkung

Die ffmpeg-Binary wird als Rohdaten per `go:embed` in die Tool-Binary eingebettet, beim ersten Start ins Cache-Verzeichnis (`os.UserCacheDir()`) entpackt und dort per `os/exec` als eigenständiger Prozess aufgerufen — es wird **nicht** gegen `libav*`/`libx264` gelinkt. Diese Trennung (separater Prozess statt statisch/dynamisch verlinkte Bibliothek) ist die in der Praxis übliche Form, ein GPL-lizenziertes ffmpeg neben einem anders lizenzierten Programm auszuliefern (`mere aggregation`, kein abgeleitetes Werk) — sollte vor der ersten öffentlichen Veröffentlichung trotzdem von jemandem mit Lizenz-Erfahrung gegengelesen werden, das ist hier keine Rechtsberatung. Quelle für die gebündelte Binary: ein vorgefertigter statischer GPL-Build (z. B. die Windows/macOS-Artefakte von BtbN's `FFmpeg-Builds`), inklusive der zugehörigen Lizenztexte im Tool-Download.

### 5. Tool lebt im selben Repository unter `tools/video-encoder/`

Ein eigenes Go-Modul (`go.mod` mit eigenem Modulpfad, nicht Teil des Server-Moduls) im selben Repo statt eines separaten Repos — Begründung: `release.yml` (Tagging, Changelog, GitHub-Release) ist bereits vorhanden und wird geteilt genutzt, ohne dass eine zweite Release-Pipeline gepflegt werden muss. Ein eigenes `go.mod` verhindert, dass Fyne/GUI-Abhängigkeiten in den Server-Build (`go build ./cmd/teamwerk`) hineingezogen werden.

### 6. Kein Feature-Flag, harter Cutover

Der Browser-Upload-Weg wird nicht schrittweise abgeschaltet, sondern in einem Deploy ersetzt (siehe Migration Plan) — ein Parallelbetrieb wäre zusätzliche Komplexität (zwei Code-Pfade, zwei Failure-Modes) für einen Übergang, der ohnehin nur bis zum nächsten Deploy dauert.

## Risks / Trade-offs

- **In-Flight-Uploads/Queue-Einträge zum Deploy-Zeitpunkt** → ein zum Cutover-Zeitpunkt bereits `queued`/`processing` stehendes, über den ALTEN Browser-Weg hochgeladenes Rohvideo (typischerweise HEVC/10-bit-Handy-Footage, nicht H.264/`yuv420p`) würde vom neuen Worker mit `unsupported_input_format` abgelehnt. Mitigation: Migration Plan verlangt, die Queue vor dem Deploy leerzufahren (siehe unten).
- **Gebündeltes ffmpeg vergrößert den Download spürbar** (typisch 40–80 MB je Plattform zusätzlich zur eigentlichen Tool-Logik) → akzeptiert, da explizit gewünscht („echtes Single-Binary"); keine Mitigation vorgesehen.
- **Native Matrix-Builds statt Cross-Compile** verlängern die Release-Pipeline um zwei zusätzliche Runner-Jobs → vertretbar, da `release.yml` ohnehin asynchron zum Nutzer-Workflow läuft.
- **AirPlay-Fix ist nur per Rauchtest verifizierbar** (kein Apple-TV-Simulator in CI) → wie bereits im bestehenden Szenario „AirPlay auf AppleTV" dokumentiert; dieser Change ändert daran nichts, macht das bestehende Szenario aber erstmals tatsächlich grün.
- **`failure_reason='unsupported_input_format'` als String statt Enum** → folgt dem bestehenden Muster (`failure_reason` ist bereits ein freies Textfeld ohne CHECK-Constraint), keine Migration nötig.

## Migration Plan

1. Vor dem Deploy dieses Changes: sicherstellen, dass keine Videos mit `status IN ('uploading','queued','processing')` in der Produktions-DB stehen (bestehende Uploads abwarten lassen oder — nach Rücksprache — manuell fehlschlagen lassen). Ein `SELECT COUNT(*) FROM videos WHERE status IN ('uploading','queued','processing')` MUST vor dem Deploy `0` liefern.
2. Deploy des Servers (Worker-Umstellung + Wegfall der Upload-Seite) und Veröffentlichung des ersten Tool-Releases erfolgen im selben Zeitfenster — ab dem Server-Deploy ist der Browser-Upload-Button entfernt, das Tool muss zu diesem Zeitpunkt bereits als Release verfügbar sein.
3. Keine Datenmigration nötig — bereits `ready` transcodierte Videos bleiben unverändert nutzbar (ihre HLS-Dateien wurden mit dem alten vollen Encode erzeugt und ändern sich nicht rückwirkend).
4. Rollback: Server-Binary-Rollback (`make deploy-rollback`) stellt den alten Worker (voller Encode) wieder her; das dann bereits verteilte Tool würde weiterhin normalisierte MP4s hochladen, die der alte Worker klaglos erneut voll encodieren würde (kein Format-Konflikt in dieser Richtung) — Rollback ist also gefahrlos möglich.
