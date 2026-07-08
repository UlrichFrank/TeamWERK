## ADDED Requirements

### Requirement: Präferenz-Filterung ist getestet
Die Präferenz-Funktionen `push.FilterByPushPref`, `push.HasEmailEnabled` und `push.GetAllPreferences` SHALL durch Unit-Tests abgedeckt sein, die das Default-Verhalten (kein Präferenz-Row ⇒ `push_enabled=true`, `email_enabled=false`) und die Kategorie-Isolation belegen.

#### Scenario: Kein Row ⇒ Push-Default aktiv
- **WHEN** `FilterByPushPref(db, [uid], "games")` ohne gespeicherte Präferenz für `uid` aufgerufen wird
- **THEN** ist `uid` im Ergebnis enthalten (Default push=true)

#### Scenario: push_enabled=0 filtert aus
- **WHEN** für `uid` eine Zeile `(games, push_enabled=0)` existiert und `FilterByPushPref(db, [uid], "games")` aufgerufen wird
- **THEN** ist `uid` NICHT im Ergebnis enthalten

#### Scenario: Kategorie-Isolation
- **WHEN** `uid` `push_enabled=0` nur für `games` hat und `FilterByPushPref(db, [uid], "trainings")` aufgerufen wird
- **THEN** ist `uid` im Ergebnis enthalten (andere Kategorie unberührt)

#### Scenario: GetAllPreferences liefert alle Kategorien mit Defaults
- **WHEN** `GetAllPreferences(db, uid)` ohne gespeicherte Zeilen aufgerufen wird
- **THEN** enthält die Map alle Kategorien (inkl. `chat`) mit `push=true`, `email=false`

### Requirement: notify-Fassade fächert Push und Email getrennt auf
`notify.Send` SHALL Push nur an Nutzer mit `push_enabled=1` und Email nur an Nutzer mit `email_enabled=1` (je Kategorie) senden; eine leere Empfängerliste SHALL keinen Versand auslösen.

#### Scenario: Getrennte Zweige
- **WHEN** `notify.Send` mit einer Kategorie und Nutzern aufgerufen wird, von denen einer nur Push, ein anderer nur Email aktiviert hat
- **THEN** erhält der erste ausschließlich Push, der zweite ausschließlich Email

#### Scenario: Leere Liste
- **WHEN** `notify.Send` mit leerer Empfängerliste aufgerufen wird
- **THEN** wird weder Push noch Email versendet

#### Scenario: Email trägt Direktlink
- **WHEN** `sendCategoryEmail` mit nicht-leerer `url` einen Nutzer mit E-Mail-Adresse verarbeitet
- **THEN** enthält der Mail-Body eine `Direktlink:`-Zeile mit `BaseURL + url`

### Requirement: Abo- und Präferenz-Endpoints sind getestet
Die Endpoints `POST/DELETE /api/push/subscribe` und `GET/PUT /api/profile/notification-preferences` SHALL Happy-Path und Fehlerfall abdecken (gemäß Test-Standard).

#### Scenario: Subscribe legt Abo an
- **WHEN** ein authentifizierter Nutzer `POST /api/push/subscribe` mit endpoint/p256dh/auth aufruft
- **THEN** antwortet der Server mit 204 und die Subscription liegt für diesen Nutzer in `push_subscriptions`

#### Scenario: Subscribe ohne Pflichtfeld
- **WHEN** `POST /api/push/subscribe` ohne `endpoint` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Unsubscribe schützt fremde Abos
- **WHEN** Nutzer B `DELETE /api/push/subscribe` mit dem Endpoint von Nutzer A aufruft
- **THEN** wird das Abo von A NICHT gelöscht (Filter `endpoint = ? AND user_id = ?`)

#### Scenario: Preferences GET liefert Defaults
- **WHEN** ein Nutzer ohne gespeicherte Präferenzen `GET /api/profile/notification-preferences` aufruft
- **THEN** enthält die Antwort alle Kategorien mit `push=true`, `email=false`

### Requirement: Chat-Push an mehrere Empfänger ist getestet
Der Chat-Push-Pfad SHALL für Gruppen mit mehreren aktiven Mitgliedern und für Broadcasts getestet sein, jeweils mit korrektem `badge`-Wert je Empfänger.

#### Scenario: Gruppe mit N Empfängern
- **WHEN** in einer Gruppenkonversation mit drei aktiven Mitgliedern eine Nachricht gesendet wird
- **THEN** wird der Push-Seam für beide Nicht-Sender-Empfänger aufgerufen (Sender ausgeschlossen)

#### Scenario: Broadcast löst Push aus
- **WHEN** ein Broadcast an eine Zielgruppe gesendet wird
- **THEN** wird der Push-Seam für die Nicht-Sender-Empfänger aufgerufen

### Requirement: Kategorie-Korrektheit je Trigger ist getestet
Für jede Domäne mit präferenzgesteuerten Benachrichtigungen (games, trainings, duties, carpooling, membership) SHALL ein Test belegen, dass der auslösende Handler die **erwartete** Kategorie an `notify.Send` übergibt.

#### Scenario: Spiel-Trigger nutzt Kategorie games
- **WHEN** ein spielbezogener Trigger (z.B. Spiel angelegt) eine Benachrichtigung auslöst
- **THEN** wird `notify.Send` mit Kategorie `games` aufgerufen

#### Scenario: Dienst-Trigger nutzt Kategorie duties
- **WHEN** ein dienstbezogener Trigger (z.B. neuer Dienst-Slot) eine Benachrichtigung auslöst
- **THEN** wird `notify.Send` mit Kategorie `duties` aufgerufen

### Requirement: Preference-Bypass-Call-Sites sind dokumentiert festgenagelt
Die Call-Sites, die `push.SendToUsers` ohne `FilterByPushPref` aufrufen (match-report-reminder, attendance-reminder, video-retention, video-ready, carpool-pairing-request, match-report-submitted), SHALL durch Tests abgesichert sein, die das aktuelle Verhalten pinnen und im Testkommentar als bewusste Design-Entscheidung-oder-offener-Punkt markieren.

#### Scenario: Bypass-Verhalten gepinnt
- **WHEN** ein Bypass-Trigger (z.B. attendance-reminder) einen Empfänger mit `push_enabled=0` für die naheliegende Kategorie hat
- **THEN** erhält der Empfänger die Push dennoch (aktuelles Verhalten), und der Test dokumentiert dies explizit als zu klärende Design-Frage
