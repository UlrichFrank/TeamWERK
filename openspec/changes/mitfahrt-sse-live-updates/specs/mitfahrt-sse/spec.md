## ADDED Requirements

### Requirement: SSE-Endpoint sendet Refresh-Signal bei Mutations

Der Server SHALL einen SSE-Endpoint `GET /api/mitfahrgelegenheiten/events` bereitstellen. Der Endpoint SHALL authentifizierte Verbindungen offen halten und `data: refresh\n\n` senden, wenn eine der folgenden Mutations stattfindet: Eintrag anlegen, Eintrag löschen, Paarung anfragen, Paarung bestätigen, Paarung ablehnen.

#### Scenario: Neuer Eintrag löst Refresh aus

- **WHEN** ein Nutzer `POST /api/mitfahrgelegenheiten` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients innerhalb von 1 Sekunde ein `data: refresh` Event

#### Scenario: Paarungsanfrage löst Refresh aus

- **WHEN** ein Nutzer `POST /api/mitfahrt-paarungen` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients ein `data: refresh` Event

#### Scenario: Keepalive verhindert Verbindungsabbruch

- **WHEN** 30 Sekunden keine Mutation stattgefunden hat
- **THEN** sendet der Server einen SSE-Kommentar (`: ping`) um die Verbindung offen zu halten

### Requirement: Frontend ersetzt Polling durch EventSource

Die `MitfahrgelegenheitenPage` SHALL keinen `setInterval`-Polling-Code mehr enthalten. Stattdessen SHALL sie eine `EventSource`-Verbindung zum SSE-Endpoint aufbauen und bei jedem `message`-Event die Daten still neu laden.

#### Scenario: Seite aktualisiert sich bei fremder Änderung

- **WHEN** ein anderer Nutzer einen Eintrag anlegt oder eine Paarung ändert
- **THEN** lädt die Seite die Daten neu ohne sichtbaren Ladespinner

#### Scenario: EventSource wird beim Verlassen aufgeräumt

- **WHEN** der Nutzer die Mitfahrgelegenheiten-Seite verlässt
- **THEN** wird die SSE-Verbindung geschlossen (`es.close()`)

### Requirement: Auth via JWT-Query-Parameter am SSE-Endpoint

Da `EventSource` keine Custom-Header unterstützt, SHALL der Auth-Middleware auch einen `?token=<jwt>`-Query-Parameter akzeptieren. Der SSE-Endpoint SHALL im authenticated-Block registriert sein.

#### Scenario: Verbindungsaufbau mit gültigem Token

- **WHEN** ein eingeloggter Nutzer `GET /api/mitfahrgelegenheiten/events?token=<valid-jwt>` aufruft
- **THEN** wird die Verbindung akzeptiert und offen gehalten

#### Scenario: Verbindungsaufbau ohne Token schlägt fehl

- **WHEN** ein nicht-authentifizierter Request den SSE-Endpoint aufruft
- **THEN** antwortet der Server mit HTTP 401
