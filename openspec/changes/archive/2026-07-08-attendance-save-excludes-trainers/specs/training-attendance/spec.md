## ADDED Requirements

### Requirement: Anwesenheit-Speichern ignoriert Trainer-Roster-Einträge
`POST /api/training-sessions/{id}/attendances` SHALL Einträge, die sich auf einen
Trainer-only-Member beziehen (im Kader-Trainerstab, nicht als Spieler im Kader), still
überspringen und die verbleibenden Spieler-Einträge speichern. Ein einzelner Trainer-Eintrag
im Paket darf das Speichern der Spieler-Einträge NICHT verhindern. Die fachliche Regel
„Trainer haben keine Anwesenheitserfassung" bleibt bestehen — für Trainer wird keine
`training_attendances`-Zeile geschrieben.

#### Scenario: Paket mit Trainer und Spieler speichert den Spieler
- **WHEN** ein Trainer `POST /api/training-sessions/{id}/attendances` für ein vergangenes Training mit einem Paket aufruft, das sowohl einen Spieler (`present=true`) als auch einen Trainer des Teams enthält
- **THEN** antwortet die API mit HTTP 204, die `present`-Angabe des Spielers ist persistiert und für den Trainer existiert keine `training_attendances`-Zeile

#### Scenario: Paket nur mit Trainer-Eintrag ist ein No-op
- **WHEN** das Paket ausschließlich Trainer-Einträge enthält
- **THEN** antwortet die API mit HTTP 204 und schreibt keine `training_attendances`-Zeile

#### Scenario: Fremdes Team weiterhin abgewiesen
- **WHEN** ein Nutzer ohne Trainer-Zugriff auf das Team die Route aufruft
- **THEN** antwortet die API mit HTTP 403
