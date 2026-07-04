## ADDED Requirements

### Requirement: Pull-basierte Delta-Synchronisation über Client-Cursor

Das System SHALL für schwere Listen-Endpoints (`GET /api/games`, `/api/duty-slots`, `/api/training-sessions`, `/api/kader`) einen Cursor-Parameter `?since=<cursor>` anbieten, der nur Datensätze mit `updated_at > cursor` in `items` liefert, sowie einen neuen Cursor zurückgibt. Ohne `?since=` SHALL sich der Endpoint unverändert (voller, paginierter Abruf) verhalten.

#### Scenario: Nur Geändertes seit Cursor

- **WHEN** ein Client `GET /api/games?since=<cursor>` aufruft und seither ein Spiel geändert wurde
- **THEN** enthält `items` genau dieses Spiel
- **AND** unveränderte Spiele fehlen in `items`

#### Scenario: Ohne Cursor unverändert

- **WHEN** ein Client `GET /api/games` ohne `?since` aufruft
- **THEN** verhält sich der Endpoint wie zuvor (volle, paginierte Liste)

### Requirement: Löschungen werden als Tombstone gemeldet

Das System SHALL seit dem Cursor gelöschte Datensätze in einer `deleted_ids`-Liste der Response melden, damit der Client sie lokal entfernen kann. Löschungen SHALL über ein append-only Tombstone-Log (bzw. Soft-Delete) nachvollziehbar sein.

#### Scenario: Gelöschter Datensatz erscheint als Tombstone

- **WHEN** seit dem Cursor ein Spiel gelöscht wurde
- **THEN** erscheint dessen ID in `deleted_ids`
- **AND** nicht in `items`

### Requirement: Voll-Refetch-Fallback bei zu altem Cursor

Das System SHALL bei einem Cursor, der älter als die Tombstone-Aufbewahrungsfrist ist, eine vollständige Response mit einem Signal (`full: true`) liefern, sodass der Client seinen lokalen Bestand neu aufbaut. Ein zu alter Cursor SHALL NIEMALS zu still fehlenden oder überzähligen Daten führen.

#### Scenario: Zu alter Cursor erzwingt Neuaufbau

- **WHEN** ein Client mit einem Cursor älter als die Aufbewahrungsfrist synchronisiert
- **THEN** antwortet das System mit `full: true` und einer vollständigen (paginierten) Menge
- **AND** der Client verwirft seinen lokalen Bestand und baut ihn neu auf

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

### Requirement: Inkrementelle Synchronisation ändert die Sichtbarkeit nicht

Das System SHALL durch Cursor-Synchronisation die Autorisierungs-/Sichtbarkeitsregeln NICHT verändern. Der aus Deltas und Tombstones lokal rekonstruierte Zustand SHALL identisch mit einem Voll-Refetch derselben sichtbaren Menge sein.

#### Scenario: Rekonstruierter Zustand entspricht Voll-Refetch

- **WHEN** ein Client eine Liste über mehrere `?since=`-Deltas plus Tombstones aufbaut
- **THEN** entspricht das Ergebnis exakt dem eines vollständigen Abrufs derselben Route
- **AND** enthält keine Datensätze, die der Nutzer nicht ohnehin sehen dürfte
