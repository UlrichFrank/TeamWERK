## MODIFIED Requirements

### Requirement: Sichtbare Team-Standard-Gruppen auflisten

Das System SHALL einen Endpoint `GET /api/chat/team-groups` bereitstellen, der für den
aufrufenden User die sichtbaren Standard-Gruppen liefert. Ein Eintrag besteht aus
`groupType` (`team`/`practice`), `teamId`, `teamName`, `kind`
(`trainer`/`spieler`/`eltern`) und `count` (Anzahl der Mitglieder ohne den Caller). Es
werden nur Gruppen der aktiven Saison (`seasons.is_active = 1`) berücksichtigt. Einträge
mit `count = 0` werden weggelassen.

Bei `groupType = 'practice'` trägt `teamId` die `kader.id` der Übungsgruppe (die keine
`teams.id` besitzt) und `teamName` deren `kader.name`. Bei `groupType = 'team'` bleibt die
Bedeutung unverändert (`teams.id` / `teams.name`).

Sichtbarkeitsregel für `groupType = 'team'` (unverändert): Ein User sieht eine
Standard-Gruppe genau dann, wenn er Rolle `admin` hat ODER die Vereinsfunktion `vorstand`
ODER `sportliche_leitung` hat ODER in `user_accessible_teams` für das Team in der aktiven
Saison eingetragen ist.

Sichtbarkeitsregel für `groupType = 'practice'`: Ein User sieht die Gruppe genau dann,
wenn er Rolle `admin` hat ODER die Vereinsfunktion `vorstand` ODER `sportliche_leitung`
hat ODER selbst Mitglied (`kader_members`) oder Trainer (`kader_trainers`) der Gruppe ist
ODER über `family_links` Elternteil eines Mitglieds der Gruppe ist. Die View
`user_accessible_teams` SHALL dafür **nicht** herangezogen werden — sie filtert
`WHERE k.team_id IS NOT NULL` und kennt Übungsgruppen deshalb nicht.

#### Scenario: Spieler sieht eigenes Team mit drei Kinds
- **WHEN** ein Spieler des Teams T1 (aktive Saison) `GET /api/chat/team-groups` aufruft
- **THEN** enthält die Antwort genau drei Einträge für T1 (Trainer, Spieler, Eltern) mit
  `groupType='team'`, sofern jedes Kind mindestens ein Mitglied außer dem Caller hat

#### Scenario: Vorstand sieht alle Teams der aktiven Saison
- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/chat/team-groups` aufruft
- **THEN** enthält die Antwort Einträge für alle Teams **und** alle Übungsgruppen der
  aktiven Saison × verfügbare Kinds

#### Scenario: Sportliche Leitung sieht alle Teams der aktiven Saison
- **WHEN** ein User mit Vereinsfunktion `sportliche_leitung` `GET /api/chat/team-groups`
  aufruft
- **THEN** enthält die Antwort Einträge für alle Teams und Übungsgruppen der aktiven Saison

#### Scenario: Mitglied einer Übungsgruppe sieht deren Gruppe
- **WHEN** ein Mitglied einer Übungsgruppe den Endpoint aufruft
- **THEN** enthält die Antwort Einträge mit `groupType='practice'` und der `kader.id` der
  Gruppe

#### Scenario: Fremder sieht die Übungsgruppe nicht
- **WHEN** ein User ohne Mitgliedschaft, Trainerrolle oder Elternbeziehung zu einer
  Übungsgruppe den Endpoint aufruft und weder `admin`, `vorstand` noch
  `sportliche_leitung` ist
- **THEN** fehlt die Gruppe in der Antwort

#### Scenario: Inaktive Saisons werden ausgeblendet
- **WHEN** ein Trainer einer **inaktiven** Saison den Endpoint aufruft und in keiner
  aktiven Saison eingetragen ist
- **THEN** ist die Liste leer

#### Scenario: Caller wird nicht mitgezählt
- **WHEN** ein Spieler `GET /api/chat/team-groups` aufruft und sein Team hat 14 Spieler
  inkl. ihm selbst
- **THEN** ist `count` für `kind=spieler` gleich 13

## ADDED Requirements

### Requirement: Mitglieder einer Übungsgruppen-Standardgruppe auflösen

Das System SHALL einen Endpoint
`GET /api/chat/practice-groups/{id}/{kind}/members` bereitstellen, der für eine sichtbare
Übungsgruppe die einzelnen Mitglieder als `[{id, name}, …]` zurückgibt. `{id}` ist die
`kader.id` der Gruppe, `name` ist `first_name + ' ' + last_name`. Der Caller selbst wird
aus der Liste gefiltert. `kind` MUSS einer von `trainer`, `spieler`, `eltern` sein.

Auflösungs-Regeln:
- `trainer` → `kader_trainers JOIN members` der Gruppe
- `spieler` → `kader_members JOIN members` der Gruppe. Eine Union mit
  `kader_extended_members` SHALL **entfallen** — Übungsgruppen haben keinen erweiterten
  Kader.
- `eltern` → `family_links.parent_user_id` zu Mitgliedern aus `kader_members` der Gruppe

Bei nicht sichtbarer Gruppe SHALL das System mit HTTP 403 antworten, bei unbekanntem
`kind` mit HTTP 400, bei nicht existierender oder nicht-`practice`-Gruppe mit HTTP 404.

#### Scenario: Mitglied liest die Spieler-Gruppe
- **WHEN** ein Mitglied einer Übungsgruppe
  `GET /api/chat/practice-groups/{id}/spieler/members` aufruft
- **THEN** wird HTTP 200 mit allen Mitgliedern der Gruppe außer dem Caller zurückgegeben

#### Scenario: Trainer-Gruppe
- **WHEN** `GET /api/chat/practice-groups/{id}/trainer/members` aufgerufen wird
- **THEN** enthält die Antwort die Trainer aus `kader_trainers` der Gruppe

#### Scenario: Eltern-Gruppe
- **WHEN** `GET /api/chat/practice-groups/{id}/eltern/members` aufgerufen wird
- **THEN** enthält die Antwort die über `family_links` verknüpften Elternteile der
  Gruppenmitglieder

#### Scenario: Fremder wird abgewiesen
- **WHEN** ein User ohne Bezug zur Gruppe den Endpoint aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Mannschaftskader über die Übungsgruppen-Route
- **WHEN** die `{id}` eines Kaders mit `kind='team'` übergeben wird
- **THEN** antwortet das System mit HTTP 404
