## 1. Migration

- [x] 1.1 `internal/db/migrations/066_chat_pinned_conversations.up.sql`: `ALTER TABLE
  conversation_members ADD COLUMN pinned_at DATETIME`, `ADD COLUMN pin_order INTEGER`
  (nächste freie Migrationsnummer vor dem Anlegen mit `ls internal/db/migrations/ | tail`
  verifizieren).
- [x] 1.2 `066_chat_pinned_conversations.down.sql`: `ALTER TABLE conversation_members DROP
  COLUMN pin_order`, `DROP COLUMN pinned_at` (Muster wie `002_kinderaccount_login.down.sql`).

## 2. Backend — Pin/Unpin/Reorder-Routen

- [x] 2.1 In `internal/chat/handler.go`: `Pin(w, r)` (`PUT /api/chat/conversations/{id}/pin`) —
  Gate via `h.isActiveMember(r, convID, claims.UserID)` (403 sonst, analog `LeaveConversation`);
  setzt `pinned_at = CURRENT_TIMESTAMP`, `pin_order = COALESCE((SELECT MAX(pin_order) FROM
  conversation_members WHERE user_id = ? AND pinned_at IS NOT NULL), 0) + 1` für
  `(conversation_id, user_id)`. Idempotent (erneutes Pinnen einer bereits gepinnten Konversation
  ändert `pin_order` nicht — kein Sprung ans Ende).
- [x] 2.2 `Unpin(w, r)` (`DELETE /api/chat/conversations/{id}/pin`) — gleiches Gate; setzt
  `pinned_at = NULL, pin_order = NULL`.
- [x] 2.3 `ReorderPinned(w, r)` (`PUT /api/chat/conversations/pinned-order`, Body `{"order":
  [id, id, ...]}`) — lädt die aktuell gepinnten `conversation_id`s des Users, vergleicht als Menge
  mit dem Body (409 `pin_order_mismatch` bei Abweichung), schreibt sonst in einer Transaktion
  `pin_order = 1..N` in Body-Reihenfolge.
- [x] 2.4 Alle drei Routen rufen `h.hub.BroadcastToUser(claims.UserID, "chat:pin-updated")`
  (kein Fan-out an andere Mitglieder — Pin ist rein privat).
- [x] 2.5 `internal/app/router.go`: drei neue Routen im Authenticated-Tier neben den
  bestehenden `/api/chat/conversations/*`-Routen registrieren.

## 3. Backend — Liste sortiert gepinnte Konversationen zuerst

- [x] 3.1 `ListConversations`: `Conversation`-Struct um `Pinned bool` und `PinOrder *int`
  (`json:"pinned"`/`"pinOrder"`) erweitern.
- [x] 3.2 `ORDER BY` zweistufig: `cm.pinned_at IS NOT NULL DESC, cm.pin_order ASC,
  COALESCE(letzte Nachricht, conversations.created_at) DESC` (dritte Klausel nur für die
  ungepinnte Gruppe wirksam, da `pin_order` dort NULL ist).

## 4. Objektrechte

- [x] 4.1 `internal/permissions/object_matrix_test.go`: Fixtures für `PUT
  /api/chat/conversations/{id}/pin` und `DELETE /api/chat/conversations/{id}/pin` ergänzen
  (`objFixture{params: convFixture}`, analog Zeile 259/260). `PUT
  /api/chat/conversations/pinned-order` hat keinen `{id}`-Pfadparameter (operiert auf der
  eigenen Pin-Menge des aufrufenden Users) und braucht deshalb keine Objektrechte-Fixture —
  eigene Tests in `internal/chat` decken den Mismatch-Fall ab (Task 6.3).

## 5. Frontend

- [x] 5.1 `pnpm -C web add @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities` (aktuelle,
  React-19-kompatible Version pinnen).
- [x] 5.2 `web/src/pages/ChatPage.tsx`: `Conversation`-Interface um `pinned: boolean` und
  `pinOrder: number | null` erweitern.
- [x] 5.3 Listen-Rendering (Zeile ~1513) in zwei Gruppen aufteilen: gepinnte Konversationen
  (mit `DndContext`/`SortableContext` aus `@dnd-kit`, Drag-Handle-Icon, sichtbares Pin-Icon)
  oben, unveränderte Liste darunter.
- [x] 5.4 Trash-Icon-Button (Zeile 1553-1559) durch `ActionMenu` (`web/src/components/
  ActionMenu.tsx`) ersetzen: Actions `Anpinnen`/`Lösen` (Label je nach `conv.pinned`) und
  `Löschen` (`variant: 'danger'`, ruft unverändert `deleteConversation(conv)`).
  `brand-*`-Tokens/lucide-Icons beachten (`Pin`/`PinOff` aus `lucide-react`).
  Icon-only-Buttons (Drag-Handle) brauchen `aria-label`.
- [x] 5.5 Drag-Ende-Handler: optimistisches Reordering im State, `PUT
  /api/chat/conversations/pinned-order` senden; bei Fehler (z. B. 409) Liste per Re-Fetch aus
  `GET /api/chat/conversations` zurücksetzen + Fehler-Toast/Alert (bestehendes Alert-Fehler-
  Muster aus `docs/agent/05-frontend.md` verwenden).
- [x] 5.6 `chat:pin-updated`-Event im bestehenden `useChatEvents`/`useEventStream('/api/chat/
  events', ...)`-Hook behandeln: Konversationsliste neu laden (Sync über eigene Tabs/Geräte).

## 6. Tests

- [x] 6.1 Happy Path: `PUT .../{id}/pin` durch aktives Mitglied → 200/204, Konversation
  erscheint danach in `GET /api/chat/conversations` mit `pinned=true` vor allen ungepinnten.
- [x] 6.2 Fehlerfall: `PUT .../{id}/pin` durch Nicht-Mitglied → 403 (Objektrechte-Matrix,
  Task 4.1).
- [x] 6.3 `ReorderPinned`: Happy Path (drei gepinnte Konversationen in neue Reihenfolge
  bringen, `GET` liefert sie danach in dieser Reihenfolge) und Fehlerfall (ID-Menge weicht ab
  → 409, keine Änderung an `pin_order`).
- [x] 6.4 Neu gepinnte Konversation hängt ans Ende der bestehenden gepinnten Reihenfolge an
  (nicht an den Anfang).
- [x] 6.5 Erneutes Pinnen einer bereits gepinnten Konversation ändert `pin_order` nicht
  (Idempotenz, kein Sprung ans Ende).
- [x] 6.6 Pin ist nicht für andere Mitglieder sichtbar: User A pinnt eine gemeinsame
  Konversation, `GET /api/chat/conversations` für User B zeigt unverändert `pinned=false` und
  unveränderte Position.
- [x] 6.7 Unpin setzt `pinned_at`/`pin_order` zurück auf NULL, Konversation fällt zurück in die
  aktivitätssortierte, ungepinnte Gruppe.
- [x] 6.8 Bestehendes Lösch-Verhalten (`DeleteConversation`) bleibt unverändert funktionsfähig
  (Regressionstest — nur die UI-Aufrufstelle ändert sich, nicht der Handler).

## 7. Verifikation

- [x] 7.1 `make test` (Backend + Vitest), `golangci-lint`, `pnpm -C web build`, `openspec
  validate --strict` vor Abschluss grün.
- [x] 7.2 Manueller Check des Drag & Drop-Verhaltens auf einem echten Touch-Gerät oder via
  Chrome DevTools MCP (Touch-Emulation) — Vitest/jsdom sieht Drag-Geste vs. Scroll-Konflikt
  strukturell nicht (siehe design.md Risiken).
