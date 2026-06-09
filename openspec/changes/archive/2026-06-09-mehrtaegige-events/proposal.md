## Why

Turniere und Trainingslager dauern mehrere Tage, lassen sich aber bisher nicht als ein zusammenhängendes Event anlegen — Nutzer müssten pro Tag ein separates Event erstellen. Ein optionales `end_date`-Feld löst das ohne neue Typen oder komplexe Datenmodelle.

## What Changes

- `games`-Tabelle erhält ein optionales `end_date DATE`-Feld (nullable)
- `POST /api/kalender` und `PUT /api/kalender/{id}` akzeptieren `end_date` im Request-Body
- `GET /api/kalender` und `GET /api/kalender/{id}` liefern `end_date` in der Response
- Im Kalender erscheint ein Event mit gesetztem `end_date` als Pill in **jeder** Tageszelle innerhalb des Bereichs `date..end_date`
- Das Event-Wizard-Formular bekommt ein optionales „Enddatum"-Feld, das nur für `generisch`-Events sichtbar ist (Turniere = `heim`/`auswärts` mit end_date, Trainingslager = `generisch` mit end_date)
- RSVP, Dienstslots und Mitfahrten bleiben unverändert — sie gelten weiterhin für das gesamte Event

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `games`: neues optionales `end_date`-Feld; GET-Responses geben es zurück; POST/PUT nehmen es entgegen
- `event-wizard`: optionales Enddatum-Feld für `generisch`-Events im Erstellungs-/Bearbeitungsformular

## Impact

- `internal/db/migrations/` — neue Migration: `end_date DATE` zu `games`
- `internal/games/handler.go` — `CreateGame`, `UpdateGame`, `ListGames`, `GetGame` anpassen
- `web/src/pages/KalenderPage.tsx` — Kalender-Rendering: mehrtägige Events in allen Tageszellen des Bereichs anzeigen; Event-Wizard-Formular: Enddatum-Feld
