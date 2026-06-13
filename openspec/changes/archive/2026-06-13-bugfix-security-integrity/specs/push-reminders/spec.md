## MODIFIED Requirements

### Requirement: Scheduled Dienst-Erinnerung (Push + optional E-Mail)
Das System SHALL zugeteilten Nutzern 48h vor einem Dienst eine Push Notification senden. Die Idempotenz-Garantie MUSS durch die Reihenfolge der Operationen sichergestellt werden: der `notification_log`-Eintrag MUSS angelegt werden, BEVOR der Push gesendet wird. Nur wenn der INSERT tatsächlich eine neue Zeile erzeugt (RowsAffected = 1), wird der Push gesendet.

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
