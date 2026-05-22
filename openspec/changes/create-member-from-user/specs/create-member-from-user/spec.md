## ADDED Requirements

### Requirement: Mitglied aus Nutzer-Account anlegen
Das System SHALL einem Admin erlauben, für einen registrierten Nutzer ohne verknüpftes Mitglied direkt einen Mitgliedsdatensatz zu erstellen. Der neue Datensatz MUSS sofort mit dem Nutzer-Account verknüpft sein (`members.user_id`).

#### Scenario: Button nur sichtbar wenn kein Mitglied verknüpft
- **WHEN** Admin die Nutzerliste `/admin/nutzer` aufruft
- **THEN** zeigt das System für jeden Nutzer ohne verknüpftes Mitglied einen „Mitglied anlegen"-Button in der Zeile

#### Scenario: Button nicht sichtbar wenn Mitglied bereits vorhanden
- **WHEN** ein Nutzer bereits ein verknüpftes Mitglied hat
- **THEN** zeigt das System keinen „Mitglied anlegen"-Button für diesen Nutzer

#### Scenario: Erfolgreiches Anlegen
- **WHEN** Admin auf „Mitglied anlegen" klickt
- **THEN** erstellt das System einen Mitgliedsdatensatz mit `status='aktiv'`, übernimmt den Namen des Accounts als Vor-/Nachname und verknüpft ihn über `user_id`

#### Scenario: Button verschwindet nach Erfolg
- **WHEN** das Mitglied erfolgreich angelegt wurde
- **THEN** verschwindet der „Mitglied anlegen"-Button in der betroffenen Zeile ohne Seitenneuladen

### Requirement: Name-Übernahme aus Account
Das System SHALL den `name`-Wert des Nutzer-Accounts als Ausgangspunkt für Vor- und Nachname des Mitglieds verwenden.

#### Scenario: Name mit Leerzeichen
- **WHEN** der Account-Name ein oder mehrere Leerzeichen enthält (z. B. „Maria Müller")
- **THEN** wird das erste Wort als `first_name` und alles danach als `last_name` gesetzt

#### Scenario: Name ohne Leerzeichen
- **WHEN** der Account-Name kein Leerzeichen enthält (z. B. „admin")
- **THEN** wird der gesamte Name als `first_name` gesetzt und `last_name` bleibt leer

### Requirement: Schutz vor Doppelanlage
Das System SHALL verhindern, dass für einen Nutzer, der bereits ein verknüpftes Mitglied hat, ein weiteres angelegt wird.

#### Scenario: Doppelter Anlageversuch via API
- **WHEN** `POST /api/admin/users/{id}/create-member` aufgerufen wird und der Nutzer bereits ein Mitglied hat
- **THEN** antwortet das System mit HTTP 409 Conflict

### Requirement: Zugriffsschutz
Der Endpoint `POST /api/admin/users/{id}/create-member` MUSS auf die Rolle `admin` beschränkt sein.

#### Scenario: Zugriff durch Nicht-Admin
- **WHEN** ein Nutzer ohne Admin-Rolle den Endpoint aufruft
- **THEN** antwortet das System mit HTTP 403 Forbidden
