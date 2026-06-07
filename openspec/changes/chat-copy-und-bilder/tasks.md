## 1. Datenbank-Migration

- [ ] 1.1 Migration `0NN_chat_messages_image_url.up.sql`: `ALTER TABLE chat_messages ADD COLUMN image_url TEXT`
- [ ] 1.2 Migration `.down.sql` anlegen (ALTER TABLE ist in SQLite nicht rückgängig machbar — Down-Migration leer oder Tabellen-Rebuild)
- [ ] 1.3 `go build ./cmd/teamwerk` muss ohne Fehler durchlaufen

## 2. Backend: Bild-Upload und -Abruf

- [ ] 2.1 Verzeichnis `/var/lib/teamwerk/chat-images/` beim Server-Start automatisch anlegen (analog zu `files/`)
- [ ] 2.2 `POST /api/chat/upload`: Datei-Upload entgegennehmen, MIME-Type prüfen (JPEG/PNG/GIF/WebP), > 10 MB mit 413 ablehnen, unter `<uuid>.<ext>` speichern, `{ imageUrl }` zurückgeben
- [ ] 2.3 `GET /api/chat/images/:filename`: Datei aus `chat-images/`-Verzeichnis ausliefern, korrekten Content-Type setzen, 404 wenn nicht gefunden
- [ ] 2.4 Beide Routen in `main.go` unter `authenticated`-Gruppe registrieren

## 3. Backend: Nachrichtenmodell erweitern

- [ ] 3.1 `Message`-Struct in `internal/chat/` um `ImageURL *string` erweitern
- [ ] 3.2 `GET /api/chat/conversations/{id}/messages`: `image_url` aus DB lesen und im Response-JSON als `imageUrl` mitliefern
- [ ] 3.3 `POST /api/chat/conversations/{id}/messages`: `imageUrl` aus Request-Body lesen, in DB schreiben; Validierung: body leer UND imageUrl leer → 400

## 4. Frontend: „Kopieren" im Kontextmenü

- [ ] 4.1 In `ChatPage.tsx` im Kontextmenü einen „Kopieren"-Button ergänzen (zwischen Antworten und Bearbeiten); nur anzeigen wenn `msg.body` nicht leer
- [ ] 4.2 `copyMsg`-Handler implementiert: `navigator.clipboard.writeText(msg.body)`, danach `setContextMenu(null)`

## 5. Frontend: Bild-Picker und Paste-Unterstützung

- [ ] 5.1 Bild-Picker-Button (Büroklammer-Icon, `Paperclip` aus lucide-react) links neben dem Texteingabefeld einfügen; `<input type="file" accept="image/*" hidden>` via Ref
- [ ] 5.2 `pendingImage`-State (`{ file: File; previewUrl: string } | null`) im `ChatPage`-State anlegen
- [ ] 5.3 Bei Dateiauswahl: `createObjectURL` für Vorschau, State setzen
- [ ] 5.4 Vorschau-Bereich unterhalb des Reply/Edit-Bars: zeigt Thumbnail + ×-Button (setzt `pendingImage` zurück)
- [ ] 5.5 `paste`-EventListener auf `window` registrieren (nur wenn Chat-Input fokussiert): `ClipboardEvent.clipboardData.files[0]` wenn Bild → `pendingImage` setzen
- [ ] 5.6 `sendMessage`: wenn `pendingImage` gesetzt, zuerst `POST /api/chat/upload`, dann `imageUrl` mit in den Nachrichten-POST aufnehmen

## 6. Frontend: Bild in MessageBubble anzeigen

- [ ] 6.1 `Message`-Interface um `imageUrl: string | null` erweitern
- [ ] 6.2 In `MessageBubble`: wenn `msg.imageUrl` gesetzt, `<img>` mit `max-w-xs rounded-lg cursor-pointer` unter dem Text rendern
- [ ] 6.3 `lightboxImage`-State (`string | null`) in `ChatPage` anlegen; Klick auf Bild setzt diesen State
- [ ] 6.4 Vollbild-Overlay: `fixed inset-0 z-50 bg-black/80 flex items-center justify-center` mit `<img>` und Schließen-Button oben rechts
