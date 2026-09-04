## ADDED Requirements

### Requirement: Erlaubte Ziele abrufen

Das System SHALL unter `GET /api/chat/broadcast-targets` die Ziele zurückgeben, die der
aufrufende User in einer Mitteilung verwenden darf — dieselbe Menge, gegen die
`POST /api/chat/broadcasts` prüft. Der Composer SHALL diese Liste anzeigen, statt die
Ziele aus Rolle oder Vereinsfunktion abzuleiten.

Die Antwort SHALL je Ziel `kind`, `teamId` (null bei vereinsweiten Zielen und bei
`alle_trainer`), ein anzeigefertiges `label` und `count` (Anzahl distinkter Empfänger)
tragen. `count` SHALL den Absender mitzählen, wenn er zur Gruppe gehört — die Zahl
beschreibt die Gruppe, nicht den Fan-out.

Ziele mit `count = 0` SHALL enthalten sein und nicht ausgefiltert werden: eine leere
Gruppe ist eine legitime Auswahl, und das Verschweigen wäre genau der stille Fall, den
der `recipients`-Zähler sichtbar machen soll.

User ohne Senderecht SHALL HTTP 403 erhalten.

#### Scenario: Vorstand sieht vereinsweite Ziele und alle Teams

- **WHEN** ein Vorstand `GET /api/chat/broadcast-targets` aufruft
- **THEN** antwortet das System mit HTTP 200
- **AND** die Liste enthält die vier vereinsweiten Ziele (`users`, `members`, `spieler`, `eltern`)
- **AND** je Team mit Kader in der aktiven Saison die Ziele `team_spieler`, `team_eltern`, `team_trainer`

#### Scenario: Trainer sieht nur die Gruppen seiner Kader

- **WHEN** ein User mit Vereinsfunktion `trainer` (ohne `vorstand`, `sportliche_leitung`, `admin`), der Kader-Trainer von Team A ist, `GET /api/chat/broadcast-targets` aufruft
- **THEN** enthält die Liste `team_spieler`, `team_eltern` und `team_trainer` für Team A sowie `alle_trainer`
- **AND** sie enthält kein vereinsweites Ziel (`users`, `members`, `spieler`, `eltern`)
- **AND** sie enthält kein Ziel eines Teams, dessen Kader-Trainer er nicht ist

#### Scenario: User ohne Senderecht

- **WHEN** ein User ohne `admin`, `vorstand`, `sportliche_leitung` und ohne Kader-Trainer-Eintrag der aktiven Saison `GET /api/chat/broadcast-targets` aufruft
- **THEN** antwortet das System mit HTTP 403

## MODIFIED Requirements

### Requirement: Broadcast senden

Das System SHALL es Usern mit Rolle `admin`, einer der Vereinsfunktionen `vorstand` bzw.
`sportliche_leitung`, oder einem Kader-Trainer-Eintrag der aktiven Saison erlauben, eine
Mitteilung zu senden. Wer keine dieser Voraussetzungen erfüllt, SHALL mit HTTP 403
abgewiesen werden.

Der Request SHALL ein nicht-leeres Array `targets` tragen. Jedes Ziel SHALL ein `kind`
und — bei team-bezogenen Zielen — eine `teamId` tragen:

| `kind` | `teamId` | Empfängermenge |
|---|---|---|
| `users` | — | alle Zeilen in `users` |
| `members` | — | alle User, zu denen eine `members`-Zeile mit `user_id` existiert |
| `spieler` | — | alle User, deren Mitglied die Vereinsfunktion `spieler` trägt |
| `eltern` | — | alle `family_links.parent_user_id` (distinkt) |
| `team_spieler` | Pflicht | Spieler des Teams (regulärer **und** erweiterter Kader der aktiven Saison) |
| `team_eltern` | Pflicht | Eltern der Spieler dieses Teams via `family_links` |
| `team_trainer` | Pflicht | Kader-Trainer dieses Teams |
| `alle_trainer` | — | Kader-Trainer aller Teams der aktiven Saison |

