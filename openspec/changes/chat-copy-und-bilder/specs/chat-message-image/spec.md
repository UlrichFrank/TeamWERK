## ADDED Requirements

### Requirement: Bild hochladen

Das System SHALL einen Endpunkt `POST /api/chat/upload` bereitstellen, der eine einzelne Bilddatei via `multipart/form-data` (Feld `image`) entgegennimmt, unter `/var/lib/teamwerk/chat-images/<uuid>.<ext>` speichert und `{ "imageUrl": "/api/chat/images/<uuid>.<ext>" }` zurückgibt. Erlaubte MIME-Types: `image/jpeg`, `image/png`, `image/gif`, `image/webp`. Maximale Dateigröße: 10 MB. Nur authentifizierte User dürfen hochladen.

#### Scenario: Erfolgreiches Bild-Upload

- **WHEN** ein authentifizierter User `POST /api/chat/upload` mit einer gültigen JPEG-Datei aufruft
- **THEN** speichert der Server die Datei unter `/var/lib/teamwerk/chat-images/<uuid>.jpg`
- **THEN** antwortet der Server mit HTTP 200 und `{ "imageUrl": "/api/chat/images/<uuid>.jpg" }`

#### Scenario: Ungültiger MIME-Type

- **WHEN** ein User eine PDF-Datei hochlädt
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Datei zu groß

- **WHEN** ein User eine Bilddatei > 10 MB hochlädt
- **THEN** antwortet der Server mit HTTP 413

#### Scenario: Nicht authentifiziert

- **WHEN** ein nicht eingeloggter User den Upload-Endpunkt aufruft
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Bild abrufen

Das System SHALL Bild-Dateien unter `GET /api/chat/images/:filename` ausliefern. Nur authentifizierte User dürfen Bilder abrufen. Der Server MUSS den korrekten `Content-Type`-Header setzen.

#### Scenario: Bild erfolgreich abrufen

- **WHEN** ein authentifizierter User `GET /api/chat/images/<uuid>.jpg` aufruft und die Datei existiert
- **THEN** sendet der Server die Bild-Bytes mit `Content-Type: image/jpeg`

#### Scenario: Bild nicht gefunden

- **WHEN** ein User einen unbekannten Dateinamen abruft
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Nicht authentifiziert

- **WHEN** ein nicht eingeloggter User ein Bild abruft
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Nachricht mit Bild senden

Das Frontend SHALL im Sende-Bereich einen Bild-Picker-Button anzeigen (Büroklammer-Icon). Über diesen Button öffnet sich der Datei-Picker (accept: image/*). Zusätzlich SHALL das Frontend auf `paste`-Events im Chat-Input-Bereich reagieren und enthaltene Bilddaten direkt hochladen. Nach erfolgreichem Upload wird die `imageUrl` gesetzt und die Nachricht (ggf. mit zusätzlichem Textinhalt) abgesendet.

#### Scenario: Bild über Datei-Picker senden

- **WHEN** ein User auf den Bild-Picker-Button klickt, eine Bilddatei auswählt und auf Senden klickt
- **THEN** wird das Bild zunächst via `POST /api/chat/upload` hochgeladen
- **THEN** wird die Nachricht mit der zurückgegebenen `imageUrl` gesendet

#### Scenario: Bild via Einfügen (Paste) aus Zwischenablage senden

- **WHEN** ein User ein Bild in die Zwischenablage kopiert und Strg+V/Cmd+V im Chat-Bereich drückt
- **THEN** wird das Bild automatisch hochgeladen und als Vorschau im Sende-Bereich angezeigt

#### Scenario: Bild-Vorschau vor dem Senden entfernen

- **WHEN** ein User ein Bild ausgewählt hat aber noch nicht gesendet hat
- **THEN** kann er die Vorschau über einen ×-Button entfernen

#### Scenario: Bild mit Textinhalt kombinieren

- **WHEN** ein User sowohl Text eingibt als auch ein Bild auswählt und sendet
- **THEN** wird eine Nachricht mit `body` und `imageUrl` gespeichert

### Requirement: Bild in Nachrichtenblase anzeigen

Das Frontend SHALL in `MessageBubble` Bilder als `<img>`-Element unterhalb des Textes rendern (oder allein, wenn kein Text vorhanden). Maximale Anzeigebreite: `max-w-xs`. Ein Tipp/Klick auf das Bild öffnet ein Vollbild-Overlay.

#### Scenario: Nachricht mit Bild anzeigen

- **WHEN** eine Nachricht `imageUrl` gesetzt hat
- **THEN** wird das Bild in der Nachrichtenblase als `<img>` mit `max-w-xs` angezeigt

#### Scenario: Vollbild-Overlay öffnen

- **WHEN** ein User auf ein Bild in einer Nachrichtenblase tippt
- **THEN** öffnet sich ein Overlay (`fixed inset-0`) mit dem Bild in maximaler Größe und einem Schließen-Button
