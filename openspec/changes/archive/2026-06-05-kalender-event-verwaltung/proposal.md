## Why

Der Kalender ist die zentrale Ansicht für alle Events in TeamWERK. Trainings sind dort sichtbar, können aber nur über eine separate Admin-Seite (`/admin/trainings`) angelegt und bearbeitet werden. Trainer müssen zwischen zwei verschiedenen UIs wechseln, obwohl der Kalender bereits einen Wizard für Heim-/Auswärtsspiele und generische Events enthält. Ziel ist eine einheitliche Event-Verwaltung direkt im Kalender.

## What Changes

- Der bestehende Kalender-Wizard (Schritt 1: Event-Typ wählen) erhält zwei neue Optionen: **Training** und **Trainingsserie**
- Neuer Wizard-Schritt für Einzeltraining: Datum, Start-/Endzeit, Ort, Team
- Neuer Wizard-Schritt für Trainingsserie: Wochentag, Start-/Endzeit, Ort, Team, Gültigkeitszeitraum
- Trainings und Serien sind **direkt im Kalender editierbar**: Klick eines Trainers/Admins auf einen Trainingstermin öffnet ein Inline-Edit-Modal statt zu `/trainings/{id}` zu navigieren
- Das Edit-Modal für Serientermine fragt den Edit-Scope: „Nur dieser Termin" / „Dieser und folgende" / „Alle Termine der Serie"
- Normale Nutzer (Spieler, Elternteil) navigieren weiterhin zur TrainingsDetailPage für RSVPs

## Capabilities

### New Capabilities

- `kalender-training-wizard`: Anlegen von Einzeltraining und Trainingsserie aus dem Kalender-Wizard heraus
- `kalender-training-edit`: Inline-Edit-Modal für Trainingstermine im Kalender (Trainer/Admin)

### Modified Capabilities

- `games` (KalenderPage): Wizard-Schritt 1 erweitert um Training und Trainingsserie; Klick-Handler für Trainings rollenabhängig
- `training-series`: kein neues Backend — bestehende Endpunkte `POST /api/training-series`, `PUT /api/training-series/{id}`, `POST /api/training-sessions`, `PUT /api/training-sessions/{id}` werden aus dem Kalender heraus genutzt

## Impact

- **Nur Frontend-Änderungen:** `KalenderPage.tsx` (Wizard + Klick-Handler)
- **Keine neuen Backend-Routen**, keine neuen DB-Migrationen
- **AdminTrainingsPage.tsx** bleibt bestehen (Übersicht aller Serien, Anwesenheitserfassung) — der Kalender-Weg ist ein zusätzlicher, kürzerer Pfad
- **Rollen:** Wizard-Optionen Training/Serie nur für Trainer und Admin sichtbar; Edit-Modal nur für Trainer/Admin
