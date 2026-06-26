# team-name-display Specification

## Purpose

Diese Spezifikation beschreibt die Capability `team-name-display`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Server liefert Display-Strings für Teamnamen

API-Endpoints, die Teams in Listen- oder Detail-Items enthalten, SHALL pro Team zwei Display-Strings ausliefern: `display_short` (z. B. „mA1") und `display_long` (z. B. „B-Jugend 2 männlich"). Bei Multi-Team-Items SHALL der Endpoint zusätzlich `team_display_short_csv` und `team_display_long_csv` (alphabetisch sortiert, komma-getrennt) ausliefern.

#### Scenario: Einzel-Team in DutyBoard-Group
- **WHEN** `GET /api/duty-board` aufgerufen wird und eine Group ist genau einem Team zugeordnet
- **THEN** enthält das Group-Item ein Feld `team_display_short` mit dem Kurznamen (z. B. „mA1") und ein Feld `team_display_long` mit dem Langnamen

#### Scenario: Doppelheimspiel in Games-Liste
- **WHEN** `GET /api/games` aufgerufen wird und ein Spiel referenziert zwei Teams gleicher age_class+gender
- **THEN** enthält das Game-Item `team_display_short_csv = "mA1, mA2"` und `team_display_long_csv = "B-Jugend 1 männlich, B-Jugend 2 männlich"`

#### Scenario: Spielgemeinschaft mit verschiedenen age_class
- **WHEN** ein Spiel referenziert zwei Teams unterschiedlicher age_class
- **THEN** enthält das Game-Item beide Display-Strings je Team in den `*_csv`-Feldern, alphabetisch sortiert

### Requirement: Kurzform leitet sich aus Saison-Kader ab

Die Kurzform eines Teams SHALL aus der aktiven Saison (`seasons.is_active=1`) berechnet werden: `<gender-Initial m/w/g><erste age_class-Buchstabe><team_number falls mehrere Teams gleicher Kombi>`.

#### Scenario: Eindeutiger Team-Identifier
- **WHEN** in der aktiven Saison nur ein Team mit `age_class='B-Jugend'` und `gender='m'` existiert
- **THEN** ist `display_short` für dieses Team „mB" (ohne team_number-Suffix)

#### Scenario: Mehrere Teams gleicher Kombi
- **WHEN** in der aktiven Saison zwei Teams mit `age_class='B-Jugend'` und `gender='m'` existieren (team_number 1 und 2)
- **THEN** ist `display_short` „mB1" bzw. „mB2"

#### Scenario: Kein Kader-Eintrag in aktiver Saison
- **WHEN** ein Team hat keinen `kader`-Eintrag in der aktiven Saison
- **THEN** ist `display_short` NULL und der Aufrufer fällt auf `teams.name` zurück (COALESCE-Pattern)

### Requirement: Frontend nutzt `formatTeamList`-Helper

Das Frontend SHALL Teamnamen ausschließlich über den zentralen Helper `formatTeamList(teams, mode)` rendern. Hardcoded Strings wie `'Mehrere'` oder `'Mehrere Teams'` sind außerhalb des Helpers nicht zulässig.

#### Scenario: Listen-Anzeige
- **WHEN** eine Listen-Seite (Termine, Mitfahrten, Chat-Filter, EventInfoModal) Teamnamen für ein Item mit 1..n Teams rendert
- **THEN** wird `formatTeamList(teams, 'short')` aufgerufen und gibt die komma-getrennte Liste der Kurznamen zurück

#### Scenario: Detail-Anzeige
- **WHEN** eine Detail-Seite (SpieltagDetailPage, TermineDetailPage Training, MeinTeam) Teamnamen rendert
- **THEN** wird `formatTeamList(teams, 'long')` aufgerufen und gibt die komma-getrennte Liste der Langnamen zurück

### Requirement: Kalender-Tile zeigt „Mehrere" als bewusste Ausnahme

Das Kalender-Spiel-Tile in `KalenderPage` SHALL bei einem Team den Kurznamen, bei mehr als einem Team den String „Mehrere" anzeigen (Inline-Label und Tooltip-Variante „Mehrere Teams"). Dieser Sonderfall ist ausdrücklich nur für das Kalender-Tile zulässig, da der verfügbare Platz die vollständige Auflistung nicht erlaubt.

#### Scenario: Einzelspiel im Kalender-Tile
- **WHEN** ein Spiel mit genau einem Team auf einer Kalender-Kachel gerendert wird
- **THEN** zeigt das Tile-Label den Kurznamen des Teams (z. B. „mA1")

#### Scenario: Doppelheimspiel im Kalender-Tile
- **WHEN** ein Spiel mit zwei oder mehr Teams auf einer Kalender-Kachel gerendert wird
- **THEN** zeigt das Tile-Label den String „Mehrere" und der Tooltip „Mehrere Teams"

#### Scenario: Aufruf via Helper
- **WHEN** das Kalender-Tile Teamnamen rendert
- **THEN** geschieht das ausschließlich über `formatTeamList(teams, 'kalender')`

### Requirement: Dashboard listet alle Teams eines Spiels auf

Der Endpoint `GET /api/dashboard` SHALL bei einem Spiel mit mehreren Teams alle Teams im `teamName`-Feld auflisten (Kurzform, komma-getrennt). Die bisherige `MIN()`-Aggregation, die nur ein Team zurückliefert, ist nicht mehr zulässig.

#### Scenario: Doppelheimspiel im Dashboard
- **WHEN** das Dashboard für einen User mit zwei Teams im Kader ein Doppelheimspiel im Time-Window enthält
- **THEN** enthält das Event-Item `teamName = "mA1, mA2"` (oder analog) und nicht nur eines der beiden Teams

#### Scenario: Einzelspiel im Dashboard
- **WHEN** das Dashboard ein Einzelspiel enthält
- **THEN** enthält `teamName` exakt den Kurznamen dieses einen Teams

### Requirement: SpieltagDetailPage rendert vorhandene Team-Daten

`SpieltagDetailPage` SHALL Teamnamen aus dem `teams[]`-Array der API-Response (`GET /api/games/{id}`) lesen und im Langform-Modus rendern. Die bisherige Referenz auf ein nicht existierendes `team_name`-Feld ist zu entfernen.

#### Scenario: Detail-Anzeige eines Spiels mit einem Team
- **WHEN** der User die Detail-Seite eines Spiels mit einem Team öffnet
- **THEN** zeigt der Header den Langnamen des Teams (z. B. „B-Jugend 2 männlich")

#### Scenario: Detail-Anzeige eines Doppelheimspiels
- **WHEN** der User die Detail-Seite eines Doppelheimspiels öffnet
- **THEN** zeigt der Header beide Langnamen komma-getrennt
