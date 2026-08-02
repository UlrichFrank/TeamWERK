## Why

Beim Öffnen bzw. Wechseln einer Konversation unter `/chat` flackert die Anzeige: für die
Dauer des Nachrichten-Fetches zeigt der Chat-Bereich den **neuen Header** über der
**Nachrichtenliste der vorherigen Konversation**. Nutzer berichten es aus Safari (macOS und
iOS), in Chrome tritt es kaum auf.

Ursache (gemessen, nicht vermutet): `openConversation` setzt `activeConv` synchron, lässt
`messages` aber unangetastet, bis `await loadMessages(...)` auflöst. Zwischen beiden liegt
die Netzwerk-Roundtrip-Zeit (lokal ~120 ms, produktiv mehr). Jeder Paint in diesem Fenster
zeigt einen Mischzustand aus zwei Konversationen.

```
Klick
 │
 ├─ setActiveConv(conv)   ← synchron  ⇒ Header rendert sofort neu
 └─ await loadMessages()  ← ~120 ms   ⇒ Liste wechselt erst danach
     ├──────── Fenster für den Mischzustand ────────┤
```

Das ist **kein Safari-Bug**, sondern ein Rennen, das Safari häufiger verliert als Chrome.
Deshalb war der Befund über Chrome nicht reproduzierbar und die Ursache lange unklar.

**Messgrundlage** (Prod-Binary + Seed-DB, Video-Aufzeichnung 25 fps, Frame-Differenz-Analyse):

| Engine | Aufnahmen | Mischzustand gemalt | Dauer |
|---|---|---|---|
| WebKit 26.5 (Safari-Engine) | 3 | **3 von 3** | ~120 ms (3 Frames) |
| Chromium 149 | 1 | 0 von 1 | — |
| WebKit, reiner Text-Chat (schnellerer Fetch) | 1 | 0 von 1 | — |

Der letzte Fall belegt den Renn-Charakter: dieselbe Engine, dasselbe Code-Pfad, nur ein
kürzerer Fetch — und der Mischzustand entfällt.

**Entlastet ist die Scroll-Anker-Logik** aus `chat-scroll-anchor`. Die rAF-Messung zeigt sie
korrekt arbeitend: beim Bild-Decode wächst `scrollHeight` um 6300 px und `scrollTop` folgt um
exakt 6300 px — der untere Rand steht still. Es gibt keinen Frame bei `scrollTop = 0`, kein
Flattern und keine verspätete `clientHeight`. Der einmalige Positionierungsschritt beim
Öffnen ist gewollt und bleibt.

## What Changes

- `openConversation` **leert die Nachrichtenliste** beim Wechsel (`setMessages([])`), bevor
  der Fetch startet. Der Chat-Bereich zeigt während des Ladens einen neutralen Ladezustand
  statt fremden Inhalts.
