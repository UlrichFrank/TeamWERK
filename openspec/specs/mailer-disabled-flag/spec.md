### Requirement: Mailer kann per Env-Flag deaktiviert werden
Das System SHALL E-Mail-Versand überspringen, wenn `MAILER_DISABLED=true` in der Umgebung gesetzt ist. Stattdessen MUSS ein Logeintrag mit Empfänger und Subject geschrieben werden. Der Rückgabewert von `Send()` SHALL `nil` sein (kein Fehler).

#### Scenario: MAILER_DISABLED=true gesetzt
- **WHEN** `MAILER_DISABLED=true` in der Umgebung gesetzt ist und `Send()` aufgerufen wird
- **THEN** wird kein SMTP-Verbindungsversuch unternommen
- **THEN** erscheint ein Logeintrag der Form `[mailer] disabled — an: <to>, Betreff: <subject>`
- **THEN** gibt `Send()` `nil` zurück

#### Scenario: MAILER_DISABLED nicht gesetzt oder leer
- **WHEN** `MAILER_DISABLED` nicht in der Umgebung gesetzt ist oder leer ist
- **THEN** verhält sich `Send()` wie bisher (SMTP-Versand)

#### Scenario: MAILER_DISABLED=false gesetzt
- **WHEN** `MAILER_DISABLED=false` gesetzt ist
- **THEN** verhält sich `Send()` wie bisher (SMTP-Versand)
