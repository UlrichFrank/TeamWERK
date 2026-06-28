# push-reminders Specification

## Purpose

Diese Spezifikation beschreibt die Capability `push-reminders`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Scheduled Push-Reminder für Spiele
Das System SHALL Mitgliedern und deren Elternteilen **zwei** Push-Reminder pro Spiel senden — einen **24h** vorher und einen **3h** vorher — sofern Push für `games` aktiviert ist und für den jeweiligen Slot noch kein Reminder protokolliert wurde. Der Auslöse-Zeitpunkt MUSS aus `games.date` + `games.time` als Wandzeit in `Europe/Berlin` gebildet und gegen die aktuelle Zeit im selben Standort verglichen werden — unabhängig von der Server-Zeitzone. Es darf NIE ein Reminder früher als 24h vor Spielbeginn versendet werden. Jeder Slot MUSS über einen eigenen `notification_log`-`ref_type` (`game_reminder_24h`, `game_reminder_3h`) idempotent sein (Insert-vor-Send: nur bei `RowsAffected == 1` wird gesendet).

#### Scenario: Spielerinnerung 24h vorher
- **WHEN** der Scheduler-Job läuft und ein Spiel beginnt in ≤24h (und noch nicht begonnen)
- **THEN** erhalten alle betroffenen Nutzer (Push `games` aktiv, kein `game_reminder_24h`-Log-Eintrag) eine Push Notification „Spielerinnerung"

#### Scenario: Spielerinnerung 3h vorher
- **WHEN** der Scheduler-Job läuft und ein Spiel beginnt in ≤3h (und noch nicht begonnen)
- **THEN** erhalten alle betroffenen Nutzer (Push `games` aktiv, kein `game_reminder_3h`-Log-Eintrag) einen zweiten Push-Reminder

#### Scenario: Wandzeit-Korrektheit über Zeitzonen-Offset
- **WHEN** der Scheduler in UTC läuft und ein Spiel im Sommer (CEST = UTC+2) um 15:00 Berlin-Zeit beginnt
- **THEN** wird der 3h-Reminder zur Berlin-Wandzeit 12:00 ausgelöst, nicht um 10:00 oder 14:00

#### Scenario: Spiel kurzfristig (<24h) angelegt
- **WHEN** ein Spiel angelegt wird, dessen Beginn bereits in weniger als 24h liegt
- **THEN** feuert der 24h-Slot beim nächsten Scheduler-Lauf sofort (das 24h-Fenster ist bereits betreten)

#### Scenario: Kein Duplikat pro Slot
- **WHEN** der Scheduler erneut läuft und für dieses Spiel+Nutzer+Slot bereits ein Reminder gesendet wurde
- **THEN** wird für diesen Slot kein weiterer Reminder gesendet (`notification_log` mit dem Slot-`ref_type` verhindert das Duplikat)

### Requirement: Scheduled Push-Reminder für Trainings
Das System SHALL Mitgliedern und deren Elternteilen **zwei** Push-Reminder pro Trainingseinheit senden — einen **24h** vorher und einen **3h** vorher — sofern Push für `trainings` aktiviert ist und die Einheit den Status `active` hat. Der Auslöse-Zeitpunkt MUSS aus `training_sessions.date` + `start_time` als Wandzeit in `Europe/Berlin` gebildet werden. Es darf NIE ein Reminder früher als 24h vor Trainingsbeginn versendet werden. Jeder Slot MUSS über einen eigenen `notification_log`-`ref_type` (`training_reminder_24h`, `training_reminder_3h`) idempotent sein.

#### Scenario: Trainingserinnerung 24h vorher
- **WHEN** der Scheduler-Job läuft und eine aktive Trainingseinheit beginnt in ≤24h
- **THEN** erhalten alle betroffenen Nutzer (Push `trainings` aktiv, kein `training_reminder_24h`-Eintrag) einen Push-Reminder

#### Scenario: Trainingserinnerung 3h vorher
- **WHEN** der Scheduler-Job läuft und eine aktive Trainingseinheit beginnt in ≤3h
- **THEN** erhalten alle betroffenen Nutzer (Push `trainings` aktiv, kein `training_reminder_3h`-Eintrag) einen zweiten Push-Reminder

