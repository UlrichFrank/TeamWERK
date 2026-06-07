## Why

Das Chat-Kontextmenü bietet bisher nur Antworten, Bearbeiten und Löschen — Nutzer können Nachrichtentext nicht direkt kopieren. Außerdem fehlt die Möglichkeit, Bilder in Gesprächen zu teilen, was für Mannschaftsfotos, Dokumente und Spieltag-Bilder praktisch wäre.

## What Changes

- Kontextmenü um „Kopieren"-Eintrag erweitert, der den Nachrichtentext via Clipboard API in die Zwischenablage schreibt
- Neuer Upload-Endpunkt für Chat-Bilder (`POST /api/chat/upload`)
- Nachrichtenmodell um optionales Feld `imageUrl` erweitert
- Sende-Button bekommt Bild-Picker (Dateiauswahl + Einfügen aus Zwischenablage via Paste-Event)
- Bilder werden als eigenständige Nachrichten gesendet (body leer, imageUrl gesetzt) oder zusammen mit Text
- Bilder werden in Nachrichtenblasen als Vorschau angezeigt; Tipp/Klick öffnet Vollbild

## Capabilities

### New Capabilities

- `chat-message-copy`: „Kopieren"-Aktion im Nachrichtenkontextmenü kopiert body in die Systemzwischenablage
- `chat-message-image`: Bilder in Chat-Nachrichten hochladen, senden und anzeigen

### Modified Capabilities

- `chat-konversationen`: Nachrichtenobjekt erhält optionales Feld `imageUrl`; `POST /api/chat/conversations/{id}/messages` akzeptiert zusätzlich `imageUrl`

## Impact

- **Backend:** Neues Handler-Methode in `internal/chat/`; Bildablage unter `/var/lib/teamwerk/chat-images/<uuid>.<ext>`; Migration für `image_url`-Spalte in `chat_messages`
- **Frontend:** `ChatPage.tsx` — Kontextmenü, Sende-Bereich, MessageBubble
- **Speicherplatz:** Bilder liegen auf VPS-Disk (Limit: 10 MB pro Bild, JPEG/PNG/GIF/WebP)
- **Keine neuen externen Dependencies**
