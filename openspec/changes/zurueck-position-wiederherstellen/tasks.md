## 1. useWindowedList-Bugfix (Voraussetzung für Dienste-Fokus)

- [x] 1.1 In `web/src/hooks/useWindowedList.ts` im Modus `scroll:'window'` den Re-Measure-Listener statt an `window` an das tatsächlich scrollende `<main>`-Element hängen (`document.querySelector('main')`, gleiche Technik wie `TerminePage.tsx:314`); Geometrie-Berechnung (`getBoundingClientRect`) unverändert lassen.
- [x] 1.2 Test ergänzen (`web/src/hooks/useWindowedList.test.ts`, neu anlegen falls nicht vorhanden), der ein Scroll-Event auf einem simulierten `<main>`-Container feuert und prüft, dass sich `start`/`end` entsprechend verschieben.

## 2. Termine: Fokus beim eigenen Karten-Klick setzen

- [ ] 2.1 In `TerminePage.tsx` vor jedem `navigate(...)`-Aufruf beim Klick auf eine Trainings- oder Spielkarte zusätzlich `setSearchParams(next mit focus=<kind>-<id>, {replace:true})` auf der aktuellen `/termine`-URL setzen.
- [ ] 2.2 Bestehenden Test in `TerminePage.test.tsx` um einen Fall ergänzen: Klick auf eine Karte setzt `focus` in der URL, bevor navigiert wird.

## 3. Dienste: Fokus-Mechanismus neu bauen (analog Termine)

- [ ] 3.1 In `DutyPage.tsx` das bestehende `parseFilters`-Analogon (team/types/past/mine/audienceAll) um einen `focus=slot-<id>`-Parameter erweitern (Parsing + Setzen via `setSearchParams`).
- [ ] 3.2 In `DutySlotList.tsx` jeder Zeile (`<tr key={s.id}>`) eine DOM-`id={`duty-slot-${s.id}`}` geben.
- [ ] 3.3 In `DutySlotList.tsx` den `<BookOpen>`-Anleitungs-Link (Zeilen ~150-157) von reinem `<Link to=...>` auf einen `onClick`-Handler umstellen, der zuerst `focus=slot-<id>` auf der `/dienste`-URL setzt (`replace:true`) und danach navigiert.
- [ ] 3.4 In `DutyPage.tsx` einen `useEffect` analog `TerminePage.tsx:322-332` ergänzen: bei vorhandenem `focus`-Param die passende Zeile per `scrollIntoView` in den Viewport holen und für ~2s per Ring-Klasse hervorheben.
- [ ] 3.5 `focusNotFound`-Hinweis analog `TerminePage.tsx` ergänzen für den Fall, dass der fokussierte Slot nicht mehr existiert/sichtbar ist.
- [ ] 3.6 Bestehenden Test `DutySlotList.instructionLink.test.tsx` um den Fall erweitern, dass der Anleitungs-Link vor der Navigation `focus=slot-<id>` in die `/dienste`-URL schreibt.
- [ ] 3.7 Neuen Test für die Scroll-/Highlight-Logik in `DutyPage.tsx` ergänzen (neue Testdatei, z. B. `DutyPage.focus.test.tsx`).

## 4. Kalender: Monat in der URL spiegeln

- [ ] 4.1 In `KalenderPage.tsx` bei `prevMonth`/`nextMonth`/„Heute" (Zeilen ~364-369) zusätzlich `setSearchParams({date: <erster Tag des neuen Monats>}, {replace:true})` aufrufen.
- [ ] 4.2 Neuen Test (`KalenderPage.dateSync.test.tsx` oder Erweiterung vorhandener Kalender-Tests) ergänzen: Monatswechsel aktualisiert `?date=`; erneutes Mounten mit dieser URL zeigt denselben Monat.

## 5. Verifikation

- [ ] 5.1 `pnpm -C web test` grün.
- [ ] 5.2 `pnpm -C web lint` und `pnpm -C web build` grün.
- [ ] 5.3 Manuell in Chrome DevTools MCP / Browser prüfen: Termine-Karte anklicken → Detailseite → Zurück → Karte ist sichtbar & hervorgehoben. Gleiches für Dienste-Anleitung. Kalender mehrere Monate weiterblättern → andere Seite → Zurück → gleicher Monat.
- [ ] 5.4 `/verify-change` durchlaufen lassen (Route→Tests, brand-Tokens, lucide-Icons — hier keine neuen Routen, aber Konventions-Check schadet nicht).
