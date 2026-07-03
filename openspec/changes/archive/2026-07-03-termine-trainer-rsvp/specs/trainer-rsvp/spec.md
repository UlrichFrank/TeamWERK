## ADDED Requirements

### Requirement: Trainer eines Kaders erscheinen in Attendances-Response

`GET /api/training-sessions/{id}/attendances` und `GET /api/games/{id}/attendances` SHALL Trainer eines am Termin beteiligten Kaders in ihrer Antwort führen. Jeder Trainer-Eintrag MUSS ein Feld `is_trainer=true` tragen; das bestehende Feld `is_extended` MUSS `false` sein.

#### Scenario: Trainer erscheint mit is_trainer=true

- **WHEN** ein authentifizierter User `GET /api/training-sessions/{id}/attendances` für eine Session mit einem in `kader_trainers` eingetragenen Trainer aufruft
- **THEN** enthält die Antwort einen Eintrag mit `member_id=<TrainerMember>`, `is_trainer=true`, `is_extended=false`

#### Scenario: Trainer erscheint auch bei Spielen

- **WHEN** ein authentifizierter User `GET /api/games/{id}/attendances` aufruft und das Spiel einem Team zugeordnet ist, dessen Kader einen Trainer hat
- **THEN** enthält die Antwort einen Trainer-Eintrag mit `is_trainer=true`

#### Scenario: Kein Trainer im Kader

- **WHEN** ein Termin einem Kader ohne Einträge in `kader_trainers` zugeordnet ist
- **THEN** enthält die Antwort keine Zeilen mit `is_trainer=true`

---

### Requirement: Trainer sind per Default confirmed (Opt-out unabhängig vom Session-Setting)

Das System SHALL für Trainer, die keine Row in `training_responses` bzw. `game_responses` haben, den `rsvp_status` in der Attendances-Response virtuell auf `confirmed` setzen — unabhängig vom Session-Setting `rsvp_opt_out`. Es wird KEINE Row in `training_responses` / `game_responses` automatisch angelegt.

#### Scenario: Trainer ohne Response gilt als confirmed

- **WHEN** ein Trainer keine Row in `training_responses` für die Session hat und die Session `rsvp_opt_out=0` gesetzt hat
- **THEN** enthält die Attendances-Response für diesen Trainer `rsvp_status='confirmed'`

#### Scenario: Explizite Absage überschreibt Default

- **WHEN** ein Trainer für dieselbe Session `POST …/respond` mit `status='declined'` aufgerufen hat
- **THEN** enthält die Attendances-Response `rsvp_status='declined'` (Row-Wert schlägt Default)

#### Scenario: Kein Default-Insert

- **WHEN** ein Trainer, der noch nie geantwortet hat, in der Attendances-Response als `confirmed` erscheint
- **THEN** existiert KEINE Row in `training_responses` mit dieser `(training_id, member_id)`-Kombination

---

### Requirement: Trainer haben keine Anwesenheits-Erfassung

Das System SHALL für Trainer kein `present`-Flag setzen oder speichern. `training_attendances` und `game_attendances` bleiben spieler-only. In der Attendances-Response ist `present` für Trainer-Zeilen immer `null`.

#### Scenario: Trainer erhält present=null

- **WHEN** ein Trainer in der Attendances-Response erscheint
- **THEN** ist `present=null`, unabhängig davon, ob der aufrufende User Trainer/Admin ist

#### Scenario: Kein POST /attendances für Trainer-Ziel-Member

- **WHEN** ein Trainer `POST /api/training-sessions/{id}/attendances` mit `member_id=<Trainer>` und `present=true` aufruft
- **THEN** antwortet der Server mit HTTP 400 (der Ziel-Member ist Kader-Trainer, kein Spieler)

#### Scenario: Kein POST /attendances für Trainer-Ziel bei Spielen

- **WHEN** ein Trainer `POST /api/games/{id}/attendances` mit `member_id=<Trainer>` aufruft
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: Trainer werden nicht in Zusagen-Zähler einbezogen

Die Felder `confirmed_count`, `declined_count`, `maybe_count` in Session- und Spiel-Responses SHALL NUR Spieler und erweiterten Kader zählen. Trainer-Rückmeldungen bleiben ausgeschlossen.

#### Scenario: Trainer-Zusage zählt nicht mit

- **WHEN** eine Session einen bestätigten Spieler und einen bestätigten Trainer hat
- **THEN** ist `confirmed_count=1` im Session-Response (nicht 2)

#### Scenario: Trainer-Absage zählt nicht mit

- **WHEN** ein Trainer explizit absagt
- **THEN** bleibt `declined_count` unverändert

---

### Requirement: Trainer können RSVP-Antworten für Termine abgeben

`POST /api/training-sessions/{id}/respond` und `POST /api/games/{id}/respond` SHALL akzeptieren, dass ein Trainer für seine eigene `member_id` oder für die `member_id` eines anderen Kader-Trainers antwortet. Diese Fähigkeit folgt dem bestehenden Muster für Standard-Rollen — die Ownership-Prüfung ist identisch zum Spieler-Fall (Selbstantwort implizit, Fremd-Antwort erlaubt für Nutzer mit `trainer`-Funktion). Die client-seitige Validierung von `reason` (Session-Setting `rsvp_require_reason`) gilt analog zum Spieler.

#### Scenario: Trainer sagt für sich selbst zu

- **WHEN** ein Trainer `POST /api/training-sessions/{id}/respond` mit `status='confirmed'` ohne `member_id` aufruft
- **THEN** antwortet der Server mit HTTP 204 und legt/aktualisiert eine `training_responses`-Row mit der Trainer-`member_id`

#### Scenario: Trainer sagt für sich selbst ab mit Grund

- **WHEN** ein Trainer `POST …/respond` mit `status='declined'` und `reason='Krank'` aufruft
- **THEN** wird die Row inkl. `reason` gespeichert (HTTP 204)

#### Scenario: Trainer antwortet für einen anderen Trainer

- **WHEN** Trainer A `POST …/respond` mit `member_id=<TrainerB>` und `status='declined'` aufruft
- **THEN** wird die Row für Trainer B angelegt/aktualisiert (HTTP 204)
