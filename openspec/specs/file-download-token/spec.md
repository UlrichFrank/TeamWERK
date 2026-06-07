# file-download-token Specification

## Purpose
Kurzlebige, signierte Download-Tokens ermöglichen es, Dateien direkt per URL zu öffnen — ohne Authorization-Header. Damit können Browser und native Viewer (PDF, Bilder) Dateien direkt über `window.open()` laden, auch in iOS-PWA-Standalone-Umgebungen, in denen kein Fetch mit Authorization-Header möglich ist.

## Requirements

### Requirement: Download-Token ausstellen

Das System MUSS authentifizierten Nutzern erlauben, ein kurzlebiges Download-Token für eine Datei auszustellen, auf die sie Lesezugriff haben. Das Token MUSS HMAC-SHA256-signiert sein (Schlüssel: `JWT_SECRET`), eine TTL von 5 Minuten haben und die File-ID sowie User-ID einschließen. Es DARF kein DB-Eintrag erzeugt werden.

Endpoint: `GET /api/files/{id}/download-token`
Response: `{ "token": "<base64url_payload>.<base64url_sig>" }`

#### Scenario: Token für lesbare Datei
- **WHEN** ein authentifizierter Nutzer `GET /api/files/42/download-token` aufruft und Lesezugriff auf den Ordner hat
- **THEN** antwortet der Server mit HTTP 200 und einem Token-String

#### Scenario: Kein Zugriff
- **WHEN** ein Nutzer `GET /api/files/42/download-token` aufruft, aber keinen Lesezugriff auf den Ordner hat
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Datei existiert nicht
- **WHEN** ein Nutzer ein Token für eine nicht existierende File-ID anfragt
- **THEN** antwortet der Server mit HTTP 404

### Requirement: Datei per Token herunterladen

Der Download-Endpoint `GET /api/files/{id}/download` MUSS zusätzlich zum Authorization-Header auch einen `?token=`-Query-Parameter akzeptieren. Das Token MUSS auf gültige Signatur, Ablauf (exp), und Übereinstimmung mit der URL-File-ID geprüft werden. Bei gültigem Token MUSS die Berechtigungsprüfung (Lesezugriff auf Ordner) erneut durchgeführt werden.

#### Scenario: Gültiges Token
- **WHEN** ein Browser `GET /api/files/42/download?token=<valid_token>` aufruft
- **THEN** antwortet der Server mit HTTP 200, dem Dateiinhalt und korrektem `Content-Type`-Header

#### Scenario: Abgelaufenes Token
- **WHEN** ein Browser ein Token verwendet, das älter als 5 Minuten ist
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Manipuliertes Token
- **WHEN** ein Browser ein Token mit ungültiger Signatur übergibt
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Token-File-ID stimmt nicht mit URL überein
- **WHEN** ein Browser ein gültiges Token für File 42 verwendet, aber `GET /api/files/99/download?token=...` aufruft
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Berechtigungen nach Token-Ausstellung entzogen
- **WHEN** einem Nutzer nach Token-Ausstellung der Lesezugriff entzogen wird und er dann mit dem Token herunterlädt
- **THEN** antwortet der Server mit HTTP 403
