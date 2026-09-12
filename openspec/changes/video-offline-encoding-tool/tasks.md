## 1. Server: Worker von Encode auf Remux umstellen

- [x] 1.1 `internal/videos/worker.go`: `buildFFmpegRenditionArgs` von `-c:v libx264 -preset medium -crf 26 ...` auf `-c:v copy -c:a copy -f hls ...` umstellen (Stream-Copy, kein Re-Encode); `-hls_time` bleibt 4
- [x] 1.2 `internal/videos/worker.go`: neue Funktion `probeInputFormat(ctx, rawPath) (codec, pixFmt string, err error)` (ffprobe auf ersten Video-Stream), aufgerufen VOR `runFFmpegRendition` in `realFFmpegTranscode`
- [x] 1.3 `internal/videos/worker.go`: bei `codec != "h264"` oder `pixFmt != "yuv420p"` → `wk.fail(id, "unsupported_input_format")` statt ffmpeg-Aufruf; Rohdatei bleibt erhalten (bestehendes Verhalten von `fail`)
- [x] 1.4 Kommentare in `buildFFmpegRenditionArgs`/`realFFmpegTranscode` aktualisieren (die bisherigen Kommentare beschreiben einen echten Encode, jetzt falsch)

## 2. Server: Tests für den Remux-Pfad

- [x] 2.1 `internal/videos/worker_test.go`: `TestBuildFFmpegRenditionArgs_Remux` — Arg-Liste enthält `-c:v copy`/`-c:a copy`, NICHT `-c:v libx264`/`-crf`
- [x] 2.2 `internal/videos/worker_test.go`: `TestProcess_UnsupportedInputFormat_MarksFailed` (Fake-`probeInputFormat`-Naht liefert nicht-H.264) → `status='failed'`, `failure_reason='unsupported_input_format'`, kein ffmpeg-Aufruf
- [x] 2.3 `internal/videos/worker_test.go`: `TestProcess_SupportedInputFormat_Remuxes` (Fake liefert H.264/yuv420p) → Happy Path bleibt grün, `status='ready'`
- [x] 2.4 Bestehende `worker_test.go`-Fälle, die den alten Encode-Argumentsatz assertieren, an die neue Remux-Arg-Liste anpassen

## 3. Tool: Go-Modul-Grundgerüst

- [ ] 3.1 Neues Verzeichnis `tools/video-encoder/` mit eigenem `go.mod` (eigener Modulpfad, unabhängig vom Server-Modul)
- [ ] 3.2 Fyne-Abhängigkeit einbinden, minimales Fenster mit Dateiauswahl-Button + Drag&Drop-Ziel + Fortschrittsbalken (noch ohne Funktionalität)
- [ ] 3.3 ffmpeg-Binaries (Windows + macOS, statischer GPL-Build) beschaffen, per `go:embed` einbetten, Lizenztext (GPL + Build-Herkunft) im Repo/Tool-About-Dialog hinterlegen
- [ ] 3.4 Erst-Start-Extraktion: eingebettete ffmpeg-Binary nach `os.UserCacheDir()` entpacken, falls dort noch keine/eine andere Version liegt; Exec-Pfad merken

## 4. Tool: Encoding

- [ ] 4.1 ffmpeg-Aufruf mit den Zielparametern bauen: scale auf 720p, `-pix_fmt yuv420p`, `-c:v libx264 -preset medium -crf 26 -maxrate 2800k -bufsize 5600k`, `-force_key_frames expr:gte(t,n_forced*4)`, festes `-profile:v`/`-level` (z. B. High@3.1), AAC-Audio (128k oder `copy` bei AAC-Quelle, analog `sourceIsAAC` in `internal/videos/worker.go`)
- [ ] 4.2 Fortschrittsanzeige aus ffmpeg-`stderr`-Progress-Parsing (z. B. `-progress pipe:1`) an die Fyne-UI koppeln, ohne UI-Blockade bei hoher Event-Frequenz
- [ ] 4.3 Fehlerbehandlung: ffmpeg-Exit-Code ≠ 0 → verständliche Fehlermeldung in der UI, kein Absturz

## 5. Tool: Auth + Upload

- [ ] 5.1 Login-Dialog (E-Mail/Passwort) → `POST /api/auth/login` über `http.Client` mit `CookieJar` (Refresh-Token-Cookie wird automatisch verwaltet)
- [ ] 5.2 Metadaten-Formular (Titel oder Spiel-Auswahl, Team, Saison) → `POST /api/videos`
- [ ] 5.3 tus-Client in Go implementieren oder vorhandene Go-tus-Client-Bibliothek einbinden; Upload gegen `/api/videos/upload/` mit denselben Metadata-Feldern wie der bisherige Browser-Client (`video_id`)
- [ ] 5.4 401-Retry-Hook: bei 401 auf einem Chunk → `POST /api/auth/refresh`, Chunk wiederholen; bei 401 auf den Refresh selbst → Upload sauber abbrechen, Re-Login anfordern
- [ ] 5.5 Resumable-Verhalten: laufende tus-Session bei Verbindungsabbruch fortsetzen (tus-Protokoll-Standardverhalten)

