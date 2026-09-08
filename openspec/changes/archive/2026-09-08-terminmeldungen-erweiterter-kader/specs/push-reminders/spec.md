## MODIFIED Requirements

### Requirement: Scheduled Push-Reminder für Spiele
Das System SHALL Mitgliedern und deren Elternteilen **zwei** Push-Reminder pro Spiel senden — einen **24h** vorher und einen **3h** vorher — sofern Push für `games` aktiviert ist und für den jeweiligen Slot noch kein Reminder protokolliert wurde. Die Empfängermenge SHALL der Regel der Capability `terminmeldung-empfaenger` folgen und damit **Stammkader und erweiterten Kader samt deren Elternteilen sowie die Trainer des Kaders** umfassen. Der Auslöse-Zeitpunkt MUSS aus `games.date` + `games.time` als Wandzeit in `Europe/Berlin` gebildet und gegen die aktuelle Zeit im selben Standort verglichen werden — unabhängig von der Server-Zeitzone. Es darf NIE ein Reminder früher als 24h vor Spielbeginn versendet werden. Jeder Slot MUSS über einen eigenen `notification_log`-`ref_type` (`game_reminder_24h`, `game_reminder_3h`) idempotent sein (Insert-vor-Send: nur bei `RowsAffected == 1` wird gesendet).

#### Scenario: Spielerinnerung 24h vorher
- **WHEN** der Scheduler-Job läuft und ein Spiel beginnt in ≤24h (und noch nicht begonnen)
- **THEN** erhalten alle betroffenen Nutzer (Push `games` aktiv, kein `game_reminder_24h`-Log-Eintrag) eine Push Notification „Spielerinnerung"

#### Scenario: Spielerinnerung 3h vorher
- **WHEN** der Scheduler-Job läuft und ein Spiel beginnt in ≤3h (und noch nicht begonnen)
- **THEN** erhalten alle betroffenen Nutzer (Push `games` aktiv, kein `game_reminder_3h`-Log-Eintrag) einen zweiten Push-Reminder

#### Scenario: Erweiterter Kader erhält die Spielerinnerung
- **WHEN** der Scheduler-Job eine Spielerinnerung für eine Mannschaft auslöst und ein Mitglied steht nur in deren erweitertem Kader
- **THEN** erhalten dieses Mitglied und seine über `family_links` verknüpften Elternteile denselben Reminder wie der Stammkader
- **THEN** greift die Idempotenz je Nutzer und Slot unverändert (`notification_log`)

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
Das System SHALL Mitgliedern und deren Elternteilen **zwei** Push-Reminder pro Trainingseinheit senden — einen **24h** vorher und einen **3h** vorher — sofern Push für `trainings` aktiviert ist und die Einheit den Status `active` hat. Die Empfängermenge SHALL der Regel der Capability `terminmeldung-empfaenger` folgen und damit **Stammkader und erweiterten Kader samt deren Elternteilen sowie die Trainer des Kaders** umfassen. Der Auslöse-Zeitpunkt MUSS aus `training_sessions.date` + `start_time` als Wandzeit in `Europe/Berlin` gebildet werden. Es darf NIE ein Reminder früher als 24h vor Trainingsbeginn versendet werden. Jeder Slot MUSS über einen eigenen `notification_log`-`ref_type` (`training_reminder_24h`, `training_reminder_3h`) idempotent sein.

#### Scenario: Trainingserinnerung 24h vorher
- **WHEN** der Scheduler-Job läuft und eine aktive Trainingseinheit beginnt in ≤24h
- **THEN** erhalten alle betroffenen Nutzer (Push `trainings` aktiv, kein `training_reminder_24h`-Eintrag) einen Push-Reminder

#### Scenario: Trainingserinnerung 3h vorher
- **WHEN** der Scheduler-Job läuft und eine aktive Trainingseinheit beginnt in ≤3h
- **THEN** erhalten alle betroffenen Nutzer (Push `trainings` aktiv, kein `training_reminder_3h`-Eintrag) einen zweiten Push-Reminder

#### Scenario: Erweiterter Kader erhält die Trainingserinnerung
- **WHEN** der Scheduler-Job eine Trainingserinnerung für eine Mannschaft auslöst und ein Mitglied steht nur in deren erweitertem Kader
- **THEN** erhalten dieses Mitglied und seine über `family_links` verknüpften Elternteile denselben Reminder wie der Stammkader

#### Scenario: Abgesagtes Training erhält keinen Reminder
- **WHEN** eine Trainingseinheit den Status `cancelled` hat
- **THEN** wird weder der 24h- noch der 3h-Reminder versendet
