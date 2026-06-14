## MODIFIED Requirements

### Requirement: Erweiterter Kader ist je Kader verwaltbar

Das System SHALL eine Tabelle `kader_extended_members` bereitstellen, die Gelegenheitsspieler einem Kader zuordnet. Erweiterte Kader-Mitglieder erscheinen NICHT in `player_memberships` und NICHT in Training-Teilnahmelisten. Sie erscheinen jedoch in `user_accessible_teams` (Teamzugang) und in `GET /api/games/{id}/participants` mit `is_extended: true`.

#### Scenario: Erweitertes Mitglied taucht nicht in player_memberships auf

- **WHEN** ein Mitglied nur in `kader_extended_members` für einen Kader eingetragen ist
- **THEN** erscheint es nicht in `player_memberships` für dieses Team

#### Scenario: Erweitertes Mitglied taucht nicht in Training-Attendance auf

- **WHEN** ein Mitglied in `kader_extended_members` für einen Kader eingetragen ist
- **THEN** erscheint es nicht in `GET /api/training-sessions/{id}/attendances`

#### Scenario: Erweitertes Mitglied hat Teamzugang

- **WHEN** ein Mitglied in `kader_extended_members` für einen Kader eingetragen ist
- **THEN** erscheint das Team in `GET /api/teams` des zugehörigen Users
- **THEN** kann der User `GET /api/teams/{id}/roster` für dieses Team aufrufen (HTTP 200)
