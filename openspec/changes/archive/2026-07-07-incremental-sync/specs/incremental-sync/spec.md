## ADDED Requirements

### Requirement: Inkrementelles Nachladen von Chat-Nachrichten

Das System SHALL `GET /api/chat/conversations/{id}/messages` um id-basierte Cursor erweitern: `?after=<msgId>` liefert nur neuere Nachrichten (append-only), `?before=<msgId>` liefert die vorhergehende Seite älterer Nachrichten. Das Frontend SHALL bei einem `chat:new-message:<id>`-Event die neue Nachricht per `?after=` anhängen, statt die Konversation vollständig neu zu laden.

#### Scenario: Nur neuere Nachrichten

- **WHEN** ein Client `GET /api/chat/conversations/{id}/messages?after=<msgId>` aufruft
- **THEN** enthält die Antwort nur Nachrichten mit `id > msgId`
- **AND** eine leere Liste, wenn keine neueren existieren

#### Scenario: Verlaufs-Seite älterer Nachrichten

- **WHEN** ein Client `GET /api/chat/conversations/{id}/messages?before=<msgId>` aufruft
- **THEN** enthält die Antwort die Seite der Nachrichten unmittelbar vor `msgId`

#### Scenario: Neue Nachricht wird angehängt statt neu geladen

- **WHEN** die aktive Konversation ein `chat:new-message:<id>`-Event empfängt
- **THEN** hängt das Frontend die betreffende Nachricht per `?after=` an
- **AND** lädt nicht die gesamte Konversation neu

### Requirement: Inkrementelles Chat-Nachladen ändert die Sichtbarkeit nicht

Das System SHALL durch die Cursor-Erweiterung des Chat-Endpoints die Autorisierungs-/Sichtbarkeitsregeln NICHT verändern. `?after=`/`?before=` SHALL genau die Teilmenge der ohnehin sichtbaren Nachrichten derselben Konversation liefern.

#### Scenario: Cursor-Abruf respektiert Konversations-Zugriff

- **WHEN** ein Client `?after=`/`?before=` auf eine Konversation aufruft, auf die er zugreifen darf
- **THEN** entspricht das Ergebnis exakt dem Ausschnitt eines vollständigen Abrufs derselben Konversation
- **AND** enthält keine Nachrichten aus Konversationen, die der Nutzer nicht sehen darf
