## Context

Motivation und Messung siehe `proposal.md` (Why). Hier nur, was den Ansatz formt.

Die betroffene Struktur in `ChatPage.tsx` (`MessageRow`):

```
<div class="flex items-center …">                       Zeile
  <div class="flex flex-col items-start flex-1">        Wrapper → Kinder shrink-to-fit
    <div class="max-w-xs sm:max-w-sm px-3 py-2">        Blase, Breite = max-content(Kinder)
      [<span>Text</span>]
      <AuthImage className="max-w-full …" naturalWidth naturalHeight />
```

`AuthImage` rendert in zwei Phasen dasselbe Layout-Slot mit **unterschiedlicher
intrinsischer Breite**:

| Phase | Element | intrinsische Breite | Beitrag zur Blasenbreite |
|---|---|---|---|
| Blob-XHR läuft | `<div style="aspect-ratio">` | keine (leer) | **0** |
| Blob da | `<img style="aspect-ratio">` | `naturalWidth` | `min(naturalWidth, max-w)` |

Das ist der gesamte Bug. Alles andere (Server-Dims, Backfill, Props, Anker) ist korrekt.

Randbedingungen aus `docs/agent/`:
- Nur `brand-*`-Tokens, keine Raw-Farben (`05-frontend.md`) — betrifft uns nicht, es kommt
  nur ein Inline-`width` hinzu.
- Scroll-Anker-Logik ist durch Messung entlastet und **nicht** anzufassen
  (`chat-scroll-anchor`, `chat-conversation-switch-flicker`, Memory
  `chat-scroll-anchoring-safari`).
- E2E-Regressionstests für Scroll-Verhalten liegen in `web/e2e/chat-scroll.spec.ts`,
  `retries: 0`, Seed-Threads aus `cmd/teamwerk/e2e_seed.go` (`07-testing.md`).

## Goals / Non-Goals

**Goals:**
- Platzhalter und fertiges Bild belegen in der Sprechblase dieselbe Fläche, ab dem ersten
  Frame, für alle Bilder mit Server-Dimensionen.
- Ein E2E-Test, der **Höhenstabilität** misst (nicht Ankerposition) und ohne den Fix rot ist.
- Die sieben bestehenden Tests in `chat-scroll.spec.ts` bleiben unverändert grün.

**Non-Goals:**
- Kein Umbau des Scroll-Ankers. Er bleibt für Bilder ohne Dims, für den Divider-Fall und
  als Sicherheitsnetz. Ob er nach diesem Fix vereinfacht werden kann, ist eine spätere
  Frage mit eigener Messung.
- Kein BlurHash / Thumbnail, kein Umbau des Blob-XHR-Ladepfads, kein Windowing.
- Kein Fix für Bilder **ohne** Dims (6-rem-Fallback bleibt). Der Backfill hat den Bestand
  abgedeckt; der Rest ist Ausnahmefall.
- Kein Entfernen von `animate-pulse` auf dem Platzhalter (Wahrnehmungsfrage, eigener Change
  falls gewünscht).

## Decisions

### Entscheidung 1: Explizite Breite am Platzhalter statt Umbau der Blase

**Was:** Bei Server-Dims bekommt der Platzhalter-`div` zusätzlich zu `aspectRatio` ein
`width: ${naturalWidth}px`. Das vorhandene `max-w-full` (`max-width: 100%`) deckelt auf die
Blasenbreite, genau wie beim `<img>`.

**Warum:** Der `<img>` verhält sich exakt so — intrinsische Breite `naturalWidth`, gedeckelt
über `max-width: 100%`. Wer dem Platzhalter dieselbe intrinsische Breite gibt, bekommt
dasselbe Layout-Ergebnis, ohne die Blase, den Wrapper oder Tailwind-Klassen anzufassen.
Gegenmessung in WebKit 26.5 (Test-HTML ohne Tailwind-Preflight):

| Fall | Platzhalter mit `width` | fertiges `<img>` |
|---|---|---|
| Nur Bild 1280×960 | 344×256 | 344×261 |
| Kleines Bild 200×300 | 224×316 | 224×321 |
| Text + Bild 1280×960 | 344×284 | 344×289 |

Die 5 px Rest sind der Inline-Descender-Abstand des `<img>` (`display: inline` im
Test-HTML). Tailwind-Preflight setzt `img { display: block }`, im echten Bundle wird die
Differenz deshalb 0 erwartet — Aufgabe 1.3 verifiziert das im E2E-Test statt es anzunehmen.

**Alternativen:**

| Alternative | Warum verworfen |
|---|---|
| Blase auf feste Breite (`w-full max-w-xs`) | Text-Blasen würden immer maximal breit — sichtbare Design-Änderung an jeder Nachricht, um ein Bild-Problem zu lösen. |
| Gemeinsamer Wrapper-`div` mit `width` um Platzhalter **und** `<img>` | Funktional gleichwertig, aber ein zusätzliches DOM-Element pro Bild, und die `className` (`rounded-lg`, `mt-2`, `cursor-pointer`) müsste auf Wrapper und Bild verteilt werden. Mehr Diff, kein Gewinn. |
| `<img>` immer rendern, `src` später setzen | Ein `<img>` ohne `src` hat ebenfalls keine intrinsische Breite → gleiches Problem. Zudem feuert `load` erst mit `src`, der Anker-Watcher (`img.complete`) würde die Phase falsch bewerten. |
| `min-width` statt `width` | `min-width: 1280px` wird durch `max-width: 100%` nicht gedeckelt (min gewinnt über max in CSS) — die Blase würde überlaufen. |

