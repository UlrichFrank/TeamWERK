## MODIFIED Requirements

### Requirement: Auth via Cookie am SSE-Endpoint
Da `EventSource` keine Custom-Header unterstützt, SHALL der SSE-Endpunkt `GET /api/events` über das HttpOnly-Refresh-Token-Cookie authentifiziert werden. Die Nutzung eines `?token=<jwt>`-Query-Parameters MUST entfernt werden, da Access Tokens in URL-Query-Parametern in Server-Logs, Browser-Verlauf und Proxy-Logs erscheinen. Das Backend MUST den Cookie-basierten Auth-Pfad in der Middleware für den SSE-Endpunkt unterstützen.

#### Scenario: Verbindungsaufbau mit gültigem Cookie
- **WHEN** ein eingeloggter Nutzer `GET /api/events` mit einem gültigen HttpOnly-Refresh-Token-Cookie aufruft
- **THEN** wird die Verbindung akzeptiert und offen gehalten

#### Scenario: Verbindungsaufbau ohne Token schlägt fehl
- **WHEN** ein nicht-authentifizierter Request den SSE-Endpoint aufruft (kein Cookie, kein Bearer Token)
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Access Token NICHT im Query-Parameter
- **WHEN** ein Client `GET /api/events?token=<jwt>` aufruft (altes Verhalten)
- **THEN** wird der `?token`-Query-Parameter NICHT als Authentifizierungsmittel akzeptiert

### Requirement: Frontend ersetzt manuellen Reload durch EventSource
Alle relevanten Pages SHALL eine `useLiveUpdates`-Verbindung zum SSE-Endpoint aufbauen und bei einem passenden `message`-Event die Daten still neu laden (ohne sichtbaren Ladespinner). Die SSE-Verbindung MUSS nach einem Access-Token-Refresh automatisch neu aufgebaut werden, um Reconnect-Schleifen mit abgelaufenen Tokens zu vermeiden.

#### Scenario: Seite aktualisiert sich bei fremder Änderung
- **WHEN** ein anderer Nutzer einen Eintrag anlegt, ändert oder löscht
- **THEN** lädt die Seite des beobachtenden Nutzers die Daten neu ohne sichtbaren Ladespinner

#### Scenario: EventSource wird beim Verlassen der Seite aufgeräumt
- **WHEN** der Nutzer eine Seite mit `useLiveUpdates` verlässt
- **THEN** wird die SSE-Verbindung geschlossen (`es.close()`)

#### Scenario: Page ignoriert nicht relevante Events
- **WHEN** ein `members`-Event eintrifft und die aktuelle Seite nur auf `duties`-Events abonniert ist
- **THEN** lädt die Seite NICHT neu

#### Scenario: EventSource wird nach Token-Refresh neu aufgebaut
- **WHEN** der Access Token durch den 401-Interceptor erneuert wurde
- **THEN** baut `useLiveUpdates` eine neue EventSource-Verbindung auf
- **THEN** gibt es keine Reconnect-Schleife mit dem abgelaufenen Token
