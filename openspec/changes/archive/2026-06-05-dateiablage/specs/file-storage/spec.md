## ADDED Requirements

### Requirement: Datei hochladen
Authentifizierte Nutzer mit `can_write` auf den Zielordner SOLLEN Dateien via `POST /api/folders/:folderId/files` hochladen können. Der Request MUSS `multipart/form-data` verwenden. Die maximale Dateigröße beträgt 50 MB. Auf Disk wird die Datei unter einem UUID-basierten Namen gespeichert; der Original-Name wird in der DB gehalten.

#### Scenario: Erfolgreicher Upload
- **WHEN** ein Nutzer mit `can_write` auf den Ordner eine Datei hochlädt
- **THEN** speichert der Server die Datei unter `/var/lib/teamwerk/files/<uuid>.<ext>` und legt einen Eintrag in `files` an

#### Scenario: Upload ohne Schreibrecht
- **WHEN** ein Nutzer ohne `can_write` auf den Zielordner eine Datei hochlädt
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Datei zu groß
- **WHEN** eine Datei größer als 50 MB hochgeladen wird
- **THEN** antwortet der Server mit HTTP 413

### Requirement: Datei herunterladen
Authentifizierte Nutzer mit `can_read` auf den enthaltenden Ordner SOLLEN Dateien via `GET /api/files/:id/download` herunterladen können. Der Server MUSS `Content-Disposition: attachment; filename="<original_name>"` setzen.

#### Scenario: Erfolgreicher Download
- **WHEN** ein Nutzer mit `can_read` `GET /api/files/:id/download` aufruft
- **THEN** streamt der Server die Datei-Bytes mit korrektem `Content-Disposition`-Header

#### Scenario: Download ohne Leserecht
- **WHEN** ein Nutzer ohne `can_read` auf den Ordner der Datei zugreift
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Datei nicht gefunden
- **WHEN** eine nicht existierende `id` referenziert wird
- **THEN** antwortet der Server mit HTTP 404

### Requirement: Ordnerinhalt auflisten
Authentifizierte Nutzer mit `can_read` SOLLEN via `GET /api/folders/:id/contents` Unterordner und Dateien auflisten können. Die Antwort MUSS `folders` (id, name, has_children) und `files` (id, name, size, mime_type, uploaded_by_name, created_at) enthalten.

#### Scenario: Erfolgreiche Auflistung
- **WHEN** ein Nutzer mit `can_read` den Inhalt eines Ordners abruft
- **THEN** erhält er Unterordner und Dateien des direkt angefragten Ordners

#### Scenario: Kein Leserecht
- **WHEN** ein Nutzer ohne `can_read` den Inhalt abruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Datei löschen
Nutzer mit `can_write` auf den enthaltenden Ordner SOLLEN Dateien via `DELETE /api/files/:id` löschen können. Admin darf immer löschen. DB-Eintrag und Datei auf Disk werden entfernt.

#### Scenario: Erfolgreiche Löschung
- **WHEN** ein Nutzer mit `can_write` `DELETE /api/files/:id` aufruft
- **THEN** löscht der Server den DB-Eintrag und die Datei auf Disk

#### Scenario: Löschen ohne Schreibrecht
- **WHEN** ein Nutzer ohne `can_write` versucht eine Datei zu löschen
- **THEN** antwortet der Server mit HTTP 403
