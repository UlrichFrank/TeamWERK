## Context

Siehe `proposal.md` – Why. Relevanter Bestand:

- `DELETE /api/chat/messages/{id}` (`internal/chat/handler.go` `DeleteMessage`) setzt `messages.deleted_at`, sichtbar für **alle** Konversationsmitglieder als Placeholder „Nachricht gelöscht". Nur Absender oder `admin`. Kein Zeitlimit (Präzedenzfall: `chat-message-edit` hat ebenfalls keins).
- Frontend `deleteMsg` (`ChatPage.tsx:1303`) ruft diese Route **ohne jede Rückfrage** auf, sowohl aus dem Desktop-Kontextmenü als auch aus `MobileMessageActionOverlay`.
- Exaktes Präzedenzmuster für „nur für mich ausblenden" existiert bereits für Mitteilungen (`broadcasts`): `broadcast_reads.hidden_at` + `DeleteBroadcast` (`UPDATE broadcast_reads SET hidden_at = … WHERE broadcast_id=? AND user_id=?`), inkl. Garbage-Collection der Mitteilung, wenn `hidden_at` bei allen Empfängern gesetzt ist. `broadcast_reads` wird dafür bei Versand für **jeden** Empfänger vorab angelegt (`INSERT OR IGNORE`), weil eine Mitteilung eine feste, zum Sendezeitpunkt bekannte Empfängerliste hat.
- Nachrichten haben dieses Muster nicht: Konversationen sind langlebig, Mitgliederzahl kann wachsen (Gruppen), Nachrichtenvolumen ist um Größenordnungen höher als Mitteilungen. Ein Vorab-`INSERT` pro Nachricht × aktives Mitglied wäre unnötiger Schreib-Overhead auf einem 1-GB-VPS ohne fachlichen Nutzen — „ausgeblendet" ist ein seltener, expliziter Nutzer-Akt, kein Default-Zustand.

## Goals / Non-Goals

