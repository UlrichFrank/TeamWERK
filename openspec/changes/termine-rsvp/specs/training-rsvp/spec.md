## MODIFIED Requirements

### Requirement: Training-RSVP-Route

Die Training-RSVP-Funktionalität SHALL über `/termine` statt über `/trainings` erreichbar sein.
Die RSVP-API-Endpunkte (`/api/training-sessions/{id}/respond`) bleiben unverändert.

#### Scenario: Spieler gibt RSVP über /termine ab
- **WHEN** ein User mit Rolle `spieler` die `/termine`-Seite aufruft
- **THEN** werden Trainings des eigenen Teams mit RSVP-Buttons angezeigt
- **THEN** führt ein RSVP-Klick intern zu `POST /api/training-sessions/{id}/respond`

#### Scenario: Trainer-Detailseite für Training über /termine
- **WHEN** ein Trainer auf eine Trainingskarte klickt
- **THEN** wird er zu `/termine/training/:id` navigiert (vorher `/trainings/:id`)
- **THEN** zeigt die Seite dieselben Inhalte wie vorher `/trainings/:id` (RSVP-Tabelle + Anwesenheit)

#### Scenario: Alter /trainings-Link redirectet
- **WHEN** ein User `/trainings` oder `/trainings/:id` aufruft
- **THEN** wird er auf `/termine` bzw. `/termine/training/:id` weitergeleitet

## REMOVED Requirements

### Requirement: /trainings als primäre Route
**Reason**: Durch `/termine` abgelöst; Trainings und Spiele werden auf einer gemeinsamen Seite angezeigt.
**Migration**: Alle Links auf `/trainings` auf `/termine` aktualisieren; React Router Redirect einrichten.
