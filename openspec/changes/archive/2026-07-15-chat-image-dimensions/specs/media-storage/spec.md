## MODIFIED Requirements

### Requirement: Bild hochladen

Das System SHALL einen Endpunkt `POST /api/media/upload` bereitstellen, der eine einzelne Bilddatei via `multipart/form-data` (Feld `image`) entgegennimmt, unter `<MEDIA_DIR>/<uuid>.<ext>` speichert, einen Datensatz in der `media`-Tabelle (`disk_name`, `mime_type`, `size`, `uploaded_by`, `width`, `height`) anlegt und `{ "mediaId": <id>, "url": "/media/<id>", "width": <int>, "height": <int> }` zurückgibt. Die zurückgegebene `url` trägt **kein** `/api`-Prefix. Erlaubte MIME-Types: `image/jpeg`, `image/png`, `image/gif`, `image/webp`. Maximale Dateigröße: 1 MB. Nur authentifizierte User dürfen hochladen. Der Server MUSS nach der MIME-Prüfung die Bild-Dimensionen per Header-Probe (ohne Full-Decode) bestimmen und mitschreiben. Scheitert die Probe (z. B. korrupter Header), wird die Datei dennoch akzeptiert; `width`/`height` bleiben in der DB NULL und werden in der Response weggelassen.

#### Scenario: Erfolgreiches Bild-Upload mit Dimensionen

- **WHEN** ein authentifizierter User `POST /api/media/upload` mit einer gültigen JPEG-Datei (≤ 1 MB, 1920×1080 px) aufruft
- **THEN** speichert der Server die Datei unter `<MEDIA_DIR>/<uuid>.jpg`, legt eine `media`-Zeile mit `width=1920`, `height=1080` an und antwortet mit HTTP 200/201 und `{ "mediaId": <id>, "url": "/media/<id>", "width": 1920, "height": 1080 }`

#### Scenario: Upload eines Bildes mit unlesbarem Header

- **WHEN** ein User eine Datei mit gültigem MIME-Type aber beschädigtem Header hochlädt
- **THEN** speichert der Server die Datei, legt eine `media`-Zeile mit `width=NULL`, `height=NULL` an und antwortet mit HTTP 200/201 und `{ "mediaId": <id>, "url": "/media/<id>" }` (ohne width/height-Felder)

#### Scenario: Ungültiger MIME-Type

- **WHEN** ein User eine PDF-Datei hochlädt
- **THEN** antwortet der Server mit HTTP 400 und legt weder Datei noch `media`-Zeile an

#### Scenario: Datei zu groß

- **WHEN** ein User eine Bilddatei > 1 MB hochlädt
- **THEN** antwortet der Server mit HTTP 413

#### Scenario: Nicht authentifiziert

- **WHEN** ein nicht eingeloggter User den Upload-Endpunkt aufruft
- **THEN** antwortet der Server mit HTTP 401

## ADDED Requirements

### Requirement: Bild-Dimensionen für Bestandsbilder nachtragen

Das System SHALL beim Start des Server-Prozesses einen einmaligen, idempotenten Backfill starten, der alle `media`-Zeilen mit `width IS NULL` durchgeht, die zugehörige Datei aus `<MEDIA_DIR>` liest, die Dimensionen per Header-Probe bestimmt und in die DB schreibt. Der Backfill läuft als Goroutine (nicht-blockierend für den HTTP-Server), sequentiell (kein Parallelismus, VPS-Speicher schonen), loggt Start, Anzahl bearbeiteter Zeilen und Ende. Fehler pro Datei (fehlende Datei, korrupter Header, WEBP-Decode-Fehler) MÜSSEN geloggt und übersprungen werden — der Backfill darf nicht abbrechen. Beim nächsten Serverstart ist der Backfill ein No-Op (keine passenden Zeilen mehr).

#### Scenario: Bestandsbild ohne Dimensionen wird nachgetragen

- **WHEN** der Server startet und in `media` existieren Zeilen mit `width IS NULL`, deren Dateien auf Disk vorhanden und lesbar sind
- **THEN** füllt der Backfill für jede dieser Zeilen `width` und `height` per `UPDATE`

#### Scenario: Fehlende Datei bricht den Backfill nicht

- **WHEN** eine `media`-Zeile mit `width IS NULL` existiert, deren Datei auf Disk fehlt
- **THEN** loggt der Backfill den Fehler, lässt `width`/`height` NULL und macht mit der nächsten Zeile weiter

#### Scenario: Backfill nach vollständigem Lauf ist No-Op

- **WHEN** der Server erneut startet und alle `media`-Zeilen bereits `width IS NOT NULL` haben
- **THEN** loggt der Backfill „nichts zu tun" und beendet sich sofort ohne Datei-I/O