**Goals:**
- Löschen einer Chat-Nachricht (Desktop + Mobile) erfordert eine explizite Bestätigung im Projekt-Modal-Stil.
- Jedes aktive Konversationsmitglied kann jede sichtbare Nachricht privat aus der eigenen Ansicht entfernen („Für mich löschen"), ohne die Nachricht für andere zu verändern.
- „Für alle löschen" (bestehendes `deleted_at`-Verhalten) bleibt fachlich identisch, nur mit Bestätigung davor.
- Konsistenz über alle Stellen, die Nachrichteninhalte für den anfragenden Nutzer ausliefern: Liste, Suche, Ungelesen-Zähler/Badge, Konversations-Vorschau.

**Non-Goals:**
- Kein Zeitlimit für „Für alle löschen" (Konsistenz mit `chat-message-edit`; eine Frist wäre eine eigene, unabhängige Entscheidung und ist hier nicht gefordert).
- Kein Undo/„Rückgängig"-Toast nach dem Löschen — die Bestätigung selbst ist die Sicherheitsmaßnahme gegen Fehlklicks, kein zweites Sicherheitsnetz.
- Keine Änderung an `EditMessage`, Umfragen, Reaktionen oder Read-Receipts.
- Kein GC/Löschen der `messages`-Zeile, wenn alle aktiven Mitglieder sie „für mich" ausgeblendet haben (anders als bei Broadcasts) — siehe Entscheidung unten.

## Decisions

### 1. Neue Tabelle `message_hides` statt Wiederverwendung von `message_reads` oder `broadcast_reads`-Musters

`message_reads` (Migration 001, PK `message_id,user_id`) existiert für Lesebestätigungen und wird nur lazy bei `MarkRead` befüllt — nicht für jede Nachricht/Nutzer-Kombination. Eine `hidden_at`-Spalte dort zu überladen würde zwei fachlich unabhängige Konzepte (Read-Receipt vs. private Sichtbarkeit) in einer Tabelle vermischen, deren Lese-Pfade (`GET /api/chat/messages/{id}/reads`, Absender-only) bewusst getrennt bleiben sollen (`chat-read-receipts`-Spec).

Neue Migration `061_message_hides`:
```sql
CREATE TABLE message_hides (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hidden_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, user_id)
);
CREATE INDEX idx_message_hides_user ON message_hides(user_id);
```
Kein Vorab-`INSERT` bei Nachrichtenerstellung (anders als `broadcast_reads`) — die Zeile entsteht ausschließlich, wenn ein Nutzer tatsächlich „Für mich löschen" wählt (`INSERT OR IGNORE`). Alle lesenden Queries filtern per `LEFT JOIN … WHERE mh.message_id IS NULL` bzw. `NOT EXISTS`, kein Inner-Join-Precondition wie bei `broadcast_reads`.

### 2. Neue Route statt Erweiterung von `DELETE /api/chat/messages/{id}`

`POST /api/chat/messages/{id}/hide` (Auth-Tier „Authenticated", jedes aktive Konversationsmitglied — kein Absender-/Admin-Gate) statt eines Body-Parameters auf der bestehenden `DELETE`-Route.

**Warum nicht ein `{scope: "me"|"everyone"}`-Body auf `DELETE`:** `DELETE`-Requests mit Body sind in Axios umständlicher (`api.delete(url, {data})`) und die bestehende `chat-message-delete`-Spec/Tests für „Für alle löschen" bleiben unangetastet, statt sie um eine Verzweigung zu erweitern. Die generische REST-Konvention des Projekts (`/api/{resource}/{id}/{action}`, siehe `docs/agent/04-api-db.md`) sieht für zusätzliche Aktionen ohnehin eigene Sub-Routen vor (Beispiel: `/poll/vote`, `/poll/close`).

**Warum nicht `DELETE` wie bei `DeleteBroadcast`:** `DeleteBroadcast` überlädt `DELETE /api/chat/broadcasts/{id}` historisch mit Hide-Semantik. Für Nachrichten existiert `DELETE /api/chat/messages/{id}` aber schon mit einer anderen, dokumentierten Bedeutung („für alle löschen", nur Absender/Admin) — die beizubehalten ist wichtiger als Konsistenz mit dem Broadcast-Endpoint-Namen.

Response: `204 No Content`. Idempotent (`INSERT OR IGNORE` — erneutes Ausblenden ist ein No-Op, kein Fehler), analog zum bestehenden idempotenten `DeleteMessage`.

Berechtigung: `h.isActiveMember(r, convID, claims.UserID)` (bestehender Helper, siehe `MarkRead`). Keine Einschränkung auf Absender/Admin — jedes Mitglied darf jede sichtbare Nachricht für sich ausblenden, auch die eigene (z. B. wer versehentlich „Für alle" nicht wollte, sondern nur die eigene Ansicht aufräumen möchte). Bereits per `deleted_at` global gelöschte Nachrichten liefern `404` (nichts auszublenden, Frontend zeigt für sie ohnehin kein Kontextmenü).

### 3. Sichtbarkeitsfilter zieht sich durch vier Lesepfade

Für den anfragenden Nutzer (`claims.UserID`) ausgeblendete Nachrichten müssen konsistent verschwinden — sonst entstünde ein Bug analog zu den bekannten Gotchas (Feature wirkt an einer Stelle, an der Nachbarstelle nicht):

1. **`messageSelect`/`ListMessages`** (`handler.go`): `WHERE`-Klausel erhält `AND NOT EXISTS (SELECT 1 FROM message_hides mh WHERE mh.message_id = m.id AND mh.user_id = ?)`. Zusätzlicher Bind-Parameter in allen drei Varianten (voll/`after`/`before`).
2. **Reply-Vorschau in derselben Query**: Die bestehende `CASE WHEN rm.deleted_at IS NOT NULL THEN '[Nachricht gelöscht]' ELSE rm.body END` bekommt einen weiteren Zweig `WHEN rm.id IN (SELECT message_id FROM message_hides WHERE user_id = ?) THEN '[Nachricht gelöscht]'` — wer eine Nachricht für sich ausgeblendet hat, soll sie auch nicht über ein Zitat in einer späteren Antwort wiedersehen. Bewusst derselbe Placeholder-Text wie bei echtem Löschen (kein Unterschied nach außen erkennbar, ob „für mich" oder „für alle" — Konsistenz mit der Bubble-Anzeige unten).
3. **`Search`** (`search.go`): zusätzlicher `AND NOT EXISTS (… message_hides …)`-Zweig neben dem bestehenden `deleted_at IS NULL`-Filter.
4. **`ComputeUnreadForUser`** (`unread.go`): zusätzlicher `AND NOT EXISTS (… message_hides …)` in der Nachrichten-Teilquery — eine ausgeblendete, vorher ungelesene Nachricht zählt nicht mehr im Badge.
5. **Konversations-`LastMessage`-Vorschau**: die Query, die `c.LastMessage` befüllt (`getConversationDetail`/Listen-Query), muss pro anfragendem Nutzer die jüngste **nicht ausgeblendete** Nachricht wählen, nicht pauschal die jüngste der Konversation.

### 4. Frontend: gemeinsamer Bestätigungsdialog statt zwei Overlay-spezifischer Implementierungen

Ein neues, kleines Modal (`DeleteMessageConfirmModal` o. ä.) folgt dem bestehenden Modal-Muster (`bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`) und wird von **beiden** Aufrufstellen (Desktop-Kontextmenü-Klick, Mobile-Overlay-Klick) geöffnet statt direkt zu löschen — ersetzt den bisherigen sofortigen `deleteMsg`-Aufruf:
- Zeigt bedingt zwei Buttons: „Für mich löschen" (immer) und „Für alle löschen" (nur wenn `canDeleteForEveryone(msg)` — bisherige `canDelete`-Logik: Absender oder Admin), plus „Abbrechen".
- `canDelete(msg)` (steuert, ob der „Löschen"-Eintrag im Kontextmenü/Overlay überhaupt erscheint) wird zu einer reinen Sichtbarkeitsfrage: **jede** nicht bereits gelöschte Nachricht zeigt „Löschen" (da „für mich" immer möglich ist) — nicht mehr auf Absender/Admin beschränkt.
- Zwei getrennte Aktionen: `hideMessageForMe(msg)` → `POST /chat/messages/{id}/hide`; `deleteMessageForEveryone(msg)` → bestehendes `DELETE /chat/messages/{id}` (unverändert).
- Live-Update: Backend broadcastet `chat:message-hidden:<convId>:<msgId>` gezielt an den handelnden Nutzer selbst (`h.hub.BroadcastToUser(claims.UserID, …)`, analog `MarkRead`s `chat:conversation-read`) für Multi-Geräte-Sync — **nicht** an andere Konversationsmitglieder (Privatsphäre: niemand soll erfahren, dass jemand eine Nachricht für sich ausgeblendet hat). `useLiveUpdates` im Frontend behandelt das Event wie einen Reload-Trigger für die aktive Konversation.
- „Für alle löschen" broadcastet weiterhin unverändert `chat:new-message:<convId>` an alle aktiven Mitglieder (bestehendes Verhalten von `DeleteMessage`).

### 5. Broadcast-Gate

`Chat.HideMessage` ruft `h.hub.BroadcastToUser` (auf den Nutzer selbst) auf und erfüllt damit das Broadcast-Gate direkt (kein Allowlist-Eintrag nötig) — anders als z. B. `Chat.DeleteBroadcast`, das aus anderem Grund (eigener SSE-Kanal) bereits in der Allowlist steht.

## Risks / Trade-offs

- **Query-Komplexität steigt an vier Stellen** (Liste, Suche, Unread, LastMessage) → Mitigation: jede Stelle bekommt einen Happy-Path- und Fehlerfall-Test (`internal/chat/*_test.go`), die exakt das „ausgeblendet für A, sichtbar für B"-Szenario abdecken; Musterreferenz ist der bestehende `deleted_at`-Filter, der dieselben vier Stellen bereits kennt (bis auf `message_hides` selbst).
- **Kein GC ausgeblendeter Nachrichten** (anders als Broadcasts) → bewusst: eine Nachricht bleibt in `messages` und für neu beitretende/andere Mitglieder unverändert vollständig sichtbar, auch wenn alle aktuellen Mitglieder sie „für sich" ausgeblendet haben. Ein GC wie bei `DeleteBroadcast` wäre hier fachlich falsch, weil eine Konversation (anders als eine einzelne Mitteilung) über die Zeit neue Mitglieder aufnehmen kann.
- **Zusätzlicher Bind-Parameter in `messageSelect`** (gemeinsamer SQL-Rumpf für drei `ListMessages`-Varianten) → Reihenfolge der `?`-Platzhalter muss in allen drei Aufrufstellen synchron gehalten werden; Regressionsschutz durch bestehende Tabellentests je Variante (voll/`after`/`before`), erweitert um den Ausblenden-Fall.

## Migration Plan

Reine additive Migration (`061_message_hides.up.sql` / `.down.sql`, `DROP TABLE message_hides`), keine Datenmigration nötig — Bestandsnachrichten sind für niemanden ausgeblendet. Kein Downtime-Risiko, läuft mit `make deploy`/`make migrate-remote-up` wie jede andere Migration.
