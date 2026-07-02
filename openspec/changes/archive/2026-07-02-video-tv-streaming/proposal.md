## Why

Videos lassen sich per AirPlay auf einen AppleTV werfen, aber **es kommt nur Ton, kein Bild**. Ursache ist die HLS-Signalisierung: `master.m3u8` enthält keine `CODECS`-Attribute, sodass tvOS den Video-Elementary-Stream nicht dekodiert. Zusätzlich gibt es drei weitere Baustellen im Streaming-Pfad:

1. **Stream-Token läuft nach 1 h ab** — bei Vollspiel-Videos (60–90 min) brechen Segmente mitten in der Wiedergabe mit HTTP 403 ab. Das trifft Safari-Native und AirPlay/Cast gleichermaßen.
2. **`pix_fmt` nicht explizit gesetzt** — für 10-bit-HEVC-Quellen (moderne iPhones) würde libx264 sonst yuv420p10 ausgeben, das tvOS nicht dekodiert. Zurzeit zufällig okay, aber tickende Bombe.
3. **Nur AirPlay, kein Chromecast** — Android- und Chrome-Nutzer haben keinen TV-Wurf-Pfad. Chromecast ist mit HLS-Basis strukturgleich zu AirPlay und lässt sich mit sehr wenig Code beim gleichen Backend nutzen.

Gleichzeitig wird die 360p-Rendition aus Speichergründen entfernt. Bestandsvideos werden ihre `master.m3u8` einmalig neu generiert (kein Re-Transcode nötig).

## What Changes

- **`master.m3u8` mit `CODECS`-Attribut** — pro Video aus `ffprobe`-Ergebnis abgeleitet (`avc1.<profile><level>` + `mp4a.40.<X>`), damit AirPlay/Cast/Smart-TVs den Video-Decoder korrekt einrichten.
- **`-pix_fmt yuv420p`** explizit an ffmpeg — verhindert 10-bit-Ausgabe bei modernen Quellen.
- **`#EXT-X-INDEPENDENT-SEGMENTS`** in `master.m3u8` — signalisiert Keyframe-Alignment (dank `-force_key_frames` inhaltlich schon erfüllt).
- **360p-Rendition entfernt** — `workerRenditions` enthält nur noch 720p; alte 360p-Verzeichnisse werden bei der Backfill-Regeneration mitgelöscht.
- **Stream-Token-TTL an Video-Dauer gebunden** — `min(4 h, duration_sec + 30 min)`, mit Untergrenze 1 h für Kurz-Clips. Videos ohne bekannte Dauer bekommen weiterhin 1 h.
- **CORS-Header auf HLS-Routen** (`Access-Control-Allow-Origin: *`) — nötig für Chromecast-Default-Receiver; Auth bleibt der `?st=`-Token, keine Cookies.
- **Chromecast-Sender im Frontend** — Cast-SDK dynamisch geladen (parallel zu `hls.js`), Cast-Button neben AirPlay im Player. Übergibt die Master-URL an den Default-Media-Receiver.
- **`x-webkit-airplay="allow"`** am `<video>` — belt-and-suspenders für AirPlay, kein funktionaler Unterschied im Default, aber expliziter.
- **Backfill für Bestandsvideos** — einmaliger Scheduler-Job (idempotent): für jedes `ready`-Video ohne `codecs`-Spalte via `ffprobe` auf `seg_001.ts` die Codec-Strings ermitteln, in DB speichern, `master.m3u8` mit CODECS neu schreiben und (falls vorhanden) `360p/`-Verzeichnis löschen.

## Capabilities

### New Capabilities

- `video-tv-streaming`: HLS-Ausspielung mit `CODECS`-Signalisierung, dauerabhängiger Token-TTL, CORS für Cast-Receiver, Chromecast-Sender im Frontend, `x-webkit-airplay`-Attribut und One-Shot-Backfill für Bestandsvideos.

### Modified Capabilities

_keine — `video-stream` und `video-transcode` aus `spielvideo-ablage` sind noch nicht archiviert; die hier neu definierten Anforderungen leben eigenständig in `video-tv-streaming`._

## Impact

