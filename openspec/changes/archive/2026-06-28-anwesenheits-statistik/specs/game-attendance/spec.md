## ADDED Requirements

### Requirement: Trainer kann Spiel-Anwesenheit nach dem Spiel erfassen

Ein Trainer eines Teams des Spiels, ein Mitglied der sportlichen Leitung oder ein Admin SHALL nach einem Spiel die tatsächliche Anwesenheit aller Kader-Mitglieder als Bulk-Operation erfassen können. Bestehende Einträge werden überschrieben (Upsert auf `UNIQUE(game_id, member_id)`).

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

### Requirement: Anwesenheitsliste eines Spiels abrufen

Ein Trainer eines Teams des Spiels, ein Mitglied der sportlichen Leitung oder ein Admin SHALL die Anwesenheitsliste eines Spiels abrufen können. Die Antwort enthält pro Mitglied der Stamm- und erweiterten Kader der beteiligten Teams: `member_id`, `member_name`, `rsvp_status` (nullable), `reason`, `present` (nullable, bool) und `is_extended` (bool).

#### Scenario: Trainer ruft Anwesenheitsliste ab

- **WHEN** ein Trainer `GET /api/games/{id}/attendances` für ein Spiel seines Teams aufruft
- **THEN** erhält er HTTP 200 mit der Mitgliederliste inkl. `rsvp_status`, `present` und `is_extended` für jedes Mitglied der beteiligten Kader

#### Scenario: Erweiterter Kader korrekt markiert

- **WHEN** ein Mitglied nur via `kader_extended_members` zum Team eines Spiels gehört
- **THEN** hat es in der Antwort `is_extended: true`

#### Scenario: Mitglied in beiden Kadern erscheint einmal

- **WHEN** ein Mitglied sowohl im Stamm- als auch im erweiterten Kader eines Teams ist
- **THEN** erscheint es in der Antwort genau einmal mit `is_extended: false`

#### Scenario: Fehlende Erfassung liefert present=null

- **WHEN** ein Mitglied in der Liste enthalten ist, für das keine `game_attendances`-Row existiert
- **THEN** ist `present` für dieses Mitglied `null`

#### Scenario: Unauthorisiertes Abrufen abgewiesen

- **WHEN** ein eingeloggter Spieler ohne Trainer-Funktion `GET /api/games/{id}/attendances` aufruft
- **THEN** antwortet das System mit HTTP 403
