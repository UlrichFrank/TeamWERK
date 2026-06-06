## ADDED Requirements

### Requirement: player_memberships View existiert

Das System SHALL einen SQLite-View `player_memberships` bereitstellen, der ausschließlich Mitglieder aus `kader_members` enthält (keine Trainer).

#### Scenario: Trainer erscheinen nicht in player_memberships

- **WHEN** ein Mitglied nur in `kader_trainers` für ein Team eingetragen ist
- **THEN** erscheint dieses Mitglied nicht in `player_memberships` für dieses Team

#### Scenario: Spieler erscheinen in player_memberships

- **WHEN** ein Mitglied in `kader_members` für ein Team eingetragen ist
- **THEN** erscheint dieses Mitglied in `player_memberships` mit den Feldern `member_id`, `team_id`, `season_id`

### Requirement: trainer_memberships View existiert

Das System SHALL einen SQLite-View `trainer_memberships` bereitstellen, der ausschließlich Mitglieder aus `kader_trainers` enthält (keine Spieler).

#### Scenario: Spieler erscheinen nicht in trainer_memberships

- **WHEN** ein Mitglied nur in `kader_members` für ein Team eingetragen ist
- **THEN** erscheint dieses Mitglied nicht in `trainer_memberships` für dieses Team

#### Scenario: Trainer erscheinen in trainer_memberships

- **WHEN** ein Mitglied in `kader_trainers` für ein Team eingetragen ist
- **THEN** erscheint dieses Mitglied in `trainer_memberships` mit den Feldern `member_id`, `team_id`, `season_id`

### Requirement: Trainings-Teilnahmeliste zeigt nur Spieler

Das System SHALL in der Teilnahmeliste eines Trainingstermins (`GET /api/training-sessions/{id}/attendances`) ausschließlich Mitglieder aus `player_memberships` anzeigen.

#### Scenario: Trainer wird nicht in Teilnahmeliste angezeigt

- **WHEN** ein Trainer-Mitglied in `kader_trainers` für das Team eingetragen ist, aber nicht in `kader_members`
- **THEN** erscheint dieses Mitglied nicht in der Antwort von `GET /api/training-sessions/{id}/attendances`

#### Scenario: Spieler wird in Teilnahmeliste angezeigt

- **WHEN** ein Spieler-Mitglied in `kader_members` für das Team eingetragen ist
- **THEN** erscheint dieses Mitglied in der Antwort von `GET /api/training-sessions/{id}/attendances`

### Requirement: RSVP-opt-out-Zählung basiert nur auf Spielern

Das System SHALL beim Berechnen von `confirmed_count` im rsvp_opt_out-Modus ausschließlich Spieler (aus `player_memberships`) zählen.

#### Scenario: Trainer zählt nicht als implizit bestätigt

- **WHEN** ein Training hat `rsvp_opt_out = 1` und ein Trainer hat keine RSVP-Antwort gegeben
- **THEN** wird der Trainer nicht zum `confirmed_count` addiert

### Requirement: team_memberships bleibt für Spielzugänglichkeit erhalten

Das System SHALL den bestehenden `team_memberships`-View unverändert beibehalten, da er Spieler und Trainer für Zugriffskontrollen im Spielplan kombiniert.

#### Scenario: Trainer sieht Spiele seines Teams

- **WHEN** ein Trainer in `kader_trainers` für ein Team eingetragen ist
- **THEN** kann er Spiele dieses Teams über die Games-API einsehen
