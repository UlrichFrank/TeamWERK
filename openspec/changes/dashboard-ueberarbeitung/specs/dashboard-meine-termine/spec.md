## ADDED Requirements

### Requirement: Tagesbasierte Terminanzeige im Dashboard

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `meineTermine` alle Events des nächsten Tages zurückgeben, an dem für den User mindestens ein Termin existiert.

Event-Quellen:
- `training_sessions` (Typ: `training`)
- `games` mit `event_type` = `heim`, `auswärts` oder `generisch` (Typ: `spiel`)

Jeder Event enthält: `id`, `type` (`training`|`spiel`), `date`, `time`, `title` (Trainingsname oder Gegner/Event-Bezeichnung), `team_name`, `detail_url` (`/termine/training/:id` bzw. `/termine/spiel/:id`).

#### Scenario: Nächster Tag hat Trainings und Spiele

- **WHEN** der nächste Tag mit Terminen sowohl `training_sessions` als auch `games` enthält
- **THEN** gibt `meineTermine` alle Events dieses Tages zurück, chronologisch nach `time` sortiert

#### Scenario: Nächster Tag hat mehr als drei Events

- **WHEN** an dem nächsten Tag mit Terminen vier oder mehr Events existieren
- **THEN** gibt `meineTermine` alle Events dieses Tages zurück (keine Kürzung auf 3)

#### Scenario: Kein kommender Termin

- **WHEN** keine zukünftigen training_sessions oder games für die Teams des Users existieren
- **THEN** ist `meineTermine` ein leeres Array

#### Scenario: Detail-URL für Training

- **WHEN** `meineTermine` einen Training-Eintrag enthält
- **THEN** ist `detail_url` im Format `/termine/training/:id`

#### Scenario: Detail-URL für Spiel

- **WHEN** `meineTermine` einen Spiel-Eintrag enthält
- **THEN** ist `detail_url` im Format `/termine/spiel/:id`

#### Scenario: User mit mehreren Teams

- **WHEN** der User (z.B. Elternteil mit zwei Kindern in verschiedenen Teams) an einem Tag Termine beider Teams hat
- **THEN** gibt `meineTermine` alle Events dieses Tages aus beiden Teams zurück
