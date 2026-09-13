# Eingebettetes ffmpeg

Dieses Verzeichnis wird per `go:embed` in das Tool eingebettet. Im Repository
liegt hier nur diese Datei; der Release-Build (`.github/workflows/release.yml`,
Job `video-encoder`) legt vor dem `go build` die zur Zielplattform passenden
Dateien daneben:

| Datei            | Inhalt                                                        |
|------------------|---------------------------------------------------------------|
| `ffmpeg.gz`      | statischer GPL-Build von ffmpeg (gzip, beim Erststart entpackt) |
| `ffmpeg.LICENSE` | Lizenztext des Builds                                          |
| `ffmpeg.README`  | Build-Herkunft/-Konfiguration                                  |
| `ffmpeg.SOURCE`  | Download-URL und SHA-256 des verwendeten Assets                |

Quelle: <https://github.com/eugeneware/ffmpeg-static/releases> (Tag und
Prüfsummen sind im Workflow gepinnt). Die Dateien sind per `.gitignore`
ausgeschlossen — nie einchecken.

Lokal ohne diese Dateien gebaut, verwendet das Tool das `ffmpeg` aus dem `PATH`.
Zum lokalen Test eines Release-artigen Builds die Dateien für die eigene
Plattform von dort herunterladen und hier ablegen (siehe `../../../README.md`).
