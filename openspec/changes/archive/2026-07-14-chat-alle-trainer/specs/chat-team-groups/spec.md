## ADDED Requirements

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
