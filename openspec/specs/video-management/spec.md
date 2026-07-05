# video-management Specification

## Purpose
TBD - created by archiving change spielvideo-ablage. Update Purpose after archive.
## Requirements
### Requirement: Sicht-Berechtigung pro Team

Ein Video MUST genau einem Team zugeordnet sein (`team_id NOT NULL`). Sichtbar für einen Nutzer ist ein Video genau dann, wenn er Rolle `admin` hat, Vereinsfunktion `vorstand` hat, aktiver Spieler des Teams ist (`team_memberships.active = 1`), Trainer des Teams ist (`team_trainers`), oder Elternteil eines aktiven Spielers des Teams ist (über `family_links`).

#### Scenario: Spieler sieht eigenes Team
- **WHEN** ein aktiver Spieler von Team A `GET /api/videos` aufruft
- **THEN** enthält die Liste Videos mit `team_id = A` und keine Videos anderer Teams

#### Scenario: Elternteil sieht Team des Kindes
- **WHEN** ein Elternteil eines aktiven Spielers von Team A `GET /api/videos?team_id=A` aufruft
- **THEN** enthält die Liste die berechtigten Videos von Team A

#### Scenario: Spieler fragt fremdes Team an
- **WHEN** ein Spieler von Team A `GET /api/videos?team_id=B` aufruft und nicht zu Team B gehört
- **THEN** ist Team B nicht in der Antwort enthalten (leere oder gefilterte Liste)

#### Scenario: Inaktive Mitgliedschaft
- **WHEN** ein Nutzer war Spieler in Team A, ist aber nicht mehr aktiv
- **THEN** sieht er die Videos von Team A nicht mehr

### Requirement: Videoliste mit Status und Paginierung

`GET /api/videos` SHALL die Liste der berechtigten Videos liefern. Die Antwort MUST `{ items, total }` enthalten. Items MUST `id`, `title`, `description`, `team_id`, `team_name`, `season_id`, `game_id`, `status`, `duration_sec`, `size_bytes`, `created_by_name`, `created_at`, `ready_at` enthalten. Filter: `team_id`, `status`, `season_id`. Paginierung: `limit` (default 50), `offset` (default 0).

#### Scenario: Standard-Liste
- **WHEN** ein berechtigter Nutzer `GET /api/videos` ohne Parameter aufruft
- **THEN** erhält er bis zu 50 Videos sortiert nach `created_at DESC` mit `total`-Angabe

#### Scenario: Filter nach Status
- **WHEN** ein Trainer `GET /api/videos?status=processing` aufruft
- **THEN** enthält die Liste nur Videos, die gerade verarbeitet werden und für ihn sichtbar sind

### Requirement: Metadata-Update

`PATCH /api/videos/{id}` SHALL Titel, Beschreibung und `game_id` ändern können. Berechtigung: Trainer des Teams, Vorstand oder Admin. Andere Felder (`team_id`, `season_id`, `status`, technische Felder) DÜRFEN NICHT änderbar sein.

#### Scenario: Trainer ändert Titel
- **WHEN** ein Trainer des Teams `PATCH /api/videos/{id}` mit neuem `title` aufruft
- **THEN** ist der Titel aktualisiert, Antwort HTTP 200

#### Scenario: Versuch team_id zu ändern
- **WHEN** der Request enthält `team_id`
- **THEN** wird das Feld ignoriert (HTTP 200, alter Wert bleibt) oder die Anfrage mit HTTP 400 abgelehnt

#### Scenario: Fremder Trainer
- **WHEN** ein Trainer, der nicht im Team-Trainerstab ist, `PATCH /api/videos/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Löschen entfernt Dateien

`DELETE /api/videos/{id}` SHALL den DB-Eintrag löschen UND alle zugehörigen Dateien (`raw/{id}.mp4` falls vorhanden, gesamtes `processed/{id}/`-Verzeichnis) physisch entfernen. Berechtigung: jeder Trainer des Teams, Vorstand oder Admin.

#### Scenario: Trainer löscht Video
- **WHEN** ein Trainer des Teams `DELETE /api/videos/{id}` aufruft
- **THEN** ist die DB-Zeile entfernt, der Ordner `processed/{id}/` ist gelöscht, `video-deleted` wurde broadcastet, HTTP 204

#### Scenario: Anderer Trainer im selben Team
- **WHEN** Trainer A das Video hochgeladen hat und Trainer B desselben Teams `DELETE` aufruft
- **THEN** wird das Video gelöscht (Trainer-Vertretung im Team erlaubt)

#### Scenario: Spieler versucht Löschen
- **WHEN** ein Spieler `DELETE /api/videos/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403, Daten bleiben unverändert

#### Scenario: Löschen während Transcode läuft
- **WHEN** ein Video mit `status='processing'` gelöscht wird
- **THEN** wird der Worker durch Status-Check abgebrochen, alle Teil-Output-Dateien werden entfernt

### Requirement: Saison-basierte Retention

Ein täglicher Scheduler-Job SHALL Videos löschen, deren zugehörige Saison ein `end_date` hat, das mehr als 90 Tage in der Vergangenheit liegt. Sieben Tage vor der geplanten Löschung MUST eine Push-Notification an alle Team-Trainer ausgehen ("Video XY wird am DD.MM. gelöscht"). Die Vorlauf-Benachrichtigung MUST idempotent sein (nur einmal pro Video).

#### Scenario: Saison endete vor 100 Tagen
- **WHEN** der Retention-Job läuft und eine Saison hatte `end_date` vor 100 Tagen
- **THEN** werden alle Videos dieser Saison gelöscht (DB + Dateien)

#### Scenario: Saison endete vor 85 Tagen
- **WHEN** der Retention-Job läuft und eine Saison hatte `end_date` vor 85 Tagen (also T-5 bis Löschung)
- **THEN** bleiben die Videos bestehen, aber Trainer haben bereits den 7-Tage-Vorlauf-Push erhalten

#### Scenario: Saison ohne end_date
- **WHEN** eine Saison hat `end_date IS NULL`
- **THEN** werden Videos dieser Saison nicht gelöscht

