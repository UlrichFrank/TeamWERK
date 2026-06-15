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

### Requirement: Admin kann erweiterte Kader-Mitglieder hinzufügen und entfernen

Das System SHALL `PUT /api/kader/{id}` um die Felder `extended_members_add` und `extended_members_remove` erweitern, analog zu `members_add` / `members_remove`.

#### Scenario: Mitglied zum erweiterten Kader hinzufügen

- **WHEN** ein Trainer/Admin `PUT /api/kader/{id}` mit `extended_members_add: [memberId]` sendet
- **THEN** wird das Mitglied in `kader_extended_members` eingetragen und erscheint in der Antwort von `GET /api/kader/{id}` unter `extended_members`

#### Scenario: Mitglied aus erweitertem Kader entfernen

- **WHEN** ein Trainer/Admin `PUT /api/kader/{id}` mit `extended_members_remove: [memberId]` sendet
- **THEN** wird das Mitglied aus `kader_extended_members` entfernt

### Requirement: Kader-Antwort enthält extended_members

Das System SHALL `GET /api/kader` und `GET /api/kader/{id}` um das Feld `extended_members` (Array mit id, name, birth_year, gender) erweitern.

#### Scenario: Kader-Detailantwort enthält erweiterte Mitglieder

- **WHEN** ein Kader erweiterte Mitglieder hat
- **THEN** enthält die Antwort von `GET /api/kader/{id}` das Feld `extended_members` mit den zugehörigen Mitgliedern

### Requirement: Admin-UI zeigt Abschnitt „Erweiterter Kader"

Das System SHALL auf `/kader` je Mannschafts-Card einen dritten Abschnitt „Erweiterter Kader" anzeigen, unterhalb des Mitglieder-Abschnitts, mit gleichem Muster: Suchfeld + Liste + Entfernen-Button.

#### Scenario: Erweitertes Mitglied erscheint in der Kader-Card

- **WHEN** ein Mitglied als erweitertes Kader-Mitglied eingetragen ist
- **THEN** erscheint es im Abschnitt „Erweiterter Kader" der entsprechenden Mannschafts-Card

#### Scenario: Erweitertes Mitglied ist entfernbar

- **WHEN** der Trainer auf × neben einem erweiterten Kader-Mitglied klickt
- **THEN** wird es aus dem erweiterten Kader entfernt und verschwindet aus der Liste
