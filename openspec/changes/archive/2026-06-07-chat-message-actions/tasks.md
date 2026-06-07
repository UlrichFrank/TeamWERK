## 1. Datenbank-Migration

- [x] 1.1 Migration `012_chat_message_actions.up.sql` anlegen: `messages` bekommt `reply_to_id INTEGER REFERENCES messages(id)`, `edited_at DATETIME`, `deleted_at DATETIME`; `broadcasts` bekommt `edited_at DATETIME`
- [x] 1.2 Migration `012_chat_message_actions.down.sql` anlegen: die vier Spalten per `ALTER TABLE … DROP COLUMN` rückgängig machen

## 2. Backend — ListMessages erweitern

- [x] 2.1 `Message`-Struct in `handler.go` um Felder `ReplyToID`, `ReplyToBody`, `ReplyToSenderName`, `EditedAt`, `DeletedAt` erweitern (alle nullable via `sql.NullXxx`)
- [x] 2.2 `ListMessages`-Query anpassen: LEFT JOIN auf `messages rm` und `users ru` für Reply-Kontext; `COALESCE(rm.body, '[Nachricht gelöscht]')` wenn `rm.deleted_at IS NOT NULL`; `body` als leerer String wenn `m.deleted_at IS NOT NULL`

## 3. Backend — SendMessage erweitern

- [x] 3.1 `SendMessage`-Handler: `replyToId`-Feld aus Request-Body lesen; validieren, dass die referenzierte Nachricht zur selben Konversation gehört (HTTP 400 sonst); Wert in `INSERT INTO messages` schreiben

## 4. Backend — EditMessage

- [x] 4.1 Handler `EditMessage` implementieren: `PUT /api/chat/messages/{id}` — liest `body` aus Request; prüft `sender_id = claimsUserID AND deleted_at IS NULL`; `UPDATE messages SET body=?, edited_at=CURRENT_TIMESTAMP WHERE id=? AND sender_id=? AND deleted_at IS NULL`; HTTP 404 wenn nicht gefunden, 204 bei Erfolg
- [x] 4.2 Route in `main.go` registrieren: `r.Put("/chat/messages/{id}", chatH.EditMessage)` (Authenticated-Gruppe)

## 5. Backend — DeleteMessage (Soft-Delete)

- [x] 5.1 Handler `DeleteMessage` implementieren: `DELETE /api/chat/messages/{id}` — prüft Mitgliedschaft; wenn `sender_id != claimsUserID` und `claims.Role != "admin"` → HTTP 403; `UPDATE messages SET deleted_at=CURRENT_TIMESTAMP WHERE id=?`; idempotent (HTTP 204 auch wenn schon gelöscht)
- [x] 5.2 Route registrieren: `r.Delete("/chat/messages/{id}", chatH.DeleteMessage)` (Authenticated-Gruppe)

## 6. Backend — EditBroadcast

- [x] 6.1 Handler `EditBroadcast` implementieren: `PUT /api/chat/broadcasts/{id}` — liest `body`; prüft `sender_id = claimsUserID`; `UPDATE broadcasts SET body=?, edited_at=CURRENT_TIMESTAMP WHERE id=? AND sender_id=?`; HTTP 403 wenn nicht Sender, 204 bei Erfolg
- [x] 6.2 Route registrieren: `r.Put("/chat/broadcasts/{id}", chatH.EditBroadcast)` (Authenticated-Gruppe)
- [x] 6.3 `ListBroadcasts`-Query und Broadcast-Struct um `EditedAt` erweitern

## 7. Frontend — Message-Interface und Rendering

- [x] 7.1 `Message`-Interface in `ChatPage.tsx` um `replyToId`, `replyToBody`, `replyToSenderName`, `editedAt`, `deletedAt` erweitern
- [x] 7.2 `Broadcast`-Interface um `editedAt` erweitern
- [x] 7.3 Nachrichten-Bubble: Deleted-Placeholder rendern wenn `deletedAt` gesetzt (Trash2-Icon, „Nachricht gelöscht", kursiv, gedämpfte Farbe, kein Kontext-Menü)
- [x] 7.4 Nachrichten-Bubble: Reply-Quote-Block rendern wenn `replyToId` gesetzt (linke farbige Border `border-brand-yellow`, `replyToSenderName` fett, `replyToBody` gekürzt auf 60 Zeichen)
- [x] 7.5 Nachrichten-Bubble: `(bearbeitet)`-Indikator unterhalb des Zeitstempels rendern wenn `editedAt` gesetzt
- [x] 7.6 Broadcast-Detailansicht: `(bearbeitet)`-Indikator anzeigen wenn `editedAt` gesetzt

## 8. Frontend — Kontext-Menü (Rechtsklick)

- [x] 8.1 Lokale `MessageContextMenu`-Komponente erstellen: positioniertes `div` mit `position: fixed`, `left/top` aus `MouseEvent`, schließt sich bei Klick außerhalb oder Escape-Taste
- [x] 8.2 Menü-Einträge: „Antworten" (CornerUpLeft-Icon, immer sichtbar außer bei gelöschten Nachrichten), „Bearbeiten" (Pencil-Icon, nur eigene nicht-gelöschte Nachrichten), „Löschen" (Trash2-Icon, eigene oder Admin)
- [x] 8.3 `onContextMenu`-Handler auf jeder Nachrichten-Bubble registrieren; `event.preventDefault()` aufrufen; Menü-Position und ausgewählte Nachricht in State speichern

## 9. Frontend — Swipe-to-Reply (Mobile)

- [x] 9.1 `touchstart`/`touchmove`/`touchend`-Handler auf Nachrichten-Bubble-Wrapper: bei horizontalem Swipe > 60px nach rechts Reply-State mit der Nachricht setzen; `transform: translateX(${Math.min(delta, 60)}px)` während des Swipes; nach `touchend` Bubble per CSS-Transition zurücksnappen
- [x] 9.2 CornerUpLeft-Icon neben der Bubble anzeigen, das beim Swipe eingeblendet wird (Opacity-Transition)

## 10. Frontend — Reply-Leiste und Edit-Leiste

- [x] 10.1 Reply-Leiste: über dem Eingabefeld, zeigt CornerUpLeft-Icon, „Antwort auf {senderName}", gekürzte Vorschau, X-Button zum Schließen; nur sichtbar wenn `replyTo`-State gesetzt
- [x] 10.2 Edit-Leiste: über dem Eingabefeld, zeigt Pencil-Icon, „Nachricht bearbeiten", X-Button; nur sichtbar wenn `editingMessage`-State gesetzt; beim Öffnen Eingabefeld mit `msg.body` befüllen
- [x] 10.3 Senden-Button: wenn `editingMessage` gesetzt → `PUT /api/chat/messages/{id}` statt `POST`; danach Edit-State zurücksetzen und Nachrichten neu laden
- [x] 10.4 Reply und Edit schließen sich gegenseitig aus (jeweils anderer State auf null setzen beim Öffnen)

## 11. Frontend — Broadcast-Bearbeitung

- [x] 11.1 In der Broadcast-Detailansicht: Pencil-Button neben dem Trash-Button nur für `bc.isSent === true` rendern
- [x] 11.2 `BroadcastEditModal`-Komponente: Textarea mit aktuellem `body`, Speichern-Button ruft `PUT /api/chat/broadcasts/{id}` auf, schließt Modal, lädt Broadcasts neu
