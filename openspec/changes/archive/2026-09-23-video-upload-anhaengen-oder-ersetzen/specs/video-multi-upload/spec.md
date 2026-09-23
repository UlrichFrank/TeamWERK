## MODIFIED Requirements

### Requirement: Mehrere Videos pro Termin/Spiel

Das System SHALL beliebig viele Videos zum selben Spiel (`game_id`) oder mit demselben
Titel/Namen erlauben. Ein neuer Upload SHALL ein bestehendes Video niemals automatisch
ersetzen, überschreiben oder dessen Daten übernehmen — das gilt weiterhin uneingeschränkt
für jeden Aufruf von `POST /api/videos`.

Ein Client (z. B. der Video-Encoder) DARF dem Nutzer jedoch eine **ausdrückliche,
aktiv zu wählende** Option „Ersetzen" anbieten: dabei bleibt der neue Upload technisch ein
ganz gewöhnlicher `POST /api/videos` ohne jede Sonderbehandlung; der Client SHALL im
Anschluss an einen bereits erfolgreich abgeschlossenen Upload eigenständig
`DELETE /api/videos/{id}` für die zuvor vorhandenen Videos desselben Spiels aufrufen. Das
System selbst SHALL diese beiden Schritte niemals verknüpfen oder automatisieren — die
Reihenfolge und die Entscheidung, ob überhaupt gelöscht wird, bleibt vollständig beim
aufrufenden Client.

#### Scenario: Zweiter Upload zum selben Spiel

- **WHEN** ein Nutzer nacheinander zwei Videos für dasselbe Spiel hochlädt
- **THEN** existieren nach beiden Uploads zwei separate Video-Zeilen mit demselben
  `game_id`, beide mit eigenen Dateien und eigenem Status

#### Scenario: Zweiter Upload derselben Datei

- **WHEN** ein Nutzer dieselbe Videodatei ein zweites Mal über den Button „Hochladen"
  hochlädt
- **THEN** wird eine neue tus-Session für die neu angelegte `video_id` gestartet und die
  zuvor angelegte Video-Zeile bleibt unangetastet

#### Scenario: `POST /api/videos` kennt keine Ersetzen-Option

- **WHEN** ein Client `POST /api/videos` mit einer `game_id` aufruft, zu der bereits
  Videos existieren
- **THEN** legt der Server ausschließlich eine neue, eigenständige Video-Zeile an; er
  löscht, ändert oder markiert keine der bestehenden Zeilen — unabhängig davon, ob der
  Client anschließend selbst welche löscht

#### Scenario: Client-gesteuertes Ersetzen bleibt zwei getrennte Aufrufe

- **WHEN** ein Nutzer im Video-Encoder-Tool „Ersetzen" wählt und der neue Upload
  erfolgreich abgeschlossen ist
- **THEN** ruft der Client danach eigenständig `DELETE /api/videos/{id}` für jedes zuvor
  vorhandene Video auf; der Server sieht darin zwei unabhängige, bereits bestehende
  Operationen und keinen neuen kombinierten Vorgang
