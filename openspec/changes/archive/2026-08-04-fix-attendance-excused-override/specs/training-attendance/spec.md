## MODIFIED Requirements

### Requirement: Trainer kann Anwesenheit nach dem Training erfassen

Ein Trainer oder Admin SHALL nach einem Training die tatsächliche Anwesenheit aller Mitglieder des Teams als Bulk-Operation erfassen können. Bestehende Einträge werden überschrieben. Ein eingehender Eintrag mit `present=false` SHALL **nicht** persistiert werden (keine `training_attendances`-Zeile angelegt oder aktualisiert), wenn für dieses Mitglied und diese Session eine Absage vorliegt (`training_responses.status='declined'`) — unabhängig davon, ob eine Abwesenheit hinterlegt ist (`absence_id`). Ein Eintrag mit `present=true` SHALL für dieses Mitglied unabhängig vom Absage-Status immer persistiert werden.

#### Scenario: Anwesenheit erfassen

- **WHEN** ein Trainer POST `/api/training-sessions/{id}/attendances` mit einem Array `[{member_id: 5, present: true}, {member_id: 7, present: false}]` aufruft
- **THEN** werden für alle angegebenen Mitglieder `training_attendances`-Rows angelegt oder aktualisiert (Upsert auf UNIQUE(training_id, member_id))

#### Scenario: Trainer kann nur für eigenes Team erfassen

- **WHEN** ein User mit `role='trainer'` versucht, Anwesenheit für eine Session eines anderen Teams zu erfassen
- **THEN** antwortet das System mit HTTP 403

#### Scenario: `present=false` für Mitglied mit genehmigter Abwesenheit wird nicht persistiert

- **WHEN** ein Trainer `POST /api/training-sessions/{id}/attendances` mit einem Eintrag `{ "member_id": 191, "present": false }` aufruft und für Mitglied 191 eine `training_responses`-Zeile mit `status='declined'` und gesetzter `absence_id` für diese Session existiert und noch keine `training_attendances`-Zeile für dieses Mitglied existiert
- **THEN** wird für Mitglied 191 **keine** `training_attendances`-Zeile angelegt
- **AND** die übrigen Einträge desselben Pakets werden normal gespeichert, die Antwort bleibt HTTP 204

#### Scenario: `present=false` für manuell abgesagtes Mitglied ohne Abwesenheit wird ebenfalls nicht persistiert

- **WHEN** ein Trainer `POST /api/training-sessions/{id}/attendances` mit einem Eintrag `{ "member_id": 42, "present": false }` aufruft und für Mitglied 42 eine `training_responses`-Zeile mit `status='declined'` und `absence_id IS NULL` (manuelle Absage ohne hinterlegte Abwesenheit) für diese Session existiert und noch keine `training_attendances`-Zeile für dieses Mitglied existiert
- **THEN** wird für Mitglied 42 **keine** `training_attendances`-Zeile angelegt

#### Scenario: `present=true` überschreibt eine Absage weiterhin, unabhängig von `absence_id`

- **WHEN** ein Trainer `POST /api/training-sessions/{id}/attendances` mit einem Eintrag `{ "member_id": 191, "present": true }` aufruft und für Mitglied 191 eine `declined`-Antwort für diese Session existiert (mit oder ohne `absence_id`)
- **THEN** wird die `training_attendances`-Zeile für Mitglied 191 mit `present=1` angelegt bzw. aktualisiert

#### Scenario: Bereits gespeicherte Anwesenheit bleibt bei nachträglichem `present=false`-Sweep unverändert entschuldigt

- **WHEN** für ein abgesagtes Mitglied noch keine `training_attendances`-Zeile existiert und ein Bulk-Save mit `present=false` für dieses Mitglied wiederholt aufgerufen wird
- **THEN** bleibt das Mitglied ohne `training_attendances`-Zeile und wird von der Statistik weiterhin als `excused` klassifiziert (Capability `attendance-statistics`)
