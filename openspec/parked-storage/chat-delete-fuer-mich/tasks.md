## 1. Migration

- [ ] 1.1 `internal/db/migrations/061_message_hides.up.sql`: Tabelle `message_hides` (`message_id`, `user_id`, `hidden_at`, PK `(message_id, user_id)`, FK `ON DELETE CASCADE` auf beide) + Index `idx_message_hides_user`
- [ ] 1.2 `internal/db/migrations/061_message_hides.down.sql`: `DROP TABLE message_hides`

## 2. Backend: neue Route „Für mich löschen"

- [ ] 2.1 `internal/chat/handler.go`: neuer Handler `HideMessage` — `chi.URLParam(r, "id")`, Existenz + `deleted_at IS NULL` prüfen (sonst 404), `h.isActiveMember(r, convID, claims.UserID)` prüfen (sonst 403), `INSERT OR IGNORE INTO message_hides (message_id, user_id) VALUES (?, ?)`, `h.hub.BroadcastToUser(claims.UserID, fmt.Sprintf("chat:message-hidden:%d:%d", convID, msgID))`, `204`
- [ ] 2.2 `internal/app/router.go`: `r.Post("/api/chat/messages/{id}/hide", h.Chat.HideMessage)` im Authenticated-Tier neben den bestehenden `chat/messages/{id}`-Routen

## 3. Backend: Sichtbarkeitsfilter an den vier Lesepfaden

- [ ] 3.1 `messageSelect` (`handler.go`): `NOT EXISTS (SELECT 1 FROM message_hides mh WHERE mh.message_id = m.id AND mh.user_id = ?)` in die `WHERE`-Klausel; neuen Bind-Parameter in allen drei `ListMessages`-Varianten (voll/`after`/`before`) ergänzen, Reihenfolge der `?`-Platzhalter prüfen
- [ ] 3.2 Reply-Vorschau in `messageSelect`: `CASE`-Zweig `WHEN rm.id IN (SELECT message_id FROM message_hides WHERE user_id = ?) THEN '[Nachricht gelöscht]'` vor `ELSE rm.body` ergänzen (weiterer Bind-Parameter)
- [ ] 3.3 `internal/chat/search.go` `Search`: `AND NOT EXISTS (… message_hides …)` neben dem bestehenden `deleted_at IS NULL`-Filter für Nachrichten-Treffer
- [ ] 3.4 `internal/chat/unread.go` `ComputeUnreadForUser`: `AND NOT EXISTS (… message_hides …)` in der Nachrichten-Teilquery
- [ ] 3.5 Konversations-`LastMessage`-Query (Listen-Query + `getConversationDetail`): pro anfragendem Nutzer die jüngste **nicht ausgeblendete** Nachricht wählen statt pauschal die jüngste der Konversation

## 4. Frontend: Bestätigungsdialog

- [ ] 4.1 Neue Komponente (z. B. `DeleteMessageConfirmModal`) im Projekt-Modal-Stil (`bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`): Props `message`, `canDeleteForEveryone`, `onHideForMe`, `onDeleteForEveryone`, `onCancel`
- [ ] 4.2 `canDelete` in `ChatPage.tsx` zu einer reinen Sichtbarkeits-Prüfung machen (jede nicht gelöschte Nachricht → „Löschen" im Kontextmenü/Overlay anzeigen); bestehende Absender/Admin-Logik als neue Funktion `canDeleteForEveryone` beibehalten
- [ ] 4.3 Desktop-Kontextmenü: Klick auf „Löschen" öffnet den Bestätigungsdialog statt `deleteMsg` direkt aufzurufen
- [ ] 4.4 Mobile `MobileMessageActionOverlay`: `onDelete` öffnet ebenfalls den Bestätigungsdialog (Overlay schließt sich, Dialog öffnet sich)
- [ ] 4.5 `deleteMsg` durch zwei Aktionen ersetzen: `hideMessageForMe` (`POST /chat/messages/{id}/hide`) und `deleteMessageForEveryone` (bestehendes `DELETE /chat/messages/{id}`, unverändert)
- [ ] 4.6 `useLiveUpdates`/SSE-Handler um `chat:message-hidden:<convId>:<msgId>` ergänzen → aktive Konversation neu laden, wenn Event für den eigenen User ankommt

## 5. Tests

- [ ] 5.1 `internal/chat/*_test.go`: Happy-Path `POST /chat/messages/{id}/hide` (aktives Mitglied, nicht Absender) → 204, `message_hides`-Zeile vorhanden
- [ ] 5.2 Fehlerfall: kein aktives Mitglied → 403; bereits für alle gelöschte Nachricht → 404; erneutes Ausblenden → 204 idempotent
- [ ] 5.3 `ListMessages`: ausgeblendete Nachricht fehlt für den ausblendenden Nutzer, ist aber für ein anderes Mitglied weiterhin vollständig sichtbar
- [ ] 5.4 Reply-Vorschau zeigt für den ausblendenden Nutzer „[Nachricht gelöscht]" bei einem Zitat der ausgeblendeten Nachricht
- [ ] 5.5 `Search`: ausgeblendete Nachricht taucht in den Suchtreffern des ausblendenden Nutzers nicht mehr auf
- [ ] 5.6 `ComputeUnreadForUser`: ausgeblendete, vorher ungelesene Nachricht zählt nicht mehr im Badge
- [ ] 5.7 Konversationsliste: `LastMessage` überspringt für den ausblendenden Nutzer eine ausgeblendete jüngste Nachricht zugunsten der nächstälteren sichtbaren
- [ ] 5.8 Bestehende `DeleteMessage`-Tests (403 fremde Nachricht, 204 Admin, Idempotenz) bleiben grün — Semantik unverändert

## 6. Abschluss

- [ ] 6.1 `make test` / `make lint` / `pnpm -C web build` grün (inkl. Architektur- und Broadcast-Gate)
- [ ] 6.2 Manueller Check in Chrome (Desktop-Breite und Mobile-Breite `sm:`-Breakpoint): Dialog-Optionen korrekt je nach eigener/fremder Nachricht und Rolle, Multi-Device-Sync des Ausblendens
- [ ] 6.3 `openspec validate --strict` für diesen Change
