# TeamWERK Video-Encoder

Desktop-Tool (Windows, macOS), mit dem Spielvideos in TeamWERK landen. Seit dem
Change `video-offline-encoding-tool` ist das der **einzige** Upload-Weg — der
Browser-Upload ist entfallen.

1. Anmelden mit dem TeamWERK-Konto (E-Mail/Passwort; Trainer, sportliche
   Leitung oder Vorstand).
2. Datei per Dialog oder Drag & Drop wählen, Mannschaft und Titel/Spiel angeben.
3. Das Tool encodiert **lokal** mit dem gebündelten ffmpeg (H.264 High@4.0,
   `yuv420p`, 720p, CRF 26, Keyframes alle 4 s, AAC) und lädt das Ergebnis über
   den bestehenden tus-Upload (`POST /api/videos` + `/api/videos/upload/`) hoch.
4. Der Server prüft Codec/Pixelformat per ffprobe und packt nur noch um
   (`-c copy` → HLS). Eine Datei, die nicht aus dem Tool stammt, lehnt er mit
   `unsupported_input_format` ab.

Der Encode-Vertrag steht an zwei Stellen und muss zusammenpassen:
`internal/encode/args.go` (hier) und `probeInputFormat` in
`internal/videos/worker.go` (Server).

## Aufbau

Eigenes Go-Modul, **nicht** Teil des Server-Moduls (Fyne soll nicht in den
Server-Build geraten) und nicht Teil von `make test`.

| Paket | Aufgabe |
|---|---|
| `main.go`, `ui.go` | Fyne-Oberfläche (einziger Teil mit CGo) |
| `internal/encode` | ffmpeg-Argumente, Quellanalyse (`ffmpeg -i`), Lauf mit Fortschritt |
| `internal/client` | Login/Refresh wie ein Browser (Cookie-Jar), API-Aufrufe, tus-Upload mit 401-Retry und Fortsetzen |
| `internal/ffmpegbin` | eingebettetes ffmpeg, Entpacken ins Cache-Verzeichnis beim ersten Start |
| `internal/progress` | Drosselung der Fortschrittsmeldungen, Restzeit |

Die `internal/`-Pakete sind reines Go und laufen in der CI
(`.github/workflows/ci.yml`, Job `video-encoder`) ohne Fenstersystem.

## Entwickeln

```bash
cd tools/video-encoder
go test ./internal/...          # mit ffmpeg/ffprobe im PATH laufen auch die Integrationstests
go run .                        # braucht C-Compiler (Xcode CLT / MinGW) für Fyne; nutzt ffmpeg aus dem PATH
```

Ohne eingebettete Binary nimmt das Tool das `ffmpeg` aus dem `PATH`. Einen
release-artigen Build mit eingebettetem ffmpeg bekommt man, indem man die
Dateien für die eigene Plattform nach `internal/ffmpegbin/bin/` legt (Namen und
Quelle: `internal/ffmpegbin/bin/README.md`, Prüfsummen im Release-Workflow).

Gegen einen lokalen `http://`-Server funktionieren Login und Upload, der
Token-Refresh nach 15 min aber nicht: das Refresh-Cookie ist `Secure`.

## Release

`.github/workflows/release.yml`, Job `video-encoder`, baut bei jedem Release
nativ pro Plattform (Fyne braucht CGo, deshalb kein Cross-Compile) und hängt an:

- `teamwerk-video-encoder-windows-amd64.exe`
- `teamwerk-video-encoder-macos.dmg` (Universal: Apple Silicon + Intel)
- `teamwerk-video-encoder-ffmpeg-licenses.zip` (Lizenz-/Herkunftstexte der gebündelten ffmpeg-Builds)

Die Download-Links auf `/videos` zeigen auf `releases/latest/download/…`
(`web/src/lib/videoEncoderDownloads.ts`) — Asset-Namen dort und im Workflow
gleich halten.

**Nicht signiert.** macOS zeigt beim ersten Start „nicht verifizierter
Entwickler“ (Rechtsklick → Öffnen), Windows SmartScreen „Unbekannter
Herausgeber“ (Weitere Informationen → Trotzdem ausführen).

## Lizenz des gebündelten ffmpeg

Gebündelt wird ein unveränderter statischer GPL-Build von ffmpeg
(<https://github.com/eugeneware/ffmpeg-static>), aufgerufen als eigenständiger
Prozess, nicht gelinkt. Lizenz- und Herkunftstext stehen im Tool unter
„Hilfe → Über“ und als Release-Asset. Vor der ersten öffentlichen Verteilung
von jemandem mit Lizenz-Erfahrung gegenlesen lassen (design.md, Entscheidung 4).