#### Scenario: Abgesagtes Training erhält keinen Reminder
- **WHEN** eine Trainingseinheit den Status `cancelled` hat
- **THEN** wird weder der 24h- noch der 3h-Reminder versendet

### Requirement: Scheduled Dienst-Erinnerung (Push + optional E-Mail)
Das System SHALL zugeteilten Nutzern 48h vor einem Dienst eine Push Notification senden. Die Idempotenz-Garantie MUSS durch die Reihenfolge der Operationen sichergestellt werden: der `notification_log`-Eintrag MUSS angelegt werden, BEVOR der Push gesendet wird. Nur wenn der INSERT tatsächlich eine neue Zeile erzeugt (RowsAffected = 1), wird der Push gesendet. Optional wird zusätzlich eine E-Mail gesendet, wenn `email_enabled=1` für Kategorie `duty_reminders`.

#### Scenario: Dienst-Push-Reminder 48h vorher
- **WHEN** der Scheduler läuft und ein Nutzer ist einem Slot in 48h zugeteilt
- **THEN** erhält der Nutzer einen Push-Reminder (sofern `push_enabled` für `duty_reminders`)

#### Scenario: Kein Duplikat bei parallelen Scheduler-Runs
- **WHEN** zwei Scheduler-Instanzen gleichzeitig laufen und denselben Nutzer+Datum-Kombination prüfen
- **THEN** erhält der Nutzer genau einen Push — nicht zwei
- **THEN** der zweite INSERT OR IGNORE schlägt fehl (RowsAffected = 0) und es wird kein Push gefeuert

#### Scenario: Kein Duplikat bei erneutem Scheduler-Run
- **WHEN** der Scheduler erneut läuft und ein Reminder für diesen Nutzer+Dienst wurde bereits gesendet
- **THEN** wird kein weiterer Reminder gesendet (`notification_log` enthält bereits den Eintrag)

#### Scenario: Dienst-E-Mail-Reminder opt-in
- **WHEN** der Scheduler-Job läuft und der Nutzer hat `email_enabled=1` für `duty_reminders`
- **THEN** erhält der Nutzer zusätzlich eine Erinnerungsmail

#### Scenario: Kein E-Mail-Reminder ohne opt-in
- **WHEN** ein Nutzer hat `email_enabled=0` (oder keinen Eintrag) für `duty_reminders`
- **THEN** erhält er keine E-Mail, nur Push (sofern Push aktiv)

### Requirement: Scheduled Push-Reminder für Fahrgemeinschaften
Das System SHALL Fahrgemeinschafts-Teilnehmer mit bestätigter Paarung **exakt 3h** vor Abfahrt per Push erinnern. Der Auslöse-Zeitpunkt MUSS aus `games.date` + `games.time` als Wandzeit in `Europe/Berlin` gebildet und gegen die aktuelle Zeit im selben Standort verglichen werden. Der Reminder MUSS über den `notification_log`-`ref_type` `carpooling_reminder` idempotent sein.

#### Scenario: Fahrgemeinschaftserinnerung 3h vorher
- **WHEN** der Scheduler läuft und ein Spiel mit bestätigter Fahrgemeinschaft beginnt in ≤3h
- **THEN** erhalten die Teilnehmer mit `status='confirmed'` einen Push-Reminder (sofern Push `carpooling` aktiv), genau einmal

#### Scenario: Wandzeit-Korrektheit der Abfahrt
- **WHEN** der Scheduler in UTC läuft und das Spiel um 15:00 Berlin-Zeit (CEST) beginnt
- **THEN** wird der Mitfahr-Reminder zur Berlin-Wandzeit 12:00 ausgelöst, nicht versetzt um den Server-Offset

<!-- HINWEIS (kein Delta): Die bestehende Anforderung "Scheduled Dienst-Erinnerung
     (Push + optional E-Mail)" in openspec/specs/push-reminders/spec.md bleibt
     bewusst UNVERÄNDERT (out of scope) und wird hier daher nicht als Delta geführt. -->

