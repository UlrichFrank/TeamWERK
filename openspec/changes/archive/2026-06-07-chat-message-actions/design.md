## Context

Der Chat in TeamWERK besteht aus `conversations` (direct/group) mit zugehörigen `messages`. Aktuell haben Nachrichten keine Interaktionsmöglichkeiten: kein Reply, kein Edit, kein Delete. Die `messages`-Tabelle hat die Felder `id, conversation_id, sender_id, body, sent_at`. Das Frontend rendert Nachrichten als statische Bubbles ohne Kontext-Menü.

Broadcasts sind One-Way-Mitteilungen in einer separaten `broadcasts`-Tabelle. Delete existiert bereits (soft via `broadcast_reads.hidden_at`). Edit fehlt.

## Goals / Non-Goals

**Goals:**
- Reply auf beliebige Nachrichten (eigene + fremde) mit Quote-Darstellung
- Edit eigener Nachrichten ohne Zeitlimit, mit `(bearbeitet)`-Indikator
- Soft-Delete eigener Nachrichten; Admin kann alle Nachrichten löschen
- Broadcasts: Edit für Sender
- Kein Breaking Change an bestehenden API-Clients (neue Felder sind additiv)

**Non-Goals:**
- Reply auf Broadcasts (One-Way-Kanal)
- Edit durch Admins fremder Nachrichten
- Lese-Bestätigungen pro Message-Edit
- Nachrichtenhistorie (kein Audit-Log der Bearbeitungen)
- Nachrichten-Forwarding

## Decisions

### 1. Soft-Delete statt Hard-Delete

Reply-Zitate würden bei Hard-Delete ihren Kontext verlieren. Soft-Delete via `deleted_at`-Timestamp hält die Zeile in der DB, das Frontend rendert einen Placeholder. Der `body` bleibt erhalten (für Admin-Moderation denkbar), wird aber nicht ausgegeben wenn `deleted_at IS NOT NULL`.

Alternative: Hard-Delete mit `reply_to_id` auf NULL setzen. Verworfen: Replies verlieren Kontext, Partial-Update ist komplexer.

### 2. reply_to_id als FK, Anzeige via JOIN

`messages.reply_to_id` referenziert `messages.id`. Der `ListMessages`-Query joined einmal auf die Parent-Zeile und gibt `replyToBody` und `replyToSenderName` denormalisiert zurück. Damit braucht das Frontend keinen separaten API-Call.

Wenn die referenzierte Nachricht gelöscht wurde, gibt der JOIN `reply_to_body = "[gelöscht]"` zurück (COALESCE im Query).

### 3. Kein eigenes Kontext-Menü-Package

Das Custom-Kontext-Menü wird direkt in `ChatPage.tsx` als lokale Komponente implementiert (positioniert via `position: fixed`, `left/top` aus `MouseEvent.clientX/Y`). Kein extra Package nötig — hält die Dependency-Liste klein (VPS-RAM-Constraint).

### 4. Swipe-to-Reply via native Touch-Events

Touch-Geste (`touchstart` / `touchmove` / `touchend`) direkt auf der Bubble-Wrapper-Div, mit CSS `transform: translateX()`. Threshold: 60px. Kein Gesture-Library — bleibt im bestehenden React-Event-Modell.

### 5. Bearbeitungs-Datum in `messages.edited_at`

Null wenn nie bearbeitet, sonst Timestamp der letzten Bearbeitung. Nur letzte Bearbeitung wird gespeichert (kein History-Log). Das ist ausreichend für den Use-Case „(bearbeitet)" anzeigen.

### 6. Broadcasts: Edit analog zu bestehender Delete-Logik

`broadcasts.edited_at` wird gesetzt, Sender kann PUT `/api/chat/broadcasts/{id}` aufrufen. Nur Sender darf bearbeiten (kein Admin-Override für Edit, nur für Nachrichten-Delete).

## Risks / Trade-offs

- **Soft-Delete füllt die DB langsam** → Für einen Vereins-Messenger mit <100 aktiven Nutzern vernachlässigbar. Kein Cleanup-Job nötig.
- **reply_to_id auf gelöschte Nachricht** → COALESCE im Query gibt `"[gelöscht]"` zurück; kein NULL-Fehler im Frontend.
- **Rechtsklick auf Mobile** → Native Context-Menu auf Mobile existiert nicht als Rechtsklick. Die mobile Geste ist Swipe-right; Rechtsklick ist nur Desktop. Kein Konflikt.
- **Race Condition Edit vs. Delete** → `UPDATE messages SET body=? WHERE id=? AND sender_id=? AND deleted_at IS NULL` — ein bereits gelöschter Message kann nicht mehr bearbeitet werden.

## Migration Plan

1. Migration `012_chat_message_actions.up.sql`:
   - `ALTER TABLE messages ADD COLUMN reply_to_id INTEGER REFERENCES messages(id)`
   - `ALTER TABLE messages ADD COLUMN edited_at DATETIME`
   - `ALTER TABLE messages ADD COLUMN deleted_at DATETIME`
   - `ALTER TABLE broadcasts ADD COLUMN edited_at DATETIME`
2. `make deploy` führt `migrate up` automatisch aus — kein manueller Schritt nötig.
3. Rollback via `012_chat_message_actions.down.sql` mit `ALTER TABLE … DROP COLUMN` (SQLite ≥ 3.35, modernc.org/sqlite unterstützt das).
