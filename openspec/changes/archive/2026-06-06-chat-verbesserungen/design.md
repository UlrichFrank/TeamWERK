## Context

Die Chat-Funktion (Migration 025, `internal/chat/handler.go`, `ChatPage.tsx`) ist vollständig implementiert. Das Schema kennt:
- `conversations` / `conversation_members` (mit `left_at` für Gruppen-Verlassen)
- `messages` / `message_reads`
- `broadcasts` / `broadcast_reads` (mit `read_at`)

Drei Lücken werden geschlossen: ein Mobile-Bug, fehlende Lösch-Semantik, fehlende Nacheinladungs-Funktion.

## Goals / Non-Goals

**Goals:**
- Broadcast-Detailansicht auf Mobile korrekt öffnen (1-Zeilen-Bug-Fix)
- "Für mich löschen" für Gespräche und Broadcasts mit automatischer Daten-Bereinigung
- Gruppen-Ersteller kann Mitglieder nachträglich hinzufügen (auch re-adden nach Verlassen)

**Non-Goals:**
- Nachrichten einzeln löschen
- Nachbearbeiten von Nachrichten oder Broadcasts
- Löschen durch Nicht-Ersteller bei Gruppen
- Broadcast-Empfänger können keine Broadcasts für andere verbergen

## Decisions

### D1: Lösch-Semantik für Gespräche via `left_at`

`conversation_members.left_at` wird für Direct Chats als "ausgeblendet" genutzt — identisch zur Gruppen-Verlassen-Logik. Der `ListConversations`-Query filtert bereits `WHERE cm.left_at IS NULL`, sodass keine Query-Änderung nötig ist.

**Alternative:** Neue Spalte `hidden_at`. Abgelehnt — `left_at` erfüllt exakt denselben Zweck, keine neue Spalte nötig.

**Cleanup-Trigger** (nach jedem `left_at`-Update):
```sql
SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND left_at IS NULL
```
Wenn 0 → `DELETE FROM conversations WHERE id = ?` (Foreign Keys + CASCADE löschen messages, message_reads, conversation_members).

### D2: Lösch-Semantik für Broadcasts via neue Spalte `hidden_at`

`broadcast_reads` erhält `hidden_at DATETIME`. Der `ListBroadcasts`-Query erhält `AND br.hidden_at IS NULL`.

**Alternative:** Zeile löschen statt Flag. Abgelehnt — die Zeile wird für den Cleanup-Check benötigt.

**Cleanup-Trigger** (nach jedem `hidden_at`-Update):
```sql
SELECT COUNT(*) FROM broadcast_reads WHERE broadcast_id = ? AND hidden_at IS NULL
```
Wenn 0 → `DELETE FROM broadcasts WHERE id = ?` (CASCADE auf broadcast_reads).

### D3: Mitglieder hinzufügen via UPSERT auf `conversation_members`

Statt INSERT OR IGNORE: UPDATE SET left_at = NULL falls der User die Gruppe bereits verlassen hatte, sonst INSERT. Das erlaubt sauberes Re-Adden.

```sql
-- Wenn bereits vorhanden (evtl. mit left_at):
UPDATE conversation_members SET left_at = NULL WHERE conversation_id = ? AND user_id = ?
-- Wenn noch nicht vorhanden:
INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)
```

SSE-Event `chat:new-message:{convId}` an den neu hinzugefügten User → Gruppe erscheint sofort in seiner Liste.

### D4: Frontend-UI für Löschen

`ActionMenu` (⋮-Dropdown, bereits als Komponente vorhanden) in jedem Listen-Item, mit einem "Löschen"-Eintrag (Trash2-Icon). Kein inline-Icon um das Layout nicht zu überladen. `window.confirm()` vor dem API-Call.

Bei aktivem Chat-Panel: Trash2 im Header (neben dem LogOut-Icon bei Gruppen).

### D5: Frontend-UI für Mitglieder hinzufügen

`UserPlus`-Icon im Gruppen-Header, nur sichtbar wenn `activeConv.createdBy === user?.id`. Öffnet ein Modal analog zu `NewConversationModal` — Suchbox + User-Liste, Einzel-Auswahl, "Hinzufügen"-Button.

## Risks / Trade-offs

- **Cleanup bei Gruppen**: Wenn alle Mitglieder verlassen haben ist die Gruppe ohnehin tot. Kein Datenverlust-Risiko für aktive User.
- **Race Condition beim Cleanup**: Zwei User löschen gleichzeitig → beide prüfen COUNT = 0 → doppeltes DELETE. SQLite serialisiert Writes, das DELETE einer nicht-existenten Row ist idempotent. Kein Problem.
- **Re-Add nach Verlassen**: User erhält wieder Zugriff auf den gesamten Nachrichtenverlauf der Gruppe. Akzeptiertes Trade-off — kein Bedarf für "Nachrichten ab Beitrittsdatum" in diesem Kontext.

## Migration Plan

1. Migration `026_chat_hidden.up.sql`: `ALTER TABLE broadcast_reads ADD COLUMN hidden_at DATETIME`
2. Bestehende `broadcast_reads`-Rows sind kompatibel (hidden_at = NULL = sichtbar)
3. Kein Rollback-Risiko: additive Schemaänderung

## Open Questions

— keine —
