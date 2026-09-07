# push-duties Specification

## Purpose

Diese Spezifikation beschreibt die Capability `push-duties`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Push bei Dienst-Ereignissen

Das System SHALL berechtigten Nutzern eine Push Notification senden, wenn neue Dienst-Slots
verfügbar sind oder ein Slot gelöscht wird, dem sie zugeteilt sind — sofern Push für
Kategorie `duties` nicht deaktiviert ist und die Benachrichtigung nicht per `silent`-Flag
unterdrückt wurde.

Die Empfängermenge bei einem neu angelegten Slot (`POST /api/duty-slots`) SHALL der
Sichtbarkeit in der Dienstbörse folgen und dazu **beide** Filter anlegen, die
`GET /api/duty-board` für nicht privilegierte Nutzer anlegt:

1. **Team-Scope** — bei gesetztem `game_id` alle Teams des Spiels (`game_teams`),
   `team_id` bleibt dort unbeachtet; bei einem Slot ohne Spiel dessen `team_id`; ohne
   beides vereinsweit. Empfänger im Team-Scope sind Spieler des Kaders, Trainer des Kaders
   (`kader_trainers`) und Eltern eines Spielers des Kaders. Die Kader-Zugehörigkeit SHALL
   ausschließlich in der **aktiven Saison** gelten — eine Kader-Zeile aus einer
   abgeschlossenen Saison begründet keine Benachrichtigung.
2. **Zielgruppe** — `COALESCE(duty_slots.audiences, duty_types.audiences)`. Ist der Wert
   NULL oder leer, gilt keine Einschränkung. Andernfalls SHALL nur benachrichtigt werden,
   wer die Zielgruppe trifft: über eine eigene Vereinsfunktion aus dem Array, oder — beim
   Eintrag `eltern` — als Elternteil eines Spielers innerhalb desselben Team-Scopes.

Der Audience-Bypass privilegierter Leser (`admin`, `?audience=all` für
`vorstand`/`vorstand_beisitzer`/`trainer`/`sportliche_leitung`) SHALL **nicht** auf die
Benachrichtigung durchschlagen: das Recht, alle Dienste zu sehen, begründet keine
Benachrichtigung über jeden neuen Dienst.

Die Benachrichtigung über einen gelöschten Slot SHALL die **Dienstart**, den
**Event-Namen** und das **Event-Datum** enthalten sowie den Namen des auslösenden Nutzers
und — falls angegeben — den Löschgrund. Der Platzhaltertext „Ein Dienst, für den du
eingetragen warst, wurde abgesagt" SHALL NICHT mehr verwendet werden.

Die `url` SHALL weiterhin auf `/dienste` zeigen: die Dienstbörse existiert nach der
Löschung, und der Empfänger kann sich dort neu eintragen.

#### Scenario: Neuer Dienst-Slot erstellt
- **WHEN** ein Admin oder Trainer einen neuen Dienst-Slot ohne Zielgruppe über `POST /api/duty-slots` anlegt
- **THEN** erhalten die Spieler des Kaders, die Trainer des Kaders und die Eltern der Spieler eine Push Notification „Neuer Dienst verfügbar"

#### Scenario: Neuer Slot ohne team_id an einem Mehr-Team-Termin
- **WHEN** ein Slot mit `game_id` eines Termins mit den Teams A und B angelegt wird
- **THEN** erhalten die berechtigten Nutzer aus Team A und Team B eine Push Notification
- **AND** erhalten Nutzer eines unbeteiligten Teams C keine

#### Scenario: Slot mit Zielgruppe „eltern"
- **WHEN** ein Slot mit `audiences=["eltern"]` für Team A angelegt wird
- **THEN** erhält ein Elternteil eines Spielers aus Team A eine Push Notification
- **AND** erhalten die Spieler und Trainer von Team A keine

#### Scenario: Slot mit Zielgruppe „trainer"
- **WHEN** ein Slot mit `audiences=["trainer"]` für Team A angelegt wird
- **THEN** erhält ein Trainer des Kaders von Team A eine Push Notification
- **AND** erhalten die Spieler von Team A keine

#### Scenario: Zielgruppe aus dem Diensttyp
- **WHEN** ein Slot ohne eigenes `audiences`-Array angelegt wird und sein Diensttyp `audiences=["eltern"]` trägt
- **THEN** gilt für die Empfängermenge die Zielgruppe des Diensttyps

#### Scenario: Kader-Zeile aus einer abgeschlossenen Saison
- **WHEN** ein Slot für Team A angelegt wird und ein Nutzer nur im Kader einer **inaktiven** Saison von Team A steht
- **THEN** erhält dieser Nutzer keine Push Notification

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