Die team-bezogenen Ziele SHALL denselben Kreis auflösen wie die gleichnamige
Standardgruppe des Chats — es SHALL keine zweite Definition von „Spieler eines Teams"
entstehen. Ein unbekanntes `kind`, eine fehlende `teamId` bei einem team-bezogenen Ziel,
eine `teamId` bei einem vereinsweiten Ziel, ein leeres `targets` sowie die früheren Werte
`all`, `team`, `role` und `legacy` SHALL mit HTTP 400 abgelehnt werden.

Das System SHALL **jedes** Ziel gegen die Ziel-Allowlist des Absenders prüfen und den
gesamten Request mit HTTP 403 ablehnen, sobald **ein** Ziel darin fehlt — es SHALL keine
Teilzustellung geben. Vereinsweite Ziele SHALL ausschließlich `admin`, `vorstand` und
`sportliche_leitung` offenstehen. Ein Trainer ohne diese Funktionen SHALL `team_*`-Ziele
nur für Teams verwenden dürfen, deren Kader-Trainer er in der aktiven Saison ist; die
Zugehörigkeit als Spieler, erweiterter Kader oder Elternteil SHALL **kein** Senderecht
begründen. `alle_trainer` SHALL jedem Absender mit Senderecht offenstehen.

Die Empfängermenge SHALL die **Vereinigung** aller gewählten Ziele sein. Jeder Empfänger
SHALL genau eine `broadcast_reads`-Zeile erhalten, auch wenn er über mehrere Ziele
getroffen wird (Elternteil zweier Kinder in verschiedenen Teams, Spieler mit
Trainerfunktion). Mitglieder ohne verknüpften User-Account (`members.user_id IS NULL`)
SHALL in keiner Zielgruppe auftauchen.

Die Auflösung SHALL **keine Vereinsfunktionen über `family_links` vererben**: ein
Elternteil ohne eigene Vereinsfunktion `spieler` gehört nicht zur Zielgruppe `spieler`,
auch wenn sein Kind sie trägt. Damit sind `spieler` und `eltern` disjunkt auflösbar —
abweichend von `folder_permissions`, wo `club_function`-Einträge auf Eltern durchschlagen.

Der Request KANN optional `mediaId` enthalten; mindestens `body` (nicht leer) **oder**
`mediaId` MUSS vorhanden sein. Ein angegebenes `mediaId` MUSS auf eine existierende
`media`-Zeile verweisen.

Die Antwort SHALL HTTP 201 mit `{ "id": <broadcastId>, "recipients": <n> }` sein, wobei
`n` die Anzahl der benachrichtigten Empfänger **ohne den Absender** ist. Der Absender
SHALL eine `broadcast_reads`-Zeile mit gesetztem `read_at` erhalten, aber weder SSE noch
Push, und SHALL nicht in `recipients` gezählt werden. Alle übrigen Empfänger SHALL ein
SSE-Event `chat:new-broadcast` und eine Push-Benachrichtigung erhalten.

Die gewählten Ziele SHALL als je eine Zeile in `broadcast_targets` gespeichert werden.

#### Scenario: Admin sendet Text-Broadcast an alle

- **WHEN** ein Admin `POST /api/chat/broadcasts` mit `{ "body": "Wichtige Info", "targets": [{"kind": "users"}] }` aufruft
- **THEN** wird der Broadcast gespeichert, alle User außer dem Absender erhalten ein SSE-Event `chat:new-broadcast`, HTTP 201
- **AND** die Antwort enthält `recipients` gleich der Anzahl dieser User

#### Scenario: Trainer sendet an Spieler und Eltern seines Teams

- **WHEN** ein Kader-Trainer von Team A `POST /api/chat/broadcasts` mit `{ "body": "Halle geändert", "targets": [{"kind": "team_spieler", "teamId": <A>}, {"kind": "team_eltern", "teamId": <A>}] }` aufruft
- **THEN** antwortet der Server mit HTTP 201
- **AND** genau die Spieler des Teams (regulärer und erweiterter Kader) und deren Eltern haben je eine `broadcast_reads`-Zeile

