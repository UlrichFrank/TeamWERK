## MODIFIED Requirements

### Requirement: Mannschafts-Scoping nach Rolle und Typ

Die wählbaren Mannschaften im Wizard SHALL von Rolle und Event-Typ abhängen. Die Datenquelle
des Pickers macht die Regel prüfbar: Für Heim-/Auswärtsspiele speist sich die Auswahl aus der
nutzergefilterten Liste `GET /api/teams`, für Sonstige Events aus der vereinsweiten Liste
`GET /api/teams/names` (alle aktiven Mannschaften der aktiven Saison).

#### Scenario: Trainer bei Heimspiel oder Auswärtsspiel
- **WHEN** ein Trainer Typ Heimspiel oder Auswärtsspiel wählt
- **THEN** enthält das Mannschafts-Dropdown nur die eigenen Mannschaften des Trainers (Quelle: `GET /api/teams`)

#### Scenario: Trainer bei Sonstigem Event
- **WHEN** ein Trainer Typ Sonstiges Event wählt
- **THEN** sind alle aktiven Mannschaften im Multi-Select wählbar (Quelle: `GET /api/teams/names`, kein Rollenfilter)
- **THEN** wird jede Mannschaft mit ihrem berechneten Kurznamen aus `buildTeamShortNames` beschriftet

#### Scenario: Admin oder Vorstand
- **WHEN** ein User mit Rolle admin oder vorstand ein Event anlegt
- **THEN** sind alle aktiven Mannschaften wählbar, unabhängig vom Typ

---

### Requirement: Backend-Validierung Trainer-Scope

Das Backend SHALL die übergebenen `team_ids` für reine Trainer (Vereinsfunktion `trainer` ohne
`sportliche_leitung`/`vorstand`, System-Rolle nicht `admin`) typabhängig prüfen:

- Bei `heim`/`auswärts` MÜSSEN **alle** `team_ids` zu den eigenen Mannschaften des Trainers
  gehören (via `kader_trainers` in der aktiven Saison).
- Bei `generisch` MUSS **mindestens eine** `team_id` zu den eigenen Mannschaften gehören; weitere
  `team_ids` dürfen beliebige aktive Mannschaften sein.

Die vollständigen Regeln inklusive `PUT`/`DELETE` stehen in der Capability
`game-mutation-team-scope`.

#### Scenario: Trainer übergibt fremde Mannschaft bei Heimspiel
- **WHEN** ein Trainer `event_type='heim'` mit `team_ids` übergibt, die nicht in seinen
  `kader_trainers`-Einträgen liegen
- **THEN** antwortet der Server mit HTTP 403 Forbidden

#### Scenario: Trainer übergibt fremde Mannschaften bei Sonstigem Event
- **WHEN** ein Trainer `event_type='generisch'` mit `team_ids` übergibt, die mindestens eine
  eigene und daneben fremde Mannschaften enthalten
- **THEN** legt der Server das Event mit allen übergebenen Mannschaften an (HTTP 201)

#### Scenario: Trainer übergibt ausschließlich fremde Mannschaften bei Sonstigem Event
- **WHEN** ein Trainer `event_type='generisch'` ausschließlich mit fremden `team_ids` übergibt
- **THEN** antwortet der Server mit HTTP 403 Forbidden
- **THEN** entsteht kein Event, das der Trainer selbst nicht sehen könnte
