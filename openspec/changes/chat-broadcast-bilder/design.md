## Context

Chat und Mitteilungen leben beide in `internal/chat/` und speichern Text in `messages` bzw. `broadcasts` (SQLite, beide `body TEXT NOT NULL CHECK(length(body) > 0 AND length(body) <= 2000)`, seit `001_initial` unverändert). Es gibt bereits zwei Upload-Systeme (`internal/files/` für Dokumente, `internal/videos/` für Videos) und clientseitige Canvas-Bildverarbeitung (`web/src/components/ImageCropModal.tsx`). Authentifizierte Inline-Bilder werden im Frontend etabliert per axios-Blob-Fetch geladen (`api.get(url, {responseType:'blob'})` → `URL.createObjectURL`, siehe `ReportImage` in `MatchReportFormPage.tsx:490`), weil `<img src>` keinen Bearer-Header sendet.

## Goals / Non-Goals

**Goals:**
- Ein **gemeinsamer** Media-Store, genutzt von Messages **und** Broadcasts.
- Bilder ≤ 1 MB (clientseitig verkleinert), Upload getrennt vom Senden.
- Reine Bild-Beiträge (leerer Text) in beiden Tabs.

**Non-Goals:**
- Videos/andere Dateitypen, Mehrfachbilder pro Beitrag, Bildbearbeitung/Crop vor dem Senden.
- CDN/Object-Storage, Thumbnails (bei ≤ 1 MB unnötig), automatisches Cleanup.
- „Kopieren"-Kontextmenü (bewusst raus — separater Wunsch).
- Bild beim Bearbeiten (Edit) einer bestehenden Nachricht ändern.

## Decisions

### D1: Gemeinsamer Media-Store statt chat-eigenem Bild-Ordner

Neues DOMAIN-Package `internal/media/` mit `NewHandler(db, mediaDir)` (legt `mediaDir` beim Start via `os.MkdirAll` an, Muster wie `files.NewHandler`). Neue Tabelle:

```sql
CREATE TABLE media (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    disk_name   TEXT    NOT NULL UNIQUE,
    mime_type   TEXT    NOT NULL,
    size        INTEGER NOT NULL,
    uploaded_by INTEGER NOT NULL REFERENCES users(id),
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
```

`POST /api/media/upload` (multipart, Feld `image`): MIME-Whitelist `image/jpeg|png|gif|webp`, `MaxBytesReader` 1 MB (> 1 MB → 413), Speicherung als `<uuid>.<ext>` unter `mediaDir`, INSERT in `media`, Response `{ "mediaId": <id>, "url": "/media/<id>" }` — **ohne `/api`-Prefix** (axios-baseURL ergänzt ihn; Konvention wie Match-Report-Bilder). Bei Schreibfehlern Datei-Rollback (`os.Remove`).

**Alternative (verworfen):** chat-eigener Store `/api/chat/images/...` (alter Proposal) — würde für Broadcasts dupliziert und ist ein dritter Upload-Mechanismus ohne Metadaten-Tracking.

### D2: Serve JWT-only, Frontend lädt per axios-Blob

`GET /api/media/{id}`: nur authentifiziert (Chat ist ohnehin nur für Eingeloggte); liest `disk_name`/`mime_type` aus `media`, setzt `Content-Type` + `X-Content-Type-Options: nosniff`, liefert Datei via `http.ServeContent`. Kein Token-Query nötig, weil das Frontend Bilder per `api.get(url,{responseType:'blob'})` → `createObjectURL` einbindet (Bearer kommt automatisch mit). Kleine wiederverwendbare Komponente `AuthImage` kapselt Fetch + Revoke.

### D3: `media_id` als FK-Spalte + gelockerter CHECK, via FK-sicherem Rebuild

Migration `028` baut `messages` und `broadcasts` neu:
- neue Spalte `media_id INTEGER REFERENCES media(id)`,
- `CHECK((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)`,
- Index `idx_messages_conv(conversation_id, sent_at DESC)` nach dem Rename neu anlegen.

