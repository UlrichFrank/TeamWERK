## Context

Chat-Nachrichten haben ein Kontextmenü (Desktop: Rechtsklick, Mobile: Long-Press) mit Antworten/Bearbeiten/Löschen. Bilder werden bisher nicht unterstützt. Alle Chat-Nachrichten liegen in `chat_messages` (SQLite); Bilder des Dokumenten-Moduls nutzen einen eigenen Ordner unter `/var/lib/teamwerk/files/`.

## Goals / Non-Goals

**Goals:**
- Nachrichtentext per Kontextmenü in Zwischenablage kopieren (Clipboard API, kein Backend-Aufwand)
- Bilder als Nachrichteninhalt hochladen, senden und inline anzeigen
- Bilder via Datei-Picker oder Einfügen aus Zwischenablage (Paste-Event) auswählen

**Non-Goals:**
- Videos oder andere Dateitypen
- Bildbearbeitung / Crop vor dem Senden
- CDN / Object Storage (Bilder liegen lokal auf VPS-Disk)
- Kompression oder Thumbnails (Upload-Limit verhindert zu große Dateien)

## Decisions

### D1: Separater Upload-Endpunkt vor dem Senden

`POST /api/chat/upload` nimmt `multipart/form-data` mit einer Bilddatei entgegen, speichert sie unter `/var/lib/teamwerk/chat-images/<uuid>.<ext>` und gibt `{ imageUrl: "/api/chat/images/<uuid>.<ext>" }` zurück. Das Frontend hängt die `imageUrl` dann an die Sende-Request.

**Alternativen:** Bild direkt im Message-POST mitsenden — würde jedoch `POST /api/chat/conversations/{id}/messages` von JSON auf multipart umstellen, was den bestehenden Reply/Edit-Flow komplizierter macht. Zwei-Schritt-Ansatz hält die Interfaces sauber.

### D2: Bildzugriff via eigener Serve-Route (kein Token-Download)

`GET /api/chat/images/:filename` gibt das Bild zurück, sofern der anfragende User Mitglied einer Konversation ist, in der dieses Bild vorkommt — oder einfacher: jeder authentifizierte User kann Chat-Bilder abrufen (da Chat sowieso nur für eingeloggte User zugänglich ist).

**Rationale:** Einfache Implementierung, passend zur Chat-Sichtbarkeit (alle eingeloggten User können miteinander chatten). Keine separaten Zugriffsrechte nötig.

### D3: `image_url` als nullable Spalte in `chat_messages`

Migration fügt `image_url TEXT` zu `chat_messages` hinzu. Eine Nachricht hat entweder `body`, `image_url` oder beides. Validierung im Backend: mindestens eines der beiden Felder muss befüllt sein.

### D4: Bilder werden in MessageBubble inline als `<img>` gerendert

Max. Anzeigebreite: `max-w-xs` (wie Textblasen). Tipp/Klick öffnet ein Overlay (Fullscreen-Modal mit weißem Hintergrund). Kein Lightbox-Package — einfaches `fixed inset-0` Modal mit `<img>` und Schließen-Button reicht.

## Risks / Trade-offs

- **Speicherplatz VPS:** Chat-Bilder liegen auf 1-GB-VPS. 10 MB Limit pro Bild + Typ-Whitelist (JPEG/PNG/GIF/WebP) begrenzen das Risiko. → Kein automatisches Cleanup vorgesehen (out of scope).
- **Clipboard API:** `navigator.clipboard.writeText` erfordert HTTPS und User-Geste. In der PWA auf HTTPS immer gegeben. Fallback (execCommand) nicht nötig. → Fehlerfall wird still ignoriert (kein sichtbarer Fehler beim Kopieren).
- **Paste-Event:** `paste` auf `window` abonnieren, wenn das Input fokussiert ist — kann mit anderen Paste-Handlern interferieren. → Nur aktiv schalten wenn das Chat-Input fokussiert ist.

## Migration Plan

1. Migration `0NN_chat_messages_image_url.up.sql`: `ALTER TABLE chat_messages ADD COLUMN image_url TEXT`
2. Neuer Ordner `/var/lib/teamwerk/chat-images/` wird beim Server-Start automatisch angelegt (wie für `files/` bereits implementiert)
3. Kein Rollback-Risiko — neue Spalte ist nullable, alter Code ignoriert sie
