## 1. Frontend — Fremdinhalt beim Wechsel unmöglich machen (`web/src/pages/ChatPage.tsx`)

- [x] 1.1 `loadingMessages`-State einführen; `openConversation` setzt vor dem Fetch
      `setMessages([])` + `setLoadingMessages(true)` (nach dem Sichern des Entwurfs, vor
      `await loadMessages(...)`)
- [x] 1.2 `loadMessages`: `setLoadingMessages(false)` im `finally` (nicht nur im Erfolgsfall —
      der bestehende `catch {}` verschluckt Fehler sonst in einen Dauer-Ladezustand)
- [x] 1.3 Render-Zweig: solange `loadingMessages` einen Ladehinweis mit `role="status"`
      anstelle der leeren Liste zeigen (brand-Tokens, `lucide-react`, keine Emojis)
- [x] 1.4 Prüfen, dass die übrigen Aufrufer von `loadMessages` (`appendNewMessages`-Fallback,
      `chat:member-left`, `toggleReaction`) den Ladezustand **nicht** auslösen — sie
      aktualisieren eine bereits offene Konversation und dürfen die Liste nicht leeren
- [x] 1.5 **Anker-Guard** (`awaitingMessagesRef`): solange die Liste absichtlich leer ist,
      steigt `applyAnchor` sofort aus — ohne Scroll und ohne `scheduleAnchorSettle()`.
      `anchorRef` bleibt gesetzt und überlebt die Leerphase. Reset auch im Fehlerfall.
      Grund siehe `design.md` „Entscheidung 5"; nachträglich aufgenommen, weil erst die
      Implementierung von 1.1 den Konflikt mit dem 600-ms-Settle sichtbar gemacht hat

## 2. Frontend — Positionierung in denselben Frame legen

- [x] 2.1 `[messages]`-Effekt (`ChatPage.tsx:705`) von `useEffect` auf `useLayoutEffect`
      umstellen; Watcher-Effekt (Zeile 751) bewusst als `useEffect` belassen
- [x] 2.2 Kommentar am Effekt ergänzen: warum Layout-Effekt (Positionierung vor dem Paint)
      und warum der Watcher es nicht sein darf (nur Listener, darf Paint nicht blockieren)

## 3. Regressionstest — Vitest (`web/src/pages/__tests__/ChatPage.switchFlicker.test.tsx`, neu)

- [x] 3.1 Test „Wechsel zeigt keine Nachrichten der vorigen Konversation": A öffnen,
      B klicken mit noch **nicht** aufgelöstem Promise für B → Header zeigt B, kein
      Nachrichtentext aus A im DOM. Vor dem Fix rot verifizieren.
- [x] 3.2 Test „Ladezustand während des Fetches": im selben Zwischenzustand ist
      `role="status"` sichtbar; nach Auflösen verschwindet er und B's Nachrichten stehen im DOM
- [x] 3.3 Bestandstests `ChatPage.openAtUnread.test.tsx` und `ChatPage.windowing.test.tsx`
      unverändert grün
- [x] 3.4 Test „Anker überlebt einen Fetch länger als das Settle-Timeout": Konversation mit
      `unreadCount > 0`, Fetch als nicht aufgelöstes Promise, Fake-Timer über 600 ms hinaus,
      dann auflösen → Divider-Positionierung greift. Ohne Guard (1.5) rot verifizieren.

## 4. Regressionstest — Playwright (`web/e2e/chat-scroll.spec.ts`)

- [x] 4.1 Test „Konversationswechsel zeigt keinen Fremdinhalt": `page.route` verzögert
      `GET /api/chat/conversations/*/messages` um 800 ms; nach dem Klick auf B gilt im
      Fenster: Header = B **und** kein Nachrichtentext aus A sichtbar; danach erscheinen
      B's Nachrichten. Vor dem Fix rot verifizieren.
- [x] 4.2 `make test-e2e` — die drei bestehenden Scroll-Tests (Ende / Divider / Chip, jeweils
      nach Bild-Decode) müssen unverändert grün sein; sie sind der Nachweis, dass die
      `useLayoutEffect`-Umstellung die Anker-Positionierung nicht beschädigt

## 5. Abschluss

- [x] 5.1 `/verify-change` (Build/Test/Lint + Projekt-Invarianten: brand-Tokens,
      lucide-Icons, keine neue Route → kein Broadcast-Gate betroffen)
- [x] 5.2 `openspec validate --strict` für diesen Change
- [ ] 5.3 Manuell in Safari (macOS **und** iOS-PWA) gegen die lokale Prod-Binary
      gegenprüfen — die ursprüngliche Meldung kam aus beiden Umgebungen, und nur dort ist
      das Rennen realistisch eng genug
