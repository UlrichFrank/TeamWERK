## ADDED Requirements

### Requirement: Trainer können auf Spiel-RSVP antworten

`POST /api/games/{id}/respond` SHALL akzeptieren, dass ein User mit Vereinsfunktion `trainer` für seine eigene `member_id` oder für die `member_id` eines anderen Trainers antwortet. Der `status` MUSS einer von `confirmed | declined | maybe` sein.

Trainer-Rows in `game_responses` werden NICHT in `confirmed_count` / `declined_count` / `maybe_count` gezählt (siehe `trainer-rsvp`-Capability).

#### Scenario: Trainer sagt für sich selbst ab

- **WHEN** ein Trainer `POST /api/games/{id}/respond` mit `{"status":"declined","reason":"Krank"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** existiert eine Row in `game_responses` mit `member_id=<Trainers Member>`, `status='declined'`, `reason='Krank'`

#### Scenario: Trainer sagt für anderen Trainer zu

- **WHEN** Trainer A `POST …/respond` mit `member_id=<TrainerB>` und `status='confirmed'` aufruft
- **THEN** antwortet der Server mit HTTP 204
