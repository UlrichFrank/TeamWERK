# chat-broadcasts Specification

## Purpose
Einweg-Mitteilungen an Zielgruppen (alle, Team, Rolle). Sender kann Broadcasts bearbeiten und löschen.
## Requirements
### Requirement: Broadcast senden

Das System SHALL es Usern mit Rolle `admin` oder einer der Vereinsfunktionen `vorstand`
bzw. `sportliche_leitung` erlauben, eine Mitteilung an eine vereinsweite Zielgruppe zu
senden. Alle drei Personas SHALL dieselben Zielgruppen wählen dürfen; es gibt keine
engere zweite Stufe. Alle übrigen User — insbesondere `trainer` ohne eine der genannten
Funktionen — SHALL mit HTTP 403 abgewiesen werden.

`targetType` SHALL genau einen der folgenden vier Werte tragen; jeder andere Wert,
insbesondere die früheren `all`, `team` und `role`, SHALL mit HTTP 400 abgelehnt werden:

| `targetType` | Empfängermenge |
|---|---|
| `users` | alle Zeilen in `users` |
| `members` | alle User, zu denen eine `members`-Zeile mit `user_id` existiert |
| `spieler` | alle User, deren Mitglied die Vereinsfunktion `spieler` trägt |
| `eltern` | alle `family_links.parent_user_id` (distinkt) |

Die Auflösung SHALL **keine Vereinsfunktionen über `family_links` vererben**: ein
Elternteil ohne eigene Vereinsfunktion `spieler` gehört nicht zur Zielgruppe `spieler`,
auch wenn sein Kind sie trägt. Damit sind `spieler` und `eltern` disjunkt auflösbar —
abweichend von `folder_permissions`, wo `club_function`-Einträge auf Eltern durchschlagen.

Jeder Empfänger SHALL genau eine `broadcast_reads`-Zeile erhalten, auch bei
Mehrfachzugehörigkeit (Elternteil mehrerer Kinder, Mitglied mit mehreren
Vereinsfunktionen). Mitglieder ohne verknüpften User-Account (`members.user_id IS NULL`)
SHALL in keiner Zielgruppe auftauchen.

Der Request KANN optional `mediaId` enthalten; mindestens `body` (nicht leer) **oder**
`mediaId` MUSS vorhanden sein. Ein angegebenes `mediaId` MUSS auf eine existierende
`media`-Zeile verweisen.

Die Antwort SHALL HTTP 201 mit `{ "id": <broadcastId>, "recipients": <n> }` sein, wobei
`n` die Anzahl der benachrichtigten Empfänger **ohne den Absender** ist. Der Absender
SHALL eine `broadcast_reads`-Zeile mit gesetztem `read_at` erhalten, aber weder SSE noch
Push, und SHALL nicht in `recipients` gezählt werden. Alle übrigen Empfänger SHALL ein
SSE-Event `chat:new-broadcast` und eine Push-Benachrichtigung erhalten.

#### Scenario: Admin sendet Text-Broadcast an alle

- **WHEN** ein Admin `POST /api/chat/broadcasts` mit `{ "body": "Wichtige Info", "targetType": "users" }` aufruft
- **THEN** wird der Broadcast gespeichert, alle User außer dem Absender erhalten ein SSE-Event `chat:new-broadcast`, HTTP 201
- **AND** die Antwort enthält `recipients` gleich der Anzahl dieser User

#### Scenario: Vorstand sendet an alle Spieler

- **WHEN** ein Vorstand `POST /api/chat/broadcasts` mit `{ "body": "Trainingsauftakt", "targetType": "spieler" }` aufruft
- **THEN** antwortet der Server mit HTTP 201 und `recipients` gleich der Anzahl der User mit Vereinsfunktion `spieler` (ohne den Absender)
- **AND** genau diese User haben eine `broadcast_reads`-Zeile

#### Scenario: Sportliche Leitung darf jede Zielgruppe

- **WHEN** ein User mit Vereinsfunktion `sportliche_leitung` (ohne `vorstand`, ohne `admin`) `POST /api/chat/broadcasts` mit `{ "body": "Info", "targetType": "users" }` aufruft
- **THEN** antwortet der Server mit HTTP 201

#### Scenario: Eltern erben die Vereinsfunktion ihres Kindes nicht

- **WHEN** ein Elternteil ohne eigene Vereinsfunktion ein Kind mit Vereinsfunktion `spieler` hat und eine Mitteilung mit `targetType: "spieler"` gesendet wird
- **THEN** erhält das Elternteil **keine** `broadcast_reads`-Zeile
- **AND** bei `targetType: "eltern"` erhält es genau eine

#### Scenario: Elternteil mehrerer Kinder wird einmal gezählt

- **WHEN** ein Elternteil über `family_links` mit zwei Kindern verknüpft ist und eine Mitteilung mit `targetType: "eltern"` gesendet wird
- **THEN** existiert für dieses Elternteil genau eine `broadcast_reads`-Zeile
- **AND** `recipients` zählt es einmal

#### Scenario: Mitglied ohne Account wird nicht erreicht

- **WHEN** ein Mitglied mit Vereinsfunktion `spieler` und `members.user_id IS NULL` existiert und eine Mitteilung mit `targetType: "spieler"` gesendet wird
- **THEN** entsteht für dieses Mitglied keine `broadcast_reads`-Zeile
- **AND** `recipients` zählt es nicht mit

#### Scenario: Trainer darf nicht mehr senden

