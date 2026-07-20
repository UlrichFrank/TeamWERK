## ADDED Requirements

### Requirement: Teilnehmer sehen Training-RSVP-Buttons unabhängig von Response

Endpoints `GET /api/training-sessions` und `GET /api/training-sessions/{id}` SHALL pro Session-Objekt ein boolesches Feld `am_i_participant` liefern. Das Feld ist `true` genau dann, wenn der aufrufende User selbst im regulären Kader (`kader_members`), im erweiterten Kader (`kader_extended_members`) oder als Trainer (`kader_trainers`) des Teams der Session für deren Saison eingetragen ist. Für Nicht-Teilnehmer ist `am_i_participant` `false`.

Die Frontend-Anzeige der eigenen RSVP-Buttons auf `/termine` SHALL an `am_i_participant` gebunden sein — **nicht** an `my_rsvp !== null`. Bei erreichtem 2h-Cutoff bleiben die Buttons sichtbar und sind `disabled` mit erklärender Notice; Cutoff-Berechtigte (admin/vorstand/trainer/sportliche_leitung) sind davon ausgenommen.

Für Eltern gilt: Die Kind-Zeilen (`children_rsvp`) sind bereits heute kader-basiert und bleiben unverändert. Ein Elternteil ohne eigene Kader-Zugehörigkeit sieht `am_i_participant=false` für sich selbst (keine Eigen-Buttons), aber weiterhin die Buttons pro Kind.

#### Scenario: Spieler ohne Response sieht `am_i_participant=true`
- **WHEN** ein Spieler im regulären Kader eines Teams `GET /api/training-sessions` aufruft und für eine Session mit `rsvp_default_players='none'` noch keine Response existiert
- **THEN** enthält das Session-Objekt `am_i_participant=true` und `my_rsvp=null`

#### Scenario: Erweiterter Kader-Spieler sieht `am_i_participant=true`
- **WHEN** ein Spieler nur über `kader_extended_members` dem Team der Session zugeordnet ist
- **THEN** ist `am_i_participant=true`

#### Scenario: Trainer sieht `am_i_participant=true`
- **WHEN** ein User via `kader_trainers` als Trainer des Teams eingetragen ist
- **THEN** ist `am_i_participant=true`

#### Scenario: Fremder Nutzer sieht `am_i_participant=false`
- **WHEN** ein User ohne Kader-Beziehung zum Team der Session diese sieht
- **THEN** ist `am_i_participant=false`
- **THEN** zeigt das Frontend keine eigenen RSVP-Buttons für ihn

#### Scenario: Elternteil ohne eigene Kader-Rolle
- **WHEN** ein Elternteil ohne eigene Kader-Zugehörigkeit die Termine-Seite aufruft
- **THEN** ist `am_i_participant=false` für alle Sessions
- **THEN** sieht der Elternteil ausschließlich Kind-Zeilen mit Buttons

#### Scenario: Spieler-Buttons sichtbar aber gesperrt nach Cutoff
- **WHEN** ein Spieler mit `am_i_participant=true` eine Session innerhalb der letzten 2 Stunden vor Beginn aufruft und den Cutoff nicht überschreiben darf
- **THEN** rendert das Frontend die drei RSVP-Buttons sichtbar, aber `disabled`, mit erklärender Notice
