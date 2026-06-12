## Why

Die „Mein Team"-Seite listet bei Nutzern mit mehreren Teams (Eltern mit Kindern in verschiedenen Mannschaften, Trainer mehrerer Teams) Trainer, Spieler und Eltern jedes Teams untereinander auf. Die Seite wird dadurch sehr lang und schwer zu scannen. Drei Tabs pro Mannschaftskarte gruppieren die drei Kategorien und reduzieren die sichtbare Datenmenge auf einmal.

## What Changes

- Die `RosterSection`-Komponente in `MeinTeamPage.tsx` erhält eine Tab-Navigation mit drei Tabs: **Team**, **Trainer**, **Eltern**
- Jede Karte verwaltet ihren Tab-Zustand lokal (unabhängig von anderen Karten)
- Standard-Tab beim Öffnen: **Team** (Spielerliste)
- Leere Tabs werden angezeigt und zeigen einen Leertext (`— keine Einträge —`)
- Keine Änderungen an API, Backend oder anderen Komponenten

## Capabilities

### New Capabilities

- `roster-section-tabs`: Tab-Navigation innerhalb der RosterSection-Karte auf der Mein-Team-Seite

### Modified Capabilities

*(keine)*

## Impact

- Nur `web/src/pages/MeinTeamPage.tsx` wird geändert
- Keine API-Änderungen, keine neuen Dependencies
