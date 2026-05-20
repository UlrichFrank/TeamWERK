## ADDED Requirements

### Requirement: Admin kann Passfoto hochladen
Das System SHALL einen Upload-Endpoint `POST /api/upload/member-photo/{id}` bereitstellen. Akzeptierte Formate: image/jpeg, image/png, image/webp. Maximale Dateigröße: 5 MB. Das Foto wird unter `storage/uploads/member-photos/{uuid}.{ext}` gespeichert, der Pfad in `members.photo_path`.

#### Scenario: Erfolgreicher Upload
- **WHEN** Admin `POST /api/upload/member-photo/{id}` mit gültigem Bild (≤ 5 MB) aufruft
- **THEN** wird die Datei gespeichert, `members.photo_path` gesetzt, Response enthält `photo_url`

#### Scenario: Datei zu groß
- **WHEN** Upload mit Datei > 5 MB
- **THEN** HTTP 413, Datei wird nicht gespeichert

#### Scenario: Ungültiger Dateityp
- **WHEN** Upload mit Nicht-Bild-Datei (z.B. PDF)
- **THEN** HTTP 400, Datei wird nicht gespeichert

### Requirement: Foto-URL im Member-Response
Das System SHALL bei `GET /api/members/{id}` ein `photo_url`-Feld zurückgeben, das auf `GET /api/uploads/{filename}` zeigt, wenn ein Foto vorhanden ist; sonst `null`.

#### Scenario: Mitglied hat Foto
- **WHEN** `GET /api/members/{id}` mit gesetztem `photo_path`
- **THEN** Response enthält `photo_url: "/api/uploads/member-photos/uuid.jpg"`

#### Scenario: Mitglied hat kein Foto
- **WHEN** `GET /api/members/{id}` ohne `photo_path`
- **THEN** Response enthält `photo_url: null`

### Requirement: Foto-Auslieferung nur für eingeloggte Nutzer
`GET /api/uploads/{filename}` MUSS Auth-Middleware durchlaufen. Unauthentifizierte Anfragen erhalten HTTP 401.

#### Scenario: Eingeloggter Nutzer ruft Foto ab
- **WHEN** gültiges JWT, `GET /api/uploads/member-photos/uuid.jpg`
- **THEN** HTTP 200, Datei mit korrektem Content-Type

#### Scenario: Unauthentifizierter Zugriff
- **WHEN** kein JWT, `GET /api/uploads/member-photos/uuid.jpg`
- **THEN** HTTP 401
