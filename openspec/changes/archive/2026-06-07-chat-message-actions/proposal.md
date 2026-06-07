## Why

Die Chat-Funktion unterstützt bisher keine Interaktionen auf einzelne Nachrichten: kein Antworten auf eine bestimmte Nachricht, kein Korrigieren eines Tippfehlers, kein Löschen einer versehentlich gesendeten Nachricht. Das ist der erwartete Standard-Funktionsumfang moderner Messenger und fehlt den Nutzern im Alltag.

## What Changes

- Nachrichten können per **Rechtsklick (Desktop)** oder **Swipe-right (Mobile)** mit einem Kontext-Menü bedient werden
- **Antworten (Reply):** Auf jede Nachricht kann geantwortet werden — eigene und fremde. Die Antwort zeigt die zitierte Ursprungsnachricht als Quote-Block.
- **Bearbeiten:** Eigene Nachrichten können jederzeit bearbeitet werden. Bearbeitete Nachrichten zeigen `(bearbeitet)`.
- **Löschen:** Eigene Nachrichten können gelöscht werden (Soft-Delete: Placeholder „Nachricht gelöscht"). Admins können alle Nachrichten löschen.
- **Broadcast-Bearbeitung:** Sender können eigene Broadcast-Mitteilungen nachträglich bearbeiten (Delete existiert bereits).

## Capabilities

### New Capabilities

- `chat-message-reply`: Reply auf Chat-Nachrichten — DB-Feld, API-Feld, UI-Quote-Block und Swipe/Rechtsklick-Trigger
- `chat-message-edit`: Bearbeiten eigener Chat-Nachrichten — DB-Feld, PUT-Endpoint, Edit-Modus im Input
- `chat-message-delete`: Soft-Delete eigener Chat-Nachrichten (Admin: alle) — DB-Feld, DELETE-Endpoint, Placeholder in der UI
- `broadcast-edit`: Bearbeiten eigener Broadcasts — DB-Feld, PUT-Endpoint, Edit-Button in der Broadcast-Ansicht

### Modified Capabilities

- `chat-konversationen`: ListMessages gibt zusätzliche Felder zurück (`replyToId`, `replyToBody`, `replyToSenderName`, `editedAt`, `deletedAt`); SendMessage akzeptiert `replyToId`
- `chat-broadcasts`: ListBroadcasts gibt `editedAt` zurück; neuer PUT-Endpoint für Sender

## Impact

- **DB:** Migration `messages`: +`reply_to_id`, +`edited_at`, +`deleted_at`; Migration `broadcasts`: +`edited_at`
- **Backend:** `internal/chat/handler.go` — neue Handler `EditMessage`, `DeleteMessage`, `EditBroadcast`; geänderte Queries in `ListMessages` und `SendMessage`
- **Frontend:** `web/src/pages/ChatPage.tsx` — neue Komponenten: Kontext-Menü, Swipe-Geste, Reply-Leiste, Edit-Leiste, Quote-Block, Deleted-Placeholder, Broadcast-Edit-Modal
- **Keine neuen Dependencies**
