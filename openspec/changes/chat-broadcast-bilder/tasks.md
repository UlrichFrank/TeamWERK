## 1. Migration 028 (FK-sicherer Rebuild)

- [x] 1.1 `028_chat_broadcast_media.up.sql`: `CREATE TABLE media (…)`; Rebuild `messages` (neue Spalte `media_id INTEGER REFERENCES media(id)`, `CHECK((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)`, alle Bestandsspalten 1:1 kopieren), Index `idx_messages_conv` neu; Rebuild `broadcasts` analog.
- [x] 1.2 `028_chat_broadcast_media.down.sql`: Rebuild auf strikte Alt-DDL zurück (`media_id` entfällt, alter CHECK), `DROP TABLE media`. Hinweis im Kommentar: Zeilen mit leerem Body (reine Bilder) müssen beim Down auf einen Platzhalter gesetzt oder gefiltert werden.
- [x] 1.3 Preservation-Test in `internal/db`: vor `028` je 1 `message` + `message_reaction` + `broadcast` + `broadcast_read` seeden, `Migrate` laufen lassen, danach prüfen: alle Zeilen erhalten, `media_id` NULL, Reaktionen nicht kaskadiert gelöscht.
- [x] 1.4 `make migrate-up` / `go test ./internal/db/...` grün.

## 2. Backend: Media-Package

- [x] 2.1 `internal/config`: Feld `MediaDir` + `getEnv("MEDIA_DIR", "./storage/media")` in `Load()`.
- [x] 2.2 `internal/media/handler.go`: `Handler{db, mediaDir}`, `NewHandler(db, mediaDir)` mit `os.MkdirAll(mediaDir, 0755)` (Muster `files.NewHandler`).
- [x] 2.3 `Upload` (`POST /api/media/upload`): `MaxBytesReader` 1 MB, `image`-FormFile, MIME-Whitelist (jpeg/png/gif/webp) sonst 400, > 1 MB → 413, `<uuid>.<ext>` speichern, INSERT `media`, Response `{ "mediaId", "url": "/media/<id>" }`; Datei-Rollback bei DB-Fehler.
- [x] 2.4 `Serve` (`GET /api/media/{id}`): `media`-Zeile lesen (404 wenn fehlt), `Content-Type` aus `mime_type` + `X-Content-Type-Options: nosniff`, `http.ServeContent`.
- [x] 2.5 `internal/app`: `Handlers.Media`-Feld ergänzen; `cmd/teamwerk/main.go`: `Media: media.NewHandler(database, cfg.MediaDir)` verdrahten.
- [x] 2.6 `internal/app/router.go`: `r.Post("/api/media/upload", h.Media.Upload)` + `r.Get("/api/media/{id}", h.Media.Serve)` im authenticated-Tier.
- [x] 2.7 Arch-Test: `domain["media"] = true` in `internal/arch/arch_test.go`.
- [x] 2.8 Broadcast-Gate: `"Media.Upload": "Upload-Vorstufe, kein Live-Update; nachfolgender SendMessage/SendBroadcast broadcastet"` in `broadcastAllowlist`.

## 3. Backend: Chat Messages um Media erweitern

- [x] 3.1 `Message`-Struct + `SELECT` in `ListMessages` um `media_id` erweitern; `MediaID *int` + `MediaURL *string` (`"/media/<id>"`) im JSON.
- [x] 3.2 `SendMessage`: `mediaId` aus Body lesen; Validierung „body **oder** mediaId"; optional Existenz von `mediaId` in `media` prüfen; `INSERT INTO messages (…, media_id)`.

## 4. Backend: Broadcasts um Media erweitern

- [x] 4.1 `Broadcast`-Struct + `SELECT` in `ListBroadcasts` um `media_id`/`MediaID`/`MediaURL` erweitern.
- [x] 4.2 `SendBroadcast`: `mediaId` lesen; Validierung „body **oder** mediaId"; `INSERT INTO broadcasts (…, media_id)`.

## 5. Frontend: Kompression + Auth-Bild

