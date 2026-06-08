# dienst-fuer-familienmitglied Specification

## Purpose
Ein Elternteil kann auf der Dienstbörse einen Dienst stellvertretend für ein verknüpftes Kind mit aktivem Proxy-Account (`can_login = 0`) beanspruchen. Der Dienst wird dem `user_id` des Kindes zugebucht, nicht dem Elternteil.

## Requirements

### Requirement: Elternteil beansprucht Dienst stellvertretend für ein Kind
Das System SHALL es einem Elternteil ermöglichen, auf der Dienstbörse einen Dienst für ein verknüpftes Kind mit aktivem Proxy-Account zu beanspruchen. Der Dienst wird dem `user_id` des Kindes zugebucht.

#### Scenario: Elternteil öffnet Claim-Dialog mit Kind-Auswahl
- **WHEN** ein Elternteil auf der Dienstbörse auf „Eintragen" für einen offenen Slot klickt
- **AND** mindestens ein verknüpftes Kind mit aktivem Proxy-Account (`can_login = 0`, `user_id` gesetzt via `family_links`) existiert
- **THEN** zeigt das System einen „Für wen?"-Dialog mit eigenem Namen als Default sowie je einem Eintrag pro Kind mit Proxy-Account

#### Scenario: Elternteil ohne Kinder mit Proxy-Account — kein Dialog
- **WHEN** ein Elternteil ohne verknüpfte Kinder mit Proxy-Account auf „Eintragen" klickt
- **THEN** wird der Dienst direkt dem Elternteil selbst zugebucht (kein Dialog)

#### Scenario: Claim für Kind wird dem Kind zugebucht
- **WHEN** ein Elternteil im Dialog das Kind auswählt und bestätigt
- **THEN** ruft das Frontend `POST /api/duty-board/{slotId}/claim` mit `{ user_id: <kind_user_id> }` auf
- **THEN** legt das Backend eine `duty_assignments`-Zeile mit der `user_id` des Kindes an
- **THEN** wird das `duty_accounts`-Konto des Kindes aktualisiert (nicht das des Elternteils)

#### Scenario: Berechtigungsprüfung — nur verknüpfte Kinder erlaubt
- **WHEN** ein Elternteil `POST /api/duty-board/{slotId}/claim` mit einer `user_id` aufruft, die nicht über `family_links` mit ihm verknüpft ist
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Kind hat Dienst bereits belegt
- **WHEN** ein Elternteil versucht, denselben Slot erneut für dasselbe Kind zu beanspruchen
- **THEN** antwortet der Server mit HTTP 409 (UNIQUE-Constraint auf `duty_assignments`)

### Requirement: Dienstbörse zeigt Proxy-Account-Namen korrekt an
Das System SHALL Proxy-Account-Inhaber (Kinder) in der Assignee-Liste eines Slots mit ihrem Namen anzeigen, ohne Kontaktdaten (E-Mail, Telefon).

#### Scenario: Kind als Assignee in der Dienstbörse
- **WHEN** ein Kind mit Proxy-Account einen Dienst übernommen hat
- **THEN** erscheint sein Name in der Assignee-Liste des Slots
- **THEN** sind keine Telefonnummern oder E-Mail-Adresse sichtbar (Proxy-Account hat keine)
