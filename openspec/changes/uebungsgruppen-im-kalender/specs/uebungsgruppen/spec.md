## MODIFIED Requirements

### Requirement: Übungsgruppe als Kader-Variante ohne Team

Das System SHALL eine **Übungsgruppe** als Zeile in `kader` mit `kind = 'practice'`
führen. Für diese Variante gilt verbindlich: `name` ist gesetzt, und `age_class`,
`gender`, `dedicated_birth_year` sowie `team_id` sind **NULL**. Für die
Mannschaftsvariante (`kind = 'team'`) gilt umgekehrt: `age_class` und `gender` sind
gesetzt, `name` ist NULL. Ein CHECK-Constraint SHALL diese Alles-oder-nichts-Regel
erzwingen; die Anwendung SHALL sich nicht darauf verlassen, die einzige Schreibstelle zu
sein.

Eine Übungsgruppe SHALL **niemals** eine `teams`-Zeile besitzen. Diese Abwesenheit ist der
Mechanismus, über den alle team-gebundenen Flächen (Spiele, Dienste, Mannschaftskasse,
Strafen, Aufgaben, Videos, Ordner-Principals, Statistiken) für Übungsgruppen unerreichbar
bleiben — ohne dass eine dieser Flächen etwas prüfen muss.

Der **iCal-Feed gehört nicht mehr dazu**: er löst Trainingstermine über
`training_sessions.kader_id` statt über `team_id` auf und gibt Übungsgruppen-Termine aus,
gesteuert über den eigenen Toggle `calendar_tokens.include_practice_groups` (Capability
`ical-feed`). Eine Fläche verlässt diesen Ausschluss ausschließlich auf diesem Weg — durch
den Wechsel des Ankers auf `kader_id` und einen expliziten Schalter, **niemals** durch das
Füllen von `training_sessions.team_id`.

Namen SHALL innerhalb einer Saison eindeutig sein (Partial-Unique-Index auf
`(season_id, name) WHERE kind='practice'`).

#### Scenario: Anlage erzeugt keinen Team-Zwilling
- **WHEN** der Vorstand `POST /api/practice-groups` mit `{"name":"Torwarttraining"}` sendet
- **THEN** entsteht eine `kader`-Zeile mit `kind='practice'`, `name='Torwarttraining'`,
  `team_id IS NULL`, `age_class IS NULL`, `gender IS NULL`,
  `dedicated_birth_year IS NULL` — und **keine** neue Zeile in `teams`

#### Scenario: Doppelter Name in derselben Saison
- **WHEN** eine Übungsgruppe „Torwarttraining" in der aktiven Saison bereits existiert und
  eine zweite mit demselben Namen angelegt wird
- **THEN** antwortet das System mit HTTP 409 und legt nichts an

#### Scenario: Gleicher Name in einer anderen Saison
- **WHEN** eine Übungsgruppe „Torwarttraining" in Saison 2025/26 existiert und dieselbe in
  Saison 2026/27 angelegt wird
- **THEN** wird sie angelegt (HTTP 201) — die Eindeutigkeit gilt je Saison

#### Scenario: Ohne aktive Saison
- **WHEN** keine Saison `is_active=1` gesetzt ist und eine Übungsgruppe angelegt wird
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Datenbank weist einen Team-Zwilling ab
- **WHEN** versucht wird, einer Zeile mit `kind='practice'` ein `team_id` zu setzen
- **THEN** schlägt die Schreiboperation am CHECK-Constraint fehl