**FK-Sicherheit:** `messages` wird von `message_reactions` (`ON DELETE CASCADE`) und sich selbst (`reply_to_id`) referenziert, `broadcasts` von `broadcast_reads`. `internal/db/db.go:Migrate` setzt für den gesamten Migrationslauf `PRAGMA foreign_keys = OFF` (db.go:59–66) — der `DROP TABLE` im Rebuild löst daher **keine** Cascade-Deletes aus. Rebuild-Reihenfolge je Tabelle: `CREATE …_new` → `INSERT … SELECT` (alle Bestandsspalten, `media_id` initial NULL) → `DROP` alt → `RENAME …_new` → Indizes neu. `.down.sql` baut die strikte Alt-DDL zurück (nur Zeilen mit nicht-leerem `body` übernehmen bzw. leere auf Platzhalter — siehe Down-Hinweis).

**Preservation-Test (Pflicht):** Da `testutil.NewDB` Migrationen auf eine **leere** DB anwendet, fängt kein Bestandstest versehentlichen Datenverlust. Ein Test in `internal/db` seedt vor `028` je eine `message` + `message_reaction` + `broadcast` + `broadcast_read`, migriert hoch und prüft, dass alle Zeilen erhalten sind und `media_id` NULL ist.

### D4: Zwei-Schritt-Senden (Upload, dann Message/Broadcast-POST)

Frontend lädt das (bereits verkleinerte) Bild via `POST /api/media/upload`, erhält `mediaId` und hängt es an den bestehenden JSON-POST (`{ body, mediaId }`). Hält die Message/Broadcast-Endpunkte bei JSON (kein Multipart), Reply/Edit-Flow unangetastet.

### D5: Clientseitige Verkleinerung auf ≤ 1 MB

Neue Util `web/src/lib/imageCompress.ts`: Bild in Canvas zeichnen, längste Kante auf max. ~1920 px deckeln, als JPEG (bzw. WebP) exportieren und Qualität iterativ senken (z.B. 0.9 → 0.5), bis das Blob ≤ 1 MB ist; Ergebnis-Blob wird hochgeladen. Transparente PNGs/GIFs: bei bereits ≤ 1 MB unverändert lassen, sonst ebenfalls verkleinern. Fällt die Verkleinerung aus (Format nicht darstellbar), greift der 1-MB-Server-Backstop.

### D6: MessageBubble/Broadcast-Anzeige + Lightbox

`media_id`/`mediaUrl` gesetzt → `AuthImage` mit `max-w-xs rounded-lg cursor-pointer` (unter dem Text bzw. allein). Klick öffnet `fixed inset-0 z-50 bg-black/80`-Overlay mit Bild + Schließen-Button (`X` aus lucide). Gilt identisch für Nachrichten und Mitteilungen.

## Risks / Trade-offs

- **Rebuild-Datenverlust:** Entschärft durch `foreign_keys=OFF` im Migrationslauf + Preservation-Test. Vor Prod-Deploy DB-Backup (Standard).
- **Speicherplatz VPS (1 GB):** ≤ 1 MB/Bild + Whitelist begrenzen; kein Cleanup (bewusst).
- **Canvas-Kompression:** Metadaten (EXIF-Rotation) können verloren gehen — vertretbar; ggf. Orientation aus EXIF vor dem Zeichnen berücksichtigen, wenn einfach.
- **Paste-Handler:** nur aktiv, wenn Chat-Input fokussiert, um andere Paste-Handler nicht zu stören.

## Migration Plan

1. `028_chat_broadcast_media.up.sql`: `CREATE TABLE media`; Rebuild `messages`; Rebuild `broadcasts`; Indizes neu.
2. `028_..._down.sql`: Rebuild zurück auf strikte Alt-DDL (Spalte `media_id` entfällt); `DROP TABLE media`.
3. `MEDIA_DIR` wird beim Start durch `media.NewHandler` angelegt (wie `files/`).
