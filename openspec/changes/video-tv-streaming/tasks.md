## 1. DB-Schema

- [x] 1.1 Migration `internal/db/migrations/017_video_codecs.up.sql`: `ALTER TABLE videos ADD COLUMN codecs TEXT;`
- [x] 1.2 Passende `.down.sql`: SQLite unterstützt `DROP COLUMN` seit 3.35 (in `modernc.org/sqlite` verfügbar) — `ALTER TABLE videos DROP COLUMN codecs;`
- [x] 1.3 `migrate up` lokal grün, `sqlite3 …` verifiziert Spalte

## 2. Transcode-Pipeline (`internal/videos/worker.go`)

- [x] 2.1 `workerRenditions` reduziert auf einen Eintrag (720p) — 360p-Struct-Literal entfernen
- [x] 2.2 `runFFmpegRendition`: Arg-Block `"-pix_fmt", "yuv420p"` **vor** `-c:v libx264` einfügen; Kommentar warum (verhindert yuv420p10 bei 10-bit-Quellen)
- [x] 2.3 Neue Funktion `probeSegmentCodecs(ctx, segmentPath string) (codecs string, err error)` in `codecs.go` — ffprobe getrennt für Video- und Audio-Stream, baut `avc1.PPCCLL,mp4a.40.X`
- [x] 2.4 Table-driven Unit-Test für `h264CodecString` und `aacCodecString` in `codecs_test.go` (Baseline/Main/High/High10 × L3.0/L3.1/L4.0/L4.1; AAC-LC/HE-AAC/HE-AACv2; case-insensitive; Fehlerpfad)
- [x] 2.5 `realFFmpegTranscode`: nach dem 720p-Lauf `probeSegmentCodecs(processedDir/720p/seg_001.ts)` → Codec-String an Worker zurückgegeben, `succeed` schreibt in DB (`UPDATE videos SET codecs=? …`)
- [x] 2.6 `writeMasterManifest` signaturseitig auf `(processedDir, codecs string)` erweitert, schreibt zusätzlich `#EXT-X-INDEPENDENT-SEGMENTS` und hängt `CODECS="<video>,<audio>"` an das STREAM-INF an
- [x] 2.7 Worker-Test **TestBuildFFmpegRenditionArgs_ContainsPixFmt**: prüft dass `buildFFmpegRenditionArgs` `-pix_fmt yuv420p` vor `-c:v` liefert (beide `aacSource`-Fälle)
- [x] 2.8 Worker-Test in `TestWorkerSerialProcessing` erweitert: nach erfolgreichem Fake-Transcode ist `videos.codecs` gesetzt, `master.m3u8` enthält CODECS + INDEPENDENT-SEGMENTS und referenziert nur `720p/index.m3u8` (kein 360p)

## 3. Stream-Token-TTL (`internal/videos/stream_token.go`)

- [x] 3.1 `Sign` signaturseitig auf `(vid, uid, durationSec int)` erweitert; TTL-Formel: `clamp(duration + 30min, 1h, 4h)`, `duration ≤ 0` → 1h (Legacy)
- [x] 3.2 Reine Funktion `computeStreamTokenTTL(durationSec int) time.Duration` extrahiert + Table-driven Unit-Test (Null/negativ/Floor-Grenze/Mittel/knapp unter/knapp über Cap/sehr lang)
- [x] 3.3 `streamTokenTTL` durch `legacyStreamTokenTTL`, `maxStreamTokenTTL`, `streamTokenSlack` ersetzt

## 4. Play-Route (`internal/videos/stream.go` bzw. wo `Play` sitzt)

