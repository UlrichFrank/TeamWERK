## Context

`TerminMatrix.tsx` rendert die Tabelle in `Card (overflow-hidden) > div.overflow-x-auto > table.border-separate`. Die Namensspalte ist `sticky left-0 z-10` und funktioniert, weil sie an den horizontalen Scroll des `overflow-x-auto`-Wrappers gebunden ist.

Den vertikalen Scroll macht dagegen `<main>` in `AppShell.tsx` (`overflow-auto`, Kommentar dort: „Scroll-Container der App ist dieses <main>"). Ein `position: sticky; top: 0` am `<thead>` greift nicht gegen `<main>`: `overflow-x: auto` erzwingt per CSS-Spezifikation `overflow-y: auto` (computed) am Wrapper, der Wrapper wird damit zum nächsten Scroll-Vorfahren des Kopfes — und scrollt selbst vertikal nie, weil er so hoch ist wie die Tabelle. Ergebnis: `sticky top-0` hätte keinen sichtbaren Effekt. Das ist der Grund, warum die Titelzeile heute nicht stehen bleibt, obwohl die Spalte es tut.

## Goals / Non-Goals

**Goals:**
- Titelzeile bleibt beim vertikalen Scrollen stehen, Namensspalte weiter beim horizontalen, Ecke „Spieler" in beide Richtungen.
- Nur CSS/Markup in `TerminMatrix.tsx`, keine Scroll-Listener.

**Non-Goals:**
- Keine fixierte Zeile für die Legende/Zählzeile über der Tabelle.
- Keine Virtualisierung der Zeilen (Kader < 40 Zeilen).
- Keine Übertragung des Musters auf andere Tabellen (`Spielmatrix`, `CrossTable`) — eigener Change, falls gewünscht.

## Decisions

**1. Die Tabelle bekommt einen eigenen Scrollbereich in beiden Achsen mit begrenzter Höhe.**
Der Wrapper wird von `overflow-x-auto` zu `overflow-auto` mit `max-height` relativ zum Sichtbereich: `max-h-[calc(100dvh-13rem)]`, einheitlich für alle Breiten. Maßgeblich ist die Legende **unter** der Tabelle: ist `<main>` ganz nach unten gescrollt (das passiert auch, wenn eine Wischgeste am Tabellenende an `<main>` weitergereicht wird), muss der Kopf trotzdem im Bild bleiben. Der Startwert `8rem` fiel dabei zweimal durch (live gemessen): mobil 390×844 (App-Kopfleiste + fünfzeilige Legende) lag der Kopf 57 px unter der Kopfleiste, bei 640 px mit Seitenleiste (dreizeilige Legende) 12 px über dem Rand. Mit `13rem` bleibt der Kopf bei 360×740, 390×844, 640×800 und 1440×830 sichtbar; ein Breakpoint-Split wäre auf dem breiten Desktop 80 px höher, aber an der `sm`-Grenze wieder falsch gewesen. Damit ist der Wrapper der Scroll-Container für **beide** Sticky-Achsen, und `sticky top-0` am Kopf wirkt.
Eine kurze Tabelle (wenige Spieler) bleibt unverändert, weil `max-height` erst bei Überlänge greift.

*Alternativen:*
- *Wrapper ohne Overflow, `<main>` scrollt horizontal mit*: dann liefe die ganze Seite (Kopfzeile, Filter) horizontal mit — inakzeptabel.
- *`overflow-x: clip`*: verhindert den Scroll-Container, schneidet aber ab statt zu scrollen — horizontales Scrollen wäre weg.
- *JS-Klon des Kopfes, per Scroll-Listener an `<main>` fixiert und horizontal synchronisiert*: funktioniert ohne verschachtelten Scrollbereich, verdoppelt aber das Kopf-Markup (Links, `aria`-Labels doppelt für Screenreader), braucht Breitenabgleich je Spalte nach jedem Datenwechsel und ist genau die Bug-Klasse (Scroll/Layout in iOS Safari), die jsdom nicht sieht. Unverhältnismäßig für den Nutzen.

**2. Z-Ebenen: Zeilenköpfe `z-10`, Spaltenköpfe `z-20`, Ecke „Spieler" `z-30`.**
Alle `<th>` in `<thead>` bekommen `sticky top-0`; die Ecke `sticky top-0 left-0`. Hintergründe sind bereits deckend (`bg-brand-surface-card`), sonst schienen die Zellen durch. `border-separate border-spacing-0` bleibt — mit `border-collapse` verlieren sticky Zellen in Chromium ihre Rahmen.

**3. `overflow-hidden` der Karte bleibt.**
Es sitzt am Eltern-Element des Scroll-Containers und beeinflusst die Sticky-Bezugsfläche nicht mehr, weil der innere Wrapper jetzt selbst in beiden Achsen scrollt. Es hält weiterhin die abgerundeten Ecken sauber. `transform-gpu` (Haarlinien-Gotcha) bleibt ebenfalls; `transform` am Vorfahren stört `position: sticky` nicht.

## Risks / Trade-offs

- [Verschachtelter Scrollbereich: am Tabellenende scrollt die Seite erst nach einer zweiten Geste weiter (Scroll-Chaining)] → Standardverhalten von Browsern; durch die Begrenzung auf die Sichtbereichshöhe ist die Tabelle ohnehin der einzige relevante Inhalt der Ansicht. Kein `overscroll-contain`, damit das Weiterreichen an `<main>` erhalten bleibt.
- [Scroll-Wiederherstellung (`useScrollRestoration`) kennt nur `<main>`, nicht den neuen Tabellen-Scroll] → Beim Zurück von der Termin-Detailseite startet die Tabelle oben. Akzeptiert; wie heute beim horizontalen Scroll.
- [`dvh` auf alten Browsern] → Fallback per Tailwind-Arbitrary-Wert ist nicht nötig, iOS ≥ 15.4 / Chromium ≥ 108 unterstützen `dvh`; notfalls greift ohne `max-height` das heutige Verhalten (kein Bruch, nur keine fixierte Zeile).
- [jsdom sieht Sticky-Verhalten nicht] → Vitest prüft nur die Klassen-Invarianten (Scroll-Container in beiden Achsen, Kopf sticky, Z-Reihenfolge); das eigentliche Verhalten wird live in Chrome (DevTools MCP, Desktop- und Mobil-Viewport) geprüft.

## Migration Plan

Reines Frontend-Deploy, kein Datenschritt. Rollback per `make deploy-rollback`.
