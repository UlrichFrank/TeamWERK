# payload-measurement Specification

## Purpose
TBD - created by archiving change payload-measurement-harness. Update Purpose after archive.
## Requirements
### Requirement: Deterministische Messumgebung

Das Mess-Werkzeug SHALL einen festen, reproduzierbaren Datensatz über die bestehenden `testutil`-Fixtures und `testutil.NewServer` aufbauen, ohne Zufallswerte oder Wanduhrzeit im Datensatz. Zwei Läufe mit demselben Code SHALL byte-identische Messgrößen liefern.

#### Scenario: Wiederholter Lauf ist stabil

- **WHEN** das Mess-Werkzeug zweimal ohne Code-Änderung ausgeführt wird
- **THEN** sind die erfassten Payload-Größen und Fan-out-Zahlen identisch

### Requirement: Payload-Messung pro Route über öffentliche Endpoints

Das Werkzeug SHALL für eine konfigurierte Liste schwerer GET-Routen die Response-Größe (Bytes) und den HTTP-Status ausschließlich über die öffentlichen HTTP-Endpoints (`testutil.NewServer`) erfassen — keine internen Shortcuts.

#### Scenario: Bytes und Status je Route

- **WHEN** das Werkzeug eine konfigurierte Route abruft
- **THEN** erfasst es die Antwortgröße in Bytes und den HTTP-Status

### Requirement: Messung der Referenzdaten-Revalidierung

Das Werkzeug SHALL Referenzrouten zweimal abrufen — den zweiten Aufruf mit `If-None-Match` des zuvor gelieferten `ETag` — und Status sowie Bytes des zweiten Aufrufs erfassen, um die Wirkung von HTTP-Caching sichtbar zu machen.

#### Scenario: Revalidierung ohne Caching

- **WHEN** eine Referenzroute ohne serverseitiges ETag zweimal abgerufen wird
- **THEN** liefert der zweite Aufruf denselben Status (200) und die volle Bytezahl

#### Scenario: Revalidierung mit Caching

- **WHEN** eine Referenzroute mit ETag zweimal abgerufen wird und der Datenstand unverändert ist
- **THEN** liefert der zweite Aufruf `304` und einen leeren Body

### Requirement: Messung des SSE-Fan-out pro Mutation

Das Werkzeug SHALL einen festen Satz von 8 benannten Clients (C1..C8) mit den in `design.md` festgelegten Vereinsfunktionen und Team-Zugehörigkeiten an `/api/events` abonnieren, je Messung genau eine feste Mutation (`members`, `games(T1)`, `settings`) auslösen und je Client die im Zeitfenster gelieferten Events (und deren Bytes) zählen. Kennzahl ist die Zahl der zugestellten Events pro Mutation, aufgeschlüsselt nach Client.

#### Scenario: Globaler Fan-out zählt alle 8 Clients

- **WHEN** die 8 Clients abonniert sind und eine Mutation ein global gebroadcastetes Event auslöst
- **THEN** erfasst das Werkzeug 8 zugestellte Events

#### Scenario: Gescopetes Event zählt nur Berechtigte

- **WHEN** die 8 Clients abonniert sind und eine Mutation ein gescopetes Event auslöst
- **THEN** erfasst das Werkzeug ausschließlich die Zustellungen an die laut Audience-Regel berechtigten Clients
- **AND** die Aufschlüsselung weist aus, welche der C1..C8 das Event erhielten

#### Scenario: Fester Client-Roster

- **WHEN** das Werkzeug den Fan-out-Satz aufbaut
- **THEN** haben C1..C8 die in `design.md` festgelegten Funktionen/Teams — reproduzierbar über Läufe hinweg

### Requirement: Report und versionierte Baseline

Das System SHALL ein Makefile-Target `make measure` bereitstellen, das die Messungen ausführt und einen Report (`metrics/PAYLOAD.md`) mit Routen-, Revalidierungs- und Fan-out-Tabelle schreibt und mit Exit 0 endet. Eine committete Baseline (`metrics/payload-baseline.md`) SHALL den Referenzstand festhalten.

#### Scenario: Report wird erzeugt

- **WHEN** `make measure` ausgeführt wird
- **THEN** entsteht `metrics/PAYLOAD.md` mit den drei Tabellen
- **AND** das Target endet mit Exit-Code 0

### Requirement: Messung verändert keine Produktionspfade

Das Werkzeug SHALL rein beobachtend sein und weder Produktions-Code-Pfade noch Auth-/Sichtbarkeitsregeln verändern.

#### Scenario: Keine Seiteneffekte auf Produktionscode

- **WHEN** das Mess-Werkzeug ausgeführt wird
- **THEN** werden keine Produktions-Handler modifiziert und keine Autorisierungsregeln umgangen

