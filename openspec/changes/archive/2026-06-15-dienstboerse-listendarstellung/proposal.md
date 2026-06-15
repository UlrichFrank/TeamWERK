## Why

Die Dienstbörse (`/dienste`) zeigt heute eine schlichte chronologische Liste mit zwei Eigenheiten, die nicht zum übrigen App-Erlebnis passen:

1. Alle Karten sind uniform gelb umrandet — der Spielcharakter (Heim / Auswärts / Sonstiges) ist auf den ersten Blick nicht erkennbar. Die `TerminePage` und die kürzlich umgebaute `MitfahrgelegenheitenPage` codieren denselben Spieltyp bereits farblich über `getEventColors()`.
2. Die Filterung ist binär (Alle Dienste / Meine Dienste) und nur für Trainer/Admin sichtbar. Es gibt keinen Team-Filter und keine Event-Typ-Filter — selbst auf der Bewegtbild-Mobile-Ansicht muss der Nutzer durch lange Listen scrollen, um „nur Heimspiele bei meinem Sohn" zu finden.

Mit dem Umbau wird die Dienstbörse visuell und funktional an die `TerminePage` und `MitfahrgelegenheitenPage` angeglichen. Ein einheitliches Vokabular über alle drei Listen-Seiten (Team-Select + Event-Typ-Pills + Meine-Pill + Vergangene-Pill) reduziert die kognitive Last und macht das Filtern unmittelbar.

## What Changes

- **BREAKING (UI)**: Die binäre Toggle-Leiste „Alle Dienste / Meine Dienste" und der Text-Link „Vergangene einblenden" werden entfernt. Stattdessen erscheint eine einheitliche Pill-Leiste analog zu `TerminePage` und `MitfahrgelegenheitenPage`.
- **Team-Select** (Dropdown) als erstes Element der Filter-Leiste — Default „Alle Teams".
- **Drei Event-Typ-Pills**: „Heim" 🏠, „Auswärts" ✈, „Sonstiges" 📅 — Mehrfachauswahl möglich, Default alle aktiv. **Kein** Trainings-Filter (Trainings-Dienste gibt es nicht).
- **Meine-Pill** (`UserCheck`-Icon) — für alle Rollen sichtbar (heute nur Trainer/Admin). Zeigt Slots, in denen der Nutzer selbst eingetragen ist.
- **Vergangene-Pill** (`History`-Icon) ersetzt den heutigen Text-Link — gleiche Optik wie auf `TerminePage`.
- **Karten-Farbcodierung** via `getEventColors(event_type)`: Heim = gelb, Auswärts = grau, Sonstiges = blau. Vergangene Gruppen behalten den heutigen Past-Override (`bg-brand-surface-card border-brand-border opacity-60`) — Past schlägt Farbe.
- **„Sonstige Dienste"** (game_id IS NULL — z. B. Vereinsfest-Aufbau, Sonderaktionen) werden als `event_type=generisch` zurückgegeben und fallen damit unter den „Sonstiges"-Filter. Bisher kam für game-lose Gruppen ein leerer `event_type`-String.
- **URL-Persistence**: Team, Event-Typen, Meine und Vergangene werden in URL-Search-Params (`?team=`, `?types=`, `?mine=1`, `?past=1`) gespeichert. Default-State = saubere URL.
- **Compact-Header** (`useCompactHeader(950)`): unter 950 px Viewport-Breite zeigen die Pills nur Icons.
- **Team-Zugriff erweitert für Vorstand**: heute sieht nur die System-Rolle `admin` *alle* Dienste. Künftig bypasst auch die Vereinsfunktion `vorstand` den Team-Filter — analog zu `GET /api/mitfahrgelegenheiten`. Alle anderen Rollen sehen weiterhin nur die Dienste ihrer (eigenen oder über Kader zugeordneten) Teams.
- **Backend-Response erweitert**: `GET /api/duty-board` liefert pro Gruppe zusätzlich `team_id` (für den Team-Filter) und für game-lose Gruppen `event_type: "generisch"`.

## Capabilities

### New Capabilities

_Keine — die Änderung modifiziert ausschließlich bestehende Capabilities._

### Modified Capabilities

- `duties`: ADDED Requirements für chronologische Listendarstellung, Event-Typ-Pill-Filter, Team-Filter, Meine-Pill (für alle Rollen), Vergangene-Pill, Farbcodierung via `getEventColors()`, URL-Persistierung, Compact-Header. Außerdem MODIFIED am bestehenden „Duty board"-Requirement: erweiterte Audienz für `vorstand`, ergänzte Response-Felder (`team_id`, normalisiertes `event_type`).

## Impact

- **Frontend**: `web/src/pages/DutyPage.tsx` wird umfassend umgebaut (Header-Leiste, Filter-Logik, Sortierung, Card-Farben). Importiert `getEventColors` aus `lib/eventColors.ts`, `buildTeamShortNames` aus `lib/teamName.ts` und `useCompactHeader` aus `hooks/useCompactHeader.ts` — alle existieren bereits.
- **Backend**: Leichte Erweiterung in `internal/duties/handler.go` (`Board`):
  - Audienz-Bypass um Vereinsfunktion `vorstand` ergänzen (heute nur System-Rolle `admin`).
  - In der Response-Struct `boardGroup` zwei zusätzliche Felder: `TeamID *int` und für game-lose Gruppen `EventType: "generisch"` statt leerer String.
- **Specs**: Eine bestehende Capability (`duties`) wird modifiziert.
- **Tests**: Backend-Erweiterung erfordert neue Tests:
  - `TestBoard_VorstandSeesAllTeams` — Nutzer mit Vereinsfunktion `vorstand` sieht Slots aller Teams (bisher nur `admin`).
  - `TestBoard_GameIDNullGroupHasGenericEventType` — game-lose Gruppen kommen mit `event_type=generisch` zurück.
  - `TestBoard_GroupContainsTeamID` — jede Gruppe enthält `team_id` (auch game-lose).
- **Datenbank**: Keine Migration nötig. Die `user_accessible_teams`-View existiert bereits und wird vom Carpooling-Endpoint bereits genutzt — sie ist hier nicht zwingend nötig (die heutige Subquery in der Board-Query erfüllt denselben Zweck), könnte aber im Zuge dieses Changes optional konsolidiert werden (nicht im Scope).
- **Deep-Links**: Bestehende URLs ohne Search-Params funktionieren weiter. Es existieren keine alten Bookmarks mit Filter-Params (der Toggle-State war heute reines `useState`).
- **Mobile**: Der Compact-Header sorgt dafür, dass die vier Pills (drei Typen + Meine + Vergangene = 5) auch bei schmalen Viewports nebeneinander passen.
