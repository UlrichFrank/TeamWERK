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

### Requirement: Notification an Dienst-Zugewiesene bei Event-Löschung

Beim Löschen eines Spiels oder generischen Ereignisses (`DELETE /api/games/{id}`) SHALL das System alle Nutzer benachrichtigen, die einen `duty_assignment` für einen Slot des betroffenen Events hatten — unabhängig vom Assignment-Status (`pending` oder `fulfilled`). Die Benachrichtigung erfolgt über die `notifications.Send`-Fassade in der Kategorie `duties`, sodass Push- und Email-Präferenzen pro Nutzer respektiert werden.

#### Scenario: Spiel mit zugewiesenen Diensten wird gelöscht

- **WHEN** ein Trainer ein Spiel mit drei Diensten löscht, von denen zwei zugesagt (`pending`) und einer erbracht (`fulfilled`) sind
- **THEN** erhalten alle drei Dienst-Zugewiesenen eine Notification mit dem Titel „Dienst entfällt" und dem Body „Dein Dienst zum {Gegnername} am {Datum} wurde gelöscht."
- **THEN** wird der Link „/dienste" mitgegeben

#### Scenario: Generisches Event mit Dienst wird gelöscht

- **WHEN** ein Trainer ein generisches Event (z.B. „Vereinsfest") mit Diensten löscht
- **THEN** erhalten die Zugewiesenen die Notification mit dem Event-Namen im Body („Dein Dienst zum Vereinsfest am 14.06. wurde gelöscht.")

#### Scenario: Event ohne Dienste wird gelöscht

- **WHEN** ein Trainer ein Event ohne zugewiesene Dienste löscht
- **THEN** wird keine `duties`-Notification verschickt
- **WHEN** das Event ein Spiel ist
- **THEN** wird trotzdem die bestehende `games`-Notification „Spiel abgesagt" an die Team-Responder verschickt

#### Scenario: Nutzer hat Email aktiv für Dienste

- **WHEN** ein Dienst-Zugewiesener `email_enabled=1` für `duties` hat und sein Event gelöscht wird
- **THEN** erhält der Nutzer eine Email mit dem persönlich formulierten Body und dem Direktlink
