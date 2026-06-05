## Context

`KalenderPage.tsx` enthält bereits einen mehrstufigen Wizard für Heim-/Auswärtsspiele und generische Events. Schritt 1 wählt den Event-Typ, Schritt 2 erfasst die Details. Trainings werden in der Kalenderansicht bereits angezeigt (via `GET /api/training-sessions`), aber der Klick navigiert zur TrainingsDetailPage. Die Backend-Endpunkte für Einzeltraining (`POST /api/training-sessions`) und Serie (`POST /api/training-series`, `PUT /api/training-series/{id}`, `PUT /api/training-sessions/{id}`) sind vollständig implementiert.

## Goals / Non-Goals

**Goals:**
- Trainer/Admin kann Training und Serie direkt aus dem Kalender anlegen
- Trainer/Admin kann einen Trainingstermin im Kalender inline bearbeiten
- Normale Nutzer sehen keine neuen Buttons und navigieren weiterhin zur Detailseite

**Non-Goals:**
- Trainings aus der AdminTrainingsPage entfernen — die Seite bleibt als vollständige Verwaltungsübersicht
- Anwesenheitserfassung im Kalender — bleibt in TrainingsDetailPage
- RSVP direkt im Kalender — bleibt in TrainingsDetailPage

## Decisions

### 1. Wizard: Schritt 1 zeigt Training/Serie nur für Trainer/Admin

**Entscheidung:** `hasFunction(user, 'manage-trainings')` (oder direkte Rollen-Prüfung auf `trainer`/`admin`) steuert ob die neuen Optionen in Schritt 1 erscheinen.

**Begründung:** Konsistent mit dem bisherigen Pattern — der Kalender-Plus-Button ist bereits nur für Trainer/Admin sichtbar.

### 2. Klick auf Training: rollenabhängiges Verhalten

**Entscheidung:**
- Spieler / Elternteil: navigate zu `/trainings/{id}` (unverändert)
- Trainer / Admin: Öffnet Edit-Modal direkt im Kalender

**Begründung:** Trainer brauchen schnellen Zugriff auf Bearbeiten, nicht auf RSVP-Liste. Spieler brauchen RSVP, nicht Bearbeiten. Getrennte UX ohne Kompromisse.

### 3. Serie-Edit-Scope im Modal

**Entscheidung:** Wenn die bearbeitete Session eine `series_id` hat, erscheinen drei Radio-Buttons: „Nur dieser Termin" / „Dieser und folgende" / „Alle Termine der Serie". Je nach Wahl wird `PUT /api/training-sessions/{id}` oder `PUT /api/training-series/{id}?scope=...` aufgerufen.

**Begründung:** Identisches Muster wie `trainingsplanung`-Design (Entscheidung 4). Konsistent mit bestehendem Backend.

### 4. Wizard-Zustand: Erweiterung des bestehenden State

**Entscheidung:** Die bestehenden Wizard-States (`wizardStep`, `eventType`, etc.) werden um `'training'` und `'serie'` als `eventType`-Werte erweitert, plus neue States für `trainingStartTime`, `trainingEndTime`, `trainingLocation`, `seriesWeekday`, `seriesValidFrom`, `seriesValidUntil`.

**Begründung:** Keine neue Komponente nötig, der Wizard ist klein genug. Der bestehende `closeDialog`/Reset-Mechanismus funktioniert unverändert.

## Risks / Trade-offs

- **KalenderPage.tsx ist bereits 757 Zeilen** — die Erweiterung wird sie weiter wachsen lassen. Mitigation: Das Edit-Modal kann als kleine separate Komponente `TrainingEditModal.tsx` ausgelagert werden.
- **Wizard-Schritt für Serie** erfordert einen Wochentag-Picker — einfaches `<select>` mit Mo–So Optionen, keine externe Dependency.
