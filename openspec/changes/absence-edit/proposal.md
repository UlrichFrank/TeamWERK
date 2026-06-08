## Why

Abwesenheiten können bisher nur angelegt und gelöscht werden — nachträgliche Korrekturen (falscher Typ, falsches Datum, fehlende Notiz) erfordern Löschen und Neu-Anlegen. Außerdem gibt es kein Feedback-Muster wenn man auf einen Abwesenheitsbalken im Kalender klickt, obwohl Spiele und Trainings per Klick ein Detail-Modal öffnen.

## What Changes

- Klick auf einen Abwesenheitsbalken öffnet ein Info-Modal analog zu Spielen und Trainings
- Das Modal zeigt: Typ (Urlaub / Verletzung), Mitgliedsname, Zeitraum, Notiz
- Ersteller und Admins sehen Bearbeiten- und Löschen-Button
- Bearbeiten öffnet ein Inline-Edit-Formular im selben Modal (Typ, Start-/Enddatum, Notiz)
- Speichern ruft `PUT /api/absences/{id}` auf mit Überlappungscheck (gleicher Typ, gleiches Mitglied, überschneidender Zeitraum → 409, eigene ID ausgenommen)
- Neues Backend-Endpoint: `PUT /api/absences/{id}`

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `member-absences`: Neue Anforderungen für Bearbeiten (`PUT /api/absences/{id}`) und Anzeigen (Klick-Interaktion im Kalender)

## Impact

- `internal/absences/handler.go` — neuer `Update`-Handler, Route in `main.go` eintragen
- `web/src/pages/KalenderPage.tsx` — `infoItem`-State um `type:'absence'` erweitern, Klick-Handler auf Balken
- `web/src/components/EventInfoModal.tsx` — Absence-Zweig ergänzen (Anzeige + Inline-Edit-Formular)
