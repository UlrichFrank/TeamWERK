## ADDED Requirements

### Requirement: Kategoriebasierte Notification-Fassade

Das System SHALL eine zentrale Funktion `notifications.Send(db, cfg, uids, category, title, body, url)` bereitstellen, die für die übergebenen Nutzer-IDs anhand der gespeicherten `notification_preferences` der angegebenen Kategorie automatisch Push und/oder Email auslöst. Aufrufer DÜRFEN NICHT mehr direkt zwischen Push- und Email-Versand entscheiden.

Die Fassade SHALL:

- nur Push an Nutzer schicken, deren `notification_preferences.push_enabled = 1` für die Kategorie ist (Default 1, wenn keine Zeile existiert)
- nur Email an Nutzer schicken, deren `notification_preferences.email_enabled = 1` für die Kategorie ist (Default 0, wenn keine Zeile existiert)
- den Email-Versand asynchron pro Empfänger als Goroutine ausführen, damit der HTTP-Response nicht blockiert
- für leere Empfängerlisten still und ohne Fehler zurückkehren

#### Scenario: Nutzer hat nur Push aktiv (Default)

- **WHEN** ein Aufrufer `notifications.Send(db, cfg, [u1], "duties", ...)` mit einem Nutzer ohne Preference-Zeile aufruft
- **THEN** erhält der Nutzer eine Push Notification
- **THEN** erhält der Nutzer keine Email

#### Scenario: Nutzer hat Email aktiv für die Kategorie

- **WHEN** ein Aufrufer `notifications.Send(db, cfg, [u1], "duties", ...)` mit einem Nutzer aufruft, der `email_enabled=1` für `duties` hat
- **THEN** wird eine Email an die in `users.email` hinterlegte Adresse verschickt
- **THEN** wird, sofern `push_enabled=1` ist, zusätzlich eine Push verschickt
- **THEN** enthält die Email den `body`-Text plus eine Zeile „Direktlink: https://intern.team-stuttgart.org{url}"

#### Scenario: Nutzer hat Push deaktiviert und Email aktiv

- **WHEN** ein Nutzer `push_enabled=0` und `email_enabled=1` für `games` hat und das System ein Spiel-Ereignis verschickt
- **THEN** erhält der Nutzer eine Email
- **THEN** erhält der Nutzer keine Push

#### Scenario: Leere Empfängerliste

- **WHEN** ein Aufrufer `notifications.Send(db, cfg, [], "duties", ...)` mit leerer Empfängerliste aufruft
- **THEN** kehrt die Funktion ohne Fehler zurück und es werden keine Nachrichten versendet

### Requirement: Migrierte Aufrufer dürfen kein direktes Push/Email mehr aufrufen

Alle Handler-Pfade, die Push oder Email an Nutzergruppen versenden, SHALL die Fassade `notifications.Send` verwenden. Direkte Aufrufe von `push.FilterByPushPref` + `push.SendToUsers` oder direktes `mailer.Send` außerhalb der Fassade SIND NUR im Scheduler-Job `duty_reminders` und im Authentifizierungs-Pfad (Welcome-Mails, Passwort-Reset) erlaubt.

#### Scenario: Event-Löschung benachrichtigt Dienst-Zugewiesene

- **WHEN** ein Trainer ein Spiel mit zugewiesenen Diensten löscht
- **THEN** ruft `games.DeleteGame` `notifications.Send(..., "duties", "Dienst entfällt", ...)` für alle `duty_assignments.user_id` der betroffenen Slots auf
- **THEN** ruft `games.DeleteGame` zusätzlich `notifications.Send(..., "games", "Spiel abgesagt", ...)` für die Team-Spielresponder auf

#### Scenario: Slot-Löschung benachrichtigt Zugewiesene

- **WHEN** ein Trainer einen einzelnen Dienst-Slot über `DELETE /api/duty-slots/{id}` löscht
- **THEN** ruft `duties.DeleteSlot` `notifications.Send(..., "duties", ...)` auf — nicht mehr `push.FilterByPushPref + SendToUsers` direkt

#### Scenario: Training-Löschung benachrichtigt Team

- **WHEN** ein Trainer ein Training über `DELETE /api/training-sessions/{id}` löscht
- **THEN** ruft `trainings.DeleteSession` `notifications.Send(..., "trainings", ...)` auf
- **THEN** werden keine Dienste verändert (Trainings haben kein Dienst-Bezug)
