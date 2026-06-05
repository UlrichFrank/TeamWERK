## MODIFIED Requirements

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
