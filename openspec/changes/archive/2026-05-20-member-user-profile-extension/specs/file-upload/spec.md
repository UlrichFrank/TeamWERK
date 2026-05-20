## ADDED Requirements

### Requirement: Upload-Verzeichnis existiert und ist beschreibbar
Das System SHALL beim Start prüfen ob das Upload-Verzeichnis existiert. Der Pfad ist konfigurierbar über die Env-Variable `UPLOAD_DIR` (Default: `./storage/uploads/`). Unterverzeichnisse `member-photos/`, `user-photos/` und `sepa-mandats/` werden beim ersten Upload automatisch erstellt.

#### Scenario: Verzeichnis fehlt beim Start
- **WHEN** `UPLOAD_DIR` existiert nicht beim Serverstart
- **THEN** Server startet trotzdem, Verzeichnis wird beim ersten Upload angelegt (mit `os.MkdirAll`)

### Requirement: Uploads werden mit UUID-Dateiname gespeichert
Das System SHALL jeden Upload mit einem zufälligen UUID als Dateiname (+ Originalextension) speichern. Originalname wird verworfen, um Path-Traversal auszuschließen.

#### Scenario: Upload speichert mit UUID
- **WHEN** Upload mit Dateiname `foto.jpg`
- **THEN** gespeichert als `member-photos/550e8400-e29b-41d4-a716-446655440000.jpg`

### Requirement: Auslieferung via authentifizierten Endpoint
`GET /api/uploads/{subpath}` liefert Dateien aus. `{subpath}` ist der relative Pfad innerhalb von `UPLOAD_DIR`. Path-Traversal (`../`) wird mit HTTP 400 abgewiesen.

#### Scenario: Path-Traversal wird abgewiesen
- **WHEN** `GET /api/uploads/../etc/passwd`
- **THEN** HTTP 400

#### Scenario: Nicht existierende Datei
- **WHEN** `GET /api/uploads/member-photos/nonexistent.jpg`
- **THEN** HTTP 404

### Requirement: Altes Foto wird beim Re-Upload gelöscht
Wenn ein Mitglied oder Nutzer bereits ein Foto hat und ein neues hochgeladen wird, SOLL die alte Datei vom Filesystem gelöscht werden. Gleiches gilt für SEPA-Dokumente.

#### Scenario: Re-Upload löscht altes Foto
- **WHEN** Mitglied hat bereits `photo_path` gesetzt und neues Bild wird hochgeladen
- **THEN** alte Datei wird gelöscht, `photo_path` auf neuen Pfad gesetzt

#### Scenario: Re-Upload SEPA löscht altes Dokument
- **WHEN** Mitglied hat bereits `sepa_mandat_path` gesetzt und neues Dokument hochgeladen
- **THEN** alte Datei wird gelöscht, `sepa_mandat_path` auf neuen Pfad gesetzt

### Requirement: SEPA-Dokument-Upload akzeptiert PDF und Bilder
`POST /api/upload/sepa-mandat/{id}` SHALL Dateitypen `application/pdf`, `image/jpeg`, `image/png`, `image/webp` akzeptieren. Maximale Dateigröße: 10 MB. Nur Admin darf diesen Endpoint aufrufen.

#### Scenario: PDF-Upload erfolgreich
- **WHEN** Admin `POST /api/upload/sepa-mandat/{id}` mit PDF ≤ 10 MB
- **THEN** Datei unter `sepa-mandats/uuid.pdf` gespeichert, `sepa_mandat_path` gesetzt

#### Scenario: Zu große Datei abgewiesen
- **WHEN** Upload mit Datei > 10 MB
- **THEN** HTTP 413

### Requirement: Member-Foto ist für alle eingeloggten Nutzer sichtbar
Das Mitgliedsfoto (Passfoto) SOLL für alle authentifizierten Nutzer über `GET /api/uploads/member-photos/...` abrufbar sein, unabhängig von Sichtbarkeitseinstellungen. Es existiert kein Visibility-Toggle für Member-Fotos.

#### Scenario: Trainer ruft Mitgliedsfoto ab
- **WHEN** Nutzer mit Rolle `trainer` ruft `GET /api/uploads/member-photos/uuid.jpg` auf
- **THEN** HTTP 200 mit Bilddaten (kein 403, kein Visibility-Check)

#### Scenario: Spieler ruft Mitgliedsfoto ab
- **WHEN** Nutzer mit Rolle `spieler` ruft `GET /api/uploads/member-photos/uuid.jpg` auf
- **THEN** HTTP 200 mit Bilddaten