- Ein `loadingMessages`-State trägt diesen Ladezustand sichtbar (statt einer leeren Fläche,
  die wie „Konversation ohne Nachrichten" aussieht).
- Ein **Anker-Guard** (`awaitingMessagesRef`) verhindert, dass die absichtlich geleerte Liste
  das 600-ms-Settle des Scroll-Ankers armiert. Ohne ihn gäbe der Timer den Anker frei, bevor
  die Nachrichten eintreffen — bei Fetches über 600 ms ginge die Divider-Position verloren.
  Nachträglich aufgenommen; der Konflikt wurde erst durch die Implementierung sichtbar
  (Herleitung in `design.md`, „Entscheidung 5").
- Der Öffnungs-/Sende-Anker wird in einem **`useLayoutEffect`** angewandt statt in einem
  `useEffect`: die neue Liste erscheint damit bereits positioniert, statt einen Frame später
  zu springen. Der fortlaufende Watcher (MutationObserver + Bild-`load`) bleibt unverändert.
- **Regressionstest** auf zwei Ebenen. Entscheidend: eine künstlich verzögerte
  Nachrichten-Antwort (`page.route` bzw. ein nicht auflösendes Promise im Vitest-Mock) macht
  das Rennen **deterministisch** — der Test braucht weder WebKit noch Frame-Analyse.

## Non-Goals

- **Kein** Umbau der Scroll-Anker-Logik aus `chat-scroll-anchor`. Sie ist durch die Messung
  entlastet; `releaseAnchor`/`scheduleAnchorSettle` bleiben inhaltlich unverändert, ebenso
  die Anker-Semantik. Der Aufrufzeitpunkt des Öffnungs-Ankers wechselt von `useEffect` auf
  `useLayoutEffect`, und `applyAnchor` bekommt **einen additiven Guard** für die neu
  eingeführte Leerphase (siehe unten) — beides ändert nicht, *wohin* verankert wird.
- **Kein** WebKit-Projekt in `web/e2e/playwright.config.ts` in diesem Change — dafür gibt es
  eine konkrete Hürde (siehe „Offene Hürde" unten), und der Regressionstest braucht es nicht.
  Als eigener Follow-up sinnvoll, hier bewusst ausgeklammert.
- **Keine** Backend-/Routen-/Schema-Änderung. Rein Frontend.
- **Kein** Vorab-Caching bereits geladener Konversationen (würde den Mischzustand ebenfalls
  verkürzen, ist aber ein Feature mit eigenem Invalidierungsproblem — siehe `design.md`).

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `chat-open-at-unread`: ergänzt um die Anforderung, dass Header und Nachrichtenliste
  während eines Konversationswechsels **nie** zu verschiedenen Konversationen gehören
  dürfen. Die bestehenden Positionierungs-Anforderungen bleiben unberührt.

## Impact

- `web/src/pages/ChatPage.tsx` — `openConversation` (Liste leeren + Ladezustand),
  neuer `loadingMessages`-State, `loadMessages` (Ladezustand zurücksetzen, auch im
  Fehlerfall), `[messages]`-Effekt `useEffect` → `useLayoutEffect`, Render-Zweig für den
  Ladezustand.
- `web/src/pages/__tests__/ChatPage.switchFlicker.test.tsx` (neu) — Vitest-Invariante.
- `web/e2e/chat-scroll.spec.ts` — ein Test mit verzögerter Nachrichten-Antwort.

## Offene Hürde (dokumentiert, nicht Teil dieses Changes)

Ein WebKit-Projekt in der E2E-Suite scheitert derzeit an der Cookie-Policy: WebKit verwirft
das `Secure`-Refresh-Cookie über `http://localhost` (Chrome hat dafür eine
localhost-Ausnahme). Beobachtet beim Messen:

```
[console] error  Error in setSecureCookie: You cannot use `Secure` on http.
[net] 401 /api/auth/refresh  →  Redirect auf /login
```

`internal/auth/handler.go` setzt `Secure: true` an drei Stellen (Zeilen 198, 341, 360) —
korrekt und **nicht** für Tests aufzuweichen. Ein WebKit-Projekt bräuchte daher entweder
HTTPS im Test-`webServer` (self-signed + `ignoreHTTPSErrors`) oder Tests, die ohne echten
Reload auskommen (In-App-Navigation). Gehört in einen eigenen Change.

## Test-Anforderungen

Keine neuen HTTP-Routen (rein Frontend). Abgesichert wird die Invariante:

> Während eines Konversationswechsels existiert **kein** Zustand, in dem der Chat-Header
> Konversation B zeigt und die Nachrichtenliste Nachrichten aus Konversation A enthält.

| Ebene | Test | Erwartung / Invariante |
|---|---|---|
| Vitest | `ChatPage.switchFlicker.test.tsx` — „Wechsel zeigt keine Nachrichten der vorigen Konversation" | Konversation A öffnen (Nachrichten sichtbar), dann B klicken, wobei der Mock für B ein **noch nicht aufgelöstes** Promise liefert. Header zeigt B; **kein** Nachrichtentext aus A ist im DOM. **Muss ohne Fix rot sein.** |
| Vitest | `ChatPage.switchFlicker.test.tsx` — „Ladezustand während des Fetches" | Im selben Zwischenzustand ist ein Ladehinweis (`role="status"`) sichtbar; nach Auflösen des Promise ist er weg und B's Nachrichten stehen im DOM. |
| Playwright | `chat-scroll.spec.ts` — „Konversationswechsel zeigt keinen Fremdinhalt" | `page.route` verzögert `GET /api/chat/conversations/*/messages` um 800 ms. Nach dem Klick auf B gilt innerhalb des Fensters: Header = B **und** kein Nachrichtentext aus A sichtbar. Danach erscheinen B's Nachrichten. **Muss ohne Fix rot sein.** |
| Playwright | Bestand `chat-scroll.spec.ts` (3 Tests) | Unverändert grün — belegt, dass die `useLayoutEffect`-Umstellung die Anker-Positionierung (Ende / Divider / Chip, jeweils nach Bild-Decode) nicht beschädigt. |
| Vitest | `ChatPage.switchFlicker.test.tsx` — „Anker überlebt einen Fetch länger als das Settle-Timeout" | Konversation mit `unreadCount > 0`, nicht aufgelöstes Fetch-Promise, Fake-Timer über 600 ms hinaus, dann auflösen → die Divider-Positionierung greift. **Muss ohne den Guard (1.5) rot sein.** |
| Vitest | Bestand `ChatPage.openAtUnread.test.tsx`, `ChatPage.windowing.test.tsx` | Unverändert grün. |
