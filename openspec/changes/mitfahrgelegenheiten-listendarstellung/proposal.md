## Why

Die Mitfahrgelegenheiten-Seite ist heute in drei Tabs (Auswärtsspiele / Heimspiele / Events) aufgeteilt. Das zwingt Nutzer, zwischen Tabs zu wechseln, um einen Überblick über alle Mitfahrt-Themen zu bekommen. Die Termine-Seite zeigt dagegen alle Events in einer einzigen chronologischen Liste mit Filter-Pills und farblicher Codierung — ein konsistenteres Muster, das auch hier passt: das nächste Spiel ist das relevanteste, unabhängig vom Event-Typ. Damit ergibt sich eine einheitliche visuelle Sprache zwischen Termine und Mitfahrgelegenheiten.

## What Changes

- **BREAKING (UI)**: Die drei Tabs (Auswärtsspiele / Heimspiele / Events) werden entfernt. Stattdessen erscheint eine einzige Liste aller Spiele.
- Header-Leiste analog zu `TerminePage`: Team-Select (mehrere Teams) + drei Event-Typ-Pills (Heim, Auswärts, Sonstiges) + Meine-Pill — **kein Training-Filter** (Mitfahrt gibt es nur für Spiele/Events).
- Game-Cards werden farblich kodiert via `getEventColors()`: Heim = gelb, Auswärts = grau, Sonstiges = blau.
- Liste ist primär chronologisch nach Datum + Uhrzeit sortiert, sekundär alphabetisch nach Team-Kürzel.
- Filter-State (Team, Typen, Meine) wird in URL-Search-Params persistiert — Deep-Linking und Reload-Stabilität wie auf Termine.
- "Meine"-Toggle wird zur Pill im gleichen Stil wie die Typ-Pills (eigenes Icon, z. B. `UserCheck`).
- Compact-Header (Icon-only Pills) bei < 950 px Viewport-Breite — gleiche Schwelle wie TerminePage.
- Vergangene Spiele bleiben weiterhin ausgeblendet (kein "Vergangene"-Toggle, da Mitfahrt für vergangene Termine sinnlos ist).
- GameCard-Innenleben (Biete/Suche-Spalten, Paarungen, In-Card-Tabs auf Mobile) bleibt unverändert.
- Backend (`GET /api/mitfahrgelegenheiten`) bleibt unverändert — liefert bereits `eventType` und `team`-Daten.

## Capabilities

### New Capabilities

_Keine — die Änderung modifiziert ausschließlich bestehende Capabilities._

### Modified Capabilities

- `mitfahrgelegenheiten-board`: Listendarstellung statt Tab-Aufteilung; neue Requirements für chronologische Sortierung, Farbcodierung via `getEventColors()`, Event-Typ-Pill-Filter und URL-Persistierung der Filter.
- `mitfahrgelegenheiten-meine-filter`: Der "Team | Meine"-Toggle wird zur Pill im selben Stil wie die Typ-Pills. Die "Tab-Counts"-Requirement entfällt vollständig, da es keine Tabs mehr gibt.

## Impact

- **Frontend**: `web/src/pages/MitfahrgelegenheitenPage.tsx` wird umfassend umgebaut (Header, Filter-Logik, Sortierung, Card-Farben). Importiert `getEventColors` aus `lib/eventColors.ts`, `buildTeamShortNames` aus `lib/teamName.ts` und `useCompactHeader` aus `hooks/useCompactHeader.ts` — alle existieren bereits.
- **Backend**: Keine Änderungen. `GET /api/mitfahrgelegenheiten` liefert bereits alle nötigen Felder (`eventType`, `team`).
- **Specs**: Zwei bestehende Specs werden modifiziert (Delta-Dateien). Keine neuen Capabilities.
- **Tests**: Rein Frontend-Aufgabe, keine neuen HTTP-Routen — der Test-Standard für neue Routen greift hier nicht. Kein neuer Test-Aufwand zwingend nötig.
- **Datenbank**: Keine Migration nötig.
- **Deep-Links**: Bestehende URLs ohne Search-Params funktionieren weiter (Default-Filter = alle Typen, Team-Modus). Alte Lesezeichen ohne `?tab=...` brechen nicht — die Tab-Param wurde nie als URL-State persistiert.
