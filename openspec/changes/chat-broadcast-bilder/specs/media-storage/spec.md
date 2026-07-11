## ADDED Requirements

### Requirement: Bild hochladen

Das System SHALL einen Endpunkt `POST /api/media/upload` bereitstellen, der eine einzelne Bilddatei via `multipart/form-data` (Feld `image`) entgegennimmt, unter `<MEDIA_DIR>/<uuid>.<ext>` speichert, einen Datensatz in der `media`-Tabelle (`disk_name`, `mime_type`, `size`, `uploaded_by`) anlegt und `{ "mediaId": <id>, "url": "/media/<id>" }` zurückgibt. Die zurückgegebene `url` trägt **kein** `/api`-Prefix. Erlaubte MIME-Types: `image/jpeg`, `image/png`, `image/gif`, `image/webp`. Maximale Dateigröße: 1 MB. Nur authentifizierte User dürfen hochladen.

#### Scenario: Erfolgreiches Bild-Upload

- **WHEN** ein authentifizierter User `POST /api/media/upload` mit einer gültigen JPEG-Datei (≤ 1 MB) aufruft
- **THEN** speichert der Server die Datei unter `<MEDIA_DIR>/<uuid>.jpg`, legt eine `media`-Zeile an
- **THEN** antwortet der Server mit HTTP 200/201 und `{ "mediaId": <id>, "url": "/media/<id>" }`

#### Scenario: Ungültiger MIME-Type

- **WHEN** ein User eine PDF-Datei hochlädt
- **THEN** antwortet der Server mit HTTP 400 und legt weder Datei noch `media`-Zeile an

#### Scenario: Datei zu groß

- **WHEN** ein User eine Bilddatei > 1 MB hochlädt
- **THEN** antwortet der Server mit HTTP 413

#### Scenario: Nicht authentifiziert

- **WHEN** ein nicht eingeloggter User den Upload-Endpunkt aufruft
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Bild abrufen

Das System SHALL Bilder unter `GET /api/media/{id}` ausliefern. Nur authentifizierte User dürfen Bilder abrufen. Der Server MUSS den in `media.mime_type` gespeicherten `Content-Type` sowie `X-Content-Type-Options: nosniff` setzen.

#### Scenario: Bild erfolgreich abrufen

- **WHEN** ein authentifizierter User `GET /api/media/{id}` aufruft und die Zeile + Datei existieren
- **THEN** sendet der Server die Bild-Bytes mit dem korrekten `Content-Type`

#### Scenario: Bild nicht gefunden

- **WHEN** ein User eine unbekannte `id` abruft
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Nicht authentifiziert

- **WHEN** ein nicht eingeloggter User ein Bild abruft
- **THEN** antwortet der Server mit HTTP 401
