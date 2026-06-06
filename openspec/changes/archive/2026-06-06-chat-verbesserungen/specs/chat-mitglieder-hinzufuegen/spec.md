## ADDED Requirements

### Requirement: Gruppenersteller kann Mitglieder hinzufügen
Der Ersteller eines Gruppen-Chats (Feld `conversations.created_by`) SHALL jederzeit weitere Personen hinzufügen können, auch wenn diese die Gruppe zuvor verlassen haben. Nicht-Ersteller haben kein Hinzufügen-Recht.

#### Scenario: Mitglied erfolgreich hinzufügen
- **WHEN** der Ersteller `POST /api/chat/conversations/{id}/members` mit `{ "userId": <id> }` aufruft
- **THEN** wird der User als aktives Mitglied eingetragen (left_at = NULL)
- **THEN** erhält der neu hinzugefügte User ein SSE-Event `chat:new-message:{convId}`
- **THEN** erscheint die Gruppe in der Konversationsliste des neu hinzugefügten Users

#### Scenario: User nach Verlassen wieder hinzufügen
- **WHEN** der Ersteller einen User hinzufügt, der die Gruppe zuvor verlassen hatte (left_at gesetzt)
- **THEN** wird `conversation_members.left_at` auf NULL zurückgesetzt (kein neuer Insert)
- **THEN** hat der User wieder Zugriff auf den gesamten Nachrichtenverlauf

#### Scenario: Nicht-Ersteller versucht hinzuzufügen
- **WHEN** ein aktives Mitglied (kein Ersteller) `POST /api/chat/conversations/{id}/members` aufruft
- **THEN** antwortet der Server mit 403 Forbidden

#### Scenario: Unzugänglicher User
- **WHEN** der Ersteller einen User hinzufügen möchte, den er laut `canContactUser`-Logik nicht kontaktieren darf
- **THEN** antwortet der Server mit 403 Forbidden

#### Scenario: Hinzufügen zu Direct Chat nicht möglich
- **WHEN** `POST /api/chat/conversations/{id}/members` für eine Direct-Conversation aufgerufen wird
- **THEN** antwortet der Server mit 400 Bad Request
