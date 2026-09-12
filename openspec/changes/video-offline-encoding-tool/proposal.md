## Why

Server-seitiges Video-Transcoding (`internal/videos/worker.go`, voller H.264-Encode via `libx264 preset=medium crf=26`) läuft auf einem VPS mit 1 GB RAM — strukturell die falsche Maschine für CPU-intensives Video-Encoding einer ganzen Spielaufnahme. Gleichzeitig ist die produktiv dokumentierte Zusage aus `openspec/specs/video-tv-streaming/spec.md` („AirPlay auf AppleTV … startet die Video-Wiedergabe, nicht nur Audio") aktuell **nicht erfüllt**: AirPlay spielt nur Ton, kein Bild. Root Cause ist nicht die CODECS-Signalisierung (die funktioniert wie spezifiziert), sondern `web/src/pages/VideoDetailPage.tsx`, das `Hls.isSupported()` VOR `video.canPlayType('application/vnd.apple.mpegurl')` prüft — auf Safari/WebKit (macOS + iOS) liefert `Hls.isSupported()` `true` (MediaSource Extensions sind vorhanden), sodass dort hls.js/MSE statt nativer HLS-Wiedergabe läuft. AirPlay kann eine MSE-gepufferte Session nicht als Ganzes an ein Apple TV übergeben — nur der Audio-Pfad routet zuverlässig.

Beide Probleme werden in einem Change gelöst: das teure Encoding wandert auf die Rechner der Filmenden (Windows/Mac), der Server macht nur noch ein günstiges Remux (Container-Wechsel, kein Re-Encode), und die AirPlay-Wiedergabe wird auf der Frontend-Seite tatsächlich korrekt (nativer HLS-Pfad zuerst).

## What Changes

- **Neues Offline-Encoding-Tool**: Single-Binary Desktop-App (Go, GUI-Framework Fyne — pure Go, kein CGo) für Windows und macOS. Einfache Oberfläche: Dateiauswahl-Dialog oder Drag & Drop, ein Fortschrittsbalken. Bündelt ffmpeg per `go:embed` (GPL-Build, beim ersten Start ins lokale Cache-Verzeichnis entpackt). Encoded lokal mit denselben Zielparametern wie heute der Server (H.264, `yuv420p`, 720p, CRF 26, preset medium, 4s-Keyframe-Erzwingung, AAC-Audio), zusätzlich mit fest gepinntem `-profile:v`/`-level` statt libx264-Default, damit das Output-Profil je Tool-Version stabil/vorhersagbar bleibt. Meldet sich über `net/http.Client` + `CookieJar` gegen die bestehenden Endpunkte `/api/auth/login`/Refresh an (verhält sich wie ein Browser) und lädt die normalisierte MP4 über den **bestehenden, unveränderten** tus-Upload-Weg (`POST /api/videos` + `/api/videos/upload/`) hoch.
- **BREAKING: Browser-Video-Upload entfällt.** `web/src/pages/VideoUploadPage.tsx` und ihre Route werden entfernt. `/videos` bekommt stattdessen einen Split-Button/Dropdown („Video hochladen ▾") analog zum bestehenden „+ Neu"-Muster auf `/nutzer` mit den Einträgen „Tool für Windows herunterladen" / „Tool für macOS herunterladen". Ab diesem Change kann ein Video ausschließlich über das Tool hochgeladen werden.
- **Server-seitiger Transcode wird zu einem Remux**: `internal/videos/worker.go` bleibt strukturell erhalten (serielle Job-Queue, Disk-Guard, Manifest-Writer, die bewährte `probeSegmentCodecs`-CODECS-Ermittlung) — nur `buildFFmpegRenditionArgs` wechselt von echtem Encode (`-c:v libx264 -preset medium -crf 26 …`) zu Stream-Copy (`-c:v copy -c:a copy -f hls …`). Der Worker MUST vor dem Remux per `ffprobe` (kein Decode, günstig) prüfen, dass die eingehende Datei tatsächlich H.264/`yuv420p` ist, und sauber mit einem neuen `failure_reason` fehlschlagen, falls nicht — Defense in Depth, weil die Formatgarantie nach Wegfall des Browser-Uploads nur noch implizit (durch das Tool) besteht.
- **AirPlay-Fix**: `VideoDetailPage.tsx` prüft künftig zuerst `video.canPlayType('application/vnd.apple.mpegurl')` und nutzt bei Erfolg native HLS-Wiedergabe (Safari/WebKit); hls.js/MSE ist nur noch der Fallback für Browser ohne native Unterstützung.
- **Distribution über bestehende Release-Infrastruktur**: `.github/workflows/release.yml` (erzeugt bereits bei jedem relevanten Merge nach `main` automatisch ein GitHub-Release) bekommt einen zusätzlichen Cross-Compile-Job, der die Tool-Binaries (Windows amd64, macOS) mit gebündeltem ffmpeg baut und als Release-Assets hochlädt. Da das Repository public ist, verlinkt das Frontend direkt auf die GitHub-Release-Asset-URLs — kein Server-Hosting, kein Auth-Proxy.
- Keine Kapitel-/Marken-Funktion (bewusst außerhalb des Scopes). Keine Änderung an der Chromecast-Anbindung (funktioniert unverändert über den Default Media Receiver + HLS-Master-URL).

## Capabilities

### New Capabilities
- `video-offline-encoding-tool`: das Desktop-Tool selbst — Encoding-Parameter, ffmpeg-Bündelung, Auth-Verhalten, Wiederverwendung des bestehenden Upload-Wegs, Distribution via GitHub Releases.

### Modified Capabilities
- `video-transcode`: „HLS-Transcode-Format" ändert sich von echtem Encode zu Remux (`-c:v copy -c:a copy`); neue Anforderung für die ffprobe-Vorprüfung des Eingangsformats vor dem Remux.
- `video-tv-streaming`: neue Anforderung, die die tatsächliche Ursache des AirPlay-Bugs behebt (native-HLS-Präferenz vor hls.js/MSE in `VideoDetailPage`).
- `video-upload`: die Anforderung „Client-seitige Progress-Throttle" (spezifisch für die entfallende `VideoUploadPage`) wird entfernt — die äquivalente Verantwortung liegt jetzt im Tool (siehe `video-offline-encoding-tool`).
- `video-upload-token-refresh`: die browser-spezifischen tus-Auth-Hook-Anforderungen (`buildAuthHooks`, `onBeforeRequest`/`onShouldRetry` in `VideoUploadPage.tsx`) entfallen mit der Seite; die äquivalente Verantwortung (Token-Refresh während eines langen Uploads) liegt jetzt im Tool (siehe `video-offline-encoding-tool`).

## Impact

- **Neues Verzeichnis**: `tools/video-encoder/` (Go-Modul innerhalb desselben Repos — Monorepo-Unterordner, weil `release.yml` bereits geteilt genutzt werden kann und keine separate CI/Release-Pipeline nötig ist).
- **Backend**: `internal/videos/worker.go` (Encode → Remux, neue ffprobe-Vorprüfung, neuer `failure_reason`-Wert), `internal/videos/codecs.go` (unverändert, weiterhin genutzt), `internal/videos/upload.go` (nur Kommentare/Fehlermeldungen, Kernlogik unverändert), `internal/db/migrations/` (falls `failure_reason` einen neuen festen Wert statt Freitext bekommt — sonst keine Migration nötig).
- **Frontend**: `web/src/pages/VideoUploadPage.tsx` (entfernt), `web/src/pages/VideosPage.tsx` (Dropdown-Button), `web/src/pages/VideoDetailPage.tsx` (AirPlay-Reihenfolge-Fix).
- **CI/CD**: `.github/workflows/release.yml` (Cross-Compile-Job + Asset-Upload für das Tool).
- **Betrieb**: `ffmpeg`/`ffprobe` bleiben vorerst auf dem VPS installiert (weiterhin für die Remux-/Vorprüfungs-Schritte und den bestehenden `internal/videos/backfill.go` für Alt-Videos benötigt) — vollständige Entfernung der ffmpeg-Abhängigkeit vom VPS ist explizit **kein** Ziel dieses Change (spätere Aufräumarbeit, siehe design.md Non-Goals).
- **Tests**: neue Tests für den Remux-Pfad und die Format-Vorprüfung in `internal/videos/*_test.go`; Anpassung/Entfernung der `VideoUploadPage.test.tsx`-Fälle, die die entfallende Seite betrafen; neue Tests für die AirPlay-Reihenfolge in `VideoDetailPage.test.tsx` (sofern vorhanden) bzw. manueller Rauchtest (native Browser-APIs sind in jsdom nicht sinnvoll testbar — siehe `docs/agent/07-testing.md`, Playwright-Kriterium).

## Test-Anforderungen

Dieser Change fügt keine neue HTTP-Route hinzu, ändert aber bestehende Geschäftslogik (`internal/videos/worker.go`) — laut `docs/agent/07-testing.md` deshalb hier festgehalten:

| Betroffene Logik | Testname | Erwartetes Ergebnis | Garantierte Invariante |
|---|---|---|---|
| `buildFFmpegRenditionArgs` (Worker) | `TestBuildFFmpegRenditionArgs_Remux` | Arg-Liste enthält `-c:v copy` und `-c:a copy`, NICHT `-c:v libx264`/`-crf` | Server führt für neue Videos keinen Video-Encode mehr aus |
| neue ffprobe-Vorprüfung im Worker | `TestProcess_UnsupportedInputFormat_MarksFailed` | Datei mit Nicht-H.264-Codec (Fake-ffprobe-Naht) → `status='failed'`, `failure_reason='unsupported_input_format'`, KEIN `-c copy`-Aufruf | Es wird nie eine Datei umgepackt, die nicht als H.264/`yuv420p` verifiziert wurde |
| neue ffprobe-Vorprüfung im Worker | `TestProcess_SupportedInputFormat_Remuxes` | Datei mit H.264/`yuv420p` (Fake-ffprobe-Naht) → Remux läuft normal, `status='ready'` | Happy Path bleibt unverändert grün |
| `probeSegmentCodecs` nach Remux | `TestRealFFmpegTranscode_Remux_PreservesCodecs` (ggf. Integrationstest mit echtem ffmpeg, falls in CI verfügbar, sonst dokumentierter Skip) | `videos.codecs` weiterhin korrekt gesetzt nach einem Remux-Lauf | AirPlay-CODECS-Signalisierung bleibt nach der Umstellung auf Remux erhalten |
| `VideoDetailPage` Player-Auswahl | `VideoDetailPage.test.tsx`: `wählt native Wiedergabe wenn canPlayType nicht leer ist` | `hls.js`-Import/Konstruktor wird NICHT aufgerufen, `video.src` wird direkt gesetzt | Kein MSE-Pfad auf Safari/WebKit |
| `VideoDetailPage` Player-Auswahl | `VideoDetailPage.test.tsx`: `fällt auf hls.js zurück wenn canPlayType leer ist` | `Hls`-Konstruktor wird aufgerufen wie bisher | Bestehendes Verhalten für Chrome/Firefox bleibt erhalten |
