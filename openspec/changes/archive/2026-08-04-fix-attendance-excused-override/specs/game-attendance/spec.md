## MODIFIED Requirements

### Requirement: Trainer kann Spiel-Anwesenheit nach dem Spiel erfassen

Ein Trainer eines Teams des Spiels, ein Mitglied der sportlichen Leitung oder ein Admin SHALL nach einem Spiel die tatsächliche Anwesenheit aller Kader-Mitglieder als Bulk-Operation erfassen können. Bestehende Einträge werden überschrieben (Upsert auf `UNIQUE(game_id, member_id)`). Ein eingehender Eintrag mit `present=false` SHALL **nicht** persistiert werden (keine `game_attendances`-Zeile angelegt oder aktualisiert), wenn für dieses Mitglied und dieses Spiel eine Absage vorliegt (`game_responses.status='declined'`) — unabhängig davon, ob eine Abwesenheit hinterlegt ist (`absence_id`). Ein Eintrag mit `present=true` SHALL für dieses Mitglied unabhängig vom Absage-Status immer persistiert werden.

#### Scenario: Trainer erfasst Anwesenheit für vergangenes Spiel

- **WHEN** ein Trainer `POST /api/games/{id}/attendances` mit `[{ "member_id": 5, "present": true }, { "member_id": 7, "present": false }]` für ein Spiel aufruft, dessen `date` in der Vergangenheit liegt
- **THEN** werden die `game_attendances`-Rows angelegt oder aktualisiert
- **AND** der Server sendet HTTP 200 und broadcastet `attendance-changed` über den Hub

#### Scenario: Zukünftiges Spiel blockiert Erfassung

- **WHEN** ein Trainer `POST /api/games/{id}/attendances` für ein Spiel aufruft, dessen `date` in der Zukunft liegt
- **THEN** antwortet das System mit HTTP 422 und einer Meldung, dass Anwesenheit erst nach dem Spiel erfasst werden kann

#### Scenario: Trainer eines fremden Teams abgewiesen

- **WHEN** ein Trainer ohne Trainer-Funktion in einem der Teams des Spiels `POST /api/games/{id}/attendances` aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Sportliche Leitung darf für jedes Team erfassen

- **WHEN** ein Mitglied mit Vereinsfunktion `sportliche_leitung` `POST /api/games/{id}/attendances` für ein beliebiges Spiel aufruft, dessen Datum in der Vergangenheit liegt
- **THEN** wird die Erfassung gespeichert und HTTP 200 zurückgegeben

#### Scenario: Unauthentifizierter Aufruf abgewiesen

- **WHEN** ein nicht eingeloggter Client `POST /api/games/{id}/attendances` aufruft
- **THEN** antwortet das System mit HTTP 401

#### Scenario: Spiel existiert nicht

- **WHEN** ein berechtigter Nutzer `POST /api/games/{id}/attendances` für eine nicht existierende `id` aufruft
- **THEN** antwortet das System mit HTTP 404

#### Scenario: `present=false` für Mitglied mit genehmigter Abwesenheit wird nicht persistiert

- **WHEN** ein Trainer `POST /api/games/{id}/attendances` mit einem Eintrag `{ "member_id": 191, "present": false }` aufruft und für Mitglied 191 eine `game_responses`-Zeile mit `status='declined'` und gesetzter `absence_id` für dieses Spiel existiert und noch keine `game_attendances`-Zeile für dieses Mitglied existiert
- **THEN** wird für Mitglied 191 **keine** `game_attendances`-Zeile angelegt
- **AND** die übrigen Einträge desselben Pakets werden normal gespeichert, die Antwort bleibt HTTP 200

#### Scenario: `present=false` für manuell abgesagtes Mitglied ohne Abwesenheit wird ebenfalls nicht persistiert

- **WHEN** ein Trainer `POST /api/games/{id}/attendances` mit einem Eintrag `{ "member_id": 42, "present": false }` aufruft und für Mitglied 42 eine `game_responses`-Zeile mit `status='declined'` und `absence_id IS NULL` (manuelle Absage ohne hinterlegte Abwesenheit) für dieses Spiel existiert und noch keine `game_attendances`-Zeile für dieses Mitglied existiert
- **THEN** wird für Mitglied 42 **keine** `game_attendances`-Zeile angelegt

#### Scenario: `present=true` überschreibt eine Absage weiterhin, unabhängig von `absence_id`

- **WHEN** ein Trainer `POST /api/games/{id}/attendances` mit einem Eintrag `{ "member_id": 191, "present": true }` aufruft und für Mitglied 191 eine `declined`-Antwort für dieses Spiel existiert (mit oder ohne `absence_id`)
- **THEN** wird die `game_attendances`-Zeile für Mitglied 191 mit `present=1` angelegt bzw. aktualisiert

#### Scenario: Bereits gespeicherte Anwesenheit bleibt bei nachträglichem `present=false`-Sweep unverändert entschuldigt

- **WHEN** für ein abgesagtes Mitglied noch keine `game_attendances`-Zeile existiert und ein Bulk-Save mit `present=false` für dieses Mitglied wiederholt aufgerufen wird
- **THEN** bleibt das Mitglied ohne `game_attendances`-Zeile und wird von der Statistik weiterhin als `excused` klassifiziert (Capability `attendance-statistics`)
