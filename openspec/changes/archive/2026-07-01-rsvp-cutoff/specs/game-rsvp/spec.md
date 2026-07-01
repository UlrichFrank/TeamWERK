## ADDED Requirements

### Requirement: Spiel-RSVP-Cutoff 18 Stunden vor Beginn

Das System SHALL `POST /api/games/{id}/respond` für Nutzer ohne Trainer-/Vorstand-/Admin-Berechtigung mit HTTP 422 ablehnen, sobald die aktuelle Zeit weniger als 18 Stunden vor dem Beginn des Spiels (`date` + `time` in Europe/Berlin) liegt. Der Cutoff sperrt jeden Statuswechsel — die erste Antwort, einen Wechsel zwischen `confirmed`/`declined`/`maybe`, und das Aktualisieren des `reason`-Feldes.

Die Fehlerantwort SHALL den Body `{"error":"rsvp_locked","message":"Spiel kann nur bis 18 Stunden vor Beginn umgesagt werden.","locks_at":"<RFC3339 UTC>"}` liefern.

#### Scenario: Spieler antwortet 2 Tage vor Spiel
- **WHEN** ein Spieler `POST /api/games/{id}/respond` mit `{"status":"confirmed"}` aufruft und das Spiel beginnt in 48 Stunden
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist `game_responses.status = 'confirmed'` für den Spieler gespeichert

#### Scenario: Spieler sagt 12 Stunden vor Spiel ab
- **WHEN** ein Spieler `POST /api/games/{id}/respond` mit `{"status":"declined"}` aufruft und das Spiel beginnt in 12 Stunden
- **THEN** antwortet der Server mit HTTP 422
- **THEN** enthält der Response-Body `error=rsvp_locked` und `locks_at` als RFC3339-UTC

#### Scenario: Spieler ändert Antwort 12 Stunden vor Spiel
- **WHEN** ein Spieler bereits `confirmed` ist und 12 Stunden vor Beginn auf `declined` wechseln will
- **THEN** antwortet der Server mit HTTP 422
- **THEN** bleibt `game_responses.status = 'confirmed'` unverändert

#### Scenario: Spieler beantwortet Spiel erstmals 12 Stunden vor Beginn
- **WHEN** ein Spieler ohne bestehende Response 12 Stunden vor Beginn antworten will
- **THEN** antwortet der Server mit HTTP 422
- **THEN** existiert weiterhin keine Zeile in `game_responses` für diesen Spieler

#### Scenario: Elternteil antwortet 12 Stunden vor Spiel für Kind
- **WHEN** ein Elternteil 12 Stunden vor Beginn `POST` mit `{"member_id": <Kind>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 422

#### Scenario: Trainer pflegt Response 12 Stunden vor Spiel
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` 12 Stunden vor Beginn `POST` mit `{"member_id": <Spieler>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist `game_responses.status = 'declined'` gespeichert mit `responded_by = <Trainer-User-ID>`

#### Scenario: Vorstand pflegt Response nach Spielbeginn
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` 1 Stunde nach Spielbeginn `POST` mit `{"member_id": <Spieler>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Sportliche Leitung darf nach Cutoff antworten
- **WHEN** ein Nutzer mit Vereinsfunktion `sportliche_leitung` 12 Stunden vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Admin darf nach Cutoff antworten
- **WHEN** ein Nutzer mit Systemrolle `admin` (ohne Vereinsfunktion) 12 Stunden vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Kassierer darf nicht nach Cutoff antworten
- **WHEN** ein Nutzer mit ausschließlicher Vereinsfunktion `kassierer` 12 Stunden vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 422

#### Scenario: Absence-Lock hat Vorrang vor Cutoff
- **WHEN** ein Spieler mit gesetztem `game_responses.absence_id` 2 Tage vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 403 (Absence-Lock, **nicht** 422)

---

### Requirement: Game-Listing liefert rsvp_locks_at

Listing- und Detail-Endpoints für Spiele SHALL pro Spiel ein Feld `rsvp_locks_at` (RFC3339, UTC) liefern, das den Zeitpunkt benennt, ab dem reguläre Mitglieder keine RSVP-Änderung mehr vornehmen können.

#### Scenario: Eigene Spiele-Liste enthält rsvp_locks_at
- **WHEN** ein User `GET /api/games/my` aufruft
- **THEN** enthält jedes Spiel-Objekt das Feld `rsvp_locks_at` als RFC3339-UTC-String

#### Scenario: Vorstand-Spiele-Liste enthält rsvp_locks_at
- **WHEN** ein User `GET /api/games` aufruft
- **THEN** enthält jedes Spiel-Objekt das Feld `rsvp_locks_at`

#### Scenario: Spiel-Detail enthält rsvp_locks_at
- **WHEN** ein User `GET /api/games/{id}` aufruft
- **THEN** enthält die Response das Feld `rsvp_locks_at`

#### Scenario: rsvp_locks_at = start - 18h
- **WHEN** ein Spiel am 30.06.2026 um 18:00 Uhr Europe/Berlin startet
- **THEN** liefert die API `rsvp_locks_at = "2026-06-29T22:00:00Z"` (00:00 Berliner Sommerzeit am 30.06. = 22:00 UTC am 29.06.)
