## Context

Der Scroll-Container ist app-weit `<main class="overflow-auto">` (`web/src/components/AppShell.tsx:442`), nicht `window` — native Browser-Scroll-Restoration greift deshalb nirgendwo. Der globale „Zurück"-Button (`AppShell.tsx:393/446`) ist ein echtes `navigate(-1)`, poppt also den realen Browser-History-Stack; die URL, die dabei wieder erscheint, ist exakt die vorherige.

`TerminePage.tsx` hat bereits eine funktionierende Fokus-Infrastruktur: `parseFilters()` liest `focus=<kind>-<id>` aus der URL (`TerminePage.tsx:111-125`), ein `useEffect` scrollt die passende Karte per `scrollIntoView` in den Viewport und legt für ~2s einen gelben Ring an (`TerminePage.tsx:322-332`). Bisher wird `focus` ausschließlich von außen gesetzt (Deep-Links aus Push-Notifications) — nie von der Seite selbst beim eigenen Karten-Klick.

`TerminePage.tsx`, `DutyPage.tsx` und `KalenderPage.tsx` halten ihre Filter bereits als URL-Suchparameter (`useSearchParams` + `setSearchParams(next, {replace:true})`); `KalenderPage.tsx` ist die Ausnahme — `year`/`month` werden nur einmalig beim Mount aus `?date=` gelesen (`initDate()`, Zeilen 148-156) und nie zurückgeschrieben.

`useWindowedList.ts` hat im Modus `scroll:'window'` (Zeilen 98-105) einen bestehenden Bug: der Re-Measure-Listener hängt an `window`, aber real scrollt `<main>`. Das Fenster (`start`/`end`) friert nach der ersten Messung ein — Zeilen außerhalb bleiben dauerhaft ungerendert, unabhängig von diesem Change. `DutySlotList.tsx` nutzt genau diesen Modus.

## Goals / Non-Goals

**Goals:**
- Zurück-Navigation von einer Detail-/Anleitungsseite zu `/termine` bzw. `/dienste` scrollt zur zuvor geöffneten Zeile und hebt sie hervor.
- Zurück-Navigation zu `/kalender` zeigt wieder den zuletzt betrachteten Monat.
- Die Lösung nutzt ausschließlich vorhandene URL-Suchparameter-Mechanik — kein neuer Speicher, kein neuer State-Layer.
- `useWindowedList` reagiert zuverlässig auf das tatsächliche Scrollen von `<main>`.

**Non-Goals:**
- Kein pixelgenaues Wiederherstellen der Scroll-Position unabhängig vom Inhalt (kein `sessionStorage`-basierter generischer Scroll-Hook — Option B aus der Exploration wurde bewusst verworfen).
- Keine Behandlung anderer Listenseiten (Mitglieder, Videos, Chat) — Scope ist auf `/termine`, `/dienste`, `/kalender` begrenzt.
- Kein Fix für potenzielle weitere `useWindowedList`-Nutzer außerhalb von `DutySlotList.tsx`; der Fix ändert das Verhalten des Hooks selbst, betrifft aber nur diesen einen Aufrufer im aktuellen Code.

## Decisions

**1. Fokus-in-URL statt generischer Scroll-Speicher.** Termine und Dienste bekommen den `focus=<kind>-<id>`-Parameter beim Klick selbst gesetzt (per `setSearchParams(next, {replace:true})` auf der aktuellen Listen-URL, bevor `navigate()` zur Detail-/Anleitungsseite aufgerufen wird). Dadurch trägt der History-Eintrag der Listenseite den Fokus schon in sich, und der bereits vorhandene (Termine) bzw. neu zu bauende (Dienste) Highlight-Mechanismus greift beim Zurücknavigieren automatisch — ohne zusätzlichen Speicher, ohne Race Conditions zwischen Unmount-Zeitpunkt und Scroll-Erfassung. Alternative wäre ein genereller `sessionStorage`-Scroll-Hook (Option B der Exploration) gewesen — verworfen, weil er blind gegenüber Listenänderungen ist (z. B. neuer Termin durch Live-Update oben eingefügt) und keine visuelle Bestätigung (Highlight) liefert, welches Element gemeint war.

