# push-duties Specification

## Purpose

Diese Spezifikation beschreibt die Capability `push-duties`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Push bei Dienst-Ereignissen

Das System SHALL berechtigten Nutzern eine Push Notification senden, wenn neue Dienst-Slots
verfügbar sind oder ein Slot gelöscht wird, dem sie zugeteilt sind — sofern Push für
Kategorie `duties` nicht deaktiviert ist und die Benachrichtigung nicht per `silent`-Flag
unterdrückt wurde.

Die Benachrichtigung über einen gelöschten Slot SHALL die **Dienstart**, den
**Event-Namen** und das **Event-Datum** enthalten sowie den Namen des auslösenden Nutzers
und — falls angegeben — den Löschgrund. Der Platzhaltertext „Ein Dienst, für den du
eingetragen warst, wurde abgesagt" SHALL NICHT mehr verwendet werden.

Die `url` SHALL weiterhin auf `/dienste` zeigen: die Dienstbörse existiert nach der
Löschung, und der Empfänger kann sich dort neu eintragen.

#### Scenario: Neuer Dienst-Slot erstellt
- **WHEN** ein Admin oder Trainer einen neuen Dienst-Slot über `POST /api/duty-slots` anlegt
- **THEN** erhalten alle berechtigten Nutzer (spieler, elternteil, trainer im Team) eine Push Notification „Neuer Dienst verfügbar"

#### Scenario: Dienst-Slot gelöscht (zugeteilte User)
- **WHEN** ein Slot über `DELETE /api/duty-slots/{id}` gelöscht wird und Nutzer dafür eingeteilt waren
- **THEN** erhalten alle bisher zugeteilten Nutzer eine Push Notification „Dienst abgesagt"
- **THEN** enthält der Body die Dienstart, den Event-Namen, das Event-Datum und den Namen des auslösenden Nutzers
- **THEN** zeigt der Klick-Link auf `/dienste`

#### Scenario: Dienst-Slot mit Grund gelöscht
- **WHEN** ein Slot mit `{"reason":"Dienst wird nicht mehr gebraucht"}` gelöscht wird
- **THEN** enthält der Body zusätzlich diesen Text

#### Scenario: Vorstand löscht einen Slot ohne Benachrichtigung
- **WHEN** ein Nutzer mit Capability `suppress_event_notification` einen Slot mit `{"silent":true}` löscht
- **THEN** erhält kein zugeteilter Nutzer eine Notification
- **THEN** wird das SSE-Live-Update trotzdem gesendet

#### Scenario: Nutzer mit deaktiviertem Push
- **WHEN** ein Dienst-Ereignis eintritt und der Nutzer hat `push_enabled=0` für `duties`
- **THEN** erhält dieser Nutzer keine Push Notification

### Requirement: Notification an Dienst-Zugewiesene bei Event-Löschung

Beim Löschen eines Spiels oder generischen Ereignisses (`DELETE /api/games/{id}`) SHALL das
System alle Nutzer benachrichtigen, die einen `duty_assignment` für einen Slot des
betroffenen Events hatten — unabhängig vom Assignment-Status (`pending` oder `fulfilled`).
Die Benachrichtigung erfolgt über die `notify.Send`-Fassade in der Kategorie `duties`,
sodass Push- und Email-Präferenzen pro Nutzer respektiert werden.

Der Body SHALL zusätzlich zum bisherigen Satz den **Namen des auslösenden Nutzers** und —
falls angegeben — den **Löschgrund** enthalten. Der Link `/dienste` SHALL erhalten bleiben.

Wird die Löschung per `silent`-Flag von einem Nutzer mit Capability
`suppress_event_notification` unterdrückt, SHALL auch diese `duties`-Notification entfallen.

#### Scenario: Spiel mit zugewiesenen Diensten wird gelöscht

- **WHEN** ein Trainer ein Spiel mit drei Diensten löscht, von denen zwei zugesagt (`pending`) und einer erbracht (`fulfilled`) sind
- **THEN** erhalten alle drei Dienst-Zugewiesenen eine Notification mit dem Titel „Dienst entfällt" und dem Body „Dein Dienst zum {Gegnername} am {Datum} wurde gelöscht."
- **THEN** enthält der Body zusätzlich den Namen des auslösenden Nutzers
- **THEN** wird der Link „/dienste" mitgegeben

#### Scenario: Spiel mit Grund gelöscht

- **WHEN** ein Trainer ein Spiel mit zugewiesenen Diensten und `{"reason":"Halle gesperrt"}` löscht
- **THEN** enthält der Body der `duties`-Notification zusätzlich den Text „Halle gesperrt"

#### Scenario: Generisches Event mit Dienst wird gelöscht

- **WHEN** ein Trainer ein generisches Event (z.B. „Vereinsfest") mit Diensten löscht
- **THEN** erhalten die Zugewiesenen die Notification mit dem Event-Namen im Body („Dein Dienst zum Vereinsfest am 14.06. wurde gelöscht.")

#### Scenario: Event ohne Dienste wird gelöscht

- **WHEN** ein Trainer ein Event ohne zugewiesene Dienste löscht
- **THEN** wird keine `duties`-Notification verschickt
- **WHEN** das Event ein Spiel ist
- **THEN** wird trotzdem die bestehende `games`-Notification „Spiel abgesagt" an die Team-Responder verschickt

#### Scenario: Stumme Löschung unterdrückt auch die Dienst-Notification

- **WHEN** ein Nutzer mit Capability `suppress_event_notification` ein Spiel mit zugewiesenen Diensten und `{"silent":true}` löscht
- **THEN** erhält weder das Team eine `games`- noch ein Zugewiesener eine `duties`-Notification

#### Scenario: Nutzer hat Email aktiv für Dienste

- **WHEN** ein Dienst-Zugewiesener `email_enabled=1` für `duties` hat und sein Event gelöscht wird
- **THEN** erhält der Nutzer eine Email mit dem persönlich formulierten Body und dem Direktlink

