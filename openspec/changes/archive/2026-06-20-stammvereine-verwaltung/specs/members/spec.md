## ADDED Requirements

### Requirement: Stammverein eines Mitglieds als Referenz
Ein Mitglied MUST einen Stammverein über `members.home_club_id` (FK auf `stammvereine`) zugeordnet bekommen können. `NULL` bedeutet „kein Stammverein". Beim Aktualisieren eines Mitglieds (`PUT /api/members/{id}`) MUST das Feld `home_club_id` (nullable Integer) akzeptiert und persistiert werden; ein gesetzter Wert MUST auf einen existierenden `stammvereine`-Eintrag verweisen, sonst HTTP 400.

#### Scenario: Stammverein zuweisen
- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `home_club_id` eines existierenden Vereins sendet
- **THEN** wird die Zuordnung gespeichert und in `GET /api/members/{id}` zurückgegeben

#### Scenario: Stammverein entfernen
- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `home_club_id: null` sendet
- **THEN** wird die Zuordnung entfernt (Mitglied gilt im Beitragslauf als `aktiv_ohne`)

#### Scenario: Ungültiger Verein
- **WHEN** ein `PUT /api/members/{id}` mit einer `home_club_id` ohne passenden `stammvereine`-Eintrag erfolgt
- **THEN** antwortet der Server mit HTTP 400
