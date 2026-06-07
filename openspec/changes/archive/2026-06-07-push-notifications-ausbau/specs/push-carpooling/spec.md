## ADDED Requirements

### Requirement: Push bei Fahrgemeinschafts-Ereignissen
Das System SHALL betroffene Nutzer per Push benachrichtigen, wenn eine Fahrgemeinschaft bestätigt oder abgesagt wird — sofern Push für Kategorie `carpooling` nicht deaktiviert.

#### Scenario: Fahrgemeinschaft bestätigt
- **WHEN** eine Fahrgemeinschafts-Anfrage angenommen wird
- **THEN** erhält der anfragende Nutzer eine Push Notification „Fahrgemeinschaft bestätigt"

#### Scenario: Fahrgemeinschaft abgesagt
- **WHEN** eine bestehende Fahrgemeinschaft storniert oder abgelehnt wird
- **THEN** erhält der betroffene Nutzer eine Push Notification „Fahrgemeinschaft abgesagt"

#### Scenario: Nutzer mit deaktiviertem Push
- **WHEN** ein Fahrgemeinschafts-Ereignis eintritt und der Nutzer hat `push_enabled=0` für `carpooling`
- **THEN** erhält dieser Nutzer keine Push Notification
