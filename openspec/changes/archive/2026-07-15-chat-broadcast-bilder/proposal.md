## Why

Chat-Nachrichten (`/chat`, Tab „Chats") und Mitteilungen (Tab „Mitteilungen"/Broadcasts) sind heute reiner Text. Für Mannschaftsfotos, Spieltag-Bilder, abfotografierte Dokumente oder Aushänge fehlt die Möglichkeit, Bilder zu teilen — in beiden Tabs. Da Chats und Mitteilungen im selben Modul (`internal/chat/`) leben und beide dasselbe Text-Body-Modell nutzen, soll die Bildunterstützung **einmal** gebaut und von beiden genutzt werden.

## What Changes

- Neuer **gemeinsamer Media-Store** (`internal/media/`, neue `media`-Tabelle): `POST /api/media/upload` nimmt ein Bild entgegen und gibt `{ mediaId, url }` zurück; `GET /api/media/{id}` liefert die Bytes (JWT-authentifiziert).
- `messages` und `broadcasts` bekommen je eine nullable Spalte `media_id` (FK → `media`). Der bisherige `CHECK(length(body) > 0 …)` wird zu `CHECK((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)` gelockert, damit reine Bild-Beiträge (leerer Text) erlaubt sind — per **FK-sicherem Tabellen-Rebuild** (Migration 028).
- `POST /api/chat/conversations/{id}/messages` und `POST /api/chat/broadcasts` akzeptieren zusätzlich `mediaId`; Validierung: `body` (nicht leer) **oder** `mediaId` muss vorhanden sein.
- `GET .../messages` und `GET /api/chat/broadcasts` liefern `mediaId` + `mediaUrl` mit.
- **Frontend (`ChatPage.tsx`)**: Bild-Picker (Büroklammer) + Einfügen aus Zwischenablage (Paste) im Sende-Bereich **beider** Tabs; Vorschau vor dem Senden mit Entfernen-Button; Inline-Anzeige in Nachrichten-/Mitteilungs-Blasen; Tipp/Klick öffnet Vollbild-Overlay.
- **Clientseitige Verkleinerung auf ≤ 1 MB** vor dem Upload (Canvas-Downscale + iterative Qualitätsreduktion, Muster aus `ImageCropModal.tsx`). Der Server erzwingt 1 MB als Backstop (> 1 MB → 413).

## Capabilities

### New Capabilities

- `media-storage`: Gemeinsamer, JWT-geschützter Bild-Upload/-Abruf über `POST /api/media/upload` und `GET /api/media/{id}`.

### Modified Capabilities

- `chat-konversationen`: Nachrichtenmodell erhält `mediaId`/`mediaUrl`; Senden akzeptiert `mediaId`; reine Bildnachrichten (leerer Body) erlaubt.
- `chat-broadcasts`: Mitteilungsmodell erhält `mediaId`/`mediaUrl`; Senden akzeptiert `mediaId`; reine Bild-Mitteilungen erlaubt.

## Impact

- **Backend:** Neues Package `internal/media/` (DOMAIN-Klassifizierung im Arch-Test); `internal/chat/handler.go` (Send/List für Messages + Broadcasts); `internal/config` (`MediaDir`); `cmd/teamwerk/main.go` + `internal/app` (Handler-Verdrahtung, `Handlers.Media`-Feld); Migration `028`.
- **Frontend:** `ChatPage.tsx`; neue Util `web/src/lib/imageCompress.ts`; ggf. kleine `AuthImage`-Hilfskomponente (axios-Blob → ObjectURL, Muster wie `ReportImage`).
- **Speicherplatz:** Bilder liegen auf VPS-Disk unter `MEDIA_DIR` (`./storage/media`), ≤ 1 MB je Bild. Kein automatisches Cleanup (out of scope).
- **Broadcast-Gate:** `POST /api/media/upload` mutiert ohne Live-Update → Allowlist-Eintrag mit Begründung (der nachfolgende `SendMessage`/`SendBroadcast` broadcastet).
- **Keine neuen externen Dependencies.**

## Abgrenzung zum bestehenden Change `chat-copy-und-bilder`

Dieser Change **ersetzt** den veralteten Proposal `chat-copy-und-bilder`. Gründe: jener referenziert eine nicht existierende Tabelle `chat_messages` (real: `messages`), erfindet einen chat-eigenen Bild-Store statt eines wiederverwendbaren Media-Stores, deckt Broadcasts nicht ab und enthält ein separat gewünschtes „Kopieren"-Feature (hier **bewusst nicht** enthalten). `chat-copy-und-bilder` ist beim Anlegen dieses Changes zu verwerfen/archivieren.
