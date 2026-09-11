## MODIFIED Requirements

### Requirement: Bild abrufen

`GET /api/media/{id}` MUST die Bild-Bytes nur an Nutzer liefern, die das referenzierende Objekt sehen dürfen: den Hochladenden, Mitglieder (auch ausgetretene) der Konversation, deren Nachricht das Bild trägt, oder Empfänger der Mitteilung, die das Bild trägt. Für alle anderen MUST der Server mit HTTP 404 antworten, damit die Existenz einer ID nicht erratbar ist. Die Antwort trägt weiterhin `Content-Type` aus der Datenbank und einen Cache-Header.

#### Scenario: Konversationsmitglied sieht das Bild
- **WHEN** ein Mitglied der Konversation das Bild einer Nachricht abruft
- **THEN** antwortet der Server mit HTTP 200 und den Bild-Bytes

#### Scenario: Fremder Nutzer sieht das Bild nicht
- **WHEN** ein eingeloggter Nutzer, der weder Hochladender noch Konversationsmitglied noch Mitteilungsempfänger ist, dieselbe ID abruft
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Mitteilungsempfänger sieht das Bild
- **WHEN** ein Empfänger einer Mitteilung mit Bild dieses abruft
- **THEN** antwortet der Server mit HTTP 200
