## ADDED Requirements

### Requirement: ListSessions filtert nach Team-Zugriffsberechtigung

`GET /api/training-sessions` SHALL nur Sessions zurückgeben, auf die der anfragende User Zugriff hat. Trainer sehen nur Sessions ihrer eigenen Teams; Admins sehen alle.

#### Scenario: Trainer sieht nur Sessions des eigenen Teams

- **WHEN** ein Trainer mit Zugriff auf Team A `GET /api/training-sessions?from=...&to=...` aufruft
- **THEN** enthält die Antwort nur Sessions von Team A, nicht von Team B

#### Scenario: Admin sieht Sessions aller Teams

- **WHEN** ein Admin `GET /api/training-sessions?from=...&to=...` aufruft
- **THEN** enthält die Antwort Sessions aller Teams im angefragten Zeitraum

#### Scenario: Unauthentifizierter Request wird abgelehnt

- **WHEN** `GET /api/training-sessions` ohne Token aufgerufen wird
- **THEN** antwortet der Server mit HTTP 401

---

### Requirement: CreateSeries generiert Sessions für alle Wochentage im Zeitraum

`POST /api/training-series` SHALL eine Serie anlegen und automatisch einzelne Sessions für jeden passenden Wochentag zwischen `from` und `until` generieren.

#### Scenario: Wöchentliche Serie erzeugt korrekte Anzahl Sessions

- **WHEN** eine Serie für Dienstag von 2026-01-06 bis 2026-01-27 angelegt wird
- **THEN** werden genau 4 Sessions angelegt (06., 13., 20., 27. Januar)

#### Scenario: Nur Trainer des Teams darf Serie anlegen

- **WHEN** ein Trainer ohne Zugriff auf das Ziel-Team `POST /api/training-series` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Admin darf Serie für beliebiges Team anlegen

- **WHEN** ein Admin `POST /api/training-series` mit einer gültigen Team-ID aufruft
- **THEN** wird die Serie erfolgreich angelegt (HTTP 201)

---

### Requirement: Respond speichert RSVP-Status eines Users für eine Session

`POST /api/training-sessions/{id}/respond` SHALL den RSVP-Status (`yes`/`no`/`maybe`) eines Users für eine Session speichern. Doppelte Responds für denselben User MUSS den bestehenden Eintrag aktualisieren (kein Duplicate-Error).

#### Scenario: Spieler gibt RSVP ab

- **WHEN** ein Spieler `POST /api/training-sessions/1/respond` mit `{"status": "yes"}` aufruft
- **THEN** ist der RSVP in der DB gespeichert und der HTTP-Status ist 200

#### Scenario: Spieler ändert bestehenden RSVP

- **WHEN** ein Spieler zuerst `yes` absagt und dann `no` absagt
- **THEN** ist in der DB nur ein Eintrag mit dem neuesten Status `no`

#### Scenario: User ohne Zugriff auf die Session kann nicht respondieren

- **WHEN** ein User `POST /api/training-sessions/{id}/respond` für eine Session eines fremden Teams aufruft
- **THEN** antwortet der Server mit HTTP 403 oder 404

---

### Requirement: SaveAttendances ist auf Trainer und Admin beschränkt

`POST /api/training-sessions/{id}/attendances` SHALL Anwesenheitsdaten speichern. Nur Trainer des zugehörigen Teams und Admins DÜRFEN diesen Endpunkt aufrufen.

#### Scenario: Trainer speichert Anwesenheit erfolgreich

- **WHEN** ein Trainer mit Zugriff auf die Session `POST /api/training-sessions/1/attendances` mit einer Liste von Member-IDs aufruft
- **THEN** werden die Anwesenheitsdaten gespeichert (HTTP 200)

#### Scenario: Spieler darf keine Anwesenheit speichern

- **WHEN** ein User mit Rolle `spieler` `POST /api/training-sessions/1/attendances` aufruft
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: ListGames gibt Spielplan für den angefragten Zeitraum zurück

`GET /api/kalender` SHALL alle Spiele für das angefragte Team und den angefragten Zeitraum zurückgeben.

#### Scenario: Spiele im Zeitraum werden zurückgegeben

- **WHEN** `GET /api/kalender?from=2026-01-01&to=2026-01-31` aufgerufen wird
- **THEN** enthält die Antwort alle Spiele im Januar 2026

#### Scenario: Leere Liste wenn keine Spiele vorhanden

- **WHEN** `GET /api/kalender?from=2030-01-01&to=2030-01-31` aufgerufen wird und keine Spiele existieren
- **THEN** antwortet der Server mit HTTP 200 und einem leeren Array

---

### Requirement: CreateGame legt ein neues Spiel an (Admin only)

`POST /api/admin/kalender` SHALL ein neues Spiel anlegen. Nur Admins DÜRFEN diesen Endpunkt aufrufen.

#### Scenario: Admin legt Heimspiel an

- **WHEN** ein Admin `POST /api/admin/kalender` mit gültigen Feldern (`team_id`, `opponent`, `date`, `is_home: true`) aufruft
- **THEN** wird das Spiel angelegt und mit HTTP 201 zurückgegeben

#### Scenario: Trainer darf kein Spiel anlegen

- **WHEN** ein Trainer `POST /api/admin/kalender` aufruft
- **THEN** antwortet der Server mit HTTP 403
