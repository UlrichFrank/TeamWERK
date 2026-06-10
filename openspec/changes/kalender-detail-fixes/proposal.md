## Why

Das `EventInfoModal` im Kalender zeigt für mehrtägige generische Events nur das Startdatum, obwohl `end_date` vorhanden ist. Außerdem wird das Feld für den Event-Namen fälschlicherweise als "Gegner" beschriftet, und alle Detail-Ansichten (Spiele, Trainings) zeigen die betroffenen Mannschaften nicht an.

## What Changes

- **Mehrtägige generische Events**: Im Detail-Modal wird bei vorhandenem `end_date` eine Datumsrange angezeigt (z.B. "7. September – 10. September 2026") statt nur dem Startdatum.
- **Label-Fix für generische Events**: Das Feld `opponent` wird für generische Events mit "Event-Name" beschriftet statt "Gegner".
- **Mannschaften in allen Detail-Ansichten**: Im Detail-Modal wird eine Zeile "Team(s)" mit den Kurznamen der betroffenen Mannschaften angezeigt — für Heim-/Auswärtsspiele, generische Events und Einzeltrainings.

## Capabilities

### New Capabilities

_(keine neuen Capabilities)_

### Modified Capabilities

- `event-info-modal`: Datum-Range für mehrtägige Events, korrektes Label für generische Events, Team-Anzeige in allen Detail-Varianten.

## Impact

- `web/src/components/EventInfoModal.tsx`: Interface-Erweiterungen + Rendering-Logik
- `web/src/pages/KalenderPage.tsx`: Übergabe von `end_date`, `teams` und `team_name` an EventInfoModal
- Kein Backend-Eingriff nötig — alle benötigten Daten (`end_date`, `teams`) sind bereits im API-Response vorhanden
