## 1. Migration & Dependency

- [x] 1.1 `go get golang.org/x/image/webp` und `go mod tidy` ausführen (neue kleine x/-Dep, ~500 KB im Binary)
- [x] 1.2 Migration `internal/db/migrations/030_media_dimensions.up.sql` schreiben: `ALTER TABLE media ADD COLUMN width INTEGER; ALTER TABLE media ADD COLUMN height INTEGER;` (beide NULL für Bestand)
- [x] 1.3 Passende `030_media_dimensions.down.sql` schreiben (SQLite `ALTER TABLE ... DROP COLUMN` erst ab 3.35 — im Zweifel Tabellen-Rebuild ohne die Spalten, so wie es die Konvention aus `docs/agent/02-workflow.md` gewohnt ist)

## 2. Upload-Probe im media-Handler

- [x] 2.1 In `internal/media/handler.go` (oder neuer `dimensions.go` daneben) Funktion `decodeDimensions(data []byte, mimeType string) (w, h int, ok bool)` bauen, die per `image.DecodeConfig` (JPEG/PNG/GIF) bzw. `webp.DecodeConfig` (`golang.org/x/image/webp`) nur den Header liest
- [x] 2.2 In `Handler.Upload` nach der MIME-Prüfung `decodeDimensions` aufrufen; `INSERT INTO media` um `width, height` erweitern (`sql.NullInt64` wenn Probe fehlschlug)
- [x] 2.3 Upload-Response um optionale `width`/`height`-Felder erweitern (`omitempty`)
- [x] 2.4 Test `TestMediaUpload_JPEG_ReturnsDimensions`: 1×1-JPEG hochladen, Response prüfen (Happy-Path)
- [x] 2.5 Test `TestMediaUpload_CorruptHeader_AcceptedWithoutDimensions`: Datei mit `image/jpeg`-Magic-Bytes aber kaputtem Rest hochladen; Upload akzeptiert, Response ohne width/height (Fehlerfall Probe)

## 3. Backfill-Job

- [x] 3.1 `internal/media/backfill.go` anlegen, exakt an `internal/videos/backfill.go` orientiert: `Backfill(ctx, db, mediaDir)`, idempotent, sequentiell, loggt Start/Anzahl/Ende
- [x] 3.2 Query `SELECT id, disk_name, mime_type FROM media WHERE width IS NULL`, pro Zeile Datei lesen (auf `maxImageBytes` cappen), `decodeDimensions`, `UPDATE` wenn erfolgreich; Fehler loggen und weiter
- [x] 3.3 In `cmd/teamwerk/main.go` in `serve()` als Goroutine starten: `go func() { media.Backfill(ctx, db, cfg.MediaDir) }()`
- [x] 3.4 Test `TestBackfill_UpdatesNullRows`: Fixture mit `width IS NULL`-Zeile und JPEG-Datei; `Backfill` läuft; DB-Zeile ist danach gefüllt
- [x] 3.5 Test `TestBackfill_SkipsAlreadyFilledRows`: Fixture mit `width IS NOT NULL`; Datei-I/O wird nicht angefasst (kein Read-Aufruf)
- [x] 3.6 Test `TestBackfill_MissingFileContinues`: Zeile mit `disk_name` ohne Datei auf Disk; Backfill loggt und läuft weiter, bricht nicht ab

## 4. Response-Erweiterung Chat + Broadcasts

- [x] 4.1 In `internal/chat/handler.go` das `Message`-Struct in `ListMessages` (Zeile ~513) um `MediaWidth *int` und `MediaHeight *int` mit `omitempty` erweitern
- [x] 4.2 Query in `ListMessages` (`messageSelect`) um `m2.width, m2.height` erweitern (JOIN existiert schon via `mediaId`) und Scan anpassen
- [x] 4.3 Analog für `GetMessage` (Einzelnachricht, gleiche Message-Struktur), wenn dort auch Bilder ausgeliefert werden — **No-Op**: `GetMessage` liefert nur id/body/deleted, keine Media-Felder
- [x] 4.4 Broadcast-Struct und -Query (`internal/chat/handler.go:957` und Broadcast-Liste) um `MediaWidth`/`MediaHeight` erweitern
- [x] 4.5 Test `TestListMessages_IncludesMediaDimensions`: Fixture-Bild mit `width=1920, height=1080`, Message referenziert es; Response enthält beide Felder
- [x] 4.6 Test `TestListMessages_OmitsMediaDimensionsWhenNull`: Bild ohne `width`/`height`; Felder fehlen in Response

## 5. AuthImage-Prop

- [x] 5.1 `AuthImage`-Signatur in `web/src/components/AuthImage.tsx` um `naturalWidth?: number`, `naturalHeight?: number` erweitern
- [x] 5.2 Rendering: wenn beide Props gesetzt sind, `aspectRatio` ab dem ersten Frame anwenden und den `Image()`-Probe-Pfad überspringen (Blob-Load bleibt); wenn nicht gesetzt, bisheriger Probe-Weg (Fallback)
- [x] 5.3 In `ChatPage.tsx` die drei `AuthImage`-Aufrufe (`ChatPage.tsx:1188`, `:1381`, `:1598` — Broadcast-Preview, Bild-Preview vor Senden, Message-Bild) mit `naturalWidth={msg.mediaWidth}` und `naturalHeight={msg.mediaHeight}` versorgen; im TypeScript-Message-Typ die neuen optionalen Felder ergänzen — Lightbox-Aufruf (`:1420`) bleibt ohne Props (Modal mit `object-contain`, kein Layout-Shift-Risiko)
- [x] 5.4 Test `AuthImage.dimensions.test.tsx`: mit `naturalWidth/Height` gesetzt → kein `Image()`-Preload; `aspectRatio` sofort im Style; ohne Props → alter Probe-Weg (Mocks für `URL.createObjectURL` reichen)

## 6. Bonus: openUser-Deep-Link auf openConversation konsolidieren

- [x] 6.1 In `web/src/pages/ChatPage.tsx` den `?openUser=<id>`-Handler (Zeilen ~367-381) umbauen: nach `POST /api/chat/conversations` das Ergebnis via `openConversation(conv)` öffnen statt inline `setActiveConv`+`loadMessages`; `setTab("chats")` bleibt
- [x] 6.2 Test in `ChatPage.deepLink.test.tsx` (neu oder existierend erweitert): Route `/chat?openUser=42` besucht; nach dem Öffnen läuft ein `scrollIntoView`-Call → landet am Ende (Muster wie im bestehenden `ChatPage.windowing.test.tsx`)

## 7. Verifikation

- [x] 7.1 `make lint && make test` grün
- [x] 7.2 `openspec validate chat-image-dimensions` grün
- [ ] 7.3 Manuell im Browser: bestehende Chat-Bild-Nachricht öffnen — kein sichtbarer Layout-Shift beim Bild-Load
- [ ] 7.4 Manuell: neues Bild hochladen und senden — beim Auto-Scroll ans Ende bleibt die Position stabil, auch nachdem der Blob dekodiert ist
- [ ] 7.5 Log-Check nach Deploy: `media.Backfill` hat gestartet und mit „nichts zu tun" bzw. „N Zeilen aktualisiert" beendet
