## MODIFIED Requirements

### Requirement: Spiel-RSVP-Cutoff 2 Stunden vor Beginn

Das System SHALL `POST /api/games/{id}/respond` für Nutzer ohne Trainer-/Vorstand-/Admin-Berechtigung mit HTTP 422 ablehnen, sobald die aktuelle Zeit weniger als 2 Stunden vor dem Beginn des Spiels (`date` + `time` in Europe/Berlin) liegt. Der Cutoff sperrt jeden Statuswechsel — die erste Antwort, einen Wechsel zwischen `confirmed`/`declined`/`maybe`, und das Aktualisieren des `reason`-Feldes.

Die Fehlerantwort SHALL den Body `{"error":"rsvp_locked","message":"Spiel kann nur bis 2 Stunden vor Beginn umgesagt werden.","locks_at":"<RFC3339 UTC>"}` liefern.

#### Scenario: Spieler antwortet 3 Stunden vor Spiel
- **WHEN** ein Spieler `POST /api/games/{id}/respond` mit `{"status":"confirmed"}` aufruft und das Spiel beginnt in 3 Stunden
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist `game_responses.status = 'confirmed'` für den Spieler gespeichert

#### Scenario: Spieler sagt 30 Minuten vor Spiel ab
- **WHEN** ein Spieler `POST /api/games/{id}/respond` mit `{"status":"declined"}` aufruft und das Spiel beginnt in 30 Minuten
- **THEN** antwortet der Server mit HTTP 422
- **THEN** enthält der Response-Body `error=rsvp_locked` und `locks_at` als RFC3339-UTC

#### Scenario: Spieler ändert Antwort 30 Minuten vor Spiel
- **WHEN** ein Spieler bereits `confirmed` ist und 30 Minuten vor Beginn auf `declined` wechseln will
- **THEN** antwortet der Server mit HTTP 422
- **THEN** bleibt `game_responses.status = 'confirmed'` unverändert

#### Scenario: Spieler beantwortet Spiel erstmals 30 Minuten vor Beginn
- **WHEN** ein Spieler ohne bestehende Response 30 Minuten vor Beginn antworten will
- **THEN** antwortet der Server mit HTTP 422
- **THEN** existiert weiterhin keine Zeile in `game_responses` für diesen Spieler

#### Scenario: Elternteil antwortet 30 Minuten vor Spiel für Kind
- **WHEN** ein Elternteil 30 Minuten vor Beginn `POST` mit `{"member_id": <Kind>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 422

#### Scenario: Trainer pflegt Response 30 Minuten vor Spiel
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` 30 Minuten vor Beginn `POST` mit `{"member_id": <Spieler>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist `game_responses.status = 'declined'` gespeichert mit `responded_by = <Trainer-User-ID>`

#### Scenario: Vorstand pflegt Response nach Spielbeginn
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` 1 Stunde nach Spielbeginn `POST` mit `{"member_id": <Spieler>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Sportliche Leitung darf nach Cutoff antworten
- **WHEN** ein Nutzer mit Vereinsfunktion `sportliche_leitung` 30 Minuten vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Admin darf nach Cutoff antworten
- **WHEN** ein Nutzer mit Systemrolle `admin` (ohne Vereinsfunktion) 30 Minuten vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Kassierer darf nicht nach Cutoff antworten
- **WHEN** ein Nutzer mit ausschließlicher Vereinsfunktion `kassierer` 30 Minuten vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 422

#### Scenario: Absence-Lock hat Vorrang vor Cutoff
- **WHEN** ein Spieler mit gesetztem `game_responses.absence_id` 2 Tage vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 403 (Absence-Lock, **nicht** 422)

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

#### Scenario: rsvp_locks_at = start - 2h
- **WHEN** ein Spiel am 30.06.2026 um 18:00 Uhr Europe/Berlin startet
- **THEN** liefert die API `rsvp_locks_at = "2026-06-30T14:00:00Z"` (16:00 Berliner Sommerzeit − 2h = 14:00 UTC)

## ADDED Requirements

### Requirement: Teilnehmer sehen RSVP-Buttons unabhängig von Response

Endpoints `GET /api/games/my` und `GET /api/games/{id}` SHALL pro Spiel-Objekt ein boolesches Feld `am_i_participant` liefern. `GET /api/games` (Vorstand-Sicht) ist ausgenommen, da diese Ansicht keine eigenen RSVP-Buttons rendert. Das Feld ist `true` genau dann, wenn der aufrufende User selbst im regulären Kader (`kader_members`), im erweiterten Kader (`kader_extended_members`) oder als Trainer (`kader_trainers`) mindestens eines am Spiel beteiligten Teams für die Saison des Spiels eingetragen ist. Für Nicht-Teilnehmer ist `am_i_participant` `false`.

Die Frontend-Anzeige der eigenen RSVP-Buttons (Zusagen/Vielleicht/Absagen) SHALL an `am_i_participant` gebunden sein — **nicht** an `my_rsvp !== null`. Ist der Cutoff (`rsvp_locks_at`) erreicht und der User nicht Cutoff-berechtigt, bleiben die Buttons sichtbar und sind `disabled` mit einer erklärenden Notice.

Für Eltern gilt: Die Kind-Zeilen (`children_rsvp`) sind bereits heute kader-basiert und ändern sich nicht. Ein Elternteil ohne eigene Kader-Zugehörigkeit sieht `am_i_participant=false` für sich selbst (keine Eigen-Buttons), aber weiterhin die Buttons pro Kind.

#### Scenario: Spieler ohne Response sieht `am_i_participant=true`
- **WHEN** ein Spieler im regulären Kader eines beteiligten Teams `GET /api/games/my` aufruft und für ein Spiel mit `rsvp_default_players='none'` noch keine Response existiert
- **THEN** enthält das Spiel-Objekt `am_i_participant=true` und `my_rsvp=null`

#### Scenario: Erweiterter Kader-Spieler sieht `am_i_participant=true`
- **WHEN** ein Spieler nur über `kader_extended_members` eines beteiligten Teams zugeordnet ist
- **THEN** ist `am_i_participant=true`

#### Scenario: Trainer sieht `am_i_participant=true`
- **WHEN** ein User via `kader_trainers` als Trainer eines beteiligten Teams eingetragen ist
- **THEN** ist `am_i_participant=true`

#### Scenario: Fremder Nutzer sieht `am_i_participant=false`
- **WHEN** ein User ohne Kader-Beziehung zu einem der beteiligten Teams das Spiel sieht (z.B. Vorstand ohne Trainer-Rolle)
- **THEN** ist `am_i_participant=false`
- **THEN** zeigt das Frontend für ihn keine eigenen RSVP-Buttons

#### Scenario: Elternteil ohne Vereinsfunktion und ohne Kader-Rolle
- **WHEN** ein Elternteil ohne eigene Kader-Zugehörigkeit die Termine-Seite aufruft
- **THEN** ist `am_i_participant=false` für alle Spiele
- **THEN** sieht der Elternteil ausschließlich Kind-Zeilen mit Buttons, keine Eigen-Buttons

#### Scenario: Spieler-Buttons sichtbar aber gesperrt nach Cutoff
- **WHEN** ein Spieler mit `am_i_participant=true` das Spiel innerhalb der letzten 2 Stunden vor Anpfiff aufruft und den Cutoff nicht überschreiben darf
- **THEN** rendert das Frontend die drei RSVP-Buttons sichtbar, aber `disabled`, mit erklärender Notice