- [x] 4.1 `Play` lädt `duration_sec` via erweitertem `loadVideoForView` (mit-selektiert, `Video.DurationSec` gefüllt)
- [x] 4.2 `Play` ruft `h.Sign(vid, uid, int(video.DurationSec))` mit tatsächlicher Dauer
- [x] 4.3 Test **TestPlay_TTL_KurzVideo_1h**: duration 1200 → `exp - now == 3600`
- [x] 4.4 Test **TestPlay_TTL_LangVideo_DauerPlus30**: duration 5400 → `exp - now == 7200`
- [x] 4.5 Test **TestPlay_TTL_SehrLangVideo_Cap4h**: duration 100000 → `exp - now == 14400`
- [x] 4.6 Test **TestPlay_TTL_LegacyOhneDuration_1h**: duration_sec NULL → 3600 s

## 5. CORS auf HLS-Routen (`internal/videos/stream.go`)

- [x] 5.1 `ServeMaster` und `ServeRenditionFile` setzen via `setHLSCORSHeaders` `Access-Control-Allow-Origin: *` und `Access-Control-Allow-Methods: GET`
- [x] 5.2 `HLSPreflight`-Handler + OPTIONS-Routen auf beiden HLS-Pfaden; `StreamTokenMiddleware` reicht OPTIONS ohne Auth durch
- [x] 5.3 Test **TestHLS_MasterCORSHeader**: GET liefert 200 und `Access-Control-Allow-Origin: *`
- [x] 5.4 Test **TestHLS_MasterCORSPreflight**: OPTIONS liefert 204 mit CORS-Headern
- [x] 5.5 Regressionscheck: `?st=`-Auth funktioniert weiterhin (bestehende `TestHLS_MissingToken_403`, `TestHLS_WrongVidToken_403`, `TestPlay_ForbiddenForOutsider` grün); zusätzlich **TestHLS_SegmentCORSHeader** für Segmente

## 6. Manifest mit CODECS (Neu-Videos)

- [x] 6.1 In `TestWorkerSerialProcessing` (Block 2) und `TestMaster_ValidToken_RewritesRenditionURLs` (Fixture-Master mit CODECS + INDEPENDENT-SEGMENTS) abgedeckt; kein 360p-Verweis mehr

## 7. Backfill für Bestandsvideos (`internal/videos/backfill.go`)

- [x] 7.1 Neue Datei `internal/videos/backfill.go` mit `RunTVCompatBackfill(ctx context.Context, db *sql.DB, storageDir string) error`
- [x] 7.2 Iteriert `SELECT id FROM videos WHERE status='ready' AND codecs IS NULL`
- [x] 7.3 Pro Video: `probeSegmentCodecs` auf `720p/seg_001.ts` (Fallback `360p/seg_001.ts` wenn 720p fehlt), `UPDATE videos SET codecs=?`, `writeMasterManifest` neu schreiben, `360p/`-Dir löschen wenn vorhanden
- [x] 7.4 Fehler pro Video geloggt (`slog.Error`), keine Rückgabe-Fehler bei Einzelfehlern; nur harte Errors (DB-Verbindung, Storage-Root unerreichbar) bubbeln hoch
- [x] 7.5 Aufruf in `cmd/teamwerk/main.go` im `serve()`-Pfad als Hintergrund-Goroutine (nicht `scheduler:run` — das ist ein Minute-Cron, nicht ein Prozess-Start); startet nach Worker, blockiert HTTP-Serving nicht
- [x] 7.6 Test **TestBackfill_Idempotent**: erste Ausführung schreibt `codecs`, entfernt 360p-Zeile aus master, löscht 360p-Dir; zweite Ausführung ruft `probeSegmentCodecsFn` nicht mehr auf (kein Videos in der Query)
- [x] 7.7 Test **TestBackfill_UeberspringtFehlerhaftesVideoOhneAbbruch**: bei zwei Videos, eines mit fehlender seg_001.ts, wird das andere trotzdem migriert
- [x] 7.8 Zusatztest **TestBackfill_LaesstNichtReadyUnberuehrt**: `queued`-Videos werden nicht angefasst — Query-Filter greift

## 8. Frontend: `x-webkit-airplay="allow"`

