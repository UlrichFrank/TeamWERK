## ADDED Requirements

### Requirement: Tägliche aggregierte Push-Erinnerung an Trainer

Ein Scheduler-Job SHALL einmal pro Tag (lokal 19:00) für jeden Trainer aller Teams in der aktiven Saison eine aggregierte Web-Push-Notification mit Titel `"Anwesenheiten fehlen"` und einem Body senden, der die Anzahl offener Erfassungen plus die ersten drei betroffenen Termine (mit Teamname und Datum) listet. Der Tap-Ziel-Pfad SHALL auf die Trainer-Anwesenheitsseite eines seiner Teams mit offenen Erfassungen verweisen. Versand erfolgt nicht-blockierend (Goroutine).

#### Scenario: Trainer mit offenen Erfassungen erhält Push

- **WHEN** der Job läuft und ein Trainer hat mindestens einen vergangenen, nicht cancelled Termin (Training oder Spiel) der aktiven Saison in einem seiner Teams ohne `attendance`-Row
- **THEN** wird `push.SendToUsers` für diesen Trainer mit Titel `"Anwesenheiten fehlen"`, dem aggregierten Body und einem Tap-Ziel `/team/{id}/anwesenheit` aufgerufen

#### Scenario: Trainer ohne offene Erfassungen erhält keine Push

- **WHEN** der Job läuft und für einen Trainer ist keine offene Erfassung in einem seiner Teams vorhanden
- **THEN** wird für diesen Trainer kein Push-Versand initiiert

### Requirement: Stop-Bedingung pro Termin

Sobald für einen Termin mindestens eine `attendance`-Row (`training_attendances` bzw. `game_attendances`) existiert, gilt der Termin als erledigt und SHALL in nachfolgenden Reminder-Pushes nicht mehr erscheinen — unabhängig davon, ob die Anwesenheit aller Mitglieder erfasst wurde.

#### Scenario: Ein Trainer erfasst, andere bekommen den Termin nicht mehr

- **WHEN** Trainer A `POST /api/games/{id}/attendances` mit einer Teilliste sendet und der Reminder-Job danach für Trainer B desselben Teams läuft
- **THEN** taucht dieses Spiel nicht mehr in der Liste der offenen Erfassungen für Trainer B auf

### Requirement: Cut-off am Saisonende

Termine außerhalb der aktiven Saison (`seasons.is_active=1`) SHALL bei der Aggregation ignoriert werden. Existiert keine aktive Saison, wird gar keine Push versendet.

#### Scenario: Saison vorbei, keine Push mehr

- **WHEN** der Job läuft und es gibt keinen Datensatz in `seasons` mit `is_active=1`, **oder** alle Trainer haben für vergangene Saisons nur Termine außerhalb der aktiven Saison
- **THEN** wird kein Push-Versand für den jeweiligen Trainer ausgelöst

### Requirement: Idempotenz über `notification_log`

Der Job SHALL maximal eine Push pro Trainer pro Kalendertag erzeugen. Vor dem Versand SHALL ein Datensatz `(user_id, kind='attendance-reminder', context=<heutiges Datum>)` in `notification_log` per `INSERT OR IGNORE` (bzw. äquivalentem Idempotenz-Mechanismus) angelegt werden; nur wenn der Eintrag neu war, wird gesendet.

#### Scenario: Job läuft mehrfach am selben Tag

- **WHEN** der Cron-Wrapper den Job mehrfach hintereinander am selben Tag startet und der Trainer offene Erfassungen hat
- **THEN** wird maximal **einmal** an diesen Trainer gesendet — alle weiteren Aufrufe bleiben ohne Versand

#### Scenario: Idempotenz pro Trainer unabhängig

- **WHEN** der Job für Trainer A bereits gesendet hat und parallel für Trainer B läuft
- **THEN** kann Trainer B weiterhin in dieser Job-Ausführung eine Push erhalten

### Requirement: Push-Body fasst maximal drei Termine konkret zusammen

Der Body SHALL die Gesamtzahl der offenen Erfassungen nennen und die ersten drei Termine als kurze Liste enthalten (Format: `"<Teamname> <Wochentag DD.MM.> (Training|Spiel)"`). Sind mehr als drei offen, SHALL ein Hinweis "… und N weitere" angehängt werden.

#### Scenario: Zwei offene Termine

- **WHEN** ein Trainer genau zwei offene Erfassungen hat
- **THEN** enthält der Body genau diese zwei Termine, ohne Hinweis-Suffix

#### Scenario: Fünf offene Termine

- **WHEN** ein Trainer fünf offene Erfassungen hat
- **THEN** enthält der Body die ersten drei Termine plus den Hinweis "… und 2 weitere"
