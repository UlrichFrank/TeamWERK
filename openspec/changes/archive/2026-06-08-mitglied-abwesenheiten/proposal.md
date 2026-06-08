## Why

Spieler und Elternteile sollen Urlaubszeiträume und Sportverbote (Verletzungen) eintragen können, damit der Verein von Anfang an weiß, wer bei Training und Spielen nicht verfügbar ist — ohne dass jede Einzel-Zusage manuell zurückgezogen werden muss.

## What Changes

- Neue Abwesenheits-Verwaltung: Spieler und Elternteile können Zeiträume (Urlaub / Verletzung) für sich bzw. ihre Kinder eintragen
- Automatischer Decline: Alle Training- und Spielzusagen im Abwesenheitszeitraum werden sofort auf „Abgesagt" gesetzt und für manuelle Änderungen gesperrt
- Neue Events im Abwesenheitszeitraum: Werden bei Erstellung sofort auto-declined
- Sichtbarkeit: Trainer sehen Abwesenheiten nur, wenn der Member das im Profil freigibt (`absences_public`); der Decline-Effekt ist immer sichtbar
- Kalender-Banner: Abwesenheiten erscheinen als farbige Linie über die betroffenen Wochentage (nur für Member selbst, Elternteile, und Trainer wenn freigegeben)
- Confirmation-Modal: Beim Anlegen einer Abwesenheit, die bestehende Zusagen überschreibt, sieht der Nutzer vorab eine Liste der betroffenen Events

## Capabilities

### New Capabilities

- `member-absences`: Abwesenheitsverwaltung — CRUD für Abwesenheitszeiträume, auto-decline-Logik, Preview-Endpoint, Kalender-Integration und Profil-Sichtbarkeits-Toggle

### Modified Capabilities

- `game-rsvp`: Responses erhalten ein `absence_id`-Feld; ist es gesetzt, ist die Antwort nicht manuell änderbar
- `training-rsvp`: Wie `game-rsvp` — zusätzliches `absence_id`-Feld, gesperrte Responses
- `members`: Neues Feld `absences_public` (Boolean, default false) steuert Trainer-Sichtbarkeit von Abwesenheiten

## Impact

- **Neue Migration (030):** Tabelle `member_absences`, Spalte `members.absences_public`, Spalten `training_responses.absence_id` + `game_responses.absence_id`
- **Neues Package:** `internal/absences/`
- **Geänderte Packages:** `internal/games/` (CreateGame prüft Abwesenheiten), `internal/trainings/` (CreateTrainingSession prüft Abwesenheiten), `internal/members/` (absences_public im Profil)
- **Frontend:** Neue Seite `AbsenzenPage`, Kalender-Banner in `KalenderPage`, Confirmation-Modal, Toggle im Profil
- **Keine neuen externen Dependencies**
