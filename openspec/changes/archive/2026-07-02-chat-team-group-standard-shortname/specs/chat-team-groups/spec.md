## MODIFIED Requirements

### Requirement: Sichtbare Team-Standard-Gruppen auflisten

Das System SHALL einen Endpoint `GET /api/chat/team-groups` bereitstellen, der für den aufrufenden User die sichtbaren Standard-Gruppen liefert. Ein Eintrag besteht aus `teamId`, `displayShort` (kanonische Team-Kurzform), `kind` (`trainer`/`spieler`/`eltern`) und `count` (Anzahl der Mitglieder ohne den Caller). Es werden nur Gruppen der aktiven Saison (`seasons.is_active = 1`) berücksichtigt. Einträge mit `count = 0` werden weggelassen.

`displayShort` MUSS die kanonische Standard-Kurzform des Teams sein: Geschlecht (`m`/`w`/`g`) + erster Buchstabe der Altersklasse + Team-Nummer **genau dann, wenn in der aktiven Saison mehrere Teams dieselbe Altersklasse und dasselbe Geschlecht teilen**. Diese Disambiguierung MUSS **saisonweit** über alle Teams gezählt werden — unabhängig davon, welche Teams der Caller sehen darf. Sie ist damit identisch zur Kurzform, die andere UIs (Kalender, Termine, Dienstbörse) für dasselbe Team anzeigen.

Sichtbarkeitsregel: Ein User sieht eine Standard-Gruppe genau dann, wenn er Rolle `admin` hat ODER die Vereinsfunktion `vorstand` ODER `sportliche_leitung` hat ODER in `user_accessible_teams` für das Team in der aktiven Saison eingetragen ist. Die Sichtbarkeit und das `count`-Feld bleiben caller-scoped; nur die Disambiguierung von `displayShort` ist saisonweit.

#### Scenario: Spieler sieht eigenes Team mit drei Kinds

- **WHEN** ein Spieler des Teams T1 (aktive Saison) `GET /api/chat/team-groups` aufruft
- **THEN** enthält die Antwort genau drei Einträge für T1 (Trainer, Spieler, Eltern), sofern jedes Kind mindestens ein Mitglied außer dem Caller hat

#### Scenario: Kurzform bleibt disambiguiert trotz eingeschränktem Zugriff

- **WHEN** in der aktiven Saison zwei männliche B-Jugend-Teams existieren (`mB1` und `mB2`), der Caller aber nur auf `mB2` Zugriff hat und `GET /api/chat/team-groups` aufruft
- **THEN** ist `displayShort` für die Einträge von `mB2` gleich `"mB2"` (die Team-Nummer wird trotz nur eines sichtbaren Teams beibehalten, weil saisonweit zwei Teams die Altersklasse+Geschlecht teilen)

#### Scenario: Kurzform ohne Nummer bei eindeutigem Team

- **WHEN** in der aktiven Saison genau ein männliches B-Jugend-Team existiert und der Caller es sieht
- **THEN** ist `displayShort` gleich `"mB"` (keine Team-Nummer, da die Altersklasse+Geschlecht saisonweit eindeutig ist)

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
