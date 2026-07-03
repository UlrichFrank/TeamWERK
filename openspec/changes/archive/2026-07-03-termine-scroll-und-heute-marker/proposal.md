## Why

Auf `/termine` springt die Liste beim Umschalten von „Vergangene anzeigen" ganz nach oben, statt die aktuell betrachtete Position zu halten. Es gibt zwar bereits eine Scroll-Restaurierung im Code (`togglePast` + `scrollRestoreRef`), sie **wirkt aber nicht**: Der eigentliche Scrollcontainer ist das `<main>` in `AppShell` (`overflow-auto`), nicht das `window`. Die Restaurierung ruft jedoch `window.scrollBy(...)` auf — ein No-Op, weil `window` nie scrollt. Beim Umschalten kollabiert zudem die `scrollHeight` des `<main>`, während die Liste kurz durch „Laden…" ersetzt ist, sodass der Browser `scrollTop` auf 0 klemmt.

Zusätzlich fehlt eine visuelle Orientierung, wo die Vergangenheit endet und die Zukunft beginnt, sobald vergangene Termine eingeblendet sind.

## What Changes

- **Scroll-Position beibehalten** beim Umschalten von „Vergangene anzeigen": Der zuvor oberste sichtbare Termin bleibt an derselben Viewport-Position; neu eingeblendete vergangene Termine erscheinen darüber (der Nutzer scrollt selbst hoch, um sie zu sehen). Fix: Restaurierung operiert auf dem Scrollcontainer `<main>` (`el.closest('main')`) statt auf `window`.
- **Trennlinie „— heute —"** vor dem ersten Termin, der nicht in der Vergangenheit liegt (`date >= today`). Die Linie erscheint nur, wenn davor mindestens ein (vergangener) Termin steht — nicht redundant am Listenanfang.

Reine Frontend-Änderung in `web/src/pages/TerminePage.tsx`. Keine neuen Routen, keine Backend-Änderung.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `termine-unified-view`: neue Anforderungen für Scroll-Erhalt beim Vergangene-Toggle und für die „heute"-Trennlinie.

## Impact

- `web/src/pages/TerminePage.tsx`
  - Scroll-Restaurierungs-Effect (Z. 279–287) und ggf. `togglePast` (Z. 289–296): Scrollziel von `window` auf `el.closest('main')` umstellen.
  - Render der Liste (Z. 440–626): „— heute —"-Divider vor dem ersten Termin mit `date >= today` einfügen (nur wenn Index > 0).
- `web/src/pages/__tests__/TerminePage.permissions.test.tsx` bzw. neue Testdatei — Verhalten des „heute"-Dividers absichern.
