## ADDED Requirements

### Requirement: Trainer kann Trainingsserien anlegen
Ein Trainer oder Admin SHALL eine Trainingsserie für sein Team anlegen können. Die Serie definiert einen festen Wochentag (0=Montag bis 6=Sonntag), Start- und Endzeit, Ort, Namen und Gültigkeitszeitraum. Das Backend generiert beim Anlegen automatisch alle `training_sessions` für jeden passenden Wochentag zwischen `valid_from` und `valid_until`.

#### Scenario: Serie anlegen generiert Sessions
- **WHEN** ein Trainer POST `/api/training-series` aufruft mit `day_of_week=1` (Dienstag), `valid_from=2026-09-01`, `valid_until=2027-06-30`
- **THEN** legt das System eine `training_series`-Row an und generiert alle Dienstag-Dates zwischen 01.09.2026 und 30.06.2027 als `training_sessions` mit `status='active'`

#### Scenario: Trainer kann nur für sein eigenes Team anlegen
- **WHEN** ein User mit `role='trainer'` versucht, eine Serie für ein Team anzulegen, dem er nicht zugewiesen ist
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Admin kann für jedes Team anlegen
- **WHEN** ein User mit `role='admin'` eine Serie für ein beliebiges Team anlegt
- **THEN** wird die Serie erfolgreich angelegt

### Requirement: Trainer kann Trainingsserien bearbeiten
Ein Trainer oder Admin SHALL eine bestehende Serie bearbeiten können. Änderungen können mit `scope='this_and_following'` (ab einem Datum) oder `scope='all'` (alle Sessions der Serie) angewendet werden. Bei Scope-Änderungen werden betroffene Sessions neu generiert.

#### Scenario: Änderung mit Scope this_and_following
- **WHEN** ein Trainer PUT `/api/training-series/{id}` mit `scope='this_and_following'` und `from_date='2026-11-01'` aufruft
- **THEN** werden alle Sessions der Serie ab dem 01.11.2026 gelöscht und mit den neuen Serienparametern neu generiert; Sessions vor diesem Datum bleiben unverändert

#### Scenario: Änderung mit Scope all
- **WHEN** ein Trainer PUT `/api/training-series/{id}` mit `scope='all'` aufruft
- **THEN** werden alle aktiven Sessions der Serie neu generiert

### Requirement: Trainer kann Trainingsserien löschen
Ein Trainer oder Admin SHALL eine Serie löschen können. Dabei werden alle zukünftigen Sessions der Serie gelöscht. Vergangene Sessions (Datum < heute) bleiben erhalten.

#### Scenario: Löschen löscht nur zukünftige Sessions
- **WHEN** ein Trainer DELETE `/api/training-series/{id}` aufruft
- **THEN** werden alle Sessions der Serie mit `date >= today` gelöscht; Sessions mit `date < today` bleiben erhalten (inklusive ihrer Responses und Attendances)
