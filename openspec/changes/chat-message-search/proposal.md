## Why

`/chat` hat aktuell keine Volltextsuche über Nachrichten — wer sich an eine Information aus einem älteren Gespräch erinnert ("wer hatte nochmal die Hallenadresse gepostet?"), muss die Konversation manuell finden und per unendlichem Scroll zurückblättern. Signal & Co. lösen das mit einer übergreifenden Suche, die alle Chats (und bei uns zusätzlich die Mitteilungen/Broadcasts) auf einmal durchsucht und direkt zum Treffer springt.

## What Changes

- Neuer Header-Control-Button (Lupe, neben der `<h1>Nachrichten</h1>`) öffnet eine übergreifende Such-Overlay.
- Neue Route `GET /api/chat/search?q=&limit=&offset=0` durchsucht den Nachrichtentext (`messages.body`) aller Konversationen, in denen der Nutzer **aktives** Mitglied ist, sowie den Text (`broadcasts.body`) aller für ihn sichtbaren Mitteilungen (`broadcast_reads.hidden_at IS NULL`). Ergebnis: `{ items: [...], total: N }`, sortiert neueste zuerst.
- Gelöschte Nachrichten (`messages.deleted_at IS NOT NULL`) werden aus der Suche ausgeschlossen — `body` wird beim Löschen nicht genullt (nur `deleted_at` gesetzt), die Suche muss das explizit filtern, sonst leckt gelöschter Inhalt über die Suchergebnisse.
- Trefferliste zeigt Konversations-/Mitteilungsname, Absender, Zeitpunkt und einen Snippet mit hervorgehobenem Treffer; ein Klick öffnet die Konversation bzw. Mitteilung.
- Für Chat-Treffer: neuer `around`-Cursor-Modus auf `GET /api/chat/conversations/{id}/messages` lädt die Konversation zentriert auf die Treffer-Nachricht (statt nur neueste zuerst), damit die getroffene Nachricht direkt sichtbar ist und kurz hervorgehoben werden kann. Für Mitteilungs-Treffer reicht das bestehende Öffnen (`openBroadcast`), da Mitteilungen nicht paginiert werden.

## Capabilities

### New Capabilities
- `chat-message-search`: übergreifende Volltextsuche über Chat-Nachrichten und Mitteilungen inkl. Sprung zur Fundstelle.

### Modified Capabilities
(keine — bestehende Chat-Capabilities ändern kein Verhalten, die Suche ist rein additiv)

## Impact

- **Backend:** `internal/chat/handler.go` (neuer Handler `Search`, Erweiterung `ListMessages` um `around`-Cursor), `internal/app/router.go` (neue Route), neue Tests in `internal/chat/`.
- **Frontend:** `web/src/pages/ChatPage.tsx` (Such-Button + Overlay-Komponente, Jump-to-Message-Logik), ggf. neue Datei `web/src/components/ChatSearchModal.tsx`.
- **DB:** keine Migration nötig — reiner `LIKE`-Scan über bestehende `messages`/`broadcasts`-Tabellen (Datenmenge eines Vereins-Chats ist klein genug, kein FTS5 nötig).
- **Keine Breaking Changes**, keine neuen externen Abhängigkeiten.
