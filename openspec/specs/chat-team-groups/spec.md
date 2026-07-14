# chat-team-groups Specification

## Purpose
TBD - created by archiving change chat-team-groups. Update Purpose after archive.
## Requirements
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

### Requirement: „Alle Trainer"-Standard-Gruppe auflisten

Das System SHALL in `GET /api/chat/team-groups` für Mitglieder des Zugriffskreises zusätzlich eine synthetische Kachel „Alle Trainer" liefern: `{ teamId: 0, displayShort: "Alle Trainer", kind: "alle_trainer", count }`, wobei `count` die Anzahl der **Kader-Trainer der aktiven Saison** ohne den Caller ist. Ist `count = 0`, wird die Kachel weggelassen.

Zwei Mengen sind zu unterscheiden:
- **Zugriffskreis** (Sichtbarkeit/Berechtigung): User, die (a) Trainer eines Kaders der **aktiven** Saison sind (`kader_trainers`, aktive Saison, `members.user_id IS NOT NULL`) ODER die Vereinsfunktion (b) `vorstand`, (c) `sportliche_leitung` ODER (d) `vorstand_beisitzer` haben. `admin` ist stets berechtigt.
- **Mitgliedermenge** (Inhalt der Gruppe): **nur** Bedingung (a) — Kader-Trainer der aktiven Saison. Vorstand, sportliche Leitung und vorstand_beisitzer sind NICHT enthalten, es sei denn, sie sind selbst Kader-Trainer dieser Saison.

Sichtbarkeitsregel: Die Kachel erscheint genau dann, wenn der Caller Rolle `admin` hat ODER im Zugriffskreis ist.

#### Scenario: Trainer sieht die „Alle Trainer"-Kachel

- **WHEN** ein Kader-Trainer der aktiven Saison `GET /api/chat/team-groups` aufruft
- **THEN** enthält die Antwort einen Eintrag `{ teamId: 0, kind: "alle_trainer", displayShort: "Alle Trainer" }`
- **THEN** ist `count` die Anzahl aller Kader-Trainer der aktiven Saison außer dem Caller

#### Scenario: Sportliche Leitung sieht die Kachel

- **WHEN** ein User mit Vereinsfunktion `sportliche_leitung` (kein Kader-Trainer) den Endpoint aufruft
- **THEN** enthält die Antwort die „Alle Trainer"-Kachel

#### Scenario: vorstand_beisitzer sieht die Kachel

- **WHEN** ein User mit Vereinsfunktion `vorstand_beisitzer` (kein Kader-Trainer) den Endpoint aufruft
- **THEN** enthält die Antwort die „Alle Trainer"-Kachel

#### Scenario: Nicht-Zugriffskreis-User sieht die Kachel nicht

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-/Beisitzer-Zugehörigkeit den Endpoint aufruft
- **THEN** enthält die Antwort KEINEN `alle_trainer`-Eintrag

### Requirement: „Alle Trainer"-Gruppe auflösen

Das System SHALL `GET /api/chat/team-groups/0/alle_trainer/members` bereitstellen (Sentinel `teamId = 0`, `kind = alle_trainer`), das die Mitgliedermenge als `[{id, name}, …]` zurückgibt (`name = first_name + ' ' + last_name`), den Caller ausgefiltert. Die Mitgliedermenge umfasst **ausschließlich** Trainer **aller** Kader der aktiven Saison (identische Logik wie `kind=trainer`, jedoch ohne Team-Filter, Dedup nach `user_id`); Vorstand, sportliche Leitung und vorstand_beisitzer erscheinen NUR, wenn sie selbst Kader-Trainer sind. Statt der teambezogenen `canSeeTeamGroup`-Prüfung gilt: der Caller MUSS im Zugriffskreis sein ODER `admin`, sonst antwortet der Server mit HTTP 403.

#### Scenario: Trainer löst „Alle Trainer" auf (teamübergreifend)

- **WHEN** ein Kader-Trainer von T1 `GET /api/chat/team-groups/0/alle_trainer/members` aufruft und T2 hat einen eigenen Trainer
- **THEN** wird HTTP 200 zurückgegeben mit den Trainern aller Teams der aktiven Saison (inkl. dem Trainer von T2), ohne den Caller

#### Scenario: Reiner Vorstand ist nicht im Ergebnis

- **WHEN** „Alle Trainer" aufgelöst wird und ein User nur `vorstand` (bzw. `sportliche_leitung`/`vorstand_beisitzer`) ist, ohne Kader-Trainer-Zuordnung
- **THEN** taucht dieser User NICHT in der Mitgliederliste auf

#### Scenario: Vorstand darf auflösen (Zugriff), Ergebnis sind Trainer

- **WHEN** ein reiner Vorstand `GET /api/chat/team-groups/0/alle_trainer/members` aufruft
- **THEN** antwortet der Server mit HTTP 200 und liefert die Kader-Trainer der aktiven Saison

#### Scenario: Nicht-Zugriffskreis-User wird abgewiesen

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-/Beisitzer-Zugehörigkeit `GET /api/chat/team-groups/0/alle_trainer/members` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Caller wird ausgefiltert

- **WHEN** ein Kader-Trainer „Alle Trainer" auflöst
- **THEN** taucht er selbst nicht in der Liste auf