- [x] 5.1 `web/src/lib/imageCompress.ts`: File → Canvas, längste Kante ≤ ~1920 px, iterative Qualitätsreduktion bis Blob ≤ 1 MB (Muster `ImageCropModal.tsx`); Rückgabe Blob + Ziel-Dateiname.
- [x] 5.2 `AuthImage`-Komponente (oder Inline-Hook): `api.get(url, {responseType:'blob'})` → `createObjectURL`, Cleanup mit `revokeObjectURL` (Muster `ReportImage`).

## 6. Frontend: Sende-Bereich (beide Tabs)

- [x] 6.1 Bild-Picker-Button (`Paperclip`, lucide) + verstecktes `<input type="file" accept="image/*">` im Sende-Bereich der Tabs „Chats" **und** „Mitteilungen".
- [x] 6.2 `pendingImage`-State + Vorschau-Leiste mit ×-Button (Entfernen) unter der Reply/Edit-Bar.
- [x] 6.3 `paste`-Listener (nur bei fokussiertem Chat-Input): Bild aus `clipboardData.files` → `pendingImage`.
- [x] 6.4 Beim Senden: `imageCompress` → `POST /api/media/upload` → `mediaId` an den Message-/Broadcast-POST anhängen; leerer Text + Bild erlaubt.

## 7. Frontend: Anzeige

- [x] 7.1 TS-Interfaces `Message` + `Broadcast` um `mediaId: number | null`, `mediaUrl: string | null`.
- [x] 7.2 In Nachrichten-Blase **und** Mitteilungs-Eintrag: `AuthImage` mit `max-w-xs rounded-lg cursor-pointer` unter dem Text (oder allein), wenn `mediaUrl` gesetzt.
- [x] 7.3 `lightbox`-State + Vollbild-Overlay (`fixed inset-0 z-50 bg-black/80`) mit Bild + `X`-Schließen-Button.
- [x] 7.4 Nur `brand-*`-Tokens + `lucide-react`; keine Raw-Farben/Emojis.

## 8. Verifikation

- [x] 8.1 `go build ./...`, `go test ./...` (inkl. Arch- + Broadcast-Gate), `golangci-lint` grün.
- [x] 8.2 `pnpm -C web build`, `pnpm -C web test`, `pnpm -C web lint` grün.
- [x] 8.3 `openspec validate chat-broadcast-bilder --strict`.

## Test-Anforderungen

| Route/Verhalten | Testname | Erwarteter Status |
|---|---|---|
| `POST /api/media/upload` gültiges JPEG | `TestUpload_OK` | 200/201 + `{mediaId,url}` |
| `POST /api/media/upload` PDF | `TestUpload_BadMime` | 400 |
| `POST /api/media/upload` > 1 MB | `TestUpload_TooLarge` | 413 |
| `POST /api/media/upload` ohne Auth | `TestUpload_Unauth` | 401 |
| `GET /api/media/{id}` vorhanden | `TestServe_OK` | 200 + korrekter Content-Type |
| `GET /api/media/{id}` unbekannt | `TestServe_NotFound` | 404 |
| `GET /api/media/{id}` ohne Auth | `TestServe_Unauth` | 401 |
| `POST .../messages` reines Bild (`body:"", mediaId`) | `TestSendMessage_ImageOnly` | 201, `media_id` gesetzt |
| `POST .../messages` leer ohne Bild | `TestSendMessage_EmptyNoMedia` | 400 |
| `GET .../messages` mit Bild | `TestListMessages_Media` | Objekt enthält `mediaId`+`mediaUrl` |
| `POST /api/chat/broadcasts` reines Bild | `TestSendBroadcast_ImageOnly` | 201, `media_id` gesetzt |
| `POST /api/chat/broadcasts` leer ohne Bild | `TestSendBroadcast_EmptyNoMedia` | 400 |
| Migration erhält Bestandsdaten | `TestMigrate028_PreservesRows` | message/reaction/broadcast/read überleben, `media_id` NULL |

**Garantierte Invarianten:** (1) Der Server akzeptiert nur Bilder ≤ 1 MB mit Whitelist-MIME. (2) Eine Nachricht/Mitteilung hat immer nicht-leeren `body` **oder** ein `media_id` (DB-CHECK + App-Validierung). (3) Die Migration `028` erhält alle Bestandszeilen und löst dank `foreign_keys=OFF` keine Cascade-Deletes aus.