**2. Dienste bekommt eine eigene, zur Termine-Logik parallele Implementierung, kein gemeinsamer Hook.** `TerminePage.tsx` unterscheidet `kind` (`training`/`game`), `DutyPage.tsx` nur `slot`-IDs — die Datenmodelle sind unterschiedlich genug (verschachtelte Gruppen bei Dienste vs. flache Liste bei Termine), dass eine vorzeitige gemeinsame Abstraktion mehr Komplexität einführen würde, als sie spart. Beide Implementierungen folgen aber demselben Muster (URL-Param lesen → `scrollIntoView` → Ring-Highlight-Klasse temporär setzen), sodass eine spätere Extraktion bei Bedarf leicht nachträglich möglich bleibt.

**3. `useWindowedList`-Fix: Listener an `<main>` statt `window`, Geometrie-Berechnung unverändert.** `getBoundingClientRect()` ist stets viewport-relativ und bleibt daher auch bei einem intern scrollenden `<main>` korrekt — nur die Event-Quelle für das Re-Measure ist falsch. Fix: im `scroll:'window'`-Modus das tatsächlich scrollende `<main>`-Element auflösen (`document.querySelector('main')`, gleiche Technik wie `TerminePage.tsx:314`) und daran `scroll`/`resize` binden statt an `window`. Kein Wechsel zu `scroll:'container'`, weil `DutySlotList.tsx` keinen eigenen Ref auf `<main>` hat (das Element gehört `AppShell`, nicht der Seite) — der Hook bleibt intern zuständig für die Auflösung.

**4. Kalender-Monat als `?date=`, geschrieben bei jeder Navigation.** Statt `year`/`month` weiterhin als reinen `useState` zu halten und nur beim Mount zu lesen, wird bei `prevMonth`/`nextMonth`/„Heute" zusätzlich `setSearchParams({date: <erster Tag des Monats>}, {replace:true})` aufgerufen — exakt das Muster, das `TerminePage.tsx`/`DutyPage.tsx` für ihre Filter bereits nutzen. `replace:true` verhindert, dass jeder einzelne Monatsklick einen eigenen History-Eintrag erzeugt (sonst müsste man ggf. mehrfach „Zurück" drücken, um `/kalender` zu verlassen).

## Risks / Trade-offs

- **[Risk]** Der fokussierte Termin/Slot könnte zwischen Hin- und Zurücknavigieren gelöscht oder aus dem sichtbaren Filter gefallen sein (z. B. Live-Update, Filterwechsel während der Detailansicht offen war). → **Mitigation:** Bereits vorhandenes Verhalten von `TerminePage.tsx` übernehmen: `focusNotFound`-Zustand zeigt einen dezenten Hinweis („Dieser Termin ist nicht verfügbar.") statt eines stillen Fehlschlags; für Dienste wird das analoge Verhalten übernommen.
- **[Risk]** `replace:true` bei jedem Kalender-Monatswechsel bedeutet, dass der Nutzer nicht per Browser-Vorwärts/Zurück durch einzelne Monate blättern kann (nur der zuletzt besuchte Monat ist in der Historie präsent). → **Mitigation:** Akzeptierter Trade-off, identisch zum bestehenden Verhalten der Filter auf Termine/Dienste (auch dort überschreibt `replace:true` die Historie) — Konsistenz mit etabliertem Muster hat Vorrang vor einer neuen Fähigkeit (Monats-Historie), die nicht angefragt wurde.
- **[Risk]** Der `useWindowedList`-Fix ändert das Verhalten eines gemeinsam genutzten Hooks. → **Mitigation:** `DutySlotList.tsx` ist aktuell der einzige Aufrufer mit `scroll:'window'`; die Geometrie-Formel bleibt unverändert, nur die Event-Bindung wird korrigiert — das Risiko einer Regression an anderer Stelle ist gering, wird aber im Testplan (`tasks.md`) explizit mit abgedeckt.

## Migration Plan

Keine Datenmigration. Rein clientseitige Änderung, mit dem nächsten Deploy sofort wirksam. Kein Rollback-Sonderfall — bei Problemen genügt ein Revert des Frontend-Commits.
