# event-type-colors Specification

## Purpose
TBD - created by archiving change event-type-colors. Update Purpose after archive.
## Requirements
### Requirement: Event-Typen werden farbkodiert dargestellt
Das System SHALL jedem der vier Event-Typen eine eigene Farbe aus der Brand-Palette zuordnen und diese konsistent in Kacheln, Filter-Buttons und Kalender-Pills verwenden.

Farbzuordnung:
- Training → `brand-green` (#6EB42E)
- Heimspiel → `brand-yellow` (#FDE400)
- Auswärtsspiel → `brand-blue` (#3E4A98)
- Generisch → `brand-text-muted` (#6B7280) für Icon/Border, `brand-gray` für Hintergrund

#### Scenario: Termin-Kachel zeigt typ-spezifische Farbe
- **WHEN** die TerminePage eine Kachel für ein Training rendert
- **THEN** hat die Kachel einen grünen oberen Rand (`border-brand-green`), grünen Icon-Farbton und grün getönten Hintergrund (`bg-brand-green/10`)

#### Scenario: Heimspiel-Kachel zeigt gelbe Farbe
- **WHEN** die TerminePage eine Kachel für ein Heimspiel rendert
- **THEN** hat die Kachel einen gelben oberen Rand, gelben Icon-Farbton und hell-gelben Hintergrund

#### Scenario: Auswärtsspiel-Kachel zeigt blaue Farbe
- **WHEN** die TerminePage eine Kachel für ein Auswärtsspiel rendert
- **THEN** hat die Kachel einen blauen oberen Rand, blauen Icon-Farbton und blau getönten Hintergrund

#### Scenario: Abgesagtes Training behält neutrales Styling
- **WHEN** ein Training den Status `cancelled` hat
- **THEN** wird es mit `border-brand-border opacity-60` dargestellt, ohne typ-spezifischen Farbakzent

### Requirement: Filter-Buttons zeigen typ-spezifische Aktivfarbe
Die Filter-Buttons in TerminePage und KalenderPage SHALL im aktiven Zustand die dem jeweiligen Event-Typ zugeordnete Farbe statt der Standard-Aktivfarbe (`bg-brand-yellow`) anzeigen.

#### Scenario: Aktiver Training-Filter-Button
- **WHEN** der Filter-Button für „Training" aktiv ist
- **THEN** wird er mit `bg-brand-green text-white border-brand-green` dargestellt

#### Scenario: Aktiver Heimspiel-Filter-Button
- **WHEN** der Filter-Button für „Heim" aktiv ist
- **THEN** wird er mit `bg-brand-yellow text-brand-black border-brand-yellow` dargestellt

#### Scenario: Aktiver Auswärts-Filter-Button
- **WHEN** der Filter-Button für „Auswärts" aktiv ist
- **THEN** wird er mit `bg-brand-blue text-white border-brand-blue` dargestellt

#### Scenario: Aktiver Generisch-Filter-Button
- **WHEN** der Filter-Button für „Sonstiges" aktiv ist
- **THEN** wird er mit `bg-brand-gray text-brand-black border-brand-gray` dargestellt

### Requirement: Kalender-Pills zeigen typ-spezifische Farben
Die Event-Pills im Monatsraster der KalenderPage SHALL typ-spezifische Hintergrund- und Icon-Farben verwenden.

#### Scenario: Training-Pill im Kalender
- **WHEN** ein Training-Event als Pill im Monatsraster dargestellt wird
- **THEN** verwendet es grüne Hintergrund- und Icon-Farben statt `bg-blue-50/border-blue-200/text-blue-500`

#### Scenario: Spiel-Pill im Kalender nach Typ
- **WHEN** ein Spiel-Event als Pill im Monatsraster dargestellt wird
- **THEN** werden Hintergrund- und Icon-Farben basierend auf `event_type` (heim/auswärts/generisch) gewählt