- [x] 8.1 `web/src/pages/VideoDetailPage.tsx`: am `<video>`-Element `x-webkit-airplay="allow"` via `{...{ 'x-webkit-airplay': 'allow' }}` gesetzt (React-idiomatischer Weg für benutzerdefinierte Bindestrich-Attribute)
- [~] 8.2 Isolierter Unit-Test für das Attribut absichtlich weggelassen — Test würde die ganze Detail-Seite mit Router+api-Mocks aufziehen; Attribut ist statisch gerendert und wird im manuellen Rauchtest (Block 10.1) verifiziert

## 9. Frontend: Cast-Sender

- [x] 9.1 `web/src/lib/cast.ts`: `loadCastSDK()` gemäß Design (Script-Tag mit `crossorigin="anonymous"`, ohne `integrity=`, mit begründendem Kommentar), Single-Flight über modul-lokales Promise, `onerror`-Fallback
- [x] 9.2 `web/src/lib/cast.ts`: `startCastSession(masterURL: string)` kapselt `CastContext.setOptions` + `requestSession` + `loadMedia` (Content-Type `application/vnd.apple.mpegurl`, StreamType BUFFERED)
- [x] 9.3 `web/src/components/CastButton.tsx`: Button mit `lucide-react`-`<Cast>`, feature-detection über `window.chrome?.cast` nach `loadCastSDK()`; brand-Tokens (Button Small: `bg-brand-yellow …`); optimistisches Rendering bis erster Ladeversuch scheitert
- [x] 9.4 `VideoDetailPage`: `<CastButton masterURL={masterURL} />` unter dem Player, `masterURL` als absoluter URL an den Receiver
- [x] 9.5 Fehlerbehandlung: SDK-Load-Fehler → Info-Alert im brand-Stil; User-Abbruch des Session-Dialogs bleibt still
- [x] 9.6 Vitest **CastButton renders when Cast API is already injected**: Mock von `window.chrome = { cast: … }` und `window.cast.framework`, Button rendert
- [x] 9.7 Vitest **CastButton renders optimistically without Cast API**: ohne Mock rendert Button trotzdem (versteckt sich erst nach fehlgeschlagenem Klick — bewusster UX-Trade-off)

## 10. Verifikation auf Testgeräten (nach Deploy — User-Rauchtest)

- [ ] 10.1 iPhone (Safari) + AppleTV im selben WLAN: Video öffnen → AirPlay aus Fullscreen-Controls → Bild kommt (jetzt mit CODECS)
- [ ] 10.2 Langes Video (> 60 min) auf iPhone Safari nativ komplett durchspielen (Token-TTL-Fix verifizieren)
- [ ] 10.3 Android/Chrome (oder Desktop-Chrome) + Chromecast/Google TV im selben WLAN: Cast-Button → Video läuft am TV
- [ ] 10.4 Regression: hls.js im Firefox weiterhin abspielbar (kein CORS-Bruch)

## 11. Doku, CHANGELOG, Commit

- [x] 11.1 `docs/agent/06-gotchas.md`: Absatz „Video-Streaming (HLS + AirPlay + Chromecast)" ergänzt — CODECS-Attribut nötig, `-pix_fmt yuv420p`, TTL an Dauer, CORS, Cast-SDK ohne SRI (mit Begründung)
- [x] 11.2 `web/public/CHANGELOG.md`: drei `[feat]`-Zeilen (AirPlay-Bild, Chromecast-Wurftaste, Token bis Video-Ende + 30 min) unter 02.07.2026
- [x] 11.3 Zusatz: OPTIONS-Permission-Matrix-Einträge in `internal/permissions/matrix_test.go` (`exPublic`) — sonst schlägt `TestPermissionMatrix_Backend` bei neuen HLS-Routen fehl
- [x] 11.4 Commit(s) gemäß `docs/agent/09-openspec.md` (feat(videos): … reworded aus „intermediate")
- [ ] 11.5 `/verify-change` und `make deploy` — User-Sache
- [ ] 11.6 `openspec archive video-tv-streaming` nach erfolgreichem Deploy — User-Sache
