## Why

Absender einer Chat-Nachricht sehen aktuell nicht, ob ihre Nachricht bei den
Empfängern angekommen und gelesen wurde. Trainer wissen nicht, ob eine
Kader-Info das Team erreicht hat; Kassierer wissen nicht, ob ein Eltern-1:1
gelesen wurde. Gleichzeitig existiert die Datenbasis dafür (`message_reads`,
`broadcast_reads`, `POST /chat/conversations/{id}/read`) bereits — sie wird
nur für die eigene Unread-Anzeige des Empfängers ausgewertet, nicht für den
Absender.

## What Changes

Absender sehen pro Nachricht einen WhatsApp-artigen Zustand:

- **`✓` gesendet** — Nachricht ist beim Server (State-of-the-Art bei uns:
  jede persistierte Nachricht).
- **`✓✓` gelesen** — mind. ein Empfänger hat die Konversation geöffnet
  (`message_reads`-Eintrag existiert).

**Direct (1:1):** Zwei-Zustands-Anzeige (`✓` / `✓✓`). Der Zeitpunkt „gelesen
um HH:MM" erscheint nur on-demand beim Tap/Klick auf die Nachrichtenblase in
einer Info-Ansicht.

**Group + Team-Group:** Aggregat `✓✓ N/M gelesen` in der Nachrichtenblase.
Tap/Klick öffnet die pro-Person-Liste („Anna ✓ 09:12 · Ben ✓ 09:15 · Carl
noch nicht gelesen"). Nur der Absender darf diese Info abrufen — die Reads
Dritter sollen für andere Empfänger nicht transparent sein.

**Broadcasts:** bewusst **außen vor**. Die Semantik dort ist eher
Zustellungsreport („178 von 200 gelesen") als WhatsApp-artiger Live-Tick
und verdient einen eigenen Change ([[broadcast-delivery-report]]) —
`broadcast_reads.read_at` existiert dafür bereits.

**Privacy:** Kein Opt-out (KISS). Im Vereinskontext sind alle Nutzer bekannt;
Trainer-/Kassierer-Bedarf überwiegt, und ein reziprokes Opt-out wie bei
WhatsApp würde einen neuen Settings-Baustein plus Handler-Filterlogik
erzwingen, ohne einen realistischen Missbrauchs-Fall im Verein zu adressieren.
Falls in der Praxis Beschwerden auftreten, ist Opt-out ein späterer Change.

**Datenmodell:** `message_reads` bekommt eine `read_at`-Spalte
(Migration 031). Bestandsdaten werden mit `CURRENT_TIMESTAMP` zum
Migrations-Zeitpunkt aufgefüllt — semantisch nicht perfekt (echtes Lesedatum
liegt in der Vergangenheit), aber der einzige echte Nutzer historischer
Timestamps wäre die Info-Ansicht, die für Bestandsnachrichten kaum interessant
ist.

**Live-Loop:** `MarkRead` broadcastet ab jetzt zusätzlich zum lesenden User
ein Coalescing-SSE-Event `chat:read-receipt` mit Payload
`{convId, readerUserId, upToMessageId, readAt}` an alle **Absender** der neu
markierten Nachrichten. Der Absender-Client aktualisiert damit alle
Nachrichten ≤ upToMessageId in dieser Konversation, statt pro Nachricht ein
Event zu bekommen (→ kein SSE-Sturm bei bulk-Reads).

**Neue Route:** `GET /api/chat/messages/{id}/reads` liefert die Reader-Liste
mit Timestamps. Zugriff nur für den Absender der Nachricht (403 sonst).

## Capabilities

### New Capabilities
- `chat-read-receipts` — neue Capability, dokumentiert die Absender-Sicht
  auf Lese-Zustände (bisher haben `chat-konversationen` und
  `chat-unread-app-badge` nur die Empfänger-Sicht abgedeckt).

### Modified Capabilities
(keine — `chat-konversationen` bleibt unangetastet; die neue Route und der
neue SSE-Event sind orthogonal zu den bestehenden Requirements dort.)

## Impact

- **Code (Backend)**: `internal/chat/handler.go` (neue Route
  `GetMessageReads`, erweiterter `MarkRead`-Fanout, `Messages`/`ListMessages`
  liefern `readCount` und `readTotal` pro Nachricht in Gruppen bzw.
  `read: bool` in 1:1). `internal/db/migrations/031_message_reads_read_at.*`.
- **Code (Frontend)**: `web/src/pages/ChatPage.tsx` rendert Tick-States in
  der eigenen Nachrichtenblase; neuer Info-Modal-Komponent (Read-by-Detail)
  für Tap-Auslöser; `useLiveUpdates`-Handler für `chat:read-receipt`.
- **Tests**: neue Handler-Tests (Happy: Absender sieht `readCount`, Detail-
  Endpoint liefert Reader-Liste; Fehler: fremder User bekommt 403 auf
  `/messages/{id}/reads`, gelöschte Nachricht 404), Broadcast-Gate-
  Ergänzung (der neue Fanout muss auf die Allowlist der Broadcast-Kanäle),
  Frontend-Component-Test für die Bubble-States.
- **Betrieb**: Migration 031 idempotent (`ALTER TABLE`-Backfill), kein
  Downtime-Risiko. Kein Deploy-Blocker.
- **Nicht betroffen**: Broadcasts (bleiben mit `broadcast_reads.read_at`
  wie sie sind), Zero-Knowledge-Krypto, Auth-Tiers, SEPA/Beitragslauf.

## Offene Punkte für die Implementierung (nicht Scope-relevant)

- Icon-Wahl: `<Check>` vs. `<CheckCheck>` aus lucide-react — beide vorhanden,
  Detail-Entscheidung in der UX-Umsetzung.
- Info-Modal-Trigger: Tap auf die eigene Blase vs. Long-Press vs. explizites
  Icon — steht ebenfalls in der UX-Runde.
