## ADDED Requirements

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
