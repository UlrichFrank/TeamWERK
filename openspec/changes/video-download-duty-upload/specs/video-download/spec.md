## Purpose

Erlaubt berechtigten Nutzern, ein fertig verarbeitetes Video als Datei herunterzuladen statt es nur zu streamen — für Offline-Archiv oder externe Nachbearbeitung.

## ADDED Requirements

### Requirement: Video-Download mit identischer Berechtigung wie Streaming

Der Server SHALL einen authentifizierten Endpoint bereitstellen, der ein fertig verarbeitetes Video (`status='ready'`) als MP4-Datei zum Download liefert. Die Berechtigung SHALL exakt derselben Prüfung folgen wie beim Streaming (`CanViewVideo`) — keine zusätzliche oder abweichende Zielgruppe. Die Datei MUST aus den vorhandenen 720p-HLS-Renditionssegmenten des Videos erzeugt werden (kein separat gespeichertes Original mehr vorhanden). Die Antwort MUST `Content-Disposition: attachment` mit einem aus dem Video-Titel abgeleiteten Dateinamen tragen.

#### Scenario: Berechtigter Nutzer lädt ein fertiges Video herunter
- **WHEN** ein Nutzer, der das Video laut `CanViewVideo` ansehen darf, den Download-Endpoint für ein Video mit `status='ready'` aufruft
- **THEN** liefert der Server HTTP 200 mit `Content-Type: video/mp4`, `Content-Disposition: attachment; filename="..."` und dem vollständigen, abspielbaren Videoinhalt

#### Scenario: Nicht berechtigter Nutzer
- **WHEN** ein Nutzer, der das Video laut `CanViewVideo` nicht ansehen darf, den Download-Endpoint aufruft
- **THEN** antwortet der Server mit demselben Statuscode, den `GET /api/videos/{id}` für diesen Nutzer liefern würde (403/404, Existenz nicht erratbar)

#### Scenario: Video noch nicht verarbeitet
- **WHEN** ein berechtigter Nutzer den Download-Endpoint für ein Video mit `status` ungleich `ready` aufruft
- **THEN** antwortet der Server mit HTTP 409 ohne einen Remux-Versuch zu starten

#### Scenario: Remux schlägt fehl
- **WHEN** der ffmpeg-Remux-Prozess für ein an sich abrufbares Video fehlschlägt (z. B. beschädigte Segmente)
- **THEN** antwortet der Server mit HTTP 500, protokolliert den Fehler serverseitig und lässt keine unvollständige Datei-Antwort an den Client durch

### Requirement: Kein dauerhaftes Caching der Download-Datei

Der Server SHALL die für den Download erzeugte MP4-Datei nicht dauerhaft auf Platte vorhalten. Jeder Download-Request SHALL die Datei neu aus den vorhandenen HLS-Segmenten erzeugen und direkt an die Response streamen, ohne zusätzlichen dauerhaften Speicherbedarf pro Video zu erzeugen.

#### Scenario: Zwei aufeinanderfolgende Downloads desselben Videos
- **WHEN** derselbe Nutzer denselben Download-Endpoint zweimal nacheinander aufruft
- **THEN** wird die MP4-Datei beim zweiten Aufruf erneut aus den HLS-Segmenten erzeugt, und es bleibt keine zusätzliche Download-Datei dauerhaft auf Platte liegen
