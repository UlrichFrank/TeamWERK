## ADDED Requirements

### Requirement: membership_requests speichert Vor- und Nachname getrennt
Das System SHALL `first_name` und `last_name` als separate Pflichtfelder in `membership_requests` speichern. Das bisherige Feld `name` wird entfernt. Bestandsdaten werden heuristisch aufgeteilt (erstes Wort = Vorname, Rest = Nachname).

#### Scenario: Migration teilt Bestandsdaten auf
- **WHEN** die Datenbankmigration 009 auf einer Datenbank mit bestehenden `membership_requests`-Einträgen ausgeführt wird
- **THEN** enthält jeder Eintrag `first_name` (erstes Wort des alten `name`) und `last_name` (Rest), und das Feld `name` existiert nicht mehr

#### Scenario: Rollback stellt name-Feld wieder her
- **WHEN** Migration 009 rückgängig gemacht wird
- **THEN** wird `name` als Concat aus `first_name || ' ' || last_name` (mit Trim) wiederhergestellt und die neuen Spalten werden entfernt
