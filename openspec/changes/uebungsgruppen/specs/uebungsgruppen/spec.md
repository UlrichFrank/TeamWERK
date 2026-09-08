## ADDED Requirements

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
Strafen, Aufgaben, Videos, Ordner-Principals, Statistiken, Kalender-Feed) für
Übungsgruppen unerreichbar bleiben — ohne dass eine dieser Flächen etwas prüfen muss.

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

### Requirement: Verwaltung von Übungsgruppen

Das System SHALL Vorstand und `admin` erlauben, Übungsgruppen anzulegen
(`POST /api/practice-groups`), umzubenennen und ihre Mitglieder sowie Trainer zu pflegen
(`PUT /api/practice-groups/{id}`), zu löschen (`DELETE /api/practice-groups/{id}`) sowie
zu lesen (`GET /api/practice-groups`, `GET /api/practice-groups/{id}`). Mitglieder werden
in `kader_members`, Trainer in `kader_trainers` geführt — dieselben Tabellen wie bei der
Mannschaftsvariante.

Jede Mutation SHALL `h.hub.Broadcast("practice-groups")` auslösen; das Frontend abonniert
über `useLiveUpdates`.

Das Löschen einer Übungsgruppe SHALL mit HTTP **409** und einem `training_count`
abgelehnt werden, solange Trainingstermine oder -serien an ihr hängen
(`training_sessions.kader_id`/`training_series.kader_id` tragen `ON DELETE RESTRICT`).
Anders als `teams` — die nie gelöscht, sondern nur über `is_active` stillgelegt werden —
sind Kader löschbar; ohne diese Guard verlöre ein Aufräumen am Saisonende die
Anwesenheits- und RSVP-Historie. Die Guard hat dieselbe Form wie die bestehende
Mitglieder-Guard in `DeleteKader` und gilt für **beide** Varianten.

Übungsgruppen SHALL vom Saisonkopierer (`POST /api/kader/copy-from-season`) übersprungen
werden: dieser keyt auf `age_class|gender` und ruft `ensureTeam` — bei zwei NULL-Werten
kollabierten alle Übungsgruppen auf denselben Schlüssel und es entstünde ein Team ohne
Altersklasse und Geschlecht.

#### Scenario: Mitglieder und Trainer pflegen
- **WHEN** der Vorstand `PUT /api/practice-groups/{id}` mit Mitglieds- und Trainer-IDs
  sendet
- **THEN** antwortet das System mit HTTP 200, die Zuordnungen stehen in `kader_members`
  bzw. `kader_trainers`, und ein `practice-groups`-Broadcast wird gesendet

#### Scenario: Ohne Vorstandsfunktion
- **WHEN** ein Nutzer ohne `vorstand` und ohne Rolle `admin` eine Übungsgruppe anlegt,
  ändert oder löscht
- **THEN** antwortet das System mit HTTP 403 und nichts wird geändert

#### Scenario: Löschen mit Terminen wird abgelehnt
- **WHEN** eine Übungsgruppe mit drei Trainingsterminen gelöscht werden soll
- **THEN** antwortet das System mit HTTP 409 und `training_count: 3`; Gruppe und Termine
  bleiben erhalten

#### Scenario: Löschen ohne Termine
- **WHEN** eine Übungsgruppe ohne Trainingstermine gelöscht wird
- **THEN** antwortet das System mit HTTP 204 und die Gruppe ist entfernt

#### Scenario: Leerer Altkader mit Historie bleibt geschützt
- **WHEN** ein Kader ohne Mitglieder, aber mit Trainings vergangener Saisons über
  `DELETE /api/kader/{id}` gelöscht werden soll
- **THEN** antwortet das System mit HTTP 409 und die Trainingshistorie bleibt erhalten

#### Scenario: Saisonkopierer lässt Übungsgruppen aus
- **WHEN** `POST /api/kader/copy-from-season` mit einer Quellsaison läuft, die
  Übungsgruppen enthält
- **THEN** entsteht in der Zielsaison keine `kind='practice'`-Zeile und keine neue
  `teams`-Zeile

#### Scenario: Liste zeigt nur die aktive Saison
- **WHEN** `GET /api/practice-groups` aufgerufen wird
- **THEN** enthält die Antwort ausschließlich Übungsgruppen der Saison mit `is_active=1`

### Requirement: Kein erweiterter Kader für Übungsgruppen

Das System SHALL Schreibzugriffe auf `kader_extended_members` für Kader mit
`kind='practice'` mit HTTP **409** ablehnen. Betroffen ist insbesondere
`PUT /api/kader/{id}` mit den Feldern `extended_members_add` / `extended_members_remove`.

Der Status 409 (nicht 404) SHALL verwendet werden: der Kader existiert, die Operation
passt nur nicht zur Variante.

Dies ist das **einzige** anwendungsseitige Gate dieses Changes. Alle übrigen
unerwünschten Flächen adressieren `/api/teams/{id}/…` und lösen über
`WHERE k.team_id = ?` auf; sie sind für eine Übungsgruppe strukturell nicht erreichbar und
brauchen deshalb keine Prüfung.

#### Scenario: Erweiterter Kader wird abgelehnt
- **WHEN** `PUT /api/kader/{id}` mit `extended_members_add` für einen Kader mit
  `kind='practice'` aufgerufen wird
- **THEN** antwortet das System mit HTTP 409 und `kader_extended_members` bleibt unverändert

#### Scenario: Mannschaftskader bleibt unberührt
- **WHEN** dieselbe Anfrage für einen Kader mit `kind='team'` gestellt wird
- **THEN** wird der erweiterte Kader wie bisher gepflegt (HTTP 200)

#### Scenario: Team-Routen sind nicht adressierbar
- **WHEN** eine beliebige Route unter `/api/teams/{id}/…` (Strafen, Kasse, Aufgaben,
  Warte, Statistiken) mit einer ID aufgerufen wird, hinter der eine Übungsgruppe steht
- **THEN** existiert keine solche `teams.id` und die Route liefert kein 2xx für die Gruppe
