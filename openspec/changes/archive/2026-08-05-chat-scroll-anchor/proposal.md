## Why

Unter `/chat` landet man beim Öffnen eines Chats nicht zuverlässig am Ende bzw. an der
letzten gelesenen Nachricht — besonders in langen, bildlastigen Threads. Ursache (im Code
verifiziert):

- Die Öffnungs-Positionierung war ein **einmaliger** Scroll (`scrollIntoView`/`scrollTop`),
  abgesichert nur durch zeitgeboxte Fenster (100–1000 ms) und einen Sticky-Snap, der
  **ausschließlich ans Ende** und **nur bei `isAtBottom`** re-verankerte.
- Chat-Bilder (`AuthImage`) laden per Blob-XHR und decoden erst Sekunden später; sie
  verschieben das Layout **nach** dem Scroll. Ein Bild **über** dem UnreadDivider drückt
  ihn dann aus dem Viewport — ohne Re-Verankerung.
- Auf **iOS Safari** (PWA-Haupteinsatzumfeld) gibt es **kein CSS scroll-anchoring**
  (`overflow-anchor`), das dieses Wachstum kompensiert → der Bug tritt dort real auf.
  Chromium hat scroll-anchoring und maskiert ihn, weshalb er schwer zu fassen war.

## What Changes

- Der einmalige Öffnungs-Scroll wird durch einen **persistenten, intent-basierten
  Scroll-Anker** ersetzt (`anchorRef`: `bottom` | `divider` | `divider-chip` | `null`).
- Der Anker wird bei **jeder** Layout-Änderung (MutationObserver + Bild-`load`)
  **fortlaufend** neu angewandt — für den Divider **genauso** wie fürs Ende.
- Freigabe des Ankers durch **echte Nutzer-Eingabe** (wheel/touchstart/keydown **und**
  Maus-Scrollbar-Drag via Scroll-Event außerhalb des programmatischen Fensters) **oder**
  wenn alle Medien fertig geladen/dekodiert sind (Settle an **Decode-Abschluss** gekoppelt,
  nicht an Wall-Clock; absolutes Zeitlimit als Backstop).
- `isAtBottom` wird bei Freigabe aus der **tatsächlichen** Scroll-Position abgeleitet
  (nicht aus dem Anker-Modus), damit ein Wheel-Escape während eines `bottom`-Ankers den
  Nutzer nicht wieder ans Ende reißt.
- `loadOlderMessages` gibt einen aktiven Anker frei (Button liegt außerhalb des
  Scroll-Containers) und hält die Position auch über decodende voran-gestellte Bilder.
- **Testwelt**: drei lange, bildlastige Seed-Threads + `dev-seed`-Subcommand (manuell im
  Browser durchklickbar) + Playwright-Regression, die die Abwesenheit von scroll-anchoring
  (Safari) emuliert und so den Bug in Chromium reproduzierbar macht.

## Non-Goals

- **Kein** globales Abschalten von Browser-scroll-anchoring — es bleibt aktiv für den Fall
  „Nutzer liest Verlauf, Bild über dem Sichtbereich decodet" auf Chrome. Der JS-Anker
  ergänzt es nur für den Öffnungs-/Safari-Fall.
- **Keine** Reaktivierung des Windowings der Chat-Liste (bleibt bewusst deaktiviert).
- **Keine** Backend-/Routen-Änderung, kein neues API-Feld für die Unread-Grenze.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `chat-open-at-unread`: die Öffnungs-Positionierung wird von einem einmaligen Scroll auf
  eine fortlaufende, decode-robuste Verankerung erweitert; neue Anforderung „Positionierung
  überlebt asynchrones Nachladen von Bildern".

## Impact

- `web/src/pages/ChatPage.tsx` — `anchorRef` + Helfer (`applyAnchor`, `releaseAnchor`,
  `scheduleAnchorSettle`, `anchorMediaPending`, `scrollBox`), Umbau von `openConversation`,
  `loadMessages`, `sendMessage`, `loadOlderMessages`, dem `[messages]`-Effekt und dem
  Sticky-Watcher (`snap` → `reposition`).
- `web/e2e/chat-scroll.spec.ts` — 3 neue Regressionstests + `emulateNoScrollAnchoring`-Helfer
  (Safari-Emulation); `retries: 0` für die Datei (unread-Zustand wird beim ersten Öffnen
  konsumiert).
- `cmd/teamwerk/e2e_seed.go` — `seedImage(withDims)`, `seedLongThread`, drei lange Threads.
- `cmd/teamwerk/dev_seed.go` (neu) + `cmd/teamwerk/main.go` — `dev-seed`-Subcommand (lokal,
  Prod-Schutz für `--db` und `MEDIA_DIR`).
- `cmd/teamwerk/e2e_seed_test.go` — Volumina/Unread-Asserts (150/150/250, 0/40/180).

## Test-Anforderungen

Keine neuen HTTP-Routen (rein Frontend + Seed-Tooling). Abgesichert wird das **Verhalten**:

| Ebene | Test | Erwartung / Invariante |
|---|---|---|
| Playwright | „lange unread-Konversation: Divider bleibt oben NACH Bild-Decode" (`chat-scroll.spec.ts`) | Unter Safari-Emulation (`overflow-anchor:none`) bleibt der UnreadDivider nach vollständigem Bild-Decode am oberen Viewport-Rand (`d.top - box.top ≤ 80`). **Muss ohne Fix rot sein** (Drift ~900 px). |
| Playwright | „lange gelesene Konversation öffnet am Ende" | Nach Bild-Decode `\|scrollHeight − clientHeight − scrollTop\| ≤ 4`. |
| Playwright | „viele-ungelesen: Chip oben, ‚Ältere laden' erhält Position" | Chip sichtbar; `scrollTop ≤ 4` nach Decode; nach „Ältere laden" bleibt der Alt-Content stabil (`\|Δtop − Δheight\| ≤ 8`). |
| Vitest | `ChatPage.openAtUnread.test.tsx` (Bestand, grün) | Divider bekommt `scrollIntoView({block:'start'})`; Ende erreicht `scrollHeight`; Chip bei `unread > Seite`. |
| Vitest | `ChatPage.windowing.test.tsx` (Bestand, grün) | Hochgescrollter Nutzer wird bei SSE **nicht** ans Ende gerissen. |
| Go | `e2e_seed_test.go` | Konversationen `lang gelesen`/`lang unread`/`viele ungelesen` mit 150/150/250 Nachrichten und Admin-`unreadCount` 0/40/180; Bilder mit und ohne Server-Dimensionen vorhanden. |
