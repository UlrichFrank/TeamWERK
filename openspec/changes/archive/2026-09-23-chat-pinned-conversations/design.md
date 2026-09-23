## Context

`GET /api/chat/conversations` (`internal/chat/handler.go:209-270`, `ListConversations`) liefert
heute alle Konversationen eines Users (`conversation_members.left_at IS NULL`) sortiert nach
`COALESCE(letzte Nachricht, conversations.created_at) DESC` — rein serverseitig, keine
persistierte Nutzer-Reihenfolge. Das Frontend (`web/src/pages/ChatPage.tsx:1513`) rendert die
Liste unverändert in Server-Reihenfolge, kein Client-Sort. Jede Zeile hat aktuell einen
sichtbaren Mülleimer-Button (`Zeile 1553-1559`, `deleteConversation(conv)`).

`conversation_members(conversation_id, user_id, joined_at, left_at)` ist bereits die
Pro-User-Pro-Konversation-Zeile (PK `(conversation_id, user_id)`) — der natürliche Ort für
Pin-Zustand, statt einer neuen Tabelle.

Chat nutzt **nicht** den globalen `h.hub.Broadcast` (siehe `internal/arch/broadcast_test.go`
Allowlist), sondern einen eigenen SSE-Kanal `/api/chat/events` mit
`h.hub.BroadcastToUser(userID, "chat:event-name")` (colon-delimited String, kein JSON) pro
Empfänger. Es gibt bereits Precedent für rein-eigene Events ohne Fan-out an andere Mitglieder,
z. B. `chat:conversation-read` (Zeile 1160). Das Frontend konsumiert über
`useChatEvents`/`useEventStream('/api/chat/events', ...)`.

Keine bestehende Drag&Drop-/Reorder-UI im Frontend (siehe Recherche zur Proposal) — die
Reihenfolge-Funktion ist die erste ihrer Art im Code.

## Goals / Non-Goals

**Goals:**
- Pin/Unpin einer Konversation, rein pro Nutzer, ohne Sichtbarkeit für andere Mitglieder.
- Manuelles Umsortieren der gepinnten Konversationen per Drag & Drop, persistiert und über
  eigene Geräte/Tabs synchron.
- Bestehendes Lösch-Verhalten (`DeleteConversation`) bleibt fachlich unverändert, wandert nur
  ins neue Kontextmenü.

**Non-Goals:**
- Keine Obergrenze für die Anzahl gepinnter Chats (YAGNI ohne konkreten Bedarf).
- Kein Pin-Status für Broadcasts (`tab === "broadcasts"`) — nur für normale Konversationen
  (`tab === "chats"`), wie im Proposal beschrieben.
- Keine automatische Aktivitäts-Sortierung *innerhalb* der gepinnten Gruppe (geklärt: rein
  manuelle Reihenfolge, sonst würde die Position bei jeder neuen Nachricht springen — genau das
  Problem, das Pinning lösen soll).
- Keine Migration/Bereinigung von Pin-Zustand beim Verlassen einer Konversation über die
  bestehende Lösch-/Verlassen-Logik hinaus (die Zeile in `conversation_members` verschwindet
  bzw. wird durch `left_at` gefiltert — Pin-Spalten laufen automatisch mit).

## Decisions

### D1: Pin-Zustand in `conversation_members`, nicht in einer neuen Tabelle

Neue Spalten `pinned_at DATETIME NULL` und `pin_order INTEGER NULL` auf
`conversation_members`. Beide `NULL` = nicht gepinnt (Default-Zustand, keine Migration von
Bestandsdaten nötig — alle Zeilen sind vor diesem Change ungepinnt). `pinned_at` wird nicht für
Sortierung gebraucht (das übernimmt `pin_order`), ist aber für „seit wann gepinnt"/Debugging
sowie als klarer Boolean-Ersatz (`pinned_at IS NOT NULL`) statt eines zusätzlichen
`is_pinned`-Flags sinnvoll — ein Flag *und* ein Sortierwert wären zwei Quellen der Wahrheit für
denselben Zustand. Alternative verworfen: eigene Tabelle `chat_pinned_conversations(user_id,
conversation_id, pin_order)` — hätte keine neue Information über `conversation_members` hinaus
und bräuchte einen zusätzlichen Join in `ListConversations`, ohne fachlichen Vorteil.

