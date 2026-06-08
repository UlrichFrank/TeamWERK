## ADDED Requirements

### Requirement: Abwesenheit anlegen
Ein Spieler (Rolle `spieler`) oder Elternteil (Rolle `elternteil`) SHALL einen Abwesenheitszeitraum (Typ `vacation` oder `injury`, Start- und Enddatum, optionale Notiz) für sich selbst bzw. ein verlinktes Kind anlegen können. Ein Elternteil MUSS via `family_links` mit dem Member verknüpft sein. Members ohne eigenen User-Account können keine Abwesenheiten erhalten.

#### Scenario: Spieler legt eigene Abwesenheit an
- **WHEN** ein eingeloggter Spieler `POST /api/absences` mit `type`, `start_date`, `end_date` aufruft
- **THEN** wird eine neue Abwesenheit für seinen verlinkten Member angelegt und HTTP 201 zurückgegeben

#### Scenario: Elternteil legt Abwesenheit für Kind an
- **WHEN** ein Elternteil `POST /api/absences` mit `member_id` eines verlinkten Kindes aufruft
- **THEN** wird die Abwesenheit für das Kind angelegt

#### Scenario: Elternteil für nicht-verlinktes Kind abgewiesen
- **WHEN** ein Elternteil eine Abwesenheit für eine `member_id` ohne `family_links`-Eintrag anlegen will
- **THEN** antwortet die API mit HTTP 403

### Requirement: Abwesenheiten auflisten
Das System SHALL dem eingeloggten Nutzer seine eigenen Abwesenheiten und — für Elternteile — die seiner verlinkten Kinder zurückgeben.

#### Scenario: Eigene Abwesenheiten abrufen
- **WHEN** ein eingeloggter Nutzer `GET /api/absences` aufruft
- **THEN** erhält er alle Abwesenheiten, für die er berechtigt ist (eigene + Kinder)

### Requirement: Abwesenheit löschen
Der Ersteller einer Abwesenheit (oder ein Admin) SHALL sie löschen können. Beim Löschen werden alle auto-declined Responses mit dieser `absence_id` per CASCADE entfernt.

#### Scenario: Eigene Abwesenheit löschen
- **WHEN** der Ersteller `DELETE /api/absences/{id}` aufruft
- **THEN** wird die Abwesenheit gelöscht und alle zugehörigen auto-declined Responses entfernt

#### Scenario: Fremde Abwesenheit löschen abgewiesen
- **WHEN** ein Nutzer eine Abwesenheit löschen will, die nicht ihm gehört und er kein Admin ist
- **THEN** antwortet die API mit HTTP 403

### Requirement: Preview vor dem Anlegen
Das System SHALL via `GET /api/absences/preview` die Events auflisten, die bei einem geplanten Zeitraum betroffen wären (bestehende `confirmed`-Responses im Zeitraum für den jeweiligen Member).

#### Scenario: Preview ohne Konflikte
- **WHEN** der Nutzer einen Zeitraum ohne bestehende Zusagen abfragt
- **THEN** gibt die API eine leere Liste zurück

#### Scenario: Preview mit Konflikten
- **WHEN** der Nutzer einen Zeitraum mit mindestens einer `confirmed` Training- oder Spiel-Zusage abfragt
- **THEN** gibt die API eine Liste der betroffenen Events (Name, Datum, Typ) zurück

### Requirement: Auto-decline beim Anlegen
Beim Anlegen einer Abwesenheit SHALL das System für alle `training_sessions` und `games` im Zeitraum, bei denen der Member im Kader ist, eine `declined`-Response mit gesetztem `absence_id` anlegen (INSERT OR REPLACE). Bestehende `confirmed`/`maybe`-Responses werden überschrieben.

#### Scenario: Bestehende Zusage wird überschrieben
- **WHEN** eine Abwesenheit angelegt wird und der Member eine `confirmed`-Response für ein Event im Zeitraum hat
- **THEN** wird die Response auf `declined` mit gesetzter `absence_id` geändert

#### Scenario: Kein Event im Zeitraum
- **WHEN** eine Abwesenheit angelegt wird und keine Events im Zeitraum liegen
- **THEN** wird die Abwesenheit ohne weitere Änderungen angelegt

### Requirement: Auto-decline bei neuen Events
Wenn eine neue `training_session` oder ein neues `game` angelegt wird, SHALL das System für alle Kader-Members mit einer Abwesenheit, die das Event-Datum überdeckt, sofort eine auto-declined Response anlegen.

#### Scenario: Training in Abwesenheitszeitraum angelegt
- **WHEN** ein Trainer ein Training anlegt, dessen Datum in der Abwesenheit eines Kader-Members liegt
- **THEN** erhält dieser Member sofort eine `declined`-Response mit gesetzter `absence_id`

### Requirement: Auto-declined Responses sind gesperrt
Eine Response mit gesetzter `absence_id` DARF von keiner Rolle (einschließlich Trainer und Admin) manuell geändert werden. Der Nutzer MUSS die Abwesenheit löschen, um wieder zusagen zu können.

#### Scenario: Manuelles Ändern einer auto-declined Response abgewiesen
- **WHEN** ein Nutzer versucht, eine Response mit `absence_id IS NOT NULL` zu ändern
- **THEN** antwortet die API mit HTTP 403

### Requirement: Kalender-Abwesenheits-Endpunkt
Das System SHALL via `GET /api/absences/calendar?from=&to=` die Abwesenheiten zurückgeben, die der eingeloggte Nutzer im Kalender sehen darf: eigene + Kinder immer; Abwesenheiten anderer Members nur wenn deren `absences_public = 1`.

#### Scenario: Trainer sieht nur öffentliche Abwesenheiten
- **WHEN** ein Trainer `GET /api/absences/calendar` aufruft
- **THEN** erhält er nur Abwesenheiten von Members mit `absences_public = 1`

#### Scenario: Spieler sieht eigene Abwesenheiten immer
- **WHEN** ein Spieler `GET /api/absences/calendar` aufruft
- **THEN** erhält er seine eigenen Abwesenheiten unabhängig von `absences_public`

### Requirement: Sichtbarkeits-Toggle im Profil
Ein Member SHALL via `PUT /api/profile/absence-visibility` steuern können, ob seine Abwesenheiten für Trainer im Kalender sichtbar sind (`absences_public`). Default ist `false`.

#### Scenario: Sichtbarkeit aktivieren
- **WHEN** ein Spieler `PUT /api/profile/absence-visibility` mit `{"public": true}` aufruft
- **THEN** wird `members.absences_public` auf `1` gesetzt

### Requirement: Kalender-Banner im Frontend
Die `KalenderPage` SHALL Abwesenheitszeiträume als farbige horizontale Linie (blassgelb, kräftigerer Rahmen) über die betroffenen Wochentage anzeigen. Bei Wochengrenze wird die Linie in separate Segmente pro Woche aufgeteilt. Die Linie ist nur sichtbar für den Member selbst, seine Elternteile, und Trainer wenn `absences_public = 1`.

#### Scenario: Abwesenheit über Wochengrenze
- **WHEN** eine Abwesenheit Mo–So einer Woche und darüber hinaus geht
- **THEN** erscheinen separate Banner-Segmente für jede betroffene Woche im Kalender
