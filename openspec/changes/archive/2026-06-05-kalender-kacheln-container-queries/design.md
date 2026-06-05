## Context

Der Monatskalender zeigt ein fixes 7-Spalten-Raster. Auf Desktop (≥768px) sind Zellen ~110px breit — genug für drei Textzeilen. Auf kleinen Displays (360px) schrumpfen sie auf ~51px, wo drei Zeilen nicht mehr lesbar sind. Die Kacheln sollen sich an die tatsächliche Zellbreite anpassen, nicht am Viewport messen.

CSS Container Queries lösen dieses Problem präzise: Eine Zelle wird als Container deklariert, Kinder reagieren auf dessen Breite.

```
Zellbreite   Darstellung
──────────   ───────────────────────────────────
< 80px       Stufe 1: nur Icon (w-3 h-3)
80–119px     Stufe 2: Icon + Kurzname + Uhrzeit
≥ 120px      Stufe 3: volle 3 Zeilen (Status quo)
```

## Goals / Non-Goals

**Goals:**
- Drei Darstellungsstufen, gesteuert durch Zellbreite
- Keine Logik-Änderung — Click-Handler, Farben, Daten bleiben unverändert
- Kein zusätzlicher JavaScript-Overhead (rein CSS)

**Non-Goals:**
- Mobile-Modal oder Peek-Layer (separates Vorhaben)
- Änderung des Routing/Navigation bei Klick
- Unterstützung von IE/älteren Browsern (Container Queries: alle modernen Browser seit 2023)

## Decisions

### Tailwind Container Query Plugin statt raw CSS

`@tailwindcss/container-queries` liefert `@lg:`, `@sm:` etc. als Utility-Klassen. Alternative wäre manuelles `@container` CSS in `index.css`.

**Entscheidung:** Plugin, weil es zum bestehenden Tailwind-Workflow passt und keine separaten CSS-Dateien einführt.

Breakpoints (custom, via `tailwind.config.js`):
```js
containers: { 'tile-sm': '80px', 'tile-md': '120px' }
```
Verwendung im JSX: `@tile-sm:block @tile-md:hidden` etc.

### Container auf der Tageszelle, nicht auf der Kachel

Die Tageszelle (`<div key={day}>`) wird Container — nicht jede einzelne Kachel. Grund: Eine Kachel nimmt `w-full` ein, ihr eigener Container-Query würde immer die Breite des Eltern-Divs messen. Die Zelle ist das stabile Messobjekt.

```tsx
// Tageszelle erhält: className="... @container"
// Kachel-Elemente: className="hidden @tile-sm:flex ..."
```

### Stufe 1 zeigt nur das Icon mit ARIA-Label

Auf `< 80px` ist kein Text lesbar. Das Icon allein (Home/MapPin/Dumbbell) vermittelt den Typ; Farbe und Border vermitteln die Mannschaft/den Typ. Ein `title`-Attribut gibt Screen-Readern und Hover-Tooltips die Vollinfo.

## Risks / Trade-offs

- **Plugin-Dependency** → bei Tailwind v4-Migration prüfen (Container Queries sind dort nativ, Plugin entfällt dann)
- **Breakpoint-Werte sind Schätzungen** → nach ersten Tests auf echten Geräten ggf. anpassen; Werte liegen nur in `tailwind.config.js`, leicht änderbar
- **Swipe-Geste auf Mobile** bleibt unverändert (pointer-Events auf der Kalender-Ebene, nicht auf den Kacheln)
