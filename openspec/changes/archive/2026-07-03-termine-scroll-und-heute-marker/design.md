## Context

Die `/termine`-Seite (`TerminePage.tsx`) rendert eine chronologisch sortierte Liste aus Trainings und Spielen. Der Toggle „Vergangene anzeigen" (`showPast`) erweitert das `from`-Fenster ein Jahr in die Vergangenheit und löst ein Neuladen aus (`useEffect([showPast])` → `load()` → `setLoading(true)`).

Es existiert bereits Scroll-Restaurierungs-Logik:
- `togglePast()` merkt sich beim Klick den obersten sichtbaren Termin (`[id^="termin-"]` mit `bottom > 0`) und dessen `getBoundingClientRect().top` in `scrollRestoreRef`.
- Ein `useEffect([loading, focus])` stellt nach dem Reload per `window.scrollBy({ top: el.getBoundingClientRect().top - restore.offset })` die Position wieder her.

**Der Bug:** Der Scrollcontainer ist nicht das `window`, sondern das `<main>` in `AppShell.tsx:388` (`class="flex-1 … overflow-auto …"` in einem `h-screen`-Flex-Layout). `window` scrollt nie, also ist `window.scrollBy` ein No-Op. Beim Umschalten wird die Liste außerdem durch „Laden…" ersetzt → `scrollHeight` von `<main>` kollabiert → Browser klemmt `scrollTop` auf 0 → sichtbarer Sprung nach oben.

## Goals / Non-Goals

**Goals:**
- Beim Vergangene-Toggle bleibt der zuvor oberste sichtbare Termin an derselben Viewport-Position.
- „— heute —"-Trennlinie vor dem ersten nicht-vergangenen Termin, nur wenn davor Vergangenes steht.

**Non-Goals:**
- Kein Umbau des AppShell-Scrollmodells (window vs. main-Container).
- Keine Änderung am Focus-Deep-Link-Verhalten (`?focus=…`), das `scrollIntoView` nutzt und weiterhin funktioniert.
- Keine Backend-Änderung, keine neuen Filter/Parameter.

## Decisions

### Entscheidung: Restaurierung auf `<main>` statt `window`

**Gewählt:** Die vorhandene Restaurierungs-Logik bleibt inhaltlich unverändert (Anchor + Viewport-Offset merken, nach Reload denselben Offset wiederherstellen). Getauscht wird nur das **Scrollziel**: Der Scrollcontainer wird zur Laufzeit über `el.closest('main')` ermittelt, und statt `window.scrollBy(...)` wird `container.scrollBy({ top: el.getBoundingClientRect().top - restore.offset, behavior: 'auto' })` aufgerufen.

**Warum funktioniert das:** `getBoundingClientRect().top` ist viewport-relativ und wird im Effect **live nach dem Layout** gemessen — unabhängig davon, ob `scrollTop` zwischenzeitlich auf 0 kollabiert ist. `scrollBy` ist relativ: Nach dem Re-Mount steht der Anchor bei seiner natürlichen `rect.top` (Container bei `scrollTop=0`), und `scrollBy(rect.top − savedOffset)` bringt ihn exakt auf `savedOffset` zurück. Neu eingeblendete vergangene Termine liegen dann oberhalb — der Nutzer scrollt selbst hoch.

**Alternative verworfen — Liste beim Toggle nicht unmounten:** Statt beim `showPast`-Reload auf „Laden…" umzuschalten, könnte man die alte Liste stehen lassen (dezenter Spinner), sodass `scrollTop` nie kollabiert und man nur den oben eingefügten Höhenversatz kompensiert. Robuster, aber größerer Eingriff in Loading-State und Render. Da der Minimal-Fix (Scrollziel korrigieren) das Symptom vollständig behebt und die bestehende Restaurierungs-Idee erhält, wird er bevorzugt.

**Edge Case:** War der Anchor selbst ein vergangener Termin und wird „Vergangene" wieder ausgeschaltet, existiert das Element nicht mehr (`if (!el) return`) → keine Restaurierung, Liste bleibt oben. Akzeptiert, da der betrachtete Termin ohnehin ausgeblendet wird.

### Entscheidung: „— heute —"-Divider, nur mit Vergangenem davor

**Gewählt:** Beim Rendern der `visibleTermine` wird der Index des ersten Termins mit `t.data.date.slice(0,10) >= today` bestimmt. Direkt vor diesem Termin wird ein nicht-anklickbarer Divider „— heute —" gerendert — **nur wenn dieser Index > 0** ist (also mindestens ein vergangener Termin darüber steht).

**Warum:** Bei ausgeblendeten Vergangenen ist der erste Termin bereits ≥ heute; ein Divider ganz oben wäre redundant. Die Bedingung `Index > 0` deckt beide Fälle sauber ab, ohne separat auf `showPast` zu prüfen.

**Konsistenz mit bestehendem Code:** `today` ist bereits als `new Date().toISOString().slice(0,10)` definiert (Z. 219) und wird für das `from`-Fenster genutzt — dieselbe Grundlage für den Vergleich. Der Divider trägt **kein** `id="termin-…"`, stört also die Anchor-Suche in `togglePast` nicht.

## Risks / Trade-offs

- [Sichtbarer Kurz-Flash bei Reload] Zwischen „Laden…" und Restaurierung kann die Liste kurz oben stehen. `behavior:'auto'` macht die Korrektur instant; akzeptabel und wie im Bestand.
- [TZ-Kante bei `toISOString()` (UTC)] `today` ist UTC-basiert wie der bestehende `from`/`to`-Code; für Datumsvergleiche auf Tagesebene in DE praktisch unkritisch und konsistent mit dem vorhandenen Verhalten.
