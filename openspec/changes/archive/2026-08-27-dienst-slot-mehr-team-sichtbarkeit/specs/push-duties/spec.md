## MODIFIED Requirements

### Requirement: Push bei Dienst-Ereignissen

Das System SHALL berechtigten Nutzern eine Push Notification senden, wenn neue Dienst-Slots
verfügbar sind oder ein Slot gelöscht wird, dem sie zugeteilt sind — sofern Push für
Kategorie `duties` nicht deaktiviert ist und die Benachrichtigung nicht per `silent`-Flag
unterdrückt wurde.

Die Empfängermenge bei einem neu angelegten Slot (`POST /api/duty-slots`) SHALL dem
Team-Scope des Slots folgen: bei gesetztem `team_id` dessen Team, bei `team_id = null`
mit gesetztem `game_id` **alle** Teams des Spiels (`game_teams`), und nur ohne beides
vereinsweit. Damit deckt sich die Benachrichtigung mit der Sichtbarkeit in der
Dienstbörse.

Die Benachrichtigung über einen gelöschten Slot SHALL die **Dienstart**, den
**Event-Namen** und das **Event-Datum** enthalten sowie den Namen des auslösenden Nutzers
und — falls angegeben — den Löschgrund. Der Platzhaltertext „Ein Dienst, für den du
eingetragen warst, wurde abgesagt" SHALL NICHT mehr verwendet werden.

Die `url` SHALL weiterhin auf `/dienste` zeigen: die Dienstbörse existiert nach der
Löschung, und der Empfänger kann sich dort neu eintragen.

#### Scenario: Neuer Dienst-Slot erstellt
- **WHEN** ein Admin oder Trainer einen neuen Dienst-Slot über `POST /api/duty-slots` anlegt
- **THEN** erhalten alle berechtigten Nutzer (spieler, elternteil, trainer im Team) eine Push Notification „Neuer Dienst verfügbar"

#### Scenario: Neuer Slot ohne team_id an einem Mehr-Team-Termin
- **WHEN** ein Slot mit `team_id: null` und `game_id` eines Termins mit den Teams A und B angelegt wird
- **THEN** erhalten die berechtigten Nutzer aus Team A und Team B eine Push Notification
- **AND** erhalten Nutzer eines unbeteiligten Teams C keine

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
