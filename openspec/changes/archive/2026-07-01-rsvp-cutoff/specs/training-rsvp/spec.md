## ADDED Requirements

### Requirement: Training-RSVP-Cutoff 2 Stunden vor Beginn

Das System SHALL `POST /api/training-sessions/{id}/respond` für Nutzer ohne Trainer-/Vorstand-/Admin-Berechtigung mit HTTP 422 ablehnen, sobald die aktuelle Zeit weniger als 2 Stunden vor dem Beginn der Session (`date` + `start_time` in Europe/Berlin) liegt. Der Cutoff sperrt jeden Statuswechsel — die erste Antwort, einen Wechsel zwischen `confirmed`/`declined`/`maybe`, und das Aktualisieren des `reason`-Feldes.

Die Fehlerantwort SHALL den Body `{"error":"rsvp_locked","message":"Training kann nur bis 2 Stunden vor Beginn umgesagt werden.","locks_at":"<RFC3339 UTC>"}` liefern.

#### Scenario: Spieler antwortet 3 Stunden vor Training
- **WHEN** ein Spieler `POST /api/training-sessions/{id}/respond` mit `{"status":"declined"}` aufruft und die Session-Start-Zeit liegt 3 Stunden in der Zukunft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist `training_responses.status = 'declined'` für den Spieler gespeichert

#### Scenario: Spieler sagt 30 Minuten vor Training ab
- **WHEN** ein Spieler `POST /api/training-sessions/{id}/respond` mit `{"status":"declined"}` aufruft und die Session-Start-Zeit liegt 30 Minuten in der Zukunft
- **THEN** antwortet der Server mit HTTP 422
- **THEN** enthält der Response-Body `error=rsvp_locked` und `locks_at` als RFC3339-UTC

#### Scenario: Spieler ändert bereits abgegebene Antwort 30 Minuten vor Training
- **WHEN** ein Spieler bereits `confirmed` ist und 30 Minuten vor Beginn auf `declined` wechseln will
- **THEN** antwortet der Server mit HTTP 422
- **THEN** bleibt `training_responses.status = 'confirmed'` unverändert

#### Scenario: Spieler beantwortet Session erstmals 30 Minuten vor Training
- **WHEN** ein Spieler ohne bestehende Response 30 Minuten vor Beginn mit `{"status":"confirmed"}` antworten will
- **THEN** antwortet der Server mit HTTP 422
- **THEN** existiert weiterhin keine Zeile in `training_responses` für diesen Spieler

#### Scenario: Elternteil antwortet 30 Minuten vor Training für Kind
- **WHEN** ein Elternteil 30 Minuten vor Beginn `POST` mit `{"member_id": <Kind>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 422

#### Scenario: Trainer pflegt Response 30 Minuten vor Training
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` 30 Minuten vor Beginn `POST` mit `{"member_id": <Spieler>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist `training_responses.status = 'declined'` gespeichert mit `responded_by = <Trainer-User-ID>`

#### Scenario: Vorstand pflegt Response nach Training-Beginn
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` 5 Minuten nach Session-Beginn `POST` mit `{"member_id": <Spieler>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Sportliche Leitung darf nach Cutoff antworten
- **WHEN** ein Nutzer mit Vereinsfunktion `sportliche_leitung` 30 Minuten vor Beginn `POST` mit `{"member_id": <Spieler>, "status":"confirmed"}` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Admin darf nach Cutoff antworten
- **WHEN** ein Nutzer mit Systemrolle `admin` (ohne Vereinsfunktion) 30 Minuten vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Kassierer darf nicht nach Cutoff antworten
- **WHEN** ein Nutzer mit ausschließlicher Vereinsfunktion `kassierer` (kein Trainer/Vorstand/Admin) 30 Minuten vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 422

#### Scenario: Absence-Lock hat Vorrang vor Cutoff
- **WHEN** ein Spieler mit gesetztem `training_responses.absence_id` 3 Stunden vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 403 (Absence-Lock, **nicht** 422)

#### Scenario: DST-Wechsel — Cutoff in Sommer- und Winterzeit korrekt
- **WHEN** eine Session am ersten Sonntag der MEZ→MESZ-Umstellung um 18:00 Uhr Ortszeit startet
- **THEN** ist `locks_at` exakt 16:00 Uhr Ortszeit (entsprechend in UTC), nicht 15:00 Uhr UTC fest

---

### Requirement: Training-Listing liefert rsvp_locks_at

Listing- und Detail-Endpoints für Trainings SHALL pro Session ein Feld `rsvp_locks_at` (RFC3339, UTC) liefern, das den Zeitpunkt benennt, ab dem reguläre Mitglieder keine RSVP-Änderung mehr vornehmen können.

#### Scenario: Sessions-Liste enthält rsvp_locks_at
- **WHEN** ein User `GET /api/training-sessions` aufruft
- **THEN** enthält jedes Session-Objekt das Feld `rsvp_locks_at` als RFC3339-UTC-String

#### Scenario: Session-Detail enthält rsvp_locks_at
- **WHEN** ein User `GET /api/training-sessions/{id}` aufruft
- **THEN** enthält die Response das Feld `rsvp_locks_at`

#### Scenario: rsvp_locks_at = start - 2h
- **WHEN** eine Session am 30.06.2026 um 18:00 Uhr Europe/Berlin startet
- **THEN** liefert die API `rsvp_locks_at = "2026-06-30T14:00:00Z"` (16:00 Berliner Sommerzeit = 14:00 UTC)
