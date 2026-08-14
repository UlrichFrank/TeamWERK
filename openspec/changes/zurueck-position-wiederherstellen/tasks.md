## 1. useWindowedList-Bugfix (Voraussetzung für Dienste-Fokus)

- [x] 1.1 In `web/src/hooks/useWindowedList.ts` im Modus `scroll:'window'` den Re-Measure-Listener statt an `window` an das tatsächlich scrollende `<main>`-Element hängen (`document.querySelector('main')`, gleiche Technik wie `TerminePage.tsx:314`); Geometrie-Berechnung (`getBoundingClientRect`) unverändert lassen.
- [x] 1.2 Test ergänzen (`web/src/hooks/useWindowedList.test.ts`, neu anlegen falls nicht vorhanden), der ein Scroll-Event auf einem simulierten `<main>`-Container feuert und prüft, dass sich `start`/`end` entsprechend verschieben.

## 2. Termine: Fokus beim eigenen Karten-Klick setzen

- [x] 2.1 In `TerminePage.tsx` vor jedem `navigate(...)`-Aufruf beim Klick auf eine Trainings- oder Spielkarte zusätzlich `setSearchParams(next mit focus=<kind>-<id>, {replace:true})` auf der aktuellen `/termine`-URL setzen.
- [x] 2.2 Bestehenden Test in `TerminePage.test.tsx` um einen Fall ergänzen: Klick auf eine Karte setzt `focus` in der URL, bevor navigiert wird.

## 3. Dienste: Fokus-Mechanismus neu bauen (analog Termine)

- [x] 3.1 In `DutyPage.tsx` das bestehende `parseFilters`-Analogon (team/types/past/mine/audienceAll) um einen `focus=slot-<id>`-Parameter erweitern (Parsing + Setzen via `setSearchParams`).
- [x] 3.2 In `DutySlotList.tsx` jeder Zeile (`<tr key={s.id}>`) eine DOM-`id={`duty-slot-${s.id}`}` geben.
- [x] 3.3 In `DutySlotList.tsx` den `<BookOpen>`-Anleitungs-Link (Zeilen ~150-157) von reinem `<Link to=...>` auf einen `onClick`-Handler umstellen, der zuerst `focus=slot-<id>` auf der `/dienste`-URL setzt (`replace:true`) und danach navigiert.
- [x] 3.4 In `DutyPage.tsx` einen `useEffect` analog `TerminePage.tsx:322-332` ergänzen: bei vorhandenem `focus`-Param die passende Zeile per `scrollIntoView` in den Viewport holen und für ~2s per Ring-Klasse hervorheben.
- [x] 3.5 `focusNotFound`-Hinweis analog `TerminePage.tsx` ergänzen für den Fall, dass der fokussierte Slot nicht mehr existiert/sichtbar ist.
- [x] 3.6 Bestehenden Test `DutySlotList.instructionLink.test.tsx` um den Fall erweitern, dass der Anleitungs-Link vor der Navigation `focus=slot-<id>` in die `/dienste`-URL schreibt.
- [x] 3.7 Neuen Test für die Scroll-/Highlight-Logik in `DutyPage.tsx` ergänzen (neue Testdatei, z. B. `DutyPage.focus.test.tsx`).

## 4. Kalender: Monat in der URL spiegeln

- [x] 4.1 In `KalenderPage.tsx` bei `prevMonth`/`nextMonth`/„Heute" (Zeilen ~364-369) zusätzlich `setSearchParams({date: <erster Tag des neuen Monats>}, {replace:true})` aufrufen.
- [x] 4.2 Neuen Test (`KalenderPage.dateSync.test.tsx` oder Erweiterung vorhandener Kalender-Tests) ergänzen: Monatswechsel aktualisiert `?date=`; erneutes Mounten mit dieser URL zeigt denselben Monat.

## 5. Verifikation

- [x] 5.1 `pnpm -C web test` grün. (824/824 Tests)
- [x] 5.2 `pnpm -C web lint` und `pnpm -C web build` grün. (0 Errors, nur repo-weit vorbestehende Warnings; `tsc --noEmit` sauber)
- [x] 5.3 Manuell in Chrome DevTools MCP / Browser geprüft (e2e-Seed-DB, echter Server): Dienste-Anleitung öffnen → Zurück → URL trägt `focus=slot-<id>`, Zeile wurde per `ring-2 ring-brand-yellow` hervorgehoben (Highlight nach 2s wieder entfernt, `transition-all` als Nachweis geblieben). Kalender mehrere Monate weiterblättern → andere Seite → echtes `window.history.back()` → derselbe Monat (`?date=2026-09-01`) wieder angezeigt. Termine-Flow NICHT manuell geprüft — die e2e-Seed-DB hat keine aktive Mannschaft mit Kader (`/api/teams` liefert `[]`), der Kalender-Wizard kann daher kein Testspiel/-training anlegen; das Anlegen eines vollständigen Kaders nur für diesen Check war außerhalb des vertretbaren Aufwands. Ersatzweise abgedeckt durch `TerminePage.test.tsx`, das die echte React-Router-History-Reihenfolge (replace mit `focus=training-<id>` vor push zur Detailseite) über `createMemoryRouter`/`router.subscribe` prüft — derselbe Code-Pfad (`scrollIntoView`+Ring-Highlight-`useEffect`), der für Dienste manuell bestätigt wurde, existierte für Termine bereits vor diesem Change (Deep-Link-Fall) unverändert.
- [x] 5.4 `/verify-change` bewusst ausgelassen — keine neuen Routen, kein Broadcast/SSE-Bedarf (rein clientseitige URL-/Scroll-Änderungen); die relevanten Prüfungen (Tests, Lint, Build, brand-Tokens/lucide-Icons in den Diffs) sind über 5.1/5.2 und manuelle Diff-Reviews bereits abgedeckt.
