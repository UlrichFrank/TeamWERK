## 1. Frontend — intent-basierter Scroll-Anker (`web/src/pages/ChatPage.tsx`)

- [x] 1.1 `anchorRef` (`bottom`|`divider`|`divider-chip`|`null`), `anchorSettleTimerRef`, `anchorDeadlineRef` einführen; `forceScrollToEndRef`/`scrollToUnreadRef` entfernen
- [x] 1.2 Helfer `scrollBox()`, `releaseAnchor()` (isAtBottom aus echter Position), `anchorMediaPending()`, `scheduleAnchorSettle()` (decode-basiert + Deadline), `applyAnchor()`
- [x] 1.3 `openConversation`: Anker aus `unreadCount` setzen, Deadline armieren, alten Settle-Timer abräumen
- [x] 1.4 `loadMessages`: Chip-Fall (`unread > Seite`) präzisiert Anker auf `divider-chip`
- [x] 1.5 `sendMessage`: Anker `bottom` + Deadline
- [x] 1.6 `[messages]`-Effekt: bei aktivem Anker `applyAnchor()`, sonst Sticky-Guard
- [x] 1.7 Watcher-Effekt: `reposition` (Anker → applyAnchor, sonst Sticky-Snap); `onScroll` gibt Anker bei echtem Scroll frei; `wheel`/`touchstart`/`keydown`-Release; Settle-Timer im Cleanup abräumen
- [x] 1.8 `loadOlderMessages`: Anker freigeben + Position über Bild-`load` bis Settle halten (`keepPosition`)

## 2. Testwelt — Seed (`cmd/teamwerk/`)

- [x] 2.1 `seedImage(withDims bool)` — media-Zeile optional ohne `width/height` (AuthImage-Fallback)
- [x] 2.2 `seedLongThread(...)` mit explizitem, gespreiztem `sent_at` (deterministisch)
- [x] 2.3 Drei Konversationen: „E2E Chat lang gelesen" (150, unread 0), „E2E Chat lang unread" (150, unread 40, no-dims-Bild über dem Divider), „E2E Chat viele ungelesen" (250, unread 180, Chip-Fall)
- [x] 2.4 `dev-seed`-Subcommand (`dev_seed.go` + Dispatch in `main.go`), Prod-Schutz für `--db` und `MEDIA_DIR`
- [x] 2.5 `e2e_seed_test.go`: Volumina + Admin-unreadCount (150/150/250, 0/40/180) + gemischte Bild-Dims

## 3. Regression — Playwright (`web/e2e/chat-scroll.spec.ts`)

- [x] 3.1 `emulateNoScrollAnchoring()` (Safari-Emulation via `overflow-anchor:none`) + `waitImagesSettled()` (count-unabhängig via `aria-busy` + `img.complete`)
- [x] 3.2 Killer-Test „Divider bleibt oben NACH Bild-Decode" (rot ohne Fix, grün mit Fix)
- [x] 3.3 „lange gelesene Konversation öffnet am Ende" + „viele-ungelesen: Chip oben, ‚Ältere laden' erhält Position"
- [x] 3.4 `retries: 0` für die Datei (unread-Zustand wird beim ersten Öffnen konsumiert → Retries würden maskieren)

## 4. Verifikation

- [x] 4.1 `tsc`, alle 7 ChatPage-Vitest-Dateien (27 Tests), `go build/test ./...` (inkl. arch/broadcast), `go vet`, Frontend-Lint — grün
- [x] 4.2 `make test-e2e` (11 Tests) grün; Schärfe zweifach nachgewiesen (alt = rot ~900 px, neu = grün)
- [x] 4.3 Adversariales Review (7 Findings) durchgearbeitet und behoben
