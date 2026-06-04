## Why

Die KalenderPage zeigt Trainings und Spiele nebeneinander, aber ein Klick auf ein Spiel öffnet immer die Dienstseite (`/kalender/{id}`). Nutzer sollen wahlweise direkt Spieldaten einsehen oder bearbeiten können, ohne zur Dienstseite wechseln zu müssen.

## What Changes

- Neuer Toggle „Spiel | Dienst" im KalenderPage-Header (analog zum „Team | Meine"-Toggle auf der Mitfahrgelegenheiten-Seite)
- **Dienst-Modus** (Standard, bisheriges Verhalten): Klick auf Spiel-Pill → navigiert zu `/kalender/{id}`
- **Spiel-Modus**: Klick auf Spiel-Pill → öffnet `GameModal` inline
  - Admin/Trainer: Bearbeitungsformular (Datum, Uhrzeit, Gegner, Teams) via `PUT /admin/games/{id}`
  - Andere Rollen: Read-only-Anzeige der gleichen Felder
- Filter „Sonstiges" wird im Spiel-Modus deaktiviert (Opacity-40, nicht klickbar) und automatisch abgewählt beim Wechsel in den Spiel-Modus
- Neues `GameModal`-Komponente (existiert noch nicht)

## Capabilities

### New Capabilities

- `game-view-edit-modal`: Inline-Modal zum Anzeigen und Bearbeiten von Spieldaten direkt im Kalender

### Modified Capabilities

*(keine bestehenden Spec-Anforderungen ändern sich)*

## Impact

- `web/src/pages/KalenderPage.tsx`: Toggle-State, bedingtes Klickverhalten, Sonstiges-Filter-Logik
- `web/src/components/GameModal.tsx`: neue Komponente (edit + read-only)
- Backend: keine Änderungen nötig (`PUT /admin/games/{id}` existiert bereits)