## 6. Tool: Tests

- [ ] 6.1 Go-Tests für die 401-Retry-Logik (Fake-HTTP-Server: erster Chunk 401, Refresh liefert neuen Token, Retry-Chunk 204)
- [ ] 6.2 Go-Tests für den Refresh-Failure-Pfad (Refresh liefert 401 → Upload bricht ab, keine Endlosschleife)
- [ ] 6.3 Test für die ffmpeg-Argument-Konstruktion (reine Funktion, analog `buildFFmpegRenditionArgs` im Server) — insbesondere `-pix_fmt yuv420p` vor `-c:v libx264` und das feste Profil/Level

## 7. Release-Pipeline

- [ ] 7.1 `.github/workflows/release.yml`: neuen Job-Block mit Matrix `[windows-latest, macos-latest]` ergänzen, der `tools/video-encoder/` nativ pro Runner baut (kein Cross-Compile, siehe design.md Entscheidung 3)
- [ ] 7.2 Gebaute Binaries als Release-Assets hochladen (`gh release upload "$VERSION" ...`), Namensschema z. B. `teamwerk-video-encoder-windows-amd64.exe` / `teamwerk-video-encoder-macos.dmg`
- [ ] 7.3 Lizenztexte (ffmpeg GPL + Build-Attribution) als zusätzliche Release-Assets oder im Tool selbst (About-Dialog) mit ausliefern

## 8. Frontend: Browser-Upload entfernen, Download-Dropdown ergänzen

- [x] 8.1 `web/src/pages/VideoUploadPage.tsx` und zugehörige Route in `App.tsx` entfernen
- [x] 8.2 `web/src/pages/VideosPage.tsx`: Split-Button/Dropdown „Video hochladen ▾" analog zum „+ Neu"-Muster in `web/src/pages/AdminUsersPage.tsx` (Zeilen ~426-452) mit Einträgen „Tool für Windows herunterladen" / „Tool für macOS herunterladen", verlinkt auf die GitHub-Release-Asset-URLs des jeweils neuesten Releases
- [x] 8.3 Verwaiste Frontend-Tests (`VideoUploadPage.test.tsx`) entfernen; neue Tests für den Dropdown-Button in `VideosPage.test.tsx` (sofern vorhanden) ergänzen

## 9. Frontend: AirPlay-Fix

- [x] 9.1 `web/src/pages/VideoDetailPage.tsx`: Prüfreihenfolge umdrehen — zuerst `video.canPlayType('application/vnd.apple.mpegurl')`, bei nicht-leerem Ergebnis `video.src` direkt setzen und `hls.js` NICHT laden/initialisieren; nur im Else-Zweig wie bisher `hls.js` dynamisch importieren
- [x] 9.2 Tests in `VideoDetailPage.test.tsx` (sofern vorhanden) ergänzen: native Wiedergabe bei nicht-leerem `canPlayType`, hls.js-Fallback bei leerem `canPlayType`
- [ ] 9.3 Manueller Rauchtest auf echtem Safari/iOS gegen ein Apple TV (native Browser-/AirPlay-APIs sind in jsdom nicht sinnvoll testbar, siehe `docs/agent/07-testing.md`)

## 10. Migration/Rollout

- [ ] 10.1 Vor dem Deploy: `SELECT COUNT(*) FROM videos WHERE status IN ('uploading','queued','processing')` auf Prod prüfen — muss `0` sein, bevor der neue Worker aktiv wird (siehe design.md Migration Plan)
- [ ] 10.2 Erstes Tool-Release veröffentlichen, BEVOR der Server mit entferntem Browser-Upload deployt wird
- [ ] 10.3 Nach dem Deploy: manueller Rauchtest mit einer echten Spielaufnahme (Tool encodieren + hochladen lassen → Video erscheint als `ready` → AirPlay-Wiedergabe auf einem echten Apple TV prüfen)

## 11. Abschluss

- [ ] 11.1 `make test` / `make lint` / `pnpm -C web build` grün (inkl. Architektur- und Broadcast-Gate)
- [ ] 11.2 `cd tools/video-encoder && go build ./...` sowie `go vet ./...` grün (eigenes Modul, nicht Teil von `make test`)
- [ ] 11.3 `openspec validate --strict` für diesen Change
