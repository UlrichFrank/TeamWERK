## ADDED Requirements

### Requirement: Spiel-Anwesenheit-Speichern ignoriert Trainer-Roster-Einträge
`POST /api/games/{id}/attendances` SHALL Einträge, die sich auf einen Trainer-only-Member
eines beteiligten Teams beziehen (im Kader-Trainerstab, nicht als Spieler im Kader), still
überspringen und die verbleibenden Spieler-Einträge speichern. Ein einzelner Trainer-Eintrag
im Paket darf das Speichern der Spieler-Einträge NICHT verhindern. Für Trainer wird keine
`game_attendances`-Zeile geschrieben.

#### Scenario: Paket mit Trainer und Spieler speichert den Spieler
- **WHEN** ein Trainer `POST /api/games/{id}/attendances` für ein vergangenes Spiel mit einem Paket aufruft, das sowohl einen Spieler (`present=true`) als auch einen Trainer eines beteiligten Teams enthält
- **THEN** antwortet die API mit HTTP 204, die `present`-Angabe des Spielers ist persistiert und für den Trainer existiert keine `game_attendances`-Zeile

#### Scenario: Paket nur mit Trainer-Eintrag ist ein No-op
- **WHEN** das Paket ausschließlich Trainer-Einträge enthält
- **THEN** antwortet die API mit HTTP 204 und schreibt keine `game_attendances`-Zeile

#### Scenario: Zukünftiges Spiel weiterhin abgewiesen
- **WHEN** ein Trainer die Route für ein Spiel mit Datum in der Zukunft aufruft
- **THEN** antwortet die API mit HTTP 422
