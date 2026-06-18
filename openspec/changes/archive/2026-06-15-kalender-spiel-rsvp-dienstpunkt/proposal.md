## Why

Spiel-Kacheln im Kalender zeigen bisher keine RSVP-Zähler (Zu-/Absagen), obwohl das Backend `confirmed_count` und `declined_count` bereits liefert. Trainingskacheln zeigen diese Zahlen bereits. Außerdem ist der Dienst-Punkt unergonomisch in der Uhrzeitzeile versteckt — er gehört sichtbar in die Teamname-Zeile.

## What Changes

- Spiel-Pill (heim, auswärts, generisch): erste Zeile zeigt `[Icon] [Teamname flex-1] [●Dienst-Punkt]` — Punkt rechts, nur ab `@tile-sm` (80 px Kachelbreite)
- Dienst-Punkt wird aus der Uhrzeitzeile entfernt
- Uhrzeitzeile erhält RSVP-Zähler: `[Zeit] [✓N] [✗N]` identisch zum Trainings-Muster, versteckt unter `@tile-sm`
- Trainings-Kacheln bleiben unverändert (haben keinen Dienst-Punkt)

## Capabilities

### New Capabilities

- `kalender-spiel-rsvp`: Spiel-Kacheln im Kalender zeigen Zu-/Absage-Zähler und einen neu positionierten Dienst-Punkt

### Modified Capabilities

*(keine bestehenden Spec-Level-Anforderungen ändern sich)*

## Impact

- Nur Frontend: `web/src/pages/KalenderPage.tsx`, Zeilen ~842–870 (game pill render)
- Kein Backend-Change, keine neuen Abhängigkeiten
- Keine API-Änderungen
