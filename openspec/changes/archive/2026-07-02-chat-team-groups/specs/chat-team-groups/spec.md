## ADDED Requirements

### Requirement: Sichtbare Team-Standard-Gruppen auflisten

Das System SHALL einen Endpoint `GET /api/chat/team-groups` bereitstellen, der für den aufrufenden User die sichtbaren Standard-Gruppen liefert. Ein Eintrag besteht aus `teamId`, `teamName`, `kind` (`trainer`/`spieler`/`eltern`) und `count` (Anzahl der Mitglieder ohne den Caller). Es werden nur Gruppen der aktiven Saison (`seasons.is_active = 1`) berücksichtigt. Einträge mit `count = 0` werden weggelassen.

Sichtbarkeitsregel: Ein User sieht eine Standard-Gruppe genau dann, wenn er Rolle `admin` hat ODER die Vereinsfunktion `vorstand` ODER `sportliche_leitung` hat ODER in `user_accessible_teams` für das Team in der aktiven Saison eingetragen ist.

#### Scenario: Spieler sieht eigenes Team mit drei Kinds

- **WHEN** ein Spieler des Teams T1 (aktive Saison) `GET /api/chat/team-groups` aufruft
- **THEN** enthält die Antwort genau drei Einträge für T1 (Trainer, Spieler, Eltern), sofern jedes Kind mindestens ein Mitglied außer dem Caller hat

#### Scenario: Vorstand sieht alle Teams der aktiven Saison

- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/chat/team-groups` aufruft
- **THEN** enthält die Antwort Einträge für alle Teams in der aktiven Saison × verfügbare Kinds

#### Scenario: Sportliche Leitung sieht alle Teams der aktiven Saison

- **WHEN** ein User mit Vereinsfunktion `sportliche_leitung` `GET /api/chat/team-groups` aufruft
- **THEN** enthält die Antwort Einträge für alle Teams in der aktiven Saison

#### Scenario: Inaktive Saisons werden ausgeblendet

- **WHEN** ein Trainer einer **inaktiven** Saison den Endpoint aufruft und in keiner aktiven Saison eingetragen ist
- **THEN** ist die Liste leer

#### Scenario: Caller wird nicht mitgezählt

- **WHEN** ein Spieler `GET /api/chat/team-groups` aufruft und sein Team hat 14 Spieler inkl. ihm selbst
- **THEN** ist `count` für `kind=spieler` gleich 13

### Requirement: Mitglieder einer Standard-Gruppe auflösen

Das System SHALL einen Endpoint `GET /api/chat/team-groups/{teamId}/{kind}/members` bereitstellen, der für eine sichtbare Standard-Gruppe die einzelnen Mitglieder als `[{id, name}, …]` zurückgibt. `name` ist `first_name + ' ' + last_name`. Der Caller selbst wird aus der Liste gefiltert. `kind` MUSS einer von `trainer`, `spieler`, `eltern` sein. Bei nicht sichtbarem Team oder unbekanntem `kind` SHALL der Server entsprechend mit 403 bzw. 400 antworten.

Auflösungs-Regeln:
- `trainer` liefert User aus `kader_trainers JOIN members` für Kader der aktiven Saison des Teams
- `spieler` liefert User aus `kader_members JOIN members` UND `kader_extended_members JOIN members` für Kader der aktiven Saison des Teams (Dedup nach `user_id`)
- `eltern` liefert User aus `family_links` (`parent_user_id`), deren `member_id` in `kader_members ∪ kader_extended_members` der aktiven Saison des Teams ist (Dedup nach `user_id`)

#### Scenario: Trainer eines Teams liest Trainer-Gruppe

- **WHEN** ein Trainer von T1 `GET /api/chat/team-groups/T1/trainer/members` aufruft
- **THEN** wird HTTP 200 zurückgegeben mit allen Trainern des Teams in der aktiven Saison außer dem Caller selbst

#### Scenario: Spieler-Gruppe enthält Stamm- und Erweiterten-Kader

- **WHEN** `GET /api/chat/team-groups/T1/spieler/members` aufgerufen wird
- **THEN** sind in der Antwort sowohl User aus `kader_members` als auch aus `kader_extended_members` enthalten (Dedup nach `user_id`)

#### Scenario: Eltern-Gruppe enthält Eltern beider Spieler-Quellen

- **WHEN** `GET /api/chat/team-groups/T1/eltern/members` aufgerufen wird
- **THEN** sind Eltern (`family_links.parent_user_id`) sowohl von `kader_members` als auch von `kader_extended_members` enthalten

#### Scenario: Spieler ruft fremdes Team auf

- **WHEN** ein Spieler von T1 `GET /api/chat/team-groups/T2/spieler/members` aufruft und in T2 nicht eingetragen ist
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Ungültiger Kind-Wert

- **WHEN** ein authentifizierter User `GET /api/chat/team-groups/T1/foobar/members` aufruft
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Caller wird aus Mitgliederliste gefiltert

- **WHEN** ein Trainer von T1 die Trainer-Gruppe seines Teams auflöst
- **THEN** taucht er selbst nicht in der zurückgegebenen Liste auf
