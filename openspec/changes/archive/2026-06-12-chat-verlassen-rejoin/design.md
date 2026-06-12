## Context

Der Chat-Bereich (`internal/chat/handler.go`) unterscheidet zwei Ausstieg-Pfade:

- `LeaveConversation` (`DELETE /members/me`) — nur Gruppen; schreibt bereits Systemnachricht + SSE
- `DeleteConversation` (`DELETE /conversations/{id}`) — Direkt- und Gruppen-Chats; setzt nur `left_at`, keine Systemnachricht, kein SSE an Gegenseite

`createDirect` sucht per JOIN nur Threads wo **beide** Parteien `left_at IS NULL` haben. Hat A gelöscht (left_at gesetzt), findet die Query nichts → neuer Thread → Duplikat.

`SendMessage` broadcastet nur an `activeMembers` (left_at IS NULL). Hat A gelöscht, bekommt A kein SSE und taucht in keiner Liste auf.

## Goals / Non-Goals

**Goals:**
- Direkt-Chat-Verlassen erzeugt Systemnachricht und benachrichtigt B via SSE
- `createDirect` setzt A per Re-join fort wenn B noch aktiv ist
- Eingehende Nachricht von B stellt A automatisch wieder her
- System-Nachrichten-Text kommt aus dem DB-Body (generisch)

**Non-Goals:**
- Kein neues DB-Schema (keine Migration)
- `LeaveConversation` (Gruppen) bleibt unverändert
- Kein "delete for both sides"-Feature
- Keine Änderungen an Auth, Rollen oder Push-Notifications

## Decisions

### D1 — Wann wird eine Direkt-Conversation dauerhaft gelöscht?

**Entscheidung:** Erst wenn **beide** Parteien `left_at IS NOT NULL` haben wird die Conversation physisch gelöscht.

**Rationale:** Solange B noch aktiv ist muss B die Systemnachricht und den Chatverlauf sehen können. Ein vorzeitiges DELETE würde die Nachricht vernichten. Löscht danach auch B, gibt es nichts mehr zu erhalten — dann erfolgt das echte DELETE.

**Alternative:** Conversation immer aufbewahren (auch wenn beide weg). Verworfen: führt zu wachsendem DB-Mülleimer ohne Mehrwert, Re-join ist nach beidseitigem Löschen explizit **nicht** gewollt.

### D2 — Re-join-Logik in createDirect

**Entscheidung:** Die Suche nach einer bestehenden Direkt-Conversation lockert den Constraint auf A: `m1.left_at` darf beliebig sein, `m2.left_at IS NULL` (B muss noch aktiv sein). Wird eine solche Conversation gefunden, wird A's `left_at = NULL` gesetzt und die bestehende Conversation zurückgegeben.

```sql
-- Neu: A darf left_at haben, B muss aktiv sein
SELECT c.id FROM conversations c
JOIN conversation_members m1 ON m1.conversation_id = c.id AND m1.user_id = ? -- A (any left_at)
JOIN conversation_members m2 ON m2.conversation_id = c.id AND m2.user_id = ? AND m2.left_at IS NULL -- B active
WHERE c.type = 'direct' LIMIT 1
```

**Rationale:** Kein Duplikat, History bleibt erhalten, B bekommt SSE dass A zurück ist.

**Alternative:** Immer neuen Thread anlegen + SSE. Verworfen: Duplikate, History-Bruch.

### D3 — Automatischer Re-join bei eingehender Nachricht

**Entscheidung:** In `SendMessage` wird **vor dem SSE-Broadcast** geprüft ob es ein Direkt-Chat ist und ob ein Mitglied `left_at IS NOT NULL` hat. Falls ja, wird `left_at = NULL` gesetzt — danach findet `activeMembers` alle Parteien und das SSE erreicht A.

**Rationale:** Minimale Änderung, kein neuer Endpunkt. Das Verhalten ist intuitiv: B schreibt → A sieht es, egal ob A vorher gelöscht hatte.

### D4 — Generisches Frontend-Rendering für Systemnachrichten

**Entscheidung:** Der hardcoded String `hat die Gruppe verlassen` wird durch `{msg.body}` ersetzt. Der Body kommt aus der DB und ist je nach Kontext `hat die Gruppe verlassen` oder `hat diesen Chat verlassen`.

**Rationale:** Eine einzige Rendering-Regel deckt alle zukünftigen Systemnachrichten ab. Kein Switch-Statement, kein weiterer hardcoded Text.

## Risks / Trade-offs

- **Re-join ohne Einwilligung** — A löscht den Chat bewusst, B schreibt einfach, A ist wieder drin. → Akzeptiert: Das ist erwünschtes Verhalten (B kann A immer anschreiben solange B aktiv ist).
- **Systemnachricht wird mitgelöscht** — Wenn A löscht und B sofort danach auch, ist die Conversation weg inkl. Systemnachricht. B hat sie möglicherweise nie gesehen. → Akzeptiert: Race Condition bei simultaner Löschung ist extrem selten und kein sicherheitskritischer Fall.
- **Kein SSE bei createDirect wenn B's left_at IS NULL** — B bekommt nur SSE wenn re-join stattfindet oder neuer Thread entsteht. Öffnet A einen bereits existierenden Thread (A und B beide aktiv) gibt es kein SSE — B war sowieso schon aktiv. → Kein Problem.
