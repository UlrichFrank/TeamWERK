# trainings-live-updates Specification

## Purpose

Diese Spezifikation beschreibt die Capability `trainings-live-updates`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Trainings-Mutationen broadcasten SSE-Event

Das Backend SHALL nach jeder erfolgreichen Trainings-Mutation `hub.Broadcast("trainings")` aufrufen. Betroffene Endpunkte: `POST /api/training-sessions`, `PUT /api/training-sessions/{id}`, `POST /api/training-sessions/{id}/respond`, `POST /api/training-sessions/{id}/attendances`, `POST /api/training-series`, `PUT /api/training-series/{id}`, `DELETE /api/training-series/{id}`.

#### Scenario: RSVP-Änderung löst Event aus

- **WHEN** ein Spieler `POST /api/training-sessions/{id}/respond` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients innerhalb von 1 Sekunde `data: trainings`

#### Scenario: Session-Update löst Event aus

- **WHEN** ein Trainer `PUT /api/training-sessions/{id}` aufruft (z.B. Absage, Zeitänderung)
- **THEN** erhalten alle verbundenen SSE-Clients `data: trainings`

#### Scenario: Kein Broadcast bei Fehler

- **WHEN** eine Trainings-Mutation mit einem Fehler endet (z.B. forbidden, DB-Fehler)
- **THEN** wird kein SSE-Event gesendet

### Requirement: Trainings-Seiten abonnieren SSE-Events

`TrainingsPage` und `TrainingsDetailPage` SHALL `useLiveUpdates` verwenden und bei einem `"trainings"`-Event ihre Daten still neu laden (ohne sichtbaren Ladespinner).

#### Scenario: TrainingsPage aktualisiert Liste bei fremder Änderung

- **WHEN** ein anderer Nutzer einen RSVP abgibt oder eine Session geändert wird
- **THEN** lädt `TrainingsPage` die Session-Liste neu ohne sichtbaren Ladespinner

#### Scenario: TrainingsDetailPage aktualisiert Detail bei fremder Änderung

- **WHEN** ein anderer Nutzer `POST /training-sessions/{id}/respond` aufruft
- **THEN** aktualisiert sich `TrainingsDetailPage` (Rückmeldungen-Liste + Zähler) ohne Reload

#### Scenario: Seiten ignorieren nicht relevante Events

- **WHEN** ein `members`- oder `duties`-Event eintrifft
- **THEN** laden `TrainingsPage` und `TrainingsDetailPage` NICHT neu
