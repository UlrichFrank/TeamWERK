## 1. State & Refs

- [x] 1.1 In `ChatPage.tsx` neuen `unreadDividerIndexRef` (Ref auf `number | null`) einführen — hält den beim Öffnen fixierten Index, damit spätere SSE-Nachrichten den Divider nicht verschieben (siehe design.md Decision 5)
- [x] 1.2 Neuer `unreadDividerRef` (Callback-Ref auf `HTMLDivElement | null`) für `scrollIntoView`
- [x] 1.3 Neuer `scrollToUnreadRef` (analog zu `forceScrollToEndRef`), damit der Auto-Scroll-Effekt beim nächsten `messages`-Update auf den Divider springt

## 2. openConversation-Logik

- [x] 2.1 In `openConversation` nach `await loadMessages(conv.id)` verzweigen:
      wenn `conv.unreadCount === 0` → `forceScrollToEndRef.current = true` (wie heute).
      Sonst → `unreadDividerIndexRef.current = Math.max(0, messages.length - conv.unreadCount)` UND `scrollToUnreadRef.current = true`
- [x] 2.2 Da `messages.length` beim Ausführen des Zweigs noch die *alten* Werte hat (setMessages hat noch nicht committed), Index aus der frischen API-Antwort direkt in `loadMessages` berechnen und über Ref weiterreichen — oder `openConversation` gibt sich den frischen Count durch, siehe unten
- [x] 2.3 `loadMessages` erweitern: nach `setMessages(msgs)` optional `unreadDividerIndexRef.current` setzen, wenn `conv.unreadCount > 0`. Signatur nimmt `unreadCount` als zweites Argument entgegen (openConversation ruft mit `conv.unreadCount` auf)

## 3. Auto-Scroll-Effekt

- [x] 3.1 Bestehenden `useEffect(..., [messages])` (im ChatPage — Sticky-Scroll-Guard) um dritten Zweig erweitern:
      **vor** dem `forceScrollToEndRef`-Check auf `scrollToUnreadRef.current` prüfen — wenn true, `unreadDividerRef.current?.scrollIntoView({ block: 'start' })`, dann Ref auf false zurücksetzen
- [x] 3.2 Sicherstellen dass `scrollToUnreadRef` beim Konversationswechsel zurückgesetzt wird (im openConversation-Pfad wenn unreadCount = 0)

## 4. UnreadDivider-Komponente

- [x] 4.1 In `ChatPage.tsx` (oder inline wie `DaySeparator`) neuen `UnreadDivider`-Komponent bauen: horizontale Linie, zentrierter Text „N ungelesene Nachrichten" (`text-brand-text-muted`, `border-brand-border-subtle`); nimmt `count: number` und `divRef: React.Ref<HTMLDivElement>` als Props
- [x] 4.2 Integration in `renderRow`-Callback der `WindowedRows` (analog zum bestehenden `sep`-Muster bei `ChatPage.tsx:986`): wenn `index === unreadDividerIndexRef.current`, `<UnreadDivider>` **vor** der Bubble rendern mit dem `unreadDividerRef` als `divRef`
- [x] 4.3 Nur zeigen wenn `unreadDividerIndexRef.current !== null` UND der Index innerhalb der geladenen Nachrichten liegt (also nicht im „alles ungelesen"-Fall)

## 5. Chip für ältere Ungelesene

- [x] 5.1 Über der `WindowedRows`-Liste (unter dem existierenden „Ältere Nachrichten laden"-Button, `ChatPage.tsx:940`) neuen Chip rendern, wenn `activeConv.unreadCount > messages.length`
- [x] 5.2 Chip-Text: `${activeConv.unreadCount - messages.length} weitere ungelesene Nachrichten älter — 'Ältere laden' klicken`. Styling: `p-2 bg-brand-info/10 border border-brand-info/30 rounded-lg text-xs text-brand-text mx-4 mt-2` (Muster wie Alert Info aus CLAUDE.md)
- [x] 5.3 Chip-Sichtbarkeit reaktiv: `useMemo` oder inline berechnen; verschwindet automatisch wenn `activeConv.unreadCount ≤ messages.length` gilt (nach `loadOlderMessages`)

## 6. Sonderfall: unreadCount > messages.length

- [x] 6.1 In der Positionierungs-Logik (Task 2.1): wenn `unreadCount > messages.length`, `unreadDividerIndexRef` auf **-1** (Sentinel) setzen, sodass 4.3 den Divider **nicht** rendert
- [x] 6.2 `scrollToUnreadRef`-Effekt in diesem Fall: fallback auf `scrollTo({ top: 0 })` des Windowed-Containers, damit der Nutzer oberhalb des ersten geladenen Eintrags (also am Chip) landet
- [x] 6.3 Test-Fixture: activeConv mit `unreadCount = 150`, 100 geladene Nachrichten — verifizieren dass Chip da ist, kein Divider

## 7. Tests

- [x] 7.1 Neuer Test-File `web/src/pages/__tests__/ChatPage.openAtUnread.test.tsx` mit Setup analog `ChatPage.windowing.test.tsx` (scrollIntoView-Spy, ResizeObserver-Polyfill, Layout-Mocks für data-windowed-scroll)
- [x] 7.2 Test „öffnet mit unreadCount=3 → scrollt zum Divider": Mock 20 Nachrichten + `unreadCount: 3`; nach Klick verifizieren dass ein Element mit Text „3 ungelesene Nachrichten" gerendert ist UND `scrollIntoView({ block: 'start' })` mindestens einmal auf einem Element aufgerufen wurde
- [x] 7.3 Test „öffnet mit unreadCount=0 → scrollt ans Ende": Mock 20 Nachrichten + `unreadCount: 0`; verifizieren dass KEIN Divider gerendert ist UND `scrollIntoView` (auf `messagesEndRef`) aufgerufen wurde — Regression gegen bisheriges Verhalten
- [x] 7.4 Test „unreadCount > geladen → Chip sichtbar, kein Divider": Mock 100 Nachrichten + `unreadCount: 150`; verifizieren dass Chip-Text „50 weitere ungelesene Nachrichten älter" sichtbar ist UND kein Divider gerendert wurde
- [x] 7.5 Bestehende `ChatPage.deepLink.test.tsx` und `ChatPage.windowing.test.tsx` durchlaufen ohne Änderungen (Regressionsschutz: neuer Code darf die alten Erwartungen nicht kippen)

## 8. Verifikation

- [x] 8.1 `make lint && make test` grün (alle Backend + 557+3 Frontend-Tests)
- [x] 8.2 `openspec validate chat-open-at-unread --type change` grün
- [ ] 8.3 Manuell im Browser: Konversation mit ungelesenen Nachrichten öffnen → landet am Divider, „N ungelesene Nachrichten" ist sichtbar
- [ ] 8.4 Manuell: Konversation ohne Ungelesenes öffnen → landet am Ende wie gewohnt
- [ ] 8.5 Manuell: SSE-Nachricht kommt während man in der Konversation ist → Divider bleibt an gleicher Stelle, neue Nachricht kommt unten dazu