#### Scenario: Trainer trifft ein Elternteil mit zwei Kindern nur einmal

- **WHEN** ein Trainer an `team_spieler` und `team_eltern` desselben Teams sendet und ein Elternteil zwei Kinder in diesem Team hat
- **THEN** existiert für dieses Elternteil genau eine `broadcast_reads`-Zeile
- **AND** `recipients` zählt es einmal

#### Scenario: Trainer darf kein fremdes Team adressieren

- **WHEN** ein Kader-Trainer von Team A `POST /api/chat/broadcasts` mit einem `team_spieler`-Ziel für Team B aufruft, dessen Kader-Trainer er nicht ist
- **THEN** antwortet der Server mit HTTP 403
- **AND** es wird kein Broadcast gespeichert

#### Scenario: Ein unerlaubtes Ziel kippt den ganzen Request

- **WHEN** ein Kader-Trainer von Team A `POST /api/chat/broadcasts` mit `targets` gleich `[{"kind": "team_spieler", "teamId": <A>}, {"kind": "users"}]` aufruft
- **THEN** antwortet der Server mit HTTP 403
- **AND** es wird kein Broadcast gespeichert und keine `broadcast_reads`-Zeile geschrieben

#### Scenario: Trainer darf nicht vereinsweit senden

- **WHEN** ein User mit Vereinsfunktion `trainer` (ohne `vorstand`, `sportliche_leitung`, `admin`) `POST /api/chat/broadcasts` mit `targets` gleich `[{"kind": "spieler"}]` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Trainer erbt kein Senderecht aus einer Eltern- oder Spielerrolle

- **WHEN** ein Kader-Trainer von Team A über `family_links` ein Kind im Team B hat und ein Ziel für Team B adressiert
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Trainer sendet an alle Trainer

- **WHEN** ein Kader-Trainer `POST /api/chat/broadcasts` mit `targets` gleich `[{"kind": "alle_trainer"}]` aufruft
- **THEN** antwortet der Server mit HTTP 201
- **AND** alle Kader-Trainer der aktiven Saison außer dem Absender erhalten eine `broadcast_reads`-Zeile

#### Scenario: Vorstand sendet an alle Spieler

- **WHEN** ein Vorstand `POST /api/chat/broadcasts` mit `{ "body": "Trainingsauftakt", "targets": [{"kind": "spieler"}] }` aufruft
- **THEN** antwortet der Server mit HTTP 201 und `recipients` gleich der Anzahl der User mit Vereinsfunktion `spieler` (ohne den Absender)
- **AND** genau diese User haben eine `broadcast_reads`-Zeile

#### Scenario: Vorstand adressiert ein Team, das er nicht trainiert

- **WHEN** ein Vorstand ohne Trainerfunktion `POST /api/chat/broadcasts` mit einem `team_spieler`-Ziel für ein beliebiges Team der aktiven Saison aufruft
- **THEN** antwortet der Server mit HTTP 201

#### Scenario: Sportliche Leitung darf jede Zielgruppe

- **WHEN** ein User mit Vereinsfunktion `sportliche_leitung` (ohne `vorstand`, ohne `admin`) `POST /api/chat/broadcasts` mit `targets` gleich `[{"kind": "users"}]` aufruft
- **THEN** antwortet der Server mit HTTP 201

#### Scenario: Eltern erben die Vereinsfunktion ihres Kindes nicht

- **WHEN** ein Elternteil ohne eigene Vereinsfunktion ein Kind mit Vereinsfunktion `spieler` hat und eine Mitteilung an `{"kind": "spieler"}` gesendet wird
- **THEN** erhält das Elternteil **keine** `broadcast_reads`-Zeile
- **AND** bei `{"kind": "eltern"}` erhält es genau eine

