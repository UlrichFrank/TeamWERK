## Why

Aktive Nutzer mit vielen Chats müssen ihre wichtigsten Gespräche (z. B. den Trainer-Chat der
eigenen Mannschaft) immer wieder aus einer nach letzter Aktivität sortierten Liste heraussuchen.
Es gibt keine Möglichkeit, einen Chat dauerhaft oben zu fixieren — die Liste ordnet sich bei
jeder neuen Nachricht in einem beliebigen anderen Gespräch komplett neu.

## What Changes

- Nutzer können einzelne Konversationen für sich selbst **anpinnen** (an den Anfang der Liste
  fixieren) und wieder lösen.
- Gepinnte Konversationen erscheinen als eigene Gruppe oben in der Chat-Liste, sortiert nach
  einer **manuell per Drag & Drop änderbaren Reihenfolge** — nicht nach letzter Nachricht.
  Ungepinnte Konversationen bleiben darunter wie bisher nach letzter Nachricht sortiert.
- Der bisherige, direkt in der Listenzeile sichtbare Lösch-Button (Mülleimer-Icon) wird durch
  ein Kontextmenü (bestehende `ActionMenu`-Komponente) ersetzt, das **Anpinnen/Lösen** und
  **Löschen** zusammenfasst. **BREAKING** (UI): der Lösch-Button ist nicht mehr direkt sichtbar,
  sondern liegt hinter dem neuen „…"-Menü.
- Pin-Status ist rein pro Nutzer und pro Gerät synchron (eigener SSE-Kanal), nicht für andere
  Konversations-Mitglieder sichtbar.

## Capabilities

### New Capabilities
- `chat-pinned-conversations`: Anpinnen/Lösen einzelner Konversationen pro Nutzer und
  manuelles Umsortieren der gepinnten Konversationen per Drag & Drop.

### Modified Capabilities
(keine — die bestehende Konversationsliste bekommt eine zusätzliche Sortier-Dimension, aber
keine Anforderung aus einer bestehenden Spec ändert sich; die Listen-Route selbst hat noch
keine eigene Spec unter `openspec/specs/`.)

## Impact

- **Backend:** `internal/chat/handler.go` (`ListConversations`, drei neue Routen: Pin/Unpin/
  Reorder), neue Migration (`conversation_members` um `pinned_at`/`pin_order` erweitern),
  `internal/app/router.go` (drei neue Routen im Authenticated-Tier), Objektrechte-Matrix
  (`internal/permissions/object_matrix_test.go`) braucht Fixtures für die neuen `{id}`-Routen.
- **Frontend:** `web/src/pages/ChatPage.tsx` (Conversation-Typ um `pinned`/`pinOrder` erweitern,
  Listen-Rendering in zwei Gruppen aufteilen, Trash-Button durch `ActionMenu` ersetzen,
  Drag & Drop für die gepinnte Gruppe), neue Runtime-Abhängigkeit `@dnd-kit/core` +
  `@dnd-kit/sortable` (kein bestehendes Drag&Drop-Muster im Frontend vorhanden).
- **Kein** Broadcast an andere Konversations-Mitglieder (rein privater Zustand); die Mutation
  broadcastet nur an den eigenen User (`h.hub.BroadcastToUser`, analog zu
  `chat:conversation-read`) für Sync über mehrere offene Tabs/Geräte.