### D2: `pin_order` als lückenhafte Ganzzahl, keine dichte Neuindizierung nötig

Pin (`PUT .../pin`) hängt hinten an: `pin_order = COALESCE(MAX(pin_order) für diesen User, 0) +
1`. Reorder (`PUT /api/chat/conversations/pinned-order`) bekommt vom Client die **vollständige
Ziel-Reihenfolge** aller aktuell gepinnten Konversations-IDs und schreibt `pin_order = 1..N` in
genau dieser Reihenfolge, in einer Transaktion. Validierung: die übergebene ID-Menge MUSS exakt
der Menge der aktuell für den User gepinnten Konversationen entsprechen (nicht mehr, nicht
weniger) — sonst `409 pin_order_mismatch` (schützt gegen Race: ein zweiter Tab hat zwischen
Laden und Drag-Ende bereits gepinnt/gelöst). Kein `PATCH` mit Einzel-Verschiebung (z. B.
„verschiebe X nach Position 3") — Drag & Drop liefert client-seitig ohnehin die komplette neue
Reihenfolge, ein inkrementelles API wäre zusätzliche Komplexität ohne Nutzen.

### D3: Kein Broadcast an andere Mitglieder, aber `BroadcastToUser` an sich selbst

Pin ist ausschließlich für den handelnden User sichtbar — kein Fan-out an die übrigen
Konversations-Mitglieder nötig oder gewünscht (sie sehen den Pin-Zustand des anderen nicht).
Alle drei neuen Mutationen (`Pin`, `Unpin`, `ReorderPinned`) rufen trotzdem
`h.hub.BroadcastToUser(claims.UserID, "chat:pin-updated")`, damit ein zweites offenes Tab/Gerät
desselben Nutzers die Liste neu lädt (identisches Muster zu `chat:conversation-read`). Chat ist
in der `broadcastAllowlist` des Architektur-Gates bereits pauschal ausgenommen (eigener Kanal),
die drei neuen Routen brauchen dort keinen zusätzlichen Eintrag.

### D4: Drag & Drop mit `@dnd-kit` (neue Abhängigkeit)

Trotz fehlendem Vorbild im Code explizit vom Nutzer gewählt (Alternative „Rauf/Runter-Pfeile im
Menü" wurde verworfen). `@dnd-kit/core` + `@dnd-kit/sortable` (React-19-kompatible Version, kein
`react-beautiful-dnd` — unmaintained, keine React-19-Unterstützung) für die Sortier-Liste der
gepinnten Gruppe. `PointerSensor` deckt Maus **und** Touch ab (kein separater Mobile-Pfad
nötig); ein sichtbarer Drag-Handle (kein „ganze Zeile ziehen") verhindert Konflikt mit dem
Öffnen der Konversation per Klick/Tap. Reorder wird **optimistisch** im Frontend angewendet
(State sofort neu sortiert) und bei Fehlschlag der PUT-Anfrage zurückgerollt (Toast +
Reload der Liste) — kein Warten auf den Server-Roundtrip mitten in der Drag-Geste.

### D5: Lösch-Button wandert ins `ActionMenu`, Icon-Button entfällt

`ActionMenu` (`web/src/components/ActionMenu.tsx`) existiert bereits und passt ohne Änderung
(`actions: {label, onClick, variant}[]`). Neue Actions pro Zeile: `Anpinnen`/`Lösen` (Label
abhängig vom aktuellen Zustand) und `Löschen` (`variant: 'danger'`, ruft die unveränderte
`deleteConversation(conv)`). Der bisherige `Trash2`-Icon-Button (Zeile 1553-1559) wird entfernt.
Der Drag-Handle für gepinnte Zeilen ist ein **zusätzliches**, separates Icon (nicht Teil des
Menüs — Drag-Handles müssen direkt greifbar sein, nicht hinter einem Klick verborgen).

## Risks / Trade-offs

- **[Risk] `@dnd-kit` ist eine neue Laufzeit-Abhängigkeit** (Bundle-Größe, Wartungslast). →
  Bewusste Nutzer-Entscheidung gegen die einfachere Pfeil-Alternative; `@dnd-kit` ist aktiv
  gepflegt und deutlich kleiner/moderner als `react-beautiful-dnd`.
- **[Risk] Drag & Drop auf Touch-Geräten kann mit vertikalem Scrollen der Liste kollidieren.**
  → `@dnd-kit`s `PointerSensor` mit `activationConstraint` (kleine Distanz-Schwelle) plus
  dedizierter Drag-Handle (kein Drag über die ganze Zeile) reduziert Fehlauslösungen beim
  Scrollen; manueller Check auf einem echten Touch-Gerät vor Abschluss (siehe Empfehlung zu
  Chrome DevTools MCP in `CLAUDE.md` für Layout-/Touch-Verhalten, das Vitest/jsdom nicht sieht).
- **[Trade-off] `pin_order`-Lücken durch Unpin/Re-Pin** (keine dichte Neuindizierung). →
  Bewusst, weil nur die relative Ordnung zählt; verhindert unnötige Schreiblast bei jedem
  Unpin.
- **[Trade-off] Kein Limit für gepinnte Chats.** → Eine sehr lange gepinnte Gruppe ist ein
  UX-, kein Korrektheitsproblem; kann bei Bedarf als eigener Folge-Change ergänzt werden.

## Migration Plan

1. Neue Migration `066_chat_pinned_conversations.up.sql`/`.down.sql`: `conversation_members`
   um `pinned_at DATETIME NULL` und `pin_order INTEGER NULL` erweitern (SQLite erlaubt
   `ALTER TABLE ... ADD COLUMN` ohne CHECK/PK-Änderung — **kein** Tabellen-Rebuild nötig, anders
   als bei `duty_reminder_log` im letzten Change). Down: `ALTER TABLE conversation_members
   DROP COLUMN pin_order` / `... DROP COLUMN pinned_at` — im Projekt bereits mehrfach
   genutztes, von `modernc.org/sqlite` unterstütztes Muster (siehe z. B.
   `002_kinderaccount_login.down.sql`).
2. Backend: `ListConversations` liefert zusätzlich `pinned` (bool) und `pinOrder` (int, nur bei
   `pinned=true` relevant); Sortierung wird zweistufig: `pinned_at IS NOT NULL DESC,
   pin_order ASC` vor der bisherigen `COALESCE(...) DESC`-Klausel.
3. Neue Routen (Authenticated-Tier, wie die übrigen `/api/chat/conversations/*`-Routen):
   `PUT /api/chat/conversations/{id}/pin`, `DELETE /api/chat/conversations/{id}/pin`,
   `PUT /api/chat/conversations/pinned-order`. Alle drei: 403/404, wenn der aufrufende User
   kein aktives Mitglied (`left_at IS NULL`) der Konversation ist.
4. Objektrechte-Matrix (`internal/permissions/object_matrix_test.go`): Fixture für
   `PUT/DELETE .../{id}/pin` ergänzen (fremde Konversation → 403/404).
5. Frontend: `Conversation`-Typ, Listen-Rendering (zwei Gruppen), `ActionMenu`-Integration,
   Drag & Drop für die gepinnte Gruppe, `@dnd-kit`-Abhängigkeit installieren (`pnpm -C web add
   @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities`).
6. Kein Sonderfall für `make deploy-rollback` — additive Migration, alte Binary ignoriert die
   neuen Spalten einfach.
