## ADDED Requirements

### Requirement: Alle aktiven Teams abrufbar für Namensberechnung

Das System SHALL einen Endpoint `GET /api/teams/names` bereitstellen, der für alle eingeloggten User alle aktiven Teams mit den Feldern zurückgibt, die für die clientseitige Kurznamensberechnung nötig sind: `id`, `age_class`, `gender`, `team_number`, `group_count`.

Das `name`-Feld der `teams`-Tabelle wird nicht zurückgegeben. Keine rollenabhängige Filterung.

#### Scenario: Spieler ruft Endpoint auf
- **WHEN** ein eingeloggter User mit Rolle `spieler` oder `elternteil` `GET /api/teams/names` aufruft
- **THEN** erhält er eine JSON-Liste aller aktiven Teams mit `id`, `age_class`, `gender`, `team_number`, `group_count`
- **THEN** enthält die Liste auch Teams, in deren Kader der User nicht ist

#### Scenario: Admin ruft Endpoint auf
- **WHEN** ein User mit Rolle `admin` `GET /api/teams/names` aufruft
- **THEN** erhält er dieselbe vollständige Liste wie ein Spieler

#### Scenario: Unauthentifizierter Zugriff
- **WHEN** ein nicht eingeloggter User `GET /api/teams/names` aufruft
- **THEN** antwortet das System mit HTTP 401

### Requirement: Frontend zeigt überall berechnete Kurznamen

Das Frontend SHALL für alle Team-Anzeigen die berechneten Kurznamen aus `buildTeamShortNames` verwenden. Kein Fallback auf rohe DB-Namen (`t.name`) für Team-Anzeige in der UI.

#### Scenario: Spieler öffnet Kalender mit generischem Event
- **WHEN** ein Spieler den Kalender öffnet und auf ein generisches Event mit mehreren Teams klickt
- **THEN** sieht er für alle Teams den berechneten Kurznamen (z.B. "mA2"), unabhängig davon ob er in diesen Teams ist

#### Scenario: Admin öffnet Kalender
- **WHEN** ein Admin den Kalender öffnet
- **THEN** sieht er dieselben Kurznamen wie ein Spieler für dieselben Teams
