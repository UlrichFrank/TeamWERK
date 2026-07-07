## Why

Der Chat verschenkt ein bereits geliefertes Delta: Das SSE-Event trägt die Nachrichten-ID (`chat:new-message:123`), aber `ChatPage.tsx` rief `loadMessages(convId)` — einen **Voll-Reload der ganzen Konversation** (bis 100 Nachrichten) — statt die eine bekannte Nachricht anzuhängen.

Dieser Change führt **id-basiertes inkrementelles Nachladen** für den Chat ein: `?after=`/`?before=` liefern nur neuere bzw. ältere Nachrichten; das SSE-Event dient nur noch als „jetzt nachfragen"-Trigger. Weil Nachrichten append-only sind, genügt die Nachrichten-**id** als Cursor — **kein** `updated_at` und **keine** Migration nötig.

**Abgrenzung:** Ursprünglich umfasste dieser Change auch die pull-basierte Delta-Synchronisation der schweren Listen (`?since=`-Cursor auf games/duty-slots/training-sessions/kader, `updated_at`-Nachrüstung, Tombstones). Diese noch offene Substanz wurde in ein eigenes Proposal **`incremental-list-sync`** herausgelöst. `incremental-sync` beschränkt sich damit auf die **fertige** Chat-Phase.

## What Changes

- **Chat inkrementell:** `GET /api/chat/conversations/{id}/messages?after=<msgId>` liefert nur neuere Nachrichten (append-only, id-basiert); `?before=<msgId>` liefert ältere für Verlaufs-Scrollen. `ChatPage` hängt bei `chat:new-message:<id>` gezielt an, statt die Konversation neu zu laden. Ohne Parameter unverändertes Verhalten.

## Capabilities

### Added Capabilities

- `incremental-sync`: id-basiertes inkrementelles Nachladen von Chat-Nachrichten (`?after=`/`?before=`) mit gezieltem Anhängen statt Voll-Reload.

## Test-Anforderungen

| Route | Testname | Erwartung / Invariante |
|---|---|---|
| `GET /api/chat/.../messages?after=` | `TestMessagesAfter_ReturnsOnlyNewer` | Nur Nachrichten mit `id > after`; leere Liste wenn nichts Neues. |
| `GET /api/chat/.../messages?before=` | `TestMessagesBefore_ReturnsOlderPage` | Liefert die Seite älterer Nachrichten vor `before` (Verlaufs-Scroll). |

**Garantierte Invariante:** Die Cursor-Erweiterung ändert **nie** die Sichtbarkeits-/Autorisierungsregeln. `?after=`/`?before=` liefern genau die Teilmenge der ohnehin sichtbaren Nachrichten derselben Konversation.

## Impact

- **Backend:** `internal/chat` — `ListMessages` um `?after=`/`?before=` erweitert (kein Schema, kein `updated_at`).
- **Frontend:** `web/src/pages/ChatPage.tsx` hängt bei `chat:new-message:<id>` das Delta an; Verlaufs-Scroll per `?before=`.
- **Abgrenzung:** Delta-Sync der schweren Listen siehe `incremental-list-sync`.
