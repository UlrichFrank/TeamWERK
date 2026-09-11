## Why

Unter iOS Safari springt der Inhalt von `/chat` beim Öffnen einer bildlastigen Konversation
mehrfach, bis alle Bilder geladen sind. Ursache (gemessen, nicht vermutet): der
`AuthImage`-Platzhalter reserviert **keine** Höhe, obwohl der Server seit
`chat-image-dimensions` die Bild-Dimensionen liefert und der Platzhalter `aspect-ratio`
setzt. Die Sprechblase ist ein Flex-Kind mit `items-start` und damit shrink-to-fit; ein
**leerer** `div` mit `aspect-ratio` trägt zu ihrer max-content-Breite **0 px** bei. Die Blase
kollabiert auf ihr Padding, `aspect-ratio` auf 0 px Breite ergibt 0 px Höhe.

Messung in Playwright (Chromium 149 und WebKit 26.5, exakte DOM-Struktur aus
`ChatPage.tsx`: Wrapper `flex-col items-start`, Blase `max-w-xs px-3 py-2`, Platzhalter
`max-w-full` + `aspect-ratio`):

| Fall | Platzhalter | fertiges `<img>` |
|---|---|---|
| Nur Bild 1280×960, **mit** Server-Dims | **24×16 px** | 344×262 px |
| Text + Bild, **mit** Server-Dims | 94×96 px | 344×289 px |
| Ohne Dims (6-rem-Fallback) | 24×112 px | 344×262 px |

Identisch in beiden Engines, also CSS-Spec-Verhalten, kein Engine-Bug. Der Fix aus
`chat-image-dimensions` (Migration 030, Backfill, `mediaWidth`/`mediaHeight`, Props) ist
korrekt durchgereicht, hat im Chat-Layout aber **nie gewirkt**.

Warum es unter iOS sichtbar ist und in Chrome nicht: jeder eintreffende Bild-Blob lässt die
Blase um ~250 px wachsen. Chromium kompensiert das über CSS scroll-anchoring **vor** dem
Paint. Safari hat kein scroll-anchoring; dort wird ein Frame mit verschobenem Inhalt gemalt,
danach korrigiert der Scroll-Anker aus `chat-scroll-anchor` im `load`-Handler den
`scrollTop`. Pro Bild ein sichtbarer Doppelframe, bei 15 gestaffelt eintreffenden Bildern 15
Sprünge. Die **Endposition** stimmt (das leisten die Anker-Tests), der **Weg dorthin** ist das
Flackern.

Der Beleg stand bereits in der Messung von `chat-conversation-switch-flicker` (design.md,
„Ergebnis 2"): `scrollHeight` 6848 → 13148 nach Bild-Decode, also **+6300 px** bei ~14 Bildern
à 450 px. Das wurde als „Anker folgt korrekt" gelesen, nicht als „Reservation fehlt". Die drei
E2E-Tests in `web/e2e/chat-scroll.spec.ts` prüfen ausschließlich die Ankerposition, nie die
Höhenstabilität, und sind deshalb grün.

## What Changes

- Der `AuthImage`-Platzhalter bekommt bei bekannten Server-Dimensionen eine **explizite
  Breite** (`width: <naturalWidth>px` gedeckelt über `max-width: 100%`), so dass er dieselbe
  intrinsische Breite in die shrink-to-fit-Blase einbringt wie später das `<img>`. Gegenmessung
  in WebKit: Platzhalter und Bild stimmen dann auf ≤ 5 px überein (Rest ist der
  Inline-Descender ohne Tailwind-Preflight; im Bundle mit `img { display: block }` erwartet 0,
  in Aufgabe verifiziert).
- Der 6-rem-Fallback für Bilder **ohne** Dimensionen bleibt unverändert (weiterhin ein
  einmaliger Shift, aus `chat-image-dimensions` bekannt und dort bewusst akzeptiert).
- **Neuer Playwright-Test** in `chat-scroll.spec.ts` unter `emulateNoScrollAnchoring`:
  `Δ scrollHeight` zwischen „alle Platzhalter gerendert" und „alle `img` dekodiert" muss für
  die Seed-Threads mit Server-Dims nahe 0 liegen. **Muss ohne Fix rot sein** (~6300 px).
- **Vitest** für `AuthImage`: Platzhalter trägt bei Server-Dims den `width`-Style, ohne Dims
  weiterhin nur `minHeight`.
- **Gotcha-Absatz** in `docs/agent/06-gotchas.md`: `aspect-ratio` auf einem leeren Element in
  einem shrink-to-fit-Container reserviert nichts; Platzhalter brauchen eine explizite Breite.

Kein BREAKING. Kein Backend, keine Migration, keine API-Änderung.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `chat-open-at-unread`: Das Requirement „Positionierung überlebt asynchrones Nachladen von
  Bildern" wird um die Zusage ergänzt, dass ein Bild-Platzhalter mit bekannten Dimensionen
  beim Decode die Layout-Höhe des Verlaufs **nicht** verändert. Die fortlaufende
  Re-Verankerung bleibt als Sicherheitsnetz für Bilder ohne Dimensionen bestehen, ist für
  Bilder mit Dimensionen aber nicht mehr der primäre Mechanismus.

## Impact

- `web/src/components/AuthImage.tsx`: Style des Platzhalter-`div` (eine Zeile plus
  Kommentar). Verhalten für `naturalWidth`/`naturalHeight` unverändert, nur die
  Breitenzuweisung kommt hinzu.
- `web/src/components/__tests__/AuthImage.dimensions.test.tsx`: eine Assertion auf den
  `width`-Style ergänzen.
- `web/e2e/chat-scroll.spec.ts`: ein neuer Test (Höhenstabilität); die bestehenden sieben
  Tests bleiben unverändert und müssen grün bleiben (belegt, dass die Anker-Logik durch die
  Breitenzuweisung nicht beschädigt wird).
- `docs/agent/06-gotchas.md`: neuer Absatz.
- **Nicht betroffen:** Scroll-Anker-Logik in `ChatPage.tsx` (`applyAnchor`, `releaseAnchor`,
  `scheduleAnchorSettle`), `WindowedRows`, Backend, `media`-Tabelle. Andere
  `AuthImage`-Aufrufer (Broadcast-Bild in `ChatPage.tsx`, Lightbox) bekommen dieselbe
  Breitenlogik; die Lightbox übergibt keine Dims und ist damit unverändert.
- **Nach dem Fix verbleibende sichtbare Übergänge** (gewollt, nicht Teil dieses Changes):
  der Ladezustand beim Konversationswechsel (`chat-conversation-switch-flicker`) und der
  `animate-pulse` auf den Platzhaltern.
