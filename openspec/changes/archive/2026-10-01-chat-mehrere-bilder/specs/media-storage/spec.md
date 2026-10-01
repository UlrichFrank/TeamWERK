## MODIFIED Requirements

### Requirement: Bild abrufen

Das System SHALL Bilder unter `GET /api/media/{id}` ausliefern. Nur authentifizierte User dürfen Bilder abrufen, und nur solche, die das referenzierende Objekt sehen dürfen: der Hochladende, Mitglieder (auch ausgetretene) der Konversation, deren Nachricht das Bild trägt, oder Empfänger der Mitteilung, die das Bild trägt. Ein Bild gilt als von einer Nachricht oder Mitteilung getragen, wenn es an **irgendeiner** Position ihres Albums steht, nicht nur an der ersten. Für alle anderen MUST der Server mit HTTP 404 antworten, damit die Existenz einer ID nicht erratbar ist. Der Server MUST den in `media.mime_type` gespeicherten `Content-Type` sowie `X-Content-Type-Options: nosniff` setzen.

#### Scenario: Bild erfolgreich abrufen

- **WHEN** ein Konversationsmitglied `GET /api/media/{id}` für das Bild einer Nachricht aufruft und die Zeile + Datei existieren
- **THEN** sendet der Server die Bild-Bytes mit dem korrekten `Content-Type`

#### Scenario: Späteres Albumbild für Konversationsmitglied

- **WHEN** ein Konversationsmitglied das dritte Bild eines Albums abruft
- **THEN** antwortet der Server mit HTTP 200 und den Bild-Bytes

#### Scenario: Bild nicht gefunden

- **WHEN** ein User eine unbekannte `id` abruft
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Nicht authentifiziert

- **WHEN** ein nicht eingeloggter User ein Bild abruft
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Fremder Nutzer sieht das Bild nicht

- **WHEN** ein eingeloggter Nutzer, der weder Hochladender noch Konversationsmitglied noch Mitteilungsempfänger ist, eine existierende ID abruft, auch die eines späteren Albumbilds
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Mitteilungsempfänger sieht das Bild

- **WHEN** ein Empfänger einer Mitteilung mit Bildern eines davon abruft, an welcher Position auch immer
- **THEN** antwortet der Server mit HTTP 200