#### Scenario: Elternteil mehrerer Kinder wird einmal gezählt

- **WHEN** ein Elternteil über `family_links` mit zwei Kindern verknüpft ist und eine Mitteilung an `{"kind": "eltern"}` gesendet wird
- **THEN** existiert für dieses Elternteil genau eine `broadcast_reads`-Zeile
- **AND** `recipients` zählt es einmal

#### Scenario: Mitglied ohne Account wird nicht erreicht

- **WHEN** ein Mitglied mit Vereinsfunktion `spieler` und `members.user_id IS NULL` existiert und eine Mitteilung an `{"kind": "spieler"}` gesendet wird
- **THEN** entsteht für dieses Mitglied keine `broadcast_reads`-Zeile
- **AND** `recipients` zählt es nicht mit

#### Scenario: Team-Ziel ohne teamId

- **WHEN** ein berechtigter User ein Ziel `{"kind": "team_spieler"}` ohne `teamId` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Alte Zielgruppen-Werte werden abgelehnt

- **WHEN** ein Vorstand `POST /api/chat/broadcasts` mit einem Ziel-`kind` gleich `"all"`, `"team"`, `"role"` oder `"legacy"` aufruft
- **THEN** antwortet der Server mit HTTP 400
- **AND** es wird kein Broadcast gespeichert

#### Scenario: Fehlende Ziele

- **WHEN** ein berechtigter User einen Broadcast ohne `targets` oder mit leerem `targets`-Array sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Reine Bild-Mitteilung senden

- **WHEN** ein Vorstand `POST /api/chat/broadcasts` mit `{ "body": "", "mediaId": <id>, "targets": [{"kind": "users"}] }` aufruft
- **THEN** wird der Broadcast mit `media_id` und leerem Body gespeichert, HTTP 201

#### Scenario: Leere Mitteilung ohne Bild wird abgelehnt

- **WHEN** ein berechtigter User einen Broadcast mit leerem `body` und ohne `mediaId` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Unberechtigter User

- **WHEN** ein User ohne `admin`/`vorstand`/`sportliche_leitung` und ohne Kader-Trainer-Eintrag der aktiven Saison einen Broadcast sendet
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Bestands-Mitteilungen bleiben zustellbar

Das System SHALL Mitteilungen, die vor der Umstellung des Zielgruppen-Vokabulars gesendet
wurden, für ihre damaligen Empfänger unverändert auslieferbar halten. Die Zustellung SHALL
ausschließlich an `broadcast_reads` hängen; die gespeicherten Ziele einer Mitteilung SHALL
nach dem Senden nirgends mehr für die Zustellung ausgewertet werden.

Die bisherige Spalte `broadcasts.target_type` SHALL durch die Zeilentabelle
`broadcast_targets` abgelöst werden; jede Bestandszeile SHALL als genau ein Ziel mit
ihrem bisherigen Wert und ohne `teamId` übernommen werden. Der Wert `legacy` SHALL
persistierbar und lesbar, aber über `POST /api/chat/broadcasts` **nicht** setzbar sein.

#### Scenario: Alte Mitteilung bleibt sichtbar

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und für ihn eine `broadcast_reads`-Zeile zu einer vor der Migration gesendeten Mitteilung existiert
- **THEN** enthält die Antwort diese Mitteilung unverändert (Body, Sender, Zeitpunkt, Lesestatus)

#### Scenario: Bestandsziel überlebt die Migration

- **WHEN** vor der Migration eine Mitteilung mit `target_type = 'users'` existierte
- **THEN** existiert danach für sie genau eine `broadcast_targets`-Zeile mit `kind = 'users'` und ohne `team_id`

#### Scenario: legacy ist nicht schreibbar

- **WHEN** ein berechtigter User `POST /api/chat/broadcasts` mit einem Ziel-`kind` gleich `"legacy"` aufruft
- **THEN** antwortet der Server mit HTTP 400
