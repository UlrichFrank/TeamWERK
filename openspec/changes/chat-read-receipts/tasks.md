# Implementation Tasks

## 1. Datenmodell

- [ ] 1.1 Migration `internal/db/migrations/031_message_reads_read_at.up.sql`:
  `ALTER TABLE message_reads ADD COLUMN read_at TIMESTAMP;` + Backfill mit
  `CURRENT_TIMESTAMP` + Index `idx_message_reads_message_id`.
- [ ] 1.2 Migration `.down.sql`: Table-Rebuild (SQLite-Idiom), Index droppen.
- [ ] 1.3 `make migrate-up` lokal ausführen, `PRAGMA table_info(message_reads)`
  prüft die neue Spalte.

## 2. Backend

- [ ] 2.1 `MarkRead`-Handler in `internal/chat/handler.go` erweitern:
  vor dem `INSERT OR IGNORE` per SELECT die Menge der neu-zu-markierenden
  `(message_id, sender_id)`-Paare ermitteln (nur wo noch kein Read existiert).
  `read_at` explizit auf `CURRENT_TIMESTAMP` setzen.
- [ ] 2.2 Nach dem INSERT: pro `(sender_id)` mit `MAX(message_id)` als
  `upToMessageId` ein SSE-Event `chat:read-receipt` mit Payload
  `{convId, readerUserId, upToMessageId, readAt}` per
  `h.hub.BroadcastToUser(sender_id, event)` senden.
- [ ] 2.3 Neue Route `GET /api/chat/messages/{id}/reads` (Handler
  `GetMessageReads`). Authorization: nur wenn `m.sender_id = claims.UserID`,
  sonst 403. Response: `[{userId, name, readAt}]` sortiert nach `readAt` ASC.
- [ ] 2.4 In `ListMessages` / `Messages` pro Nachricht `readCount` (Anzahl
  Reader != Sender) und `readTotal` (aktive Konversations-Mitglieder != Sender)
  in die Response aufnehmen. Für Direct-Konversationen kollabiert das auf
  `read: bool` (readCount >= 1).
- [ ] 2.5 Router-Eintrag `r.Get("/chat/messages/{id}/reads", chatH.GetMessageReads)`
  im auth-tier `Authenticated` (jeder eingeloggte Nutzer, Handler-interne
  Sender-Prüfung).

## 3. Tests (Backend)

- [ ] 3.1 `TestGetMessageReads_Sender_OK`: Absender ruft
  `/messages/{id}/reads` ab, Response enthält Liste mit `readAt`.
- [ ] 3.2 `TestGetMessageReads_ForeignUser_403`: anderer Konversations-User
  bekommt 403.
- [ ] 3.3 `TestGetMessageReads_MessageMissing_404`: gelöschte oder unbekannte
  Message-ID.
- [ ] 3.4 `TestMarkRead_BroadcastsReadReceiptToSenders`: MarkRead in Gruppen-
  Konversation mit 2 Sendern → SSE-Fanout an beide Sender, jeweils genau ein
  Event mit korrektem `upToMessageId`.
- [ ] 3.5 `TestListMessages_IncludesReadCounters`: Response enthält
  `readCount`, `readTotal` pro Nachricht.
- [ ] 3.6 Broadcast-Gate `internal/arch/broadcast_test.go` läuft weiter grün
  (der neue Fanout ergänzt den bestehenden, keine Allowlist-Änderung).

## 4. Frontend

- [ ] 4.1 Message-Type in `web/src/pages/ChatPage.tsx` (bzw. Message-Interface)
  um `readCount`, `readTotal`, `read` erweitern.
- [ ] 4.2 Rendering in der eigenen Nachrichten-Bubble:
  - Direct: `<Check>` (gesendet) → `<CheckCheck className="text-brand-info">`
    (gelesen).
  - Group/Team-Group: `<CheckCheck>` + Text `N/M gelesen` (oder nur
    `<Check>` wenn N=0).
- [ ] 4.3 Neuer Info-Modal-Komponent `MessageReadsModal` (in
  `web/src/components/`), lädt on-demand per `api.get(/chat/messages/{id}/reads)`,
  rendert Reader-Liste mit `readAt` (Format `HH:MM`).
- [ ] 4.4 Tap/Klick auf die eigene Bubble öffnet das Modal (mobile: Tap
  auf die Bubble; Desktop: Click auf die Bubble mit Cursor-Pointer).
- [ ] 4.5 `useLiveUpdates`-Handler für `chat:read-receipt`: Payload parsen,
  betroffenen State `readReceipts` mergen, betroffene Bubbles re-rendern.
- [ ] 4.6 Kein Rendering für fremde Nachrichten (nur eigene bekommen Ticks).

## 5. Tests (Frontend)

- [ ] 5.1 Component-Test `MessageBubble` mit `read: true` → `CheckCheck`
  gerendert.
- [ ] 5.2 Component-Test `MessageBubble` mit `readCount=3, readTotal=8`
  → Text `3/8 gelesen`.
- [ ] 5.3 `MessageReadsModal`-Test: Fetch mockt, Reader-Liste sortiert nach
  `readAt` gerendert.
- [ ] 5.4 `ChatPage`-Integration: SSE-Event `chat:read-receipt` mit
  `upToMessageId=5` markiert Nachrichten 1..5 als gelesen, spätere unverändert.

## 6. Dokumentation

- [ ] 6.1 `docs/agent/06-gotchas.md`: neuer Absatz „Chat-Read-Receipts" —
  Kurzform des Live-Loop-Designs (Coalescing, Sender-only für Detail-Route).
- [ ] 6.2 `docs/agent/04-api-db.md`: neue Route `GET /chat/messages/{id}/reads`
  in Auth-Tier-Tabelle ergänzen.

## 7. Verifikation

- [ ] 7.1 `/verify-change` — grüne Build/Test/Lint + Projekt-Invarianten.
- [ ] 7.2 Lokaler Manual-Test: Konversation zwischen zwei Test-Usern öffnen,
  Nachricht senden, in zweiter Session lesen, Live-Update im ersten Client
  beobachten.
- [ ] 7.3 Info-Modal in Gruppen-Konversation manuell öffnen und die
  Reader-Liste prüfen.

## 8. Merge & Post-Deploy

- [ ] 8.1 PR öffnen, CI grün, Review.
- [ ] 8.2 `make deploy` (embed.FS + systemctl restart, Migration 031 läuft
  automatisch).
- [ ] 8.3 Kurze Bekanntmachung im internen Kanal, dass Read-Receipts jetzt
  aktiv sind.
- [ ] 8.4 `openspec archive chat-read-receipts`.
