## MODIFIED Requirements

### Requirement: Sicht-Berechtigung pro Team

Ein Video MUST genau einem Team zugeordnet sein (`team_id NOT NULL`). Sichtbar für einen Nutzer ist ein Video genau dann, wenn er Rolle `admin` hat, Vereinsfunktion `vorstand` hat, aktiver Spieler des Teams ist (`team_memberships.active = 1`), Trainer des Teams ist (`team_trainers`), Elternteil eines aktiven Spielers des Teams ist (über `family_links`), **Mitglied des erweiterten Kaders des Teams ist** (`kader_extended_members`, aktive Saison) **oder Elternteil eines solchen Mitglieds** (über `family_links`).

Für die beiden Zweige des erweiterten Kaders SHALL als Statusfilter `members.status <> 'ausgetreten'` gelten, NICHT `= 'aktiv'`: Förderkinder tragen `members.status = 'foerderkind'` und sind die typische Besetzung des erweiterten Kaders — ein Gleichheitsfilter auf `aktiv` schlösse genau die Zielgruppe aus.

Die Berechtigung SHALL für Liste, Detailabruf und Stream dieselbe sein. Sie gibt ausschließlich Lese- und Streamzugriff; Hochladen, Ändern und Löschen bleiben Trainer, Vorstand und Admin vorbehalten.

#### Scenario: Spieler sieht eigenes Team
- **WHEN** ein aktiver Spieler von Team A `GET /api/videos` aufruft
- **THEN** enthält die Liste Videos mit `team_id = A` und keine Videos anderer Teams

#### Scenario: Elternteil sieht Team des Kindes
- **WHEN** ein Elternteil eines aktiven Spielers von Team A `GET /api/videos?team_id=A` aufruft
- **THEN** enthält die Liste die berechtigten Videos von Team A

#### Scenario: Erweiterter Kader sieht die Videos seiner Mannschaft
- **WHEN** ein Mitglied, das nur im erweiterten Kader von Team A steht, `GET /api/videos` aufruft
- **THEN** enthält die Liste die Videos von Team A
- **THEN** liefert `GET /api/videos/{id}` eines dieser Videos HTTP 200 und keinen 403

#### Scenario: Elternteil eines erweiterten Kader-Mitglieds
- **WHEN** ein Elternteil eines nur erweiterten Mitglieds von Team A die Videoliste abruft
- **THEN** enthält sie die Videos von Team A

#### Scenario: Förderkind wird nicht über den Status ausgefiltert
- **WHEN** ein Mitglied im erweiterten Kader von Team A den Status `foerderkind` trägt
- **THEN** sieht es die Videos von Team A

#### Scenario: Ausgetretenes erweitertes Mitglied
- **WHEN** ein Mitglied im erweiterten Kader von Team A den Status `ausgetreten` trägt
- **THEN** sieht es die Videos von Team A nicht

#### Scenario: Spieler fragt fremdes Team an
- **WHEN** ein Spieler von Team A `GET /api/videos?team_id=B` aufruft und nicht zu Team B gehört
- **THEN** ist Team B nicht in der Antwort enthalten (leere oder gefilterte Liste)

#### Scenario: Inaktive Mitgliedschaft
- **WHEN** ein Nutzer war Spieler in Team A, ist aber nicht mehr aktiv
- **THEN** sieht er die Videos von Team A nicht mehr
