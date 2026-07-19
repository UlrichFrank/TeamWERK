# serien-abmeldung Specification

## Purpose

Diese Spezifikation beschreibt die Capability `serien-abmeldung`: die serien-gebundene, dauerhafte Abmeldung eines Spielers von einer Trainings-Serie durch Trainer, sportliche Leitung oder Admin, samt der maßgeblichen Ableitung, wann eine Session für ein Mitglied als abgemeldet gilt.

## Requirements

### Requirement: Trainer kann einen Spieler serien-gebunden dauerhaft abmelden

Ein Trainer des zugehörigen Teams, ein Mitglied mit Vereinsfunktion `sportliche_leitung` oder ein Admin SHALL für eine `training_series` einen Spieler dauerhaft abmelden können. Die Abmeldung besteht aus `member_id`, optionalem `start_date` (NULL = ab Serien-Beginn), optionalem `end_date` (NULL = permanent bis Serien-Ende) und optionalem `reason`. Sie wird in `member_series_unavailabilities` gespeichert; `team_id` wird nicht redundant persistiert, sondern über `training_series.team_id` abgeleitet. Ein Spieler oder Elternteil SHALL **keine** Abmeldung anlegen, ändern oder löschen können.

#### Scenario: Trainer legt permanente Abmeldung für sein Team an

- **WHEN** ein Trainer eines Teams `POST /api/training-series/{id}/unavailabilities` mit `member_id` und ohne `end_date` für eine Serie dieses Teams aufruft
- **THEN** wird eine Zeile in `member_series_unavailabilities` mit `end_date = NULL` angelegt und HTTP 201 zurückgegeben

#### Scenario: Trainer setzt befristete Abmeldung mit Grund

- **WHEN** ein Trainer `POST .../unavailabilities` mit `start_date`, `end_date` und `reason="spielt A-Jugend"` aufruft
- **THEN** wird die Abmeldung mit dem angegebenen Zeitraum und Grund angelegt (HTTP 201)

#### Scenario: Trainer eines fremden Teams abgewiesen

- **WHEN** ein Trainer, der nicht dem Team der Serie zugeordnet ist (kein `kader_trainers`-Eintrag für `training_series.team_id`), `POST .../unavailabilities` aufruft
- **THEN** antwortet das System mit HTTP 403 und legt keine Zeile an

#### Scenario: Spieler darf nicht abmelden

- **WHEN** ein eingeloggter Spieler oder Elternteil `POST /api/training-series/{id}/unavailabilities` aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Sportliche Leitung und Admin dürfen für jedes Team

- **WHEN** ein Mitglied mit `sportliche_leitung` oder ein Admin `POST .../unavailabilities` für eine beliebige Serie aufruft
- **THEN** wird die Abmeldung angelegt (HTTP 201)

### Requirement: Abmeldungen einer Serie auflisten

Das System SHALL via `GET /api/training-series/{id}/unavailabilities` die Abmeldungen der Serie an berechtigte Nutzer (Trainer des Teams, sportliche Leitung, Admin) zurückgeben, jeweils mit `member_id`, `member_name`, `start_date`, `end_date`, `reason` und `created_at`.

#### Scenario: Trainer listet Abmeldungen seiner Serie

- **WHEN** ein Trainer des Teams `GET /api/training-series/{id}/unavailabilities` aufruft
- **THEN** erhält er HTTP 200 mit allen Abmeldungen der Serie

#### Scenario: Fremder Trainer abgewiesen

- **WHEN** ein Trainer eines anderen Teams die Liste abruft
- **THEN** antwortet das System mit HTTP 403

### Requirement: Abmeldung löschen

Ein Trainer des zugehörigen Teams, sportliche Leitung oder Admin SHALL eine Abmeldung via `DELETE /api/training-series/{id}/unavailabilities/{uid}` löschen können. Nach dem Löschen zählt der Spieler ab dem nächsten Statistik-/RSVP-Zugriff wieder normal (kein persistenter Nebenzustand, der aufgeräumt werden müsste).

#### Scenario: Trainer löscht Abmeldung

- **WHEN** ein Trainer des Teams `DELETE /api/training-series/{id}/unavailabilities/{uid}` aufruft
- **THEN** wird die Zeile entfernt und HTTP 200/204 zurückgegeben

#### Scenario: Löschen einer fremden Serie abgewiesen

- **WHEN** ein Trainer eines anderen Teams die Abmeldung löschen will
- **THEN** antwortet das System mit HTTP 403 und die Zeile bleibt bestehen

#### Scenario: Abmeldung, die nicht zur Serie gehört

- **WHEN** `{uid}` nicht zur Serie `{id}` gehört
- **THEN** antwortet das System mit HTTP 404

### Requirement: Ableitung der Betroffenheit einer Session

Das System SHALL definieren, dass eine Trainings-Session (mit `series_id = S` und `date = D`) für ein Mitglied `X` genau dann von einer Abmeldung betroffen ist, wenn eine Zeile in `member_series_unavailabilities` existiert mit `member_id = X`, `training_series_id = S`, `(start_date IS NULL OR start_date <= D)` und `(end_date IS NULL OR end_date >= D)`. Einzeltermine ohne Serie (`series_id IS NULL`) SHALL nie betroffen sein. Diese Ableitung ist die maßgebliche Referenz für RSVP-Sperre, Attendance-Ausschluss und Statistik.

#### Scenario: Session innerhalb des Abmelde-Fensters ist betroffen

- **WHEN** eine Session der Serie mit Datum innerhalb `[start_date, end_date]` (bzw. offen bei NULL) einer Abmeldung des Mitglieds liegt
- **THEN** gilt die Session für dieses Mitglied als abgemeldet

#### Scenario: Session außerhalb des Fensters ist nicht betroffen

- **WHEN** das Session-Datum vor `start_date` oder nach `end_date` einer befristeten Abmeldung liegt
- **THEN** gilt die Session für dieses Mitglied nicht als abgemeldet

#### Scenario: Einzeltermin ohne Serie nie betroffen

- **WHEN** eine Session `series_id IS NULL` hat
- **THEN** kann keine Serien-Abmeldung auf sie zutreffen

#### Scenario: Überlappende Abmeldungen sind harmlos

- **WHEN** für dasselbe Mitglied und dieselbe Serie zwei Abmeldungen mit überlappenden, aber verschiedenen `start_date` existieren
- **THEN** gilt die Session als abgemeldet, sobald mindestens eine Zeile das Datum abdeckt (kein Fehler, kein Doppel-Effekt)
