## Why

Alle Termin-Kacheln und Kalender-Pills sehen aktuell identisch aus (gelber Rahmen, neutrales Grau). Nutzer müssen am Icon ablesen, um welchen Typ es sich handelt. Farbkodierung macht Termine auf einen Blick unterscheidbar und nutzt die vorhandene Brand-Palette konsequent.

## What Changes

- Neue Datei `web/src/lib/eventColors.ts` mit zentralem `EVENT_COLORS`-Mapping (vier Typen → Farb-Tokens für Kachel, Filter-Button, Kalender-Pill)
- `TerminePage.tsx`: Kacheln und Filter-Buttons erhalten typ-spezifische Farben
- `KalenderPage.tsx`: Event-Pills im Monatsraster erhalten typ-spezifische Farben; Training-Pills ersetzen bisher verwendetes `bg-blue-50`

## Capabilities

### New Capabilities

- `event-type-colors`: Farbkodierung der vier Event-Typen (Training, Heimspiel, Auswärtsspiel, Generisch) in Kacheln, Filter-Buttons und Kalender-Pills

### Modified Capabilities

*(keine bestehenden Specs ändern ihre Anforderungen)*

## Impact

- `web/src/lib/eventColors.ts` (neu)
- `web/src/pages/TerminePage.tsx` (Styling)
- `web/src/pages/KalenderPage.tsx` (Styling)
- Keine Backend-Änderungen, keine neuen Abhängigkeiten, keine Tailwind-Config-Änderungen
