## ADDED Requirements

### Requirement: Trainer kann Anwesenheit nach dem Training erfassen
Ein Trainer oder Admin SHALL nach einem Training die tatsächliche Anwesenheit aller Mitglieder des Teams als Bulk-Operation erfassen können. Bestehende Einträge werden überschrieben.

#### Scenario: Anwesenheit erfassen
- **WHEN** ein Trainer POST `/api/training-sessions/{id}/attendances` mit einem Array `[{member_id: 5, present: true}, {member_id: 7, present: false}]` aufruft
- **THEN** werden für alle angegebenen Mitglieder `training_attendances`-Rows angelegt oder aktualisiert (Upsert auf UNIQUE(training_id, member_id))

#### Scenario: Trainer kann nur für eigenes Team erfassen
- **WHEN** ein User mit `role='trainer'` versucht, Anwesenheit für eine Session eines anderen Teams zu erfassen
- **THEN** antwortet das System mit HTTP 403

### Requirement: Trainer kann Anwesenheitsliste einer Session abrufen
Ein Trainer oder Admin SHALL die Anwesenheitsliste einer Session abrufen können, die beide Dimensionen zeigt: RSVP-Status (was angesagt wurde) und tatsächliche Anwesenheit.

#### Scenario: Anwesenheitsliste abrufen
- **WHEN** ein Trainer GET `/api/training-sessions/{id}/attendances` aufruft
- **THEN** erhält er eine Liste aller Teammitglieder mit jeweils `rsvp_status` (aus `training_responses`) und `present` (aus `training_attendances`, null wenn noch nicht erfasst)

#### Scenario: Diskrepanz sichtbar
- **WHEN** ein Mitglied `rsvp_status='confirmed'` hat, aber `present=false`
- **THEN** sind beide Werte in der Liste sichtbar, sodass Trainer Zusagen ohne Erscheinen erkennen kann

### Requirement: Anwesenheitserfassung nur für vergangene oder aktuelle Sessions
Das System SHALL verhindern, dass Anwesenheit für Sessions in der Zukunft erfasst wird.

#### Scenario: Zukunfts-Session blockiert
- **WHEN** ein Trainer POST `/api/training-sessions/{id}/attendances` für eine Session aufruft, deren `date` in der Zukunft liegt
- **THEN** antwortet das System mit HTTP 422 und der Meldung, dass Anwesenheit erst nach dem Termin erfasst werden kann
