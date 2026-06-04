## ADDED Requirements

### Requirement: Training-Detail zeigt vollständige Kaderliste für alle authentifizierten User

`GET /api/training-sessions/{id}/attendances` SHALL für alle authentifizierten User zugänglich sein, die entweder selbst Kader-Mitglied des Teams sind oder ein Kind im Kader haben. Bisher war dieser Endpoint Trainer-only.

#### Scenario: Spieler ruft Training-Detail ab

- **WHEN** ein User mit Rolle `spieler`, der Kader-Mitglied des betreffenden Teams ist, `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** antwortet der Server mit HTTP 200
- **THEN** enthält die Antwort alle Kader-Mitglieder mit deren RSVP-Status

#### Scenario: Elternteil ruft Training-Detail ab

- **WHEN** ein Elternteil, dessen Kind Kader-Mitglied ist, `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** antwortet der Server mit HTTP 200
- **THEN** enthält die Antwort alle Kader-Mitglieder mit deren RSVP-Status

#### Scenario: Fremder User wird abgelehnt

- **WHEN** ein User, der weder Kader-Mitglied noch Elternteil eines Kader-Mitglieds ist, `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: `present`-Feld nur für Trainer sichtbar

Das Feld `present` in der Attendances-Response SHALL nur für Trainer und Admins einen Wert enthalten. Für Spieler und Eltern ist `present` immer `null`.

#### Scenario: Nicht-Trainer erhält kein present-Flag

- **WHEN** ein Spieler oder Elternteil `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `present` für alle Einträge `null`

#### Scenario: Trainer erhält present-Flags

- **WHEN** ein Trainer `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** enthält `present` den tatsächlich gespeicherten Wert (true/false/null)

---

### Requirement: Kommentar-Sichtbarkeit auf der Training-Detail-Seite

`GET /api/training-sessions/{id}/attendances` SHALL ein Feld `reason` zurückgeben, gefiltert nach Rolle:
- Trainer/Admin: alle Kommentare aller Spieler
- Spieler: nur der eigene Kommentar
- Elternteil: nur Kommentare der eigenen Kinder

#### Scenario: Trainer sieht alle Kommentare

- **WHEN** ein Trainer `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `reason` für alle Einträge mit vorhandenem Kommentar befüllt

#### Scenario: Spieler sieht nur eigenen Kommentar

- **WHEN** ein Spieler `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `reason` nur für den eigenen Eintrag befüllt; alle anderen haben `reason: null`

#### Scenario: Elternteil sieht nur Kinder-Kommentare

- **WHEN** ein Elternteil `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `reason` nur für Einträge der eigenen Kinder befüllt; alle anderen haben `reason: null`
