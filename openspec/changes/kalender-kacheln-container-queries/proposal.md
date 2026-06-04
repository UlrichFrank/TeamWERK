## Why

Auf kleinen Bildschirmen sind die Kalender-Kacheln zu eng für drei Textzeilen — der 7-Spalten-Raster bleibt aber gewünscht. Die Kacheln sollen sich an ihre tatsächlich verfügbare Breite anpassen, nicht an den Viewport, weil die Zellbreite allein von der Kalenderbreite abhängt.

## What Changes

- Kalender-Kacheln (Spiele und Trainings) reagieren mit CSS Container Queries auf die Breite ihrer Zelle
- Drei Darstellungsstufen: Icon-only / Icon + Team + Zeit / volle 3-Zeilen-Ansicht (wie heute)
- Die Kalender-Zelle wird als Container deklariert (`@container`)
- Tailwind v3 Container-Query-Plugin (`@tailwindcss/container-queries`) wird eingebunden
- Breakpoints: `< 80px` → Stufe 1, `80–119px` → Stufe 2, `≥ 120px` → Stufe 3

## Capabilities

### New Capabilities

- `kalender-kachel-groessen`: Drei adaptive Darstellungsstufen für Kalender-Kacheln basierend auf Container-Breite

### Modified Capabilities

_(keine Anforderungsänderungen an bestehenden Specs)_

## Impact

- `web/src/pages/KalenderPage.tsx`: Kachel-Markup, Container-Klasse auf Tageszellen
- `web/tailwind.config.js`: Plugin `@tailwindcss/container-queries` hinzufügen
- `web/package.json`: neue Dev-Dependency `@tailwindcss/container-queries`
- Kein Backend-Änderungsbedarf
