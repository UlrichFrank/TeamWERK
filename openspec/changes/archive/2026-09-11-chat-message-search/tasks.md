## 1. Backend: Such-Endpunkt

- [x] 1.1 `GET /api/chat/search` in `internal/chat/handler.go`: `q` (Pflicht, HTTP 400 wenn leer), `limit` (default 50, max 200), `offset` (default 0) parsen — Grenzen wie in `internal/members/handler.go`.
- [x] 1.2 Teilquery Chat-Nachrichten: `messages` JOIN `conversations`/`conversation_members` (nur `left_at IS NULL` für den anfragenden Nutzer), `deleted_at IS NULL`, `body LIKE ? ESCAPE '\'` mit escaptem Suchbegriff (`%`, `_`, `\` maskieren).
- [x] 1.3 Teilquery Mitteilungen: `broadcasts` JOIN `broadcast_reads` (`user_id = ?`, `hidden_at IS NULL`), `body LIKE ? ESCAPE '\'`.
- [x] 1.4 Ergebnisse in Go mergen, nach `sent_at DESC, id DESC` sortieren, `offset`/`limit` anwenden, `total` aus der ungekürzten Trefferzahl beider Teilmengen berechnen.
- [x] 1.5 Snippet-Extraktion: rune-genauer Ausschnitt (~140 Zeichen) um die erste Fundstelle im `body`, mit `…` an abgeschnittenen Enden (eigene Helper-Funktion, testbar isoliert vom Handler).
- [x] 1.6 Response-Felder je Treffer: `kind` (`message`/`broadcast`), `id`, `conversationId` (nur bei `message`), `conversationName` (Konversationsname bzw. für Direct-Chats Name des Gegenübers), `senderName`, `snippet`, `sentAt`.
- [x] 1.7 Route in `internal/app/router.go` unter der Authenticated-Chat-Gruppe registrieren.

## 2. Backend: Around-Cursor für Jump-to-Message

- [x] 2.1 `ListMessages` um Query-Parameter `around` erweitern (mutually exclusive zu `after`/`before`, wie bestehend zwischen `after`/`before`).
- [x] 2.2 Zwei Teilqueries wie in Entscheidung 6 (design.md): `id <= around` absteigend limitiert auf die Hälfte der Seitengröße, `id > around` aufsteigend auf die andere Hälfte; danach in Go zusammenführen und aufsteigend sortieren.
- [x] 2.3 Response um `hasOlder`/`hasNewer` ergänzen (statt implizitem Seitengröße-Vergleich).
- [x] 2.4 Bestehende Sichtbarkeitsprüfung (`isMember`) unverändert vor dem Query-Dispatch beibehalten.

## 3. Backend: Tests

- [x] 3.1 Happy-Path: Suchbegriff matcht Nachricht in eigener aktiver Konversation → im Ergebnis.
- [x] 3.2 Happy-Path: Suchbegriff matcht Mitteilung → im Ergebnis.
- [x] 3.3 Negativ: Suchbegriff matcht Nachricht in Konversation ohne (aktive) Mitgliedschaft → nicht im Ergebnis.
- [x] 3.4 Negativ: Suchbegriff matcht `body` einer gelöschten Nachricht (`deleted_at` gesetzt) → nicht im Ergebnis (`TestSearch_GeloeschteNachrichtNichtGefunden` aus design.md).
- [x] 3.5 Negativ: leerer `q`-Parameter → HTTP 400.
- [x] 3.6 Edge Case: Suchbegriff enthält `%`/`_` → wird literal behandelt, kein Wildcard-Verhalten.
- [x] 3.7 `around`-Cursor: liefert Nachrichten vor und nach der Zielnachricht, korrekt sortiert, `hasOlder`/`hasNewer` korrekt gesetzt.
- [x] 3.8 `around` + `after`/`before` gleichzeitig → HTTP 400.

## 4. Frontend: Such-UI

- [x] 4.1 Header-Control-Button (Lupe, `HEADER_CTRL_ICON`) neben `<h1>Nachrichten</h1>` in `web/src/pages/ChatPage.tsx`, öffnet Such-Modal.
- [x] 4.2 Neue Komponente `web/src/components/ChatSearchModal.tsx`: Eingabefeld (debounced, min. 2 Zeichen), Trefferliste (Konversations-/Mitteilungsname, Absender, Zeitpunkt via `relativeTime.ts`/`parseServerTime` falls Timestamp ohne Zonenkennung, Snippet mit hervorgehobenem Treffer-Substring).
- [x] 4.3 Klick auf Chat-Treffer: Modal schließen, `activeConv` setzen, Nachrichten über neuen `around`-Cursor laden (nicht über `loadMessages`), zur Zielnachricht `scrollIntoView({block:"center"})` + Highlight-Pulse-Klasse (2s, dann entfernen).
- [x] 4.4 Klick auf Mitteilungs-Treffer: Modal schließen, `tab` auf `"broadcasts"`, bestehendes `openBroadcast` aufrufen.
- [x] 4.5 Leerer/Ladezustand/Fehlerzustand im Modal (analog bestehender Fehlerbehandlung in `ChatPage.tsx`, `errorMessage`-Helper).
- [x] 4.6 `data-message-id`-Attribut auf Message-Row ergänzen (Voraussetzung für `scrollIntoView`-Ziel), falls noch nicht vorhanden.

## 5. Verifikation

- [x] 5.1 `make test` (Go, inkl. Broadcast-/Push-Fanout-/Arch-Gates — neue Route berührt keine Mutation, `broadcastAllowlist` daher unverändert).
- [x] 5.2 `pnpm -C web build && pnpm -C web test && pnpm -C web lint`.
- [x] 5.3 Playwright-Fall ergänzen: Suchtreffer öffnen scrollt zur richtigen Nachricht (Scroll-Verhalten laut `docs/agent/07-testing.md` auf E2E-Ebene, nicht Vitest) — `make test-e2e`.
- [x] 5.4 (per Playwright-Screenshots in Chromium geprüft, Desktop 1280 px + Mobile 390 px: Modal, Hervorhebung, Sprung) Manuelle Prüfung in Chrome: Suche über mehrere Konversationen + Mitteilungen, Highlight-Verhalten, Mobile-Ansicht (`sm:`-Breakpoint, Card-Layout falls zutreffend).
- [x] 5.5 `/verify-change` vor Abschluss (Route→Tests, brand-Tokens, lucide-Icons, `openspec validate`).
