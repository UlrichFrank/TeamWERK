## Why

Chat-Bilder ohne bekannte Aspect-Ratio verschieben beim Bild-Load die
Scroll-Position: der `AuthImage`-Placeholder ist 6 rem hoch, das echte Bild
kann 30 – 400 px hoch werden. Der Auto-Scroll ans Ende der Konversation
scrollt zunächst korrekt, danach wächst der Container asynchron mit jedem
Blob-Load und der Nutzer landet doch nicht am Ende. Symptom: Klick auf
Konversation in der Chat-Übersicht → landet mitten im Verlauf.

Der bereits gebaute Client-Fix (Aspect-Ratio-Probe im Browser nach
Blob-Download) verhindert *folgende* Shifts, aber nicht den ersten
Übergang vom Placeholder zum ausgemessenen Bild. Nur wenn der Client die
Dimensionen ab dem ersten Frame kennt, entfällt der Shift komplett — der
gleiche Ansatz wie WhatsApp/Signal/Telegram.

## What Changes

- Migration 030: `media.width` und `media.height` als NULLABLE INTEGER
  (Bestandszeilen bleiben NULL bis der Backfill sie füllt).
- Upload-Handler probet nach der MIME-Prüfung mit `image.DecodeConfig`
  (JPEG/PNG/GIF stdlib) bzw. `webp.DecodeConfig` (via
  `golang.org/x/image/webp`, ~1 zusätzliche Zeile Import) die Dimensionen
  aus dem in-memory Byte-Slice und schreibt sie mit ins `INSERT`.
- Einmaliger Backfill-Job als Goroutine in `serve()` (Muster wie
  `internal/videos/backfill.go`): `SELECT id, disk_name, mime_type FROM
  media WHERE width IS NULL` → Datei lesen → probe → `UPDATE`.
  Idempotent, silently no-op nach erstem Lauf.
- Chat-Message- und Broadcast-Responses liefern `mediaWidth` und
  `mediaHeight` (beide `omitempty` — alte Bilder ohne Dims bekommen
  keinen Wert).
- `AuthImage` bekommt optionale `naturalWidth`/`naturalHeight`-Props:
  wenn gesetzt, wird `aspectRatio` ab dem ersten Frame gesetzt und der
  Client-seitige Probe entfällt (Fallback bleibt für alte Bilder ohne
  Backfill-Coverage bzw. andere Aufrufer).
- Bonus (unabhängig vom Bilderthema, gleicher Root-Cause „nicht am Ende
  gelandet"): `?openUser=<id>`-Deep-Link in `ChatPage.tsx` konsolidiert
  sich auf `openConversation()` statt eigenem `setActiveConv`+
  `loadMessages`-Duplikat, damit auch dieser Pfad `forceScrollToEndRef`
  setzt.

**Kein BREAKING**: Alte Frontend-Clients ignorieren die neuen Felder;
Bestandsbilder ohne Dimensionen fallen automatisch auf den bestehenden
Client-Probe zurück.

## Capabilities

### New Capabilities

Keine.

### Modified Capabilities

- `media-storage`: Upload probet und persistiert Bild-Dimensionen;
  `GET /api/media/{id}` unverändert (nur die Metadaten in der DB
  wachsen).
- `chat-konversationen`: Message-Response liefert zusätzlich
  `mediaWidth`/`mediaHeight` (nur bei Bild-Nachrichten mit bekannten
  Dimensionen).
- `chat-broadcasts`: Broadcast-Response liefert zusätzlich
  `mediaWidth`/`mediaHeight`.

## Impact

- **DB**: Migration 030 (2 ALTER TABLE, beide NULL-safe). Kein
  Tabellen-Rebuild nötig.
- **Backend**: neue Dependency `golang.org/x/image/webp` (kleine, stabile
  x/-Bibliothek — akzeptabel; keine cgo-Anforderung). `internal/media`
  bekommt ~30 Zeilen Probe-Logik. Neuer Backfill-File in
  `internal/media/backfill.go` (~80 Zeilen, Copy-Pattern von
  `internal/videos/backfill.go`).
- **API**: Response-Schemas von `GET /api/chat/conversations/{id}/messages`,
  `GET /api/chat/messages/{id}` und `GET /api/chat/broadcasts` bekommen
  zwei optionale Felder. Additive Änderung, kein Breaking.
- **Frontend**: `AuthImage`-Signatur bekommt zwei optionale Props;
  `ChatPage.tsx` reicht sie durch. `openUser`-Konsolidierung als
  Bonus-Task.
- **Reihenfolge**: Das noch nicht archivierte Change `chat-broadcast-bilder`
  legt die `media`-Tabelle und die betroffenen Response-Schemas erst an.
  Diese Delta-Specs bauen darauf auf; das Change wird nach
  `chat-broadcast-bilder` archiviert.
- **Kein zusätzlicher Speicher**: Header-Probe liest nur die ersten paar
  hundert Bytes des in-memory Uploads, kein Full-Decode. Backfill liest
  jede Datei genau einmal.
