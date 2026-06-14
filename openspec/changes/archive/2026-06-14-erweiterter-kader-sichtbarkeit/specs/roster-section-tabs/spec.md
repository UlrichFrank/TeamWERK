## ADDED Requirements

### Requirement: Team-Tab zeigt Abschnitt „Erweiterter Kader"

Der Team-Tab auf der Mein-Team-Seite SHALL unterhalb der regulären Spielertabelle einen Abschnitt „Erweiterter Kader" anzeigen, wenn `extended_players` in der Roster-Antwort mindestens einen Eintrag enthält. Ist `extended_players` leer, wird kein Abschnitt gerendert (kein Leer-Text, kein leerer Block).

#### Scenario: Team hat abgesetzte Spieler

- **WHEN** `GET /api/teams/{id}/roster` gibt `extended_players` mit mindestens einem Eintrag zurück
- **WHEN** der Nutzer den Tab „Team" aktiviert
- **THEN** zeigt die Karte unterhalb der regulären Spielertabelle einen Abschnitt mit Heading „Erweiterter Kader"
- **THEN** listet der Abschnitt die abgesetzten Spieler mit Trikotnummer und Name (gleiche Spalten wie reguläre Spieler)

#### Scenario: Team hat keine abgesetzten Spieler

- **WHEN** `GET /api/teams/{id}/roster` gibt `extended_players: []` zurück
- **WHEN** der Nutzer den Tab „Team" aktiviert
- **THEN** zeigt die Karte keinen „Erweiterter Kader"-Abschnitt