- **Code Backend:** `internal/videos/worker.go` (ffmpeg-Args, Rendition-Liste, Codec-Extraktion, Manifest-Writer), `internal/videos/stream.go` (CORS-Header), `internal/videos/stream_token.go` (dauerabhängige TTL), `internal/videos/handler.go` bzw. `crud.go` (Play übergibt duration), neuer Scheduler-Backfill-Job in `internal/videos/` oder `internal/scheduler/`.
- **DB-Migration:** `internal/db/migrations/00N_video_codecs.up.sql/.down.sql` — Spalte `videos.codecs TEXT NULL`.
- **Frontend:** `web/src/pages/VideoDetailPage.tsx` (Cast-Button + `x-webkit-airplay`), neuer Helper `web/src/lib/cast.ts` (dynamischer Import des Cast-SDK, Init-Guard). `hls.js`-Pfad unverändert.
- **Externe Abhängigkeiten:** Cast-SDK wird über `<script src="https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1">` **erst nach Klick** auf den Cast-Button geladen — kein Google-Ping ohne User-Aktion, DSGVO-neutral im Default-Zustand.
- **SRI (Subresource Integrity) für `cast_sender.js`:** bewusst **nicht** gesetzt. Google versioniert das Script serverseitig ohne stabile Hashes und pusht regelmäßige Kompatibilitäts-Updates für Chromecast-Firmware; ein fixer `integrity=`-Hash würde bei jedem Google-Push die Cast-Integration brechen. Restrisiko: kompromittierter Google-CDN-Endpunkt liefert bösartiges JS. Mitigationen: (a) Script wird nur auf `VideoDetailPage` und nur nach explizitem User-Klick geladen (keine globale Angriffsfläche, keine passive Auslieferung an alle Nutzer), (b) `crossorigin="anonymous"` gesetzt (verhindert Cookie-Leak an gstatic), (c) Verzicht auf die Route reduziert Risiko auf null, wir würden dann aber Chromecast nicht anbieten. Alternative Self-Host-Route (mit SRI) ist bewusst verworfen: Google-Kompat-Updates müssten manuell nachgezogen werden, Ausfallrisiko liegt bei uns statt bei Google. Design bewertet gegen [Web Fundamentals: Cast SDK Loading](https://developers.google.com/cast/docs/web_sender/integrate).
- **Speicher-Ersparnis:** ~22 % weniger HLS-Bytes pro Video (360p vs. 720p Bandbreiten-Verhältnis 0,8 : 2,8 Mbps).
- **Kein ABR-Fallback mehr:** bei sehr schwacher Netzanbindung stallt der Player, statt runterzuschalten. Bewusste Trade-Off-Entscheidung.
- **Doku:** Video-Gotcha in `docs/agent/06-gotchas.md` bekommt einen Absatz zu CODECS, dauerabhängiger TTL und Cast/AirPlay-CORS.

## Test-Anforderungen

Für jede neue/geänderte Route gilt Happy-Path + Fehlerfall (siehe `docs/agent/07-testing.md`). Route → Testname → Status:

- `GET /api/videos/{id}/play` — **Play_KurzVideo_TTL_1h**: 200, `exp - now ≈ 3600 s` bei `duration_sec ≤ 1800`.
- `GET /api/videos/{id}/play` — **Play_LangVideo_TTL_DauerPlus30**: 200, `exp - now ≈ duration_sec + 1800` bei `duration_sec` zwischen 1800 und 12 600.
- `GET /api/videos/{id}/play` — **Play_SehrLangVideo_TTL_Cap4h**: 200, `exp - now ≈ 14 400 s` bei `duration_sec > 12 600`.
- `GET /api/videos/{id}/hls/master.m3u8?st=…` — **ServeMaster_EnthältCodecs**: 200, Body enthält `CODECS="avc1.…,mp4a.40.…"` und `#EXT-X-INDEPENDENT-SEGMENTS`.
- `GET /api/videos/{id}/hls/master.m3u8?st=…` — **ServeMaster_CORS_Header**: 200 und Header `Access-Control-Allow-Origin: *`.
- `OPTIONS /api/videos/{id}/hls/master.m3u8` — **ServeMaster_Preflight**: 204 mit CORS-Headern (falls Chromecast-Receiver vorab OPTIONS schickt; Test verifiziert Verhalten, nicht ob es geschickt wird).
- Worker-Test — **Transcode_SchreibtCodecsInDB**: nach `succeed` steht `videos.codecs = "avc1.…,mp4a.40.…"` in der DB und `master.m3u8` referenziert nur noch `720p/index.m3u8`.
- Worker-Test — **Transcode_ffmpegArgsEnthaltenPixFmt**: injizierte `transcode`-Fake bekommt die vollständige Arg-Liste zu sehen und findet darin `-pix_fmt yuv420p` (Naht analog zu bestehenden Worker-Tests).
- Backfill-Test — **Backfill_IdempotentUndAktualisiertManifest**: erste Ausführung setzt `codecs` und schreibt neues `master.m3u8`; zweite Ausführung ist No-Op (kein DB-Update, keine Datei-Änderung).

Frontend:

- Vitest **VideoDetailPage_ZeigtCastButton_wennApiVerfuegbar**: Mock von `window.chrome.cast` und Rendering des Cast-Launchers.
- Vitest **VideoDetailPage_video_hat_airplay_attribut**: rendert `x-webkit-airplay="allow"`.

Die garantierte Invariante insgesamt: **AirPlay auf AppleTV zeigt Bild** (nicht als Test automatisierbar, aber Kombination CODECS + `pix_fmt yuv420p` + INDEPENDENT-SEGMENTS ist die reproduzierbare Ursache-Kette gemäß Apple HLS Authoring Spec).
