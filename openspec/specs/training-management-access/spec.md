### Requirement: Trainer-Zugriff auf Trainings basiert auf kader_trainers

Das System SHALL einem Nutzer Trainer-Zugriff auf ein Team erteilen, wenn er in `kader_trainers` als Trainer eines Kaders dieses Teams eingetragen ist (unabhängig von Saison). Admins haben immer Vollzugriff.

#### Scenario: Trainer sieht Trainingsserien seines Teams

- **WHEN** ein Nutzer mit `kader_trainers`-Eintrag für Team X `GET /api/training-series` aufruft
- **THEN** werden alle Serien für Team X zurückgegeben

#### Scenario: Trainer sieht keine Trainingsserien fremder Teams

- **WHEN** ein Nutzer mit `kader_trainers`-Eintrag nur für Team X `GET /api/training-series` aufruft
- **THEN** werden keine Serien für Team Y zurückgegeben

#### Scenario: Trainer darf Serie für sein Team anlegen

- **WHEN** ein Nutzer mit `kader_trainers`-Eintrag für Team X `POST /api/training-series` mit `team_id = X` aufruft
- **THEN** wird die Serie angelegt (HTTP 201)

#### Scenario: Trainer darf keine Serie für fremdes Team anlegen

- **WHEN** ein Nutzer mit `kader_trainers`-Eintrag nur für Team X `POST /api/training-series` mit `team_id = Y` aufruft
- **THEN** wird HTTP 403 zurückgegeben

#### Scenario: Admin darf Serie für beliebiges Team anlegen

- **WHEN** ein Admin `POST /api/training-series` mit beliebiger `team_id` aufruft
- **THEN** wird die Serie angelegt (HTTP 201)

### Requirement: Mitgliederliste für Trainer ist auf kader_trainers-Teams beschränkt

Das System SHALL einem Trainer (`HasFunction("trainer")`) bei `GET /api/members` nur Mitglieder zurückgeben, die in einem Team aktiv sind, für das der Nutzer in `kader_trainers` als Trainer eingetragen ist.

#### Scenario: Trainer sieht nur eigene Team-Mitglieder

- **WHEN** ein Trainer mit Zugriff auf Team X `GET /api/members` aufruft
- **THEN** werden nur Mitglieder zurückgegeben, die eine `team_membership` für Team X haben

#### Scenario: Admin sieht alle Mitglieder

- **WHEN** ein Admin `GET /api/members` aufruft
- **THEN** werden alle nicht-ausgetretenen Mitglieder zurückgegeben

### Requirement: Trainer-Benachrichtigung bei Beitrittsantrag über kader_trainers

Das System SHALL bei einem neuen Beitrittsantrag für Team X alle Nutzer per E-Mail benachrichtigen, die in `kader_trainers` als Trainer eines Kaders von Team X eingetragen sind.

#### Scenario: Trainer erhält E-Mail bei Beitrittsantrag

- **WHEN** ein Beitrittsantrag für Team X eingeht
- **THEN** erhalten alle Nutzer mit `kader_trainers`-Eintrag für Team X eine Benachrichtigungs-E-Mail
