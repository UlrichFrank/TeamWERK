## ADDED Requirements

### Requirement: Mehrere Videos pro Termin/Spiel

Das System SHALL beliebig viele Videos zum selben Spiel (`game_id`) oder mit demselben
Titel/Namen erlauben. Ein neuer Upload SHALL niemals ein bestehendes Video ersetzen,
überschreiben oder dessen Daten übernehmen.

#### Scenario: Zweiter Upload zum selben Spiel

- **WHEN** ein Nutzer nacheinander zwei Videos für dasselbe Spiel hochlädt
- **THEN** existieren nach beiden Uploads zwei separate Video-Zeilen mit demselben
  `game_id`, beide mit eigenen Dateien und eigenem Status

#### Scenario: Zweiter Upload derselben Datei

- **WHEN** ein Nutzer dieselbe Videodatei ein zweites Mal über den Button „Hochladen"
  hochlädt
- **THEN** wird eine neue tus-Session für die neu angelegte `video_id` gestartet und die
  zuvor angelegte Video-Zeile bleibt unangetastet

### Requirement: Frischer Upload resumt keine fremde Session

Der Button „Hochladen" (frischer Upload) SHALL für die neu per `POST /api/videos` angelegte
`video_id` immer eine neue tus-Session starten und NICHT `resumeFromPreviousUpload`
verwenden. Nur der explizite Button „Upload fortsetzen" SHALL eine vorhandene Session
fortsetzen, wobei er die ursprüngliche `video_id` aus den Session-Metadaten behält.

#### Scenario: Frischer Upload trotz vorhandener Resume-Session

- **WHEN** für die gewählte Datei eine unterbrochene frühere tus-Session vorliegt und der
  Nutzer auf „Hochladen" (statt „Upload fortsetzen") klickt
- **THEN** wird die frühere Session ignoriert und ein neuer Upload für die neue `video_id`
  gestartet, sodass keine fremde Video-Zeile bespielt wird

#### Scenario: Upload fortsetzen behält video_id

- **WHEN** der Nutzer für eine unterbrochene Datei „Upload fortsetzen" klickt
- **THEN** wird die frühere Session mit ihrer ursprünglichen `video_id` fortgesetzt und
  keine neue Video-Zeile angelegt

### Requirement: Spiel-Zuordnung eines Videos änderbar

Berechtigte Nutzer (Trainer/sportliche Leitung/Vorstand des Teams bzw. Admin) SHALL die
Spiel-Zuordnung eines bestehenden Videos über `PATCH /api/videos/{id}` ändern oder
entfernen können. Das Feld `game_id` SHALL Tri-State sein: fehlt = unverändert, `null` =
Zuordnung entfernen, Zahl = auf dieses Spiel setzen. Das Bearbeiten-Modal SHALL einen
Spiel-Selector inklusive Option „Kein Spiel zuordnen" anbieten.

#### Scenario: Zuordnung setzen (Happy Path)

- **WHEN** ein berechtigter Nutzer `PATCH /api/videos/{id}` mit `game_id` = gültige Spiel-ID sendet
- **THEN** antwortet der Server mit 200 und `videos.game_id` ist auf diese ID gesetzt

#### Scenario: Zuordnung entfernen

- **WHEN** ein berechtigter Nutzer `PATCH /api/videos/{id}` mit `game_id` = `null` sendet
- **THEN** antwortet der Server mit 200 und `videos.game_id` ist `NULL`

#### Scenario: Zuordnung unverändert lassen

- **WHEN** ein `PATCH /api/videos/{id}` ohne das Feld `game_id` gesendet wird
- **THEN** bleibt der bestehende `game_id`-Wert unverändert

#### Scenario: Ohne Berechtigung

- **WHEN** ein Nutzer ohne Verwaltungsrecht für das Team `PATCH /api/videos/{id}` mit `game_id` sendet
- **THEN** antwortet der Server mit 403 und `videos.game_id` bleibt unverändert

### Requirement: Bereinigung hängengebliebener Uploads

Das System SHALL Videos, die länger als 24 Stunden im Status `uploading` verharren, per
Scheduler auf `status='failed'` mit `failure_reason` „Upload abgebrochen" setzen, damit
keine dauerhaften Geister-Einträge „Wird hochgeladen" entstehen. Videos, die den Upload
regulär abgeschlossen haben (`queued`/`processing`/`ready`), SHALL der Job nicht anfassen.

#### Scenario: Alte uploading-Zeile wird als fehlgeschlagen markiert

- **WHEN** der Scheduler-Tick läuft und ein Video seit mehr als 24 h `status='uploading'` hat
- **THEN** wird dessen Status auf `failed` mit `failure_reason` „Upload abgebrochen" gesetzt

#### Scenario: Frische uploading-Zeile bleibt unangetastet

- **WHEN** ein Video erst vor wenigen Minuten angelegt wurde und noch `status='uploading'` hat
- **THEN** lässt der Scheduler-Job es unverändert
