# Design: Chat Read Receipts

## Datenmodell

Migration `031_message_reads_read_at`:

```sql
-- .up.sql
ALTER TABLE message_reads ADD COLUMN read_at TIMESTAMP;
UPDATE message_reads SET read_at = CURRENT_TIMESTAMP WHERE read_at IS NULL;
-- SQLite kann NOT NULL nachträglich nicht setzen — Constraint per CHECK.
CREATE INDEX idx_message_reads_message_id ON message_reads(message_id);
```

Der neue Index beschleunigt `GET /messages/{id}/reads` und die
Aggregat-Berechnung (`COUNT(*) WHERE message_id = ?`) in der Message-Listen-
Abfrage, die sonst über den (message_id, user_id)-Primärschlüssel auf
einzelne Reads zugreift.

Der `.down.sql`-Pfad droppt die Spalte per Table-Rebuild (SQLite-Idiom).

## Live-Loop-Coalescing

Warum ein Coalescing-Event pro (convId, readerId) statt pro Nachricht:

```
   Bulk-Read Szenario: Empfänger öffnet Konv. mit 30 ungelesenen Nachrichten

   Pro-Nachricht-Fanout          Coalescing-Fanout
   ═══════════════════           ═════════════════
                                                
   30 × SSE("chat:read-          1 × SSE("chat:read-receipt",
   receipt", msgId=X)             upToMessageId=<max>)
                                                
   → Absender-Client              → Absender-Client
     mergt 30 States              mergt via ≤ upToMessageId
     synchron                     einmal
```

Der Handler ermittelt beim `INSERT OR IGNORE` die Menge der tatsächlich neu
markierten Message-IDs, gruppiert nach Absender und feuert pro
(sender_user_id, conv_id) genau ein Event mit dem höchsten neu-markierten
message_id-Wert als `upToMessageId`. Zusätzlicher Aufwand ~1 SELECT im
Handler, aber der SSE-Kanal bleibt schmal.

## Fanout in Gruppen

In einer 8er-Gruppe, in der alle acht Personen den Kader-Post öffnen, feuert
das `chat:read-receipt`-Event 8 Mal — je einmal pro Reader an den Absender.
Das ist bewusst so: der Absender-Client will pro Reader die Read-Info separat
mergen, um „6/8 gelesen" korrekt zu zählen. Alternative wäre ein
Server-seitiges Coalescing über einen Zeit-Fenster („Sammle alle Reads
innerhalb 500ms und feuere ein Merge-Event"), das aber Komplexität ohne
klaren Nutzen bringt: acht Events in 10 Sekunden sind kein Sturm.

## Authorization für die Detail-Route

```
GET /api/chat/messages/{id}/reads
```

Nur der **Absender** der Nachricht (`m.sender_id = claims.UserID`) darf die
Reader-Liste sehen. Andere Konversations-Mitglieder bekommen 403. Grund: die
Reads Dritter dürfen für andere Teilnehmer nicht transparent werden — sonst
wüsste jedes Mitglied, wer wann welche Nachricht gelesen hat, was über die
WhatsApp-Semantik (Info nur an den Sender) hinausginge und den KISS-No-Opt-
out-Entschluss unfair machen würde.

Löschungs-Fälle: Wenn der Sender selbst die Konversation verlassen hat
(`conversation_members.left_at IS NOT NULL`), bekommt er trotzdem 200 mit
der Liste. Er ist Sender, seine Zugehörigkeit ist irrelevant. Wenn die
Nachricht per Message-Delete entfernt wurde (Spec: `chat-message-delete`),
liefert die Route 404.

## Frontend-Merge-Logik

`ChatPage.tsx` hält einen State `readReceipts: Map<convId, Map<userId,
{upToMessageId, readAt}>>`. Beim initialen Load der Konversation kommt der
Zustand aus der `GET /messages`-Response (jede Nachricht trägt `readCount`
und `readTotal`; für die letzte eigene Nachricht wird zusätzlich `readByAll`
berechnet). Beim SSE-Event wird der Map-Eintrag aktualisiert und die
betroffenen Bubbles re-rendert.

Die Info-Modal-Ansicht lädt on-demand per `GET /messages/{id}/reads` beim
Tap auf die Bubble.

## Broadcast-Gate-Konformität

Der `MarkRead`-Handler broadcastet bereits heute (`chat:conversation-read`
an den Reader selbst). Der neue Fanout an den Absender fügt einen zweiten
`BroadcastToUser`-Aufruf hinzu — Broadcast-Gate ist zufrieden, keine
Allowlist-Änderung nötig. Der neue Route `GET /messages/{id}/reads` ist
read-only (GET), fällt also von vornherein aus dem Broadcast-Gate (das nur
POST/PUT/PATCH/DELETE prüft).

## Was bewusst nicht drin ist

- **Delivered-Zustand** (WhatsApp-Zwischen-Häkchen ✓✓ grau) — wir haben
  keine Client-Side-Delivery-Bestätigung und die Info „Server hat's
  angenommen" ist trivial (jede persistierte Nachricht). Zweistufig statt
  dreistufig bleibt ehrlich.
- **Opt-out** — siehe Proposal, KISS.
- **Broadcasts** — eigener späterer Change.
- **Push-basiertes Read** — Push-Notifications gelten explizit **nicht** als
  Lese-Bestätigung. Der Nutzer muss die Konversation öffnen, damit
  `POST /conversations/{id}/read` ausgelöst wird.
