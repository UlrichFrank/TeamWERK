## Why

Beim Verwalten von Kadern ist es schwierig, auf einen Blick zu sehen, welche Positionen unterbesetzt sind. Trainers und Admins müssen die Member-Liste durchscannen, um zu zählen, wie viele Spieler jede Position spielen können. Eine visuelle Position-Status-Anzeige ermöglicht schnelle Überblick-Diagnostik ohne Manual-Counting.

## What Changes

- AdminKaderPage zeigt für jede Handball-Position kompakte visuelle Status-Indikatoren zwischen dem „Jahrgänge"-Toggle und der „Trainer"-Suche
- Jede Position wird mit einer Abkürzung (TW, LA, RA, RL, RM, RR, KL) und Status-Kreisen angezeigt:
  - 1 roter Kreis: 0 Spieler mit dieser Position
  - 1 gelber Kreis: 1 Spieler
  - 2 grüne Kreise: 2 Spieler
  - 3 blaue Kreise: 3+ Spieler
- Keine API-Änderungen nötig (Position-Daten existieren bereits auf Members)

## Capabilities

### New Capabilities

- `position-occupancy-display`: Visuelle Anzeige der Position-Besetzung pro Kader mit Status-Indikatoren auf der AdminKaderPage

### Modified Capabilities

- (keine)

## Impact

- **Frontend:** AdminKaderPage.tsx (React-Komponente)
- **Keine API-Änderungen:** Position-Daten werden aus Member-Objekten gelesen (bereits vorhanden)
- **Styling:** Compact Circles, sehr kleine Größe (wenig Platz-Verschwendung)
- **Rollen:** Nur Admin sieht diese Seite — keine Rollen-Impacts
