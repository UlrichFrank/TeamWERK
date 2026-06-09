## MODIFIED Requirements

### Requirement: Direct-Konversation erstellen oder öffnen

Das System SHALL beim Erstellen einer Direct-Konversation prüfen ob bereits eine Konversation zwischen den beiden Usern existiert — auch wenn A die Konversation bereits verlassen hat (A's `left_at IS NOT NULL`), solange B noch aktiv ist (B's `left_at IS NULL`). Falls eine solche Konversation gefunden wird, SHALL A per Re-join wiederhergestellt werden (A's `left_at = NULL`) und der bestehende Datensatz zurückgegeben werden. Ist keine Konversation vorhanden (beide hatten gelöscht, Conversation ist permanent entfernt), wird eine neue Konversation angelegt. In beiden Fällen (Re-join und Neu-Anlage) SHALL B ein SSE-Event `chat:new-message:{convId}` erhalten damit die Konversation in Bs Liste erscheint.

#### Scenario: Erste Direct-Konversation zwischen zwei Usern

- **WHEN** User A `POST /api/chat/conversations` mit `{ type: "direct", userId: B }` aufruft und keine frühere Konversation existiert
- **THEN** wird eine neue Konversation angelegt und beide User als Mitglieder eingetragen
- **THEN** gibt der Server HTTP 201 mit dem Konversations-Objekt zurück
- **THEN** erhält B ein SSE-Event `chat:new-message:{convId}`

#### Scenario: Bestehende Direct-Konversation erneut öffnen (beide aktiv)

- **WHEN** User A `POST /api/chat/conversations` mit `{ type: "direct", userId: B }` aufruft und beide `left_at IS NULL` haben
- **THEN** gibt der Server HTTP 200 mit der bestehenden Konversation zurück (kein Duplikat, kein SSE)

#### Scenario: A hatte Konversation verlassen — Re-join statt Duplikat

- **WHEN** User A `POST /api/chat/conversations` mit `{ type: "direct", userId: B }` aufruft, A hat `left_at IS NOT NULL` aber B ist noch aktiv (`left_at IS NULL`)
- **THEN** wird A's `left_at = NULL` gesetzt (Re-join)
- **THEN** gibt der Server HTTP 200 mit der bestehenden Konversation zurück (kein neuer Thread, kein Verlust der History)
- **THEN** erhält B ein SSE-Event `chat:new-message:{convId}`

#### Scenario: Beide hatten gelöscht — neuer Thread

- **WHEN** User A `POST /api/chat/conversations` mit `{ type: "direct", userId: B }` aufruft und die frühere Konversation dauerhaft gelöscht wurde (beide hatten `left_at` gesetzt)
- **THEN** wird eine neue Konversation angelegt und beide User als Mitglieder eingetragen
- **THEN** gibt der Server HTTP 201 zurück
- **THEN** erhält B ein SSE-Event `chat:new-message:{convId}`

#### Scenario: Spieler kann nur User aus eigenem Team anschreiben

- **WHEN** ein Spieler eine Direct-Konversation mit einem User aus einem anderen Team versucht
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Nachricht senden

Das System SHALL das Senden einer Nachricht in einer Konversation erlauben. Der Request kann optional `replyToId` enthalten. Die referenzierte Nachricht MUSS zur selben Konversation gehören. Nach erfolgreichem Speichern SHALL der Server via SSE **alle** aktiven Mitglieder einschließlich des Senders selbst benachrichtigen (damit andere Geräte/Tabs des Senders die Nachricht erhalten) und Push Notifications an Mitglieder senden die gerade offline sind. Bei Direkt-Konversationen SHALL der Server vor dem SSE-Broadcast prüfen ob das andere Mitglied `left_at IS NOT NULL` hat — falls ja, SHALL `left_at = NULL` gesetzt werden (Auto-Re-join), damit das andere Mitglied das SSE erhält und die Konversation wieder in seiner Liste erscheint.

#### Scenario: Nachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ body: "Hallo!" }` aufruft
- **THEN** wird die Nachricht gespeichert und HTTP 201 zurückgegeben
- **THEN** erhalten alle aktiven Mitglieder (einschließlich Sender) ein SSE-Event `chat:new-message:<conversationId>`

#### Scenario: Sender-Gerät B erhält Echtzeit-Update

- **WHEN** der Sender auf Gerät A eine Nachricht sendet und gleichzeitig auf Gerät B eingeloggt ist
- **THEN** empfängt Gerät B das SSE-Event `chat:new-message:{convId}` und zeigt die neue Nachricht an

#### Scenario: Nachricht mit Reply senden

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit gültigem `replyToId` aufruft
- **THEN** wird `messages.reply_to_id` auf den angegebenen Wert gesetzt

#### Scenario: Ungültige Reply-Referenz

- **WHEN** `replyToId` auf eine Nachricht in einer anderen Konversation zeigt
- **THEN** antwortet das Backend mit HTTP 400

#### Scenario: Ausgetretenes Mitglied kann nicht senden

- **WHEN** ein User der die Gruppe verlassen hat eine Nachricht sendet
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Leere Nachricht wird abgelehnt

- **WHEN** ein User eine Nachricht mit leerem `body` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: B schreibt in Direkt-Chat, A hatte gelöscht — Auto-Re-join

- **WHEN** User B in einem Direkt-Chat eine Nachricht sendet und User A's `left_at IS NOT NULL`
- **THEN** wird A's `left_at = NULL` gesetzt bevor der SSE-Broadcast erfolgt
- **THEN** erhält A ein SSE-Event `chat:new-message:{convId}` und die Konversation erscheint wieder in A's Liste