### Entscheidung 2: E2E misst Höhe, nicht Position — mit verzögerten Medien-Antworten

**Was:** Neuer Test in `chat-scroll.spec.ts`: `page.route` hält `GET /api/media/*` an, bis
der Test die Ausgangshöhe gemessen hat. Ablauf:

1. „E2E Chat mit Bildern" öffnen (4 Bilder, **alle** mit Dims — `seedChatMedia`
   ruft `seedImage(…, true)`).
2. Warten, bis 4 Platzhalter (`[aria-busy="true"]`) im Container stehen, `scrollHeight`
   messen (`h0`).
3. Routen freigeben, `waitAllImagesLoaded(page, 4)`, `scrollHeight` messen (`h1`).
4. `|h1 − h0| ≤ 4`.

**Warum die Verzögerung:** Lokal ist der Blob-Fetch im einstelligen Millisekunden-Bereich;
ohne Anhalten gibt es keinen stabilen Moment, in dem alle vier Platzhalter sichtbar sind.
Dasselbe Muster (künstliche Verzögerung macht ein Rennen deterministisch) hat sich in
`chat-conversation-switch-flicker` bewährt.

**Warum „E2E Chat mit Bildern" und nicht die langen Threads:** Die langen Seed-Threads
mischen absichtlich Bilder mit und ohne Dims (Bug-Trigger für den Anker). Dort ist ein
Höhen-Delta **erwartet** (6-rem-Fallback). Ein Test auf Δ≈0 braucht einen Thread, in dem
jedes Bild Dims hat. Ohne Fix erwartetes Delta dort: 4 × (444 − 16) ≈ 1700 px → klar rot.

**Warum keine `emulateNoScrollAnchoring`:** Höhe ist von scroll-anchoring unabhängig; die
Emulation ist hier weder nötig noch schädlich. Wir lassen sie weg, damit der Test nicht
suggeriert, er prüfe Positionierung.

**Alternativen:**

| Alternative | Warum verworfen |
|---|---|
| Vitest/jsdom | jsdom hat kein Layout; die shrink-to-fit-Breite ist genau die Layout-Physik, die es nicht sieht (`07-testing.md`). Vitest prüft nur, dass der `width`-Style gesetzt ist (Aufgabe 1.2). |
| Bestehende Anker-Tests um Höhen-Assertion erweitern | Deren Threads enthalten Bilder ohne Dims → Δ≠0 ist dort korrekt. Vermischung würde die Aussage beider Tests trüben. |
| Neuer Seed-Thread nur mit Dims-Bildern | Unnötig, „E2E Chat mit Bildern" erfüllt die Bedingung bereits. |

### Entscheidung 3: Kein Eingriff in den Anker, auch nicht „aufräumen"

Mit korrekt reservierten Platzhaltern feuert `reposition()` beim Decode zwar weiterhin
(`load`-Event), findet aber ein unverändertes `scrollHeight` vor und schreibt denselben
`scrollTop` zurück — ein No-op. Die Anker-Logik bleibt für den Divider-Fall, für Bilder ohne
Dims und als Backstop nötig. Ein Rückbau wäre ein eigener Change mit eigener Messung
(Memory `chat-scroll-anchoring-safari`: nicht „vereinfachen" mit dem Argument, der Browser
mache das schon).

## Risks / Trade-offs

- **Preflight-Annahme (0 px Differenz) trifft nicht zu** → Aufgabe 1.3 misst es im echten
  Bundle; bleibt eine Differenz, wird `display: block` explizit am `<img>` gesetzt
  (`block`-Klasse), nicht die Toleranz erhöht.
- **`naturalWidth` in Bild-Pixeln vs. CSS-Pixel** → Der `<img>` verwendet dieselbe Zahl als
  intrinsische Breite (1 Bild-Pixel = 1 CSS-Pixel ohne `srcset`/DPR-Deskriptoren). Kein
  Unterschied zwischen Platzhalter und Bild, auf Retina wie auf Non-Retina.
- **Bilder ohne Dims flackern weiter** → bewusst akzeptiert (Non-Goal); Backfill hat den
  Bestand abgedeckt. Der Gotcha-Absatz benennt das.
- **E2E-Timing: Platzhalter-Zählung bevor Route freigegeben wird** → `page.route` blockiert
  die Antworten hart; die Platzhalter stehen, solange die Route hält. Kein Wall-Clock-Fenster.
- **Andere `AuthImage`-Aufrufer** (Broadcast-Bild mit Dims, Lightbox ohne Dims) → Broadcast
  profitiert identisch (gleiche Blasenstruktur), Lightbox bleibt unverändert (keine Dims →
  kein `width`).

## Migration Plan

Reines Frontend-Bundle. Deploy = `make deploy`. Rollback = vorheriges Bundle, dann wieder
Flackern, kein Datenrisiko. Kein Feature-Flag.

## Open Questions

- Keine, die Spec, Ansatz oder Tasks ändern würden. Ob der Scroll-Anker nach diesem Fix für
  den `bottom`-Fall überhaupt noch korrigiert (statt No-op), ließe sich mit dem
  rAF-Sampling aus `chat-conversation-switch-flicker` messen — außerhalb dieses Changes.
