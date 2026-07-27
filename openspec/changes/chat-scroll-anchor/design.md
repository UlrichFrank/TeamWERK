## Kontext

Der Fix betrifft ausschließlich die Client-Scroll-Logik in `web/src/pages/ChatPage.tsx`.
Serverseitig gibt es **keine** Unread-Grenze im Message-Response — der Client leitet den
Divider-Index aus `unreadCount` + Seitengröße (100) ab (bestehendes Verhalten, unverändert).

## Entscheidung 1: Intent-basierter Anker statt Einmal-Scroll

**Problem:** Ein einmaliger `scrollIntoView`/`scrollTop` beim Öffnen landet daneben, sobald
Bilder erst danach decoden und das Layout verschieben. Zeitgeboxte Fenster (100–1000 ms)
sind fragil, weil Decode-Zeiten auf realen Netzen darüber liegen.

**Entscheidung:** Ein persistenter `anchorRef` (`bottom` | `divider` | `divider-chip` |
`null`) hält die *Absicht*. `applyAnchor()` wird sowohl beim Öffnen (`[messages]`-Effekt)
als auch **fortlaufend** vom Watcher (MutationObserver + Bild-`load`, capture) aufgerufen,
solange der Anker gesetzt ist — und verankert den **Divider genauso** wie das **Ende**
(vorher gab es Re-Verankerung nur fürs Ende und nur bei `isAtBottom`).

## Entscheidung 2: Settle an Decode-Abschluss koppeln, nicht an Wall-Clock

**Problem (aus adversarialem Review):** Ein reines Wall-Clock-Settle (z. B. 1200 ms)
reintroduziert exakt die alte Fragilität — ein langsames Bild über dem Divider, dessen
Inter-Event-Gap das Timeout überschreitet, gibt den Anker vorzeitig frei; beim späteren
Decode gibt es keinen Re-Pin mehr (Divider-Modus hat keinen `isAtBottom`-Self-Heal wie der
Bottom-Modus).

**Entscheidung:** `scheduleAnchorSettle()` gibt den Anker erst frei, wenn **keine Medien
mehr ausstehen** — `anchorMediaPending()` prüft AuthImage-Platzhalter (`aria-busy`, während
Blob-XHR) **und** `img.complete` (Decode). Solange etwas lädt, wird erneut geprüft statt
freigegeben. Ein absolutes Zeitlimit (`anchorDeadlineRef`, 15 s ab Anker-Setzung) verhindert
Endlos-Halten bei einem hängenden Bild, das nie `load`/`error` feuert.

## Entscheidung 3: Freigabe-Trigger

- **Nutzer-Eingabe** (`wheel`/`touchstart`/`keydown`) → sofortige Freigabe (der Nutzer soll
  während des Nachladens nicht gegen den Re-Pin ankämpfen).
- **Scroll-Event außerhalb des programmatischen Fensters** → Freigabe. Deckt den
  **Maus-Scrollbar-Drag / Track-Klick** ab, der KEIN wheel/touch/key feuert (Review-Finding).
  `applyAnchor` schiebt `programmaticScrollUntilRef` bei jeder Anwendung um 300 ms vor, damit
  die von uns ausgelösten Scrolls nicht als Nutzer-Scroll fehlgedeutet werden.
- Bei Freigabe wird `isAtBottom` aus der **echten** Position berechnet (nicht aus dem Modus),
  sonst risse ein Wheel-Escape während eines `bottom`-Ankers den Nutzer wieder ans Ende.

## Entscheidung 4: Browser-scroll-anchoring NICHT global abschalten

Chrome kompensiert Wachstum über dem Sichtbereich automatisch (nützlich, wenn der Nutzer im
Verlauf liest und ein Bild oben decodet). Das bleibt aktiv. Der JS-Anker ergänzt es nur für
den Öffnungs-/Sende-Fall und trägt die Positionierung dort selbst — nötig für **Safari/iOS**,
das kein scroll-anchoring hat und wo der Bug real auftritt.

## Entscheidung 5: E2E muss Safari emulieren

Ein Chromium-Playwright-Test ist ohne Weiteres **nicht scharf**: Chromes scroll-anchoring
maskiert die Drift, der Test wäre grün gegen alt UND neu. `emulateNoScrollAnchoring()` setzt
`overflow-anchor: none` auf `[data-windowed-scroll]` und reproduziert damit das reale
Safari-Verhalten. Nachgewiesen: ohne Fix driftet der Divider ~900 px (rot), mit Fix bleibt er
oben (grün). Seed-Daten enthalten gezielt ein Bild **ohne** Server-Dimensionen über dem
Divider (idx 69 in „E2E Chat lang unread"), das den Placeholder→Bild-Shift auslöst.

## Alternativen (verworfen)

- **Nur `overflow-anchor: none` + JS für alles:** würde die Verlaufs-Lese-Kompensation auf
  Chrome verschlechtern; mehr JS-Verantwortung ohne Gewinn.
- **`img.decode()` vor dem Öffnungs-Scroll abwarten:** Bilder sind zum Öffnungszeitpunkt oft
  noch nicht im DOM (AuthImage setzt `src` erst nach dem Blob-XHR) → fragil. Fortlaufende
  Re-Verankerung ist robuster.
