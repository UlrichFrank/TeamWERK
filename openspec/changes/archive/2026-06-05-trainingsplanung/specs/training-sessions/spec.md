## ADDED Requirements

### Requirement: Trainer kann Einzeltermine anlegen
Ein Trainer oder Admin SHALL einen Einzeltermin (ohne Serie) anlegen können, z.B. für ein Sondertraining oder Testspiel-Vorbereitung.

#### Scenario: Einzeltermin anlegen
- **WHEN** ein Trainer POST `/api/training-sessions` mit `date`, `start_time`, `end_time`, `team_id` aufruft (ohne `series_id`)
- **THEN** wird eine `training_session` mit `series_id=NULL` und `status='active'` angelegt

### Requirement: Trainer kann einzelne Sessions bearbeiten
Ein Trainer oder Admin SHALL eine einzelne Session bearbeiten können (z.B. abweichender Ort, andere Uhrzeit). Dies gilt sowohl für Einzel- als auch für Serien-Sessions.

#### Scenario: Session-Override innerhalb einer Serie
- **WHEN** ein Trainer PUT `/api/training-sessions/{id}` aufruft und z.B. den Ort ändert
- **THEN** wird genau diese Session aktualisiert; andere Sessions der gleichen Serie bleiben unverändert

### Requirement: Trainer kann Sessions absagen
Ein Trainer oder Admin SHALL eine einzelne Session absagen können. Eine abgesagte Session bleibt in der DB (mit `status='cancelled'`), erscheint aber im Kalender als abgesagt.

#### Scenario: Session absagen
- **WHEN** ein Trainer DELETE `/api/training-sessions/{id}` oder PUT mit `status='cancelled'` aufruft
- **THEN** wird `status='cancelled'` und `cancel_reason` gesetzt; bestehende Responses bleiben erhalten

### Requirement: Spieler und Eltern sehen die Sessions ihres Teams
Ein authentifizierter User SHALL die Trainingssessions seines Teams (bzw. des Teams seiner Kinder) über GET `/api/training-sessions` abrufen können. Der Endpoint unterstützt Filter nach `team_id`, `from` und `to` (Datumsbereich).

#### Scenario: Spieler sieht eigene Team-Sessions
- **WHEN** ein User mit `role='spieler'` GET `/api/training-sessions` aufruft
- **THEN** erhält er alle aktiven Sessions der Teams, denen er als Mitglied angehört, mit dem Response-Summary (confirmed_count, declined_count, pending_count) und seinem eigenen RSVP-Status

#### Scenario: Elternteil sieht Sessions der Teams seiner Kinder
- **WHEN** ein User mit `role='elternteil'` GET `/api/training-sessions` aufruft
- **THEN** erhält er alle Sessions der Teams, in denen seine Kinder (via `family_links`) Mitglieder sind

#### Scenario: Trainer sieht alle Sessions seines Teams
- **WHEN** ein User mit `role='trainer'` GET `/api/training-sessions` aufruft
- **THEN** erhält er alle Sessions der ihm zugewiesenen Teams, inkl. vollständiger Response-Liste mit Begründungen
