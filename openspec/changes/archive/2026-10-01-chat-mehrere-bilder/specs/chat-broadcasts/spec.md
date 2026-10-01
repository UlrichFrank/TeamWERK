## MODIFIED Requirements

### Requirement: Empfangene Broadcasts abrufen

Das System SHALL die sichtbaren Broadcasts eines Users zurückgeben. Zu jedem Broadcast werden geliefert: `id`, `senderName`, `body`, `media` (Liste der Bilder in Album-Reihenfolge, leer ohne Bild; je Eintrag `id`, `url = "/media/<id>"`, `width`/`height` nur bei bekannter Dimension, sonst weggelassen), `mediaId`/`mediaUrl`/`mediaWidth`/`mediaHeight` (Angaben zum **ersten** Bild, Kompatibilität für ältere Clients; null bzw. weggelassen ohne Bild), `sentAt`, `isRead`, `isSent`, `editedAt`.

#### Scenario: Broadcast mit mehreren Bildern

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und ein Broadcast drei Bilder A, B, C trägt
- **THEN** enthält `media` genau A, B, C in dieser Reihenfolge, und `mediaId`/`mediaUrl` bezeichnen A

#### Scenario: Broadcast mit Bild und bekannten Dimensionen

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und ein Broadcast ein Bild trägt, dessen `media`-Zeile `width=800`, `height=600` hat
- **THEN** enthält der `media`-Eintrag `width=800`, `height=600`, und das Broadcast-Objekt `mediaId`, `mediaUrl = "/media/<mediaId>"`, `mediaWidth=800`, `mediaHeight=600`

#### Scenario: Broadcast mit Bild ohne bekannte Dimensionen

- **WHEN** ein Broadcast mit einem Bild abgerufen wird, dessen `media`-Zeile `width IS NULL` hat
- **THEN** fehlen `width`/`height` im `media`-Eintrag sowie `mediaWidth`/`mediaHeight` im Broadcast-Objekt; `mediaId` und `mediaUrl` sind gesetzt

#### Scenario: Bestands-Broadcast mit Einzelbild

- **WHEN** ein vor Einführung der Alben gesendeter Broadcast mit einem Bild abgerufen wird
- **THEN** enthält `media` genau diesen einen Eintrag

#### Scenario: Broadcast ohne Bild abrufen

- **WHEN** ein Broadcast ohne Bild abgerufen wird
- **THEN** ist `media` eine leere Liste, `mediaId` und `mediaUrl` sind null; `mediaWidth`/`mediaHeight` fehlen

### Requirement: Mitteilung an vereinsweite Ziele und Team-Gruppen senden

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

Der Request KANN Bilder enthalten, als `mediaIds` (Liste mit 1 bis 10 media-IDs,
Reihenfolge = Album-Reihenfolge) oder als `mediaId` (Kurzform für genau ein Bild); beide
Felder zugleich SHALL mit HTTP 400 abgelehnt werden. Mindestens `body` (nicht leer)
**oder** ein Bild MUSS vorhanden sein. Jede übergebene media-ID MUSS auf eine existierende
`media`-Zeile verweisen, die der Absender selbst hochgeladen hat und die noch keiner
Nachricht und keiner anderen Mitteilung zugeordnet ist; dieselbe ID darf nicht doppelt
vorkommen. Verletzt der Request eine dieser Bedingungen oder enthält er mehr als 10
Bilder, SHALL der Server mit HTTP 400 antworten und keinen Broadcast speichern. Die
Push-Vorschau einer Mitteilung ohne Text lautet „Bild" bei einem und „N Bilder" bei
N > 1 Bildern; je Empfänger geht genau eine Push hinaus.

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

- **WHEN** ein Vorstand `POST /api/chat/broadcasts` mit `{ "body": "", "mediaId": <id>, "targets": [{"kind": "users"}] }` aufruft und das Bild selbst hochgeladen hat
- **THEN** wird der Broadcast mit diesem einen Bild und leerem Body gespeichert, HTTP 201

#### Scenario: Mitteilung mit mehreren Bildern senden

- **WHEN** ein Vorstand `{ "body": "Saisonabschluss", "mediaIds": [a, b, c, d], "targets": [{"kind": "users"}] }` sendet und alle Bilder selbst hochgeladen hat
- **THEN** wird **ein** Broadcast mit vier Bildern in der Reihenfolge a, b, c, d gespeichert, HTTP 201, und je Empfänger genau eine Push gesendet

#### Scenario: Mitteilung mit mehr als zehn Bildern wird abgelehnt

- **WHEN** ein berechtigter User einen Broadcast mit 11 media-IDs sendet
- **THEN** antwortet der Server mit HTTP 400 und speichert keinen Broadcast

#### Scenario: Mitteilung mit fremdem Bild wird abgelehnt

- **WHEN** ein berechtigter User eine media-ID sendet, die ein anderer Nutzer hochgeladen hat oder die bereits an einer Nachricht oder Mitteilung hängt
- **THEN** antwortet der Server mit HTTP 400 und speichert keinen Broadcast

#### Scenario: Leere Mitteilung ohne Bild wird abgelehnt

- **WHEN** ein berechtigter User einen Broadcast mit leerem `body` und ohne Bild sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Unberechtigter User

- **WHEN** ein User ohne `admin`/`vorstand`/`sportliche_leitung` und ohne Kader-Trainer-Eintrag der aktiven Saison einen Broadcast sendet
- **THEN** antwortet der Server mit HTTP 403
