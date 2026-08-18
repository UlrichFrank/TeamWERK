## MODIFIED Requirements

### Requirement: Auth via Cookie am SSE-Endpoint

Da `EventSource` keine Custom-Header unterstützt, SHALL **jeder** SSE-Endpunkt des Systems
über das HttpOnly-Refresh-Token-Cookie authentifiziert werden — heute `GET /api/events`
(globale Live-Updates) und `GET /api/chat/events` (Chat-Kanal). Die Nutzung eines
`?token=<jwt>`-Query-Parameters MUST entfernt werden, da Access Tokens in
URL-Query-Parametern in Server-Logs, Browser-Verlauf und Proxy-Logs erscheinen. Das Backend
MUST den Cookie-basierten Auth-Pfad in der Middleware für SSE-Endpunkte unterstützen.

Ein Client MUST an keinem SSE-Endpunkt einen Access Token in der URL mitschicken, auch nicht
zusätzlich zum Cookie: ein Parameter, den der Server ignoriert, wird trotzdem protokolliert.

#### Scenario: Verbindungsaufbau mit gültigem Cookie

- **WHEN** ein eingeloggter Nutzer `GET /api/events` mit einem gültigen HttpOnly-Refresh-Token-Cookie aufruft
- **THEN** wird die Verbindung akzeptiert und offen gehalten

#### Scenario: Verbindungsaufbau ohne Token schlägt fehl

- **WHEN** ein nicht-authentifizierter Request den SSE-Endpoint aufruft (kein Cookie, kein Bearer Token)
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Access Token NICHT im Query-Parameter

- **WHEN** ein Client `GET /api/events?token=<jwt>` aufruft (altes Verhalten)
- **THEN** wird der `?token`-Query-Parameter NICHT als Authentifizierungsmittel akzeptiert

#### Scenario: Auch der Chat-Kanal trägt keinen Token in der URL

- **WHEN** das Frontend den Chat-Ereigniskanal öffnet
- **THEN** lautet die aufgerufene URL exakt `/api/chat/events` ohne Query-Parameter
- **THEN** wird die Verbindung allein über das HttpOnly-Refresh-Token-Cookie authentifiziert

## ADDED Requirements

### Requirement: Lebenszyklus einer SSE-Verbindung im Frontend

Ein SSE-Kanal im Frontend SHALL an die **Nutzer-Identität** gebunden sein: er wird
aufgebaut, sobald ein angemeldeter Nutzer feststeht, und bei jedem Identitätswechsel
(Login, Logout, Start und Ende einer Impersonation) geschlossen und neu aufgebaut. Ein
Kanal, der einmalig beim Mounten der Komponente aufgebaut wird, erfüllt diese Anforderung
NICHT — er überlebt den Identitätswechsel und liefert danach die Ereignisse des vorherigen
Nutzers.

Bricht die Verbindung ab, SHALL der Kanal mit begrenztem Backoff erneut verbunden werden.
Ein endgültiges Aufgeben für die Lebensdauer der Seite ist NICHT zulässig: SSE-Verbindungen
brechen im Normalbetrieb regelmäßig ab (Bildschirm aus, Wechsel zwischen WLAN und
Mobilfunk, Suspendieren einer Homescreen-PWA, Timeout im Reverse Proxy), und ohne Kanal
verliert jede daran hängende Anzeige still ihre Aktualität.

Beim Verlassen der Komponente SHALL die Verbindung geschlossen und ein laufender
Reconnect-Timer abgeräumt werden.

#### Scenario: Verbindung wird nach Abbruch wiederhergestellt

- **WHEN** eine bestehende SSE-Verbindung abbricht (Netzwechsel, Standby, Proxy-Timeout)
- **THEN** baut das Frontend die Verbindung nach einer Wartezeit erneut auf
- **THEN** werden nach erfolgreichem Neuaufbau wieder Ereignisse verarbeitet

#### Scenario: Wiederholt scheiternde Verbindung eskaliert die Wartezeit

- **WHEN** mehrere Verbindungsversuche hintereinander scheitern
- **THEN** wächst die Wartezeit zwischen den Versuchen bis zu einer Obergrenze
- **THEN** entsteht keine ununterbrochene Folge von Verbindungsversuchen

#### Scenario: Identitätswechsel baut den Kanal neu auf

- **WHEN** ein Administrator eine Impersonation startet oder beendet
- **THEN** wird die bestehende SSE-Verbindung geschlossen
- **THEN** wird eine neue Verbindung für die nun geltende Identität aufgebaut

#### Scenario: Abmelden schließt den Kanal

- **WHEN** der Nutzer sich abmeldet
- **THEN** wird die SSE-Verbindung geschlossen und kein Reconnect mehr versucht
