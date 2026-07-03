## ADDED Requirements

### Requirement: Trainer können auf Trainings-RSVP antworten

`POST /api/training-sessions/{id}/respond` SHALL akzeptieren, dass ein User mit Vereinsfunktion `trainer` für seine eigene `member_id` oder für die `member_id` eines anderen Trainers antwortet. Der `status` MUSS einer von `confirmed | declined | maybe` sein.

Trainer-Rows in `training_responses` werden NICHT in `confirmed_count` / `declined_count` / `pending_count` gezählt (siehe `trainer-rsvp`-Capability).

#### Scenario: Trainer sagt für sich selbst zu

- **WHEN** ein Trainer `POST /api/training-sessions/{id}/respond` mit `{"status":"confirmed"}` aufruft (implizit auf eigene `member_id`)
- **THEN** antwortet der Server mit HTTP 204
- **THEN** existiert eine Row in `training_responses` mit `member_id=<Trainers Member>` und `status='confirmed'`

#### Scenario: Trainer sagt für anderen Trainer ab

- **WHEN** Trainer A `POST …/respond` mit `member_id=<TrainerB>` und `status='declined'` aufruft
- **THEN** antwortet der Server mit HTTP 204 und legt die Row für Trainer B an