- **WHEN** ein User mit Vereinsfunktion `trainer` (ohne `vorstand`, ohne `sportliche_leitung`, ohne `admin`) `POST /api/chat/broadcasts` mit beliebiger Zielgruppe aufruft
- **THEN** antwortet der Server mit HTTP 403
- **AND** es wird kein Broadcast gespeichert

#### Scenario: Alte Zielgruppen-Werte werden abgelehnt

- **WHEN** ein Vorstand `POST /api/chat/broadcasts` mit `targetType` gleich `"all"`, `"team"` oder `"role"` aufruft
- **THEN** antwortet der Server mit HTTP 400
- **AND** es wird kein Broadcast gespeichert

#### Scenario: Fehlende Zielgruppe

- **WHEN** ein berechtigter User einen Broadcast ohne `targetType` oder mit unbekanntem Wert sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Reine Bild-Mitteilung senden

- **WHEN** ein Vorstand `POST /api/chat/broadcasts` mit `{ "body": "", "mediaId": <id>, "targetType": "users" }` aufruft
- **THEN** wird der Broadcast mit `media_id` und leerem Body gespeichert, HTTP 201

#### Scenario: Leere Mitteilung ohne Bild wird abgelehnt

- **WHEN** ein berechtigter User einen Broadcast mit leerem `body` und ohne `mediaId` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Unberechtigter User

- **WHEN** ein User ohne `admin`/`vorstand`/`sportliche_leitung` einen Broadcast sendet
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Empfangene Broadcasts abrufen

Das System SHALL die sichtbaren Broadcasts eines Users zurückgeben. Zu jedem Broadcast werden geliefert: `id`, `senderName`, `body`, `mediaId` (null wenn kein Bild), `mediaUrl` (null wenn kein Bild; sonst `"/media/<mediaId>"`), `mediaWidth` (nur bei Bild-Broadcasts mit bekannter Dimension; sonst weggelassen), `mediaHeight` (nur bei Bild-Broadcasts mit bekannter Dimension; sonst weggelassen), `sentAt`, `isRead`, `isSent`, `editedAt`.

#### Scenario: Broadcast mit Bild und bekannten Dimensionen

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und ein Broadcast `media_id` gesetzt hat, dessen `media`-Zeile `width=800`, `height=600` hat
- **THEN** enthält das Broadcast-Objekt `mediaId`, `mediaUrl = "/media/<mediaId>"`, `mediaWidth=800`, `mediaHeight=600`

#### Scenario: Broadcast mit Bild ohne bekannte Dimensionen

- **WHEN** ein Broadcast mit `media_id` abgerufen wird, dessen `media`-Zeile `width IS NULL` hat
- **THEN** enthält das Broadcast-Objekt `mediaId`, `mediaUrl`; `mediaWidth` und `mediaHeight` fehlen im JSON-Objekt

#### Scenario: Broadcast ohne Bild abrufen

- **WHEN** ein Broadcast ohne `media_id` abgerufen wird
- **THEN** sind `mediaId` und `mediaUrl` beide null; `mediaWidth`/`mediaHeight` fehlen

### Requirement: Broadcast als gelesen markieren

Das System SHALL es Empfängern erlauben einen Broadcast als gelesen zu markieren. Dies beeinflusst den Ungelesen-Badge im Nav.

#### Scenario: Broadcast öffnen markiert als gelesen

- **WHEN** ein User einen Broadcast öffnet und `POST /api/chat/broadcasts/{id}/read` aufruft
- **THEN** wird `broadcast_reads.read_at` für diesen User gesetzt
- **THEN** erscheint der Broadcast als gelesen in der Liste

### Requirement: Kein Rückkanal bei Broadcasts

Das System SHALL keinerlei Reply-Funktionalität für Broadcasts bereitstellen. Der Endpoint zur Konversationserstellung (`POST /api/chat/conversations`) darf nicht über einen Broadcast-Kontext erreichbar sein. Im Frontend wird kein Reply-Eingabefeld angezeigt.

#### Scenario: Kein Reply-Endpoint für Broadcasts

- **WHEN** ein User versucht auf einen Broadcast zu antworten
- **THEN** existiert kein API-Endpoint für diese Aktion (HTTP 404 oder nicht vorhanden)

### Requirement: Bestands-Mitteilungen bleiben zustellbar

Das System SHALL Mitteilungen, die vor der Umstellung des Zielgruppen-Vokabulars gesendet
wurden, für ihre damaligen Empfänger unverändert auslieferbar halten. Die Zustellung SHALL
ausschließlich an `broadcast_reads` hängen; `broadcasts.target_type` SHALL nach dem Senden
nirgends mehr ausgewertet werden.

Bestandswerte SHALL wie folgt abgebildet werden: `all` → `users`, `team` und `role` →
`legacy`. Der Wert `legacy` SHALL persistierbar und lesbar, aber über
`POST /api/chat/broadcasts` **nicht** setzbar sein.

#### Scenario: Alte Mitteilung bleibt sichtbar

- **WHEN** ein User `GET /api/chat/broadcasts` aufruft und für ihn eine `broadcast_reads`-Zeile zu einer vor der Migration gesendeten Mitteilung existiert
- **THEN** enthält die Antwort diese Mitteilung unverändert (Body, Sender, Zeitpunkt, Lesestatus)

#### Scenario: legacy ist nicht schreibbar

- **WHEN** ein berechtigter User `POST /api/chat/broadcasts` mit `targetType: "legacy"` aufruft
- **THEN** antwortet der Server mit HTTP 400

