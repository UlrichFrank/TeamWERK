## MODIFIED Requirements

### Requirement: Bild abrufen

Das System SHALL Bilder unter `GET /api/media/{id}` ausliefern. Nur authentifizierte User dürfen Bilder abrufen, und nur solche, die das referenzierende Objekt sehen dürfen: der Hochladende, Mitglieder (auch ausgetretene) der Konversation, deren Nachricht das Bild trägt, oder Empfänger der Mitteilung, die das Bild trägt. Für alle anderen MUST der Server mit HTTP 404 antworten, damit die Existenz einer ID nicht erratbar ist. Der Server MUST den in `media.mime_type` gespeicherten `Content-Type` sowie `X-Content-Type-Options: nosniff` setzen.

#### Scenario: Bild erfolgreich abrufen

- **WHEN** ein Konversationsmitglied `GET /api/media/{id}` für das Bild einer Nachricht aufruft und die Zeile + Datei existieren
- **THEN** sendet der Server die Bild-Bytes mit dem korrekten `Content-Type`

#### Scenario: Bild nicht gefunden

- **WHEN** ein User eine unbekannte `id` abruft
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Nicht authentifiziert

- **WHEN** ein nicht eingeloggter User ein Bild abruft
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Fremder Nutzer sieht das Bild nicht

- **WHEN** ein eingeloggter Nutzer, der weder Hochladender noch Konversationsmitglied noch Mitteilungsempfänger ist, eine existierende ID abruft
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Mitteilungsempfänger sieht das Bild

- **WHEN** ein Empfänger einer Mitteilung mit Bild dieses abruft
- **THEN** antwortet der Server mit HTTP 200
