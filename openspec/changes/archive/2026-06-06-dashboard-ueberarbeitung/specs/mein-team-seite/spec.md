## ADDED Requirements

### Requirement: Team-Roster-Endpoint

Das System SHALL einen Endpoint `GET /api/teams/:id/roster` bereitstellen, der für ein Team die vollständige Personenübersicht zurückgibt.

Rückgabe:
- `team`: `id`, `name`
- `trainers`: Array mit `name`, `email` (aus `users`)
- `players`: Array mit `name`, `jersey_number` (nullable), `status`, `email` (aus verlinktem `user`, nullable)
- `parents`: Array mit `name`, `email`, `children` (Array der Namen der verlinkten Kinder im Team)

Berechtigung: Alle authentifizierten User mit Zugriff auf das Team (via `user_accessible_teams`). Kein Zugriff für User ohne Team-Verbindung → HTTP 403.

#### Scenario: Trainer ruft Roster ab

- **WHEN** ein Trainer `GET /api/teams/1/roster` aufruft für ein Team, dem er zugehört
- **THEN** gibt das System HTTP 200 mit `trainers`, `players` und `parents` zurück

#### Scenario: Spieler ohne Team-Zugriff

- **WHEN** ein Spieler den Roster eines Teams abruft, dem er nicht angehört
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Eltern erscheinen mit Kindernamen

- **WHEN** der Roster ein Team enthält, bei dem Elternteile via `family_links` verknüpft sind
- **THEN** enthält jeder Elternteil-Eintrag in `children` die Namen der verlinkten Kinder, die diesem Team angehören

### Requirement: Mein-Team-Seite im Frontend

Das System SHALL eine Seite `/mein-team` bereitstellen, die je Team des eingeloggten Users eine gestapelte Tabelle mit Trainern, Spielern und Eltern anzeigt.

Anforderungen:
- Ein Abschnitt pro Team (ohne Tabs zur Umschaltung)
- Drei Abschnitte je Team: Trainer, Spieler, Eltern
- Angezeigte Felder: Name, E-Mail (verlinkt als `mailto:`)
- Für Spieler zusätzlich: Trikotnummer und Aktiv-Status

#### Scenario: Dashboard-Link je Team

- **WHEN** der User in der Dashboard-Kachel "Mein Team" ist
- **THEN** sieht er einen Link pro Team, auf das er Zugriff hat, der zur `/mein-team`-Seite führt

#### Scenario: User mit zwei Teams

- **WHEN** ein Trainer zwei Teams hat
- **THEN** zeigt `/mein-team` zwei gestapelte Abschnitte, einen je Team

#### Scenario: Kein Team vorhanden

- **WHEN** der User keinem Team zugeordnet ist
- **THEN** zeigt `/mein-team` einen Hinweis "Kein Team zugeordnet"
