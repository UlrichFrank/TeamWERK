### Requirement: SSE-Endpoint sendet typisierte Refresh-Signale

Der Server SHALL einen SSE-Endpoint `GET /api/events` bereitstellen. Der Endpoint SHALL authentifizierte Verbindungen offen halten und typisierte Event-Strings senden (`data: <event-typ>\n\n`), wenn eine Mutation in einem der folgenden Bereiche stattfindet: `mitfahrgelegenheiten`, `members`, `duties`, `games`, `settings`, `trainings`.

#### Scenario: Neuer Mitfahrgelegenheiten-Eintrag löst Event aus

- **WHEN** ein Nutzer `POST /api/mitfahrgelegenheiten` (Upsert) aufruft
- **THEN** erhalten alle verbundenen SSE-Clients innerhalb von 1 Sekunde `data: mitfahrgelegenheiten`

#### Scenario: Paarungsanfrage löst Event aus

- **WHEN** ein Nutzer `POST /api/mitfahrt-paarungen` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: mitfahrgelegenheiten`

#### Scenario: Mitglieds-Mutation löst Event aus

- **WHEN** ein Admin oder Trainer ein Mitglied anlegt, bearbeitet oder dessen Status ändert
- **THEN** erhalten alle verbundenen SSE-Clients `data: members`

#### Scenario: Dienst-Mutation löst Event aus

- **WHEN** ein Admin oder Trainer einen Dienst-Slot anlegt, bearbeitet oder löscht, oder eine Zuweisung erfüllt/als Geldersatz markiert
- **THEN** erhalten alle verbundenen SSE-Clients `data: duties`

#### Scenario: Trainings-Mutation löst Event aus

- **WHEN** ein Nutzer eine Trainings-Session oder Trainingsserie erstellt, bearbeitet oder löscht, oder einen RSVP abgibt
- **THEN** erhalten alle verbundenen SSE-Clients `data: trainings`

#### Scenario: Keepalive verhindert Verbindungsabbruch

- **WHEN** 30 Sekunden keine Mutation stattgefunden hat
- **THEN** sendet der Server einen SSE-Kommentar (`: ping`) um die Verbindung offen zu halten

### Requirement: Auth via JWT-Query-Parameter am SSE-Endpoint

Da `EventSource` keine Custom-Header unterstützt, SHALL die Auth-Middleware auch einen `?token=<jwt>`-Query-Parameter akzeptieren. Der SSE-Endpoint SHALL im authenticated-Block registriert sein.

#### Scenario: Verbindungsaufbau mit gültigem Token

- **WHEN** ein eingeloggter Nutzer `GET /api/events?token=<valid-jwt>` aufruft
- **THEN** wird die Verbindung akzeptiert und offen gehalten

#### Scenario: Verbindungsaufbau ohne Token schlägt fehl

- **WHEN** ein nicht-authentifizierter Request den SSE-Endpoint aufruft
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Frontend ersetzt manuellen Reload durch EventSource

Alle relevanten Pages (Mitfahrgelegenheiten, Mitglieder, Dienste, Spielplan) SHALL eine `useLiveUpdates`-Verbindung zum SSE-Endpoint aufbauen und bei einem passenden `message`-Event die Daten still neu laden (ohne sichtbaren Ladespinner).

#### Scenario: Seite aktualisiert sich bei fremder Änderung

- **WHEN** ein anderer Nutzer einen Eintrag anlegt, ändert oder löscht
- **THEN** lädt die Seite des beobachtenden Nutzers die Daten neu ohne sichtbaren Ladespinner

#### Scenario: EventSource wird beim Verlassen der Seite aufgeräumt

- **WHEN** der Nutzer eine Seite mit `useLiveUpdates` verlässt
- **THEN** wird die SSE-Verbindung geschlossen (`es.close()`)

#### Scenario: Page ignoriert nicht relevante Events

- **WHEN** ein `members`-Event eintrifft und die aktuelle Seite nur auf `duties`-Events abonniert ist
- **THEN** lädt die Seite NICHT neu
## Requirements
### Requirement: SSE-Endpoint sendet Versions-Event beim Verbindungsaufbau

Der SSE-Handler SHALL beim Aufbau jeder neuen Verbindung als erstes Event `data: __version:<hash>\n\n` senden, bevor reguläre Mutations-Events gesendet werden. Der `<hash>` ist der zur Compile-Zeit eingebettete Build-Hash.

#### Scenario: Neuer Client empfängt Versions-Event beim Connect

- **WHEN** ein authentifizierter Client `GET /api/events?token=<jwt>` aufruft
- **THEN** sendet der Server innerhalb von 100ms das Event `data: __version:<hash>`
- **THEN** folgen danach reguläre Mutations-Events (keepalive, domain-events)

#### Scenario: Reconnect nach Server-Neustart sendet neuen Hash

- **WHEN** ein Client nach einem Server-Neustart die SSE-Verbindung neu aufbaut
- **THEN** sendet der neue Server seinen aktuellen Build-Hash als `__version:`-Event
- **THEN** unterscheidet sich dieser Hash vom Hash des vorherigen Servers (da neues Binary)

#### Scenario: Bestehende useLiveUpdates-Nutzung bleibt unverändert

- **WHEN** eine Seite `useLiveUpdates` nutzt und ein `__version:`-Event empfängt
- **THEN** wird das Event NICHT an den `onEvent`-Callback weitergeleitet
- **THEN** verarbeitet `useLiveUpdates` nur Events ohne `__version:`-Prefix

