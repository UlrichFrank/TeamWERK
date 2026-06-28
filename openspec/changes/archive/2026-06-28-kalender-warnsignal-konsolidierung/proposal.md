## Why

Im Kalender-Grid können bei einem Spiel-Tile bis zu drei Aufmerksamkeits-Signale gleichzeitig erscheinen (AlertTriangle für offene Dienst-Slots, EventNoteIndicator-Icon für manuelle Hinweistexte, farbiger Duty-Dot für Slot-Füllstand), obwohl sie alle dasselbe „schau hier nochmal hin" bedeuten. Das verwirrt und kostet auf kleinen Mobile-Tiles wertvollen Platz. Zusätzlich fehlt im Detail-Modal (/termine) der Hinweis auf offene Slots komplett.

## What Changes

- **Tile:** Pro Spiel-Tile nur noch **ein** AlertTriangle, das erscheint wenn `note.trim() !== ''` **oder** `filled_count < total_count`; Tooltip/aria-label enthält beide Gründe falls zutreffend.
- **Tile:** Farbigen Duty-Dot (Slot-Füllstand-Kreis) entfernen.
- **Tile:** `EventNoteIndicator`-Icon aus der unteren Zeile der Spiel-Tiles entfernen (Signal ist im konsolidierten AlertTriangle aufgegangen).
- **Training-Tiles:** Unverändert (haben keinen Duty-Dot, `EventNoteIndicator` bleibt).
- **EventInfoModal:** `Game`-Interface um `slot_count`, `filled_count`, `total_count` erweitern; unter dem bestehenden Hinweistext eine generierte Zeile „X offene Dienst-Slots" ergänzen — visuell abgesetzt, kein zusätzliches Icon.

## Capabilities

### New Capabilities

- `kalender-warnsignal`: Konsolidiertes Warn-Signal für Spiel-Tiles im Kalender (ein Icon, kombinierter Tooltip) sowie generierte Slot-Info im Detail-Modal.

### Modified Capabilities

*(keine Änderungen an bestehenden Specs)*

## Impact

- `web/src/pages/KalenderPage.tsx` — Tile-Rendering für Games
- `web/src/components/EventInfoModal.tsx` — Game-Interface + Note-Sektion
- Keine API-Änderungen, keine neuen Migrationen, keine neuen Abhängigkeiten
