## ADDED Requirements

### Requirement: Push bei Dienst-Ereignissen
Das System SHALL berechtigten Nutzern eine Push Notification senden, wenn neue Dienst-Slots verfügbar sind oder ein Slot gelöscht wird, dem sie zugeteilt sind — sofern Push für Kategorie `duties` nicht deaktiviert.

#### Scenario: Neuer Dienst-Slot erstellt
- **WHEN** ein Admin oder Trainer einen neuen Dienst-Slot über `POST /api/duty-slots` anlegt
- **THEN** erhalten alle berechtigten Nutzer (spieler, elternteil, trainer im Team) eine Push Notification „Neuer Dienst verfügbar"

#### Scenario: Dienst-Slot gelöscht (zugeteilte User)
- **WHEN** ein Slot über `DELETE /api/duty-slots/{id}` gelöscht wird und Nutzer dafür eingeteilt waren
- **THEN** erhalten alle bisher zugeteilten Nutzer eine Push Notification „Dienst abgesagt"

#### Scenario: Nutzer mit deaktiviertem Push
- **WHEN** ein Dienst-Ereignis eintritt und der Nutzer hat `push_enabled=0` für `duties`
- **THEN** erhält dieser Nutzer keine Push Notification
