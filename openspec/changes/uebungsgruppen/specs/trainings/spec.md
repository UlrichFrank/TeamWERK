## ADDED Requirements

### Requirement: Der Kader ist der Besitzer eines Trainings

Das System SHALL `training_sessions.kader_id` und `training_series.kader_id` (beide
NOT NULL, `REFERENCES kader(id) ON DELETE RESTRICT`) als **alleinigen** Besitzer eines
Trainingstermins bzw. einer Trainingsserie führen. Jede Auflösung der Frage „wer gehört zu
diesem Training?" — Sichtbarkeit, RSVP-Berechtigung, Anwesenheitserfassung,
Serien-Abmeldung, Erinnerungen, SSE-Zielmenge — SHALL über `kader_id` erfolgen, nicht mehr
über die Kombination `(team_id, season_id)`.

Damit gilt ein einziger Pfad für Mannschaftstrainings und Übungsgruppentrainings.

#### Scenario: Training für eine Übungsgruppe anlegen
- **WHEN** ein Trainer einer Übungsgruppe `POST /api/training-sessions` für diese Gruppe
  sendet
- **THEN** wird der Termin mit `kader_id` der Gruppe und `team_id IS NULL` angelegt
  (HTTP 201)

#### Scenario: Mitglied sieht den Termin
- **WHEN** ein Mitglied einer Übungsgruppe seine Trainingsliste abruft
- **THEN** enthält sie die Termine dieser Gruppe

#### Scenario: Nicht-Mitglied sieht den Termin nicht
- **WHEN** ein Nutzer, der weder Mitglied noch Trainer noch Elternteil eines Mitglieds
  dieser Übungsgruppe ist, seine Trainingsliste abruft
- **THEN** fehlen die Termine der Gruppe

#### Scenario: Eltern sehen den Termin
- **WHEN** ein über `family_links` verknüpftes Elternteil eines Gruppenmitglieds die
  Trainingsliste abruft
- **THEN** enthält sie die Termine der Gruppe

#### Scenario: RSVP und Anwesenheit funktionieren
- **WHEN** ein Mitglied einer Übungsgruppe auf einen Termin antwortet und der Trainer
  anschließend die Anwesenheit erfasst
- **THEN** werden Antwort und Anwesenheit gespeichert (HTTP 200/201), wie bei einem
  Mannschaftstraining

#### Scenario: Fremder darf nicht antworten
- **WHEN** ein Nutzer ohne Zugehörigkeit zur Übungsgruppe auf deren Termin antwortet
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Erinnerungen erreichen die Gruppe
- **WHEN** ein Übungsgruppentermin in 24 bzw. 3 Stunden beginnt
- **THEN** erhalten Mitglieder, deren Eltern und Trainer der Gruppe dieselben
  Erinnerungen wie bei einem Mannschaftstermin

#### Scenario: Trainings vergangener Saisons bleiben aufgelöst
- **WHEN** die aktive Saison wechselt und ein Training der Vorsaison abgerufen wird
- **THEN** wird sein Besitzer weiterhin über `kader_id` aufgelöst — die Auflösung hängt an
  der Saison des Trainings, nicht an der aktiven

#### Scenario: Kader mit Trainings ist nicht löschbar
- **WHEN** ein Kader gelöscht werden soll, an dem Trainingstermine oder -serien hängen
- **THEN** antwortet das System mit HTTP 409 und weder Kader noch Trainings werden entfernt

#### Scenario: Bestandsverhalten für Mannschaften unverändert
- **WHEN** ein Mannschaftstraining angelegt, gelesen, beantwortet oder abgehakt wird
- **THEN** ist das Verhalten identisch zu vorher — inklusive der Sichtbarkeit für den
  erweiterten Kader und dessen Eltern

### Requirement: `team_id` eines Trainings ist eine Projektion, kein Datenfeld

Das System SHALL `training_sessions.team_id` und `training_series.team_id` als
**nullable** führen. Der Wert SHALL gesetzt sein, wenn der besitzende Kader zu einer
Mannschaft gehört (`kader.team_id IS NOT NULL`), und **NULL**, wenn der Besitzer eine
Übungsgruppe ist.

`team_id IS NULL` bedeutet ausdrücklich „gehört einer Übungsgruppe" und SHALL **nicht** als
Datenfehler behandelt oder durch einen Backfill gefüllt werden. Diese NULL trägt den
gesamten Ausschluss der Übungsgruppen aus den team-gebundenen Auswertungen; sie zu füllen
würde alle betroffenen Flächen gleichzeitig öffnen.

Konsumenten außerhalb von `internal/trainings` — Anwesenheitsstatistik
(`internal/attendance`), Abwesenheitskalender (`internal/absences`), iCal-Feed
(`internal/calendar`), Dashboard, Videos, Scheduler-Statistiken — SHALL weiterhin über
`team_id` auflösen und Übungsgruppen dadurch ohne zusätzlichen Filter ausschließen.

#### Scenario: Statistik ignoriert Übungsgruppen
- **WHEN** `GET /api/teams/{id}/attendance-stats` für eine Mannschaft abgerufen wird und
  Mitglieder dieser Mannschaft zusätzlich in einer Übungsgruppe trainieren
- **THEN** fließen ausschließlich die Mannschaftstrainings in die Statistik ein

#### Scenario: Kalender-Feed ignoriert Übungsgruppen
- **WHEN** ein Mitglied einer Übungsgruppe seinen iCal-Feed abruft
- **THEN** enthält er keine Übungsgruppentermine

#### Scenario: Abwesenheitskalender ignoriert Übungsgruppen
- **WHEN** der Abwesenheitskalender einer Mannschaft abgerufen wird
- **THEN** werden Übungsgruppentermine nicht berücksichtigt

#### Scenario: Live-Updates erreichen die Gruppe trotzdem
- **WHEN** ein Übungsgruppentermin angelegt, geändert oder gelöscht wird
- **THEN** erhalten die Mitglieder, Eltern und Trainer der Gruppe ein SSE-Update — die
  Zielmenge wird über `kader_id` aufgelöst, nicht über das NULL-`team_id`

### Requirement: Migration bestehender Trainings ist verlustfrei und laut

Das System SHALL beim Backfill jeden bestehenden Trainingstermin und jede Serie genau dem
Kader zuordnen, dessen `(team_id, season_id)` mit dem des Datensatzes übereinstimmt. Diese
Zuordnung ist eindeutig, weil `kader` je Saison höchstens eine Zeile pro Team führt.

Findet sich zu einem Datensatz **kein** Kader, SHALL die Migration **abbrechen** (`kader_id`
ist NOT NULL) statt den Datensatz stillschweigend zu verlieren oder mit NULL zu füllen.

#### Scenario: Bestandstermine behalten ihren Besitzer
- **WHEN** Migration `058` auf einer Bestandsdatenbank läuft
- **THEN** trägt jeder Termin danach den `kader_id` des Kaders seines
  `(team_id, season_id)`, und `team_id` bleibt unverändert gesetzt

#### Scenario: Waisen-Datensatz bricht die Migration ab
- **WHEN** ein Trainingstermin existiert, dessen `(team_id, season_id)` keinen Kader hat
- **THEN** schlägt die Migration fehl und die Datenbank bleibt auf dem alten Stand
