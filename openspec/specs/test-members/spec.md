# test-members Specification

## Purpose
Fachliche Testabdeckung für die Mitgliederverwaltung: paginierte Mitgliederliste mit Suche und Ausblendung Ausgetretener, Trainer-Scope, Familienlink-Verwaltung sowie Proxy-Account-Erstellung.

## Requirements

### Requirement: Mitgliederliste mit Paginierung und Suche
Das System SHALL Mitglieder mit Offset/Limit-Paginierung und serverseitiger Suche liefern. Ausgetretene Mitglieder (status='ausgetreten') MÜSSEN immer ausgeblendet werden.

#### Scenario: Paginierung funktioniert
- **WHEN** Vorstand GET /api/members?limit=10&offset=10 bei 25 aktiven Mitgliedern
- **THEN** 10 Einträge im items-Array, total=25

#### Scenario: Suche nach Name
- **WHEN** Vorstand GET /api/members?search=müller
- **THEN** Nur Mitglieder mit "müller" im Namen

#### Scenario: Ausgetretene nicht sichtbar
- **WHEN** Vorstand GET /api/members bei 2 aktiven + 1 ausgetretenem Mitglied
- **THEN** Nur 2 Mitglieder, total=2

### Requirement: Trainer-Scope für Mitgliederliste
Das System SHALL Trainern nur Mitglieder des eigenen Teams anzeigen (via kader_trainers-Verknüpfung).

#### Scenario: Trainer sieht nur eigenes Team
- **WHEN** Trainer (via kader_trainers mit Team A verknüpft) GET /api/members
- **THEN** Nur Mitglieder von Team A

### Requirement: Familienlink-Verwaltung
Das System SHALL maximal 2 Erziehungsberechtigte pro Mitglied erlauben. Ein doppelt angelegter Link wird via INSERT OR IGNORE still ignoriert. Ein nicht existierender Link liefert HTTP 404 beim Löschen.

#### Scenario: Familienlink anlegen
- **WHEN** POST /api/admin/family-links mit gültigem parent_user_id und member_id
- **THEN** HTTP 204, family_links-Eintrag vorhanden

#### Scenario: Dritter Elternteil abgelehnt
- **WHEN** POST /api/admin/family-links wenn Mitglied bereits 2 Elternteile hat
- **THEN** HTTP 409

#### Scenario: Duplikat ist idempotent
- **WHEN** POST /api/admin/family-links mit bereits bestehendem Link
- **THEN** HTTP 204, weiterhin nur 1 Eintrag in DB

#### Scenario: Nicht-existierenden Link löschen
- **WHEN** DELETE /api/admin/family-links mit nicht vorhandenem Link
- **THEN** HTTP 404

### Requirement: Proxy-Account-Erstellung
Das System SHALL für Mitglieder ohne Account einen Proxy-User anlegen (can_login=0). Ist bereits ein User verknüpft, schlägt die Anfrage mit HTTP 409 fehl.

#### Scenario: Proxy-Account anlegen
- **WHEN** POST /api/admin/members/{id}/proxy-account für Mitglied ohne user_id
- **THEN** HTTP 201, neuer User mit can_login=0, members.user_id aktualisiert

#### Scenario: Doppelter Proxy abgelehnt
- **WHEN** POST /api/admin/members/{id}/proxy-account für Mitglied mit vorhandenem user_id
- **THEN** HTTP 409
