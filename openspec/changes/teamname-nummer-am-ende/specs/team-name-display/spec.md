# Spec Delta: team-name-display

## MODIFIED Requirements

### Requirement: Server liefert Display-Strings für Teamnamen

API-Endpoints, die Teams in Listen- oder Detail-Items enthalten, SHALL pro Team zwei Display-Strings ausliefern: `display_short` (z. B. „mA1") und `display_long` (z. B. „B-Jugend männlich 2"). Bei Multi-Team-Items SHALL der Endpoint zusätzlich `team_display_short_csv` und `team_display_long_csv` (alphabetisch sortiert, komma-getrennt) ausliefern.

#### Scenario: Einzel-Team in DutyBoard-Group
- **WHEN** `GET /api/duty-board` aufgerufen wird und eine Group ist genau einem Team zugeordnet
- **THEN** enthält das Group-Item ein Feld `team_display_short` mit dem Kurznamen (z. B. „mA1") und ein Feld `team_display_long` mit dem Langnamen

#### Scenario: Doppelheimspiel in Games-Liste
- **WHEN** `GET /api/games` aufgerufen wird und ein Spiel referenziert zwei Teams gleicher age_class+gender
- **THEN** enthält das Game-Item `team_display_short_csv = "mA1, mA2"` und `team_display_long_csv = "B-Jugend männlich 1, B-Jugend männlich 2"`

#### Scenario: Spielgemeinschaft mit verschiedenen age_class
- **WHEN** ein Spiel referenziert zwei Teams unterschiedlicher age_class
- **THEN** enthält das Game-Item beide Display-Strings je Team in den `*_csv`-Feldern, alphabetisch sortiert

### Requirement: SpieltagDetailPage rendert vorhandene Team-Daten

`SpieltagDetailPage` SHALL Teamnamen aus dem `teams[]`-Array der API-Response (`GET /api/games/{id}`) lesen und im Langform-Modus rendern. Die bisherige Referenz auf ein nicht existierendes `team_name`-Feld ist zu entfernen.

#### Scenario: Detail-Anzeige eines Spiels mit einem Team
- **WHEN** der User die Detail-Seite eines Spiels mit einem Team öffnet
- **THEN** zeigt der Header den Langnamen des Teams (z. B. „B-Jugend männlich 2")

#### Scenario: Detail-Anzeige eines Doppelheimspiels
- **WHEN** der User die Detail-Seite eines Doppelheimspiels öffnet
- **THEN** zeigt der Header beide Langnamen komma-getrennt

## ADDED Requirements

### Requirement: Langform stellt die Mannschaftsnummer ans Ende

Die Langform eines Teams (`display_long`) SHALL aus dem Kader der aktiven Saison nach dem Schema `<Altersklasse> <Geschlecht> <Nummer>` gebildet werden, mit `Geschlecht` ∈ {„männlich", „weiblich", „gemischt"}. Die Nummer SHALL genau dann angehängt werden, wenn in der aktiven Saison mehr als ein Kader derselben Altersklasse und desselben Geschlechts existiert — dann trägt **jede** dieser Mannschaften ihre Nummer, auch Mannschaft 1. Eine Schreibweise mit der Nummer zwischen Altersklasse und Geschlecht („C-Jugend 1 männlich") SHALL nirgends mehr ausgeliefert werden.

#### Scenario: Zwei Mannschaften derselben Kombination
- **WHEN** in der aktiven Saison zwei Kader mit `age_class='C-Jugend'`, `gender='m'` und `team_number` 1 bzw. 2 existieren
- **THEN** ist `display_long` „C-Jugend männlich 1" bzw. „C-Jugend männlich 2"

#### Scenario: Einzige Mannschaft der Kombination
- **WHEN** in der aktiven Saison genau ein Kader mit `age_class='A-Jugend'`, `gender='f'` existiert
- **THEN** ist `display_long` „A-Jugend weiblich" (ohne Nummer)

#### Scenario: Kein Kader in der aktiven Saison
- **WHEN** ein Team hat keinen `kader`-Eintrag in der aktiven Saison
- **THEN** fällt `display_long` auf den gespeicherten `teams.name` zurück

### Requirement: Eigene Teams liefern Display-Strings

`GET /api/teams/my` SHALL je Team zusätzlich zu `id`, `name` und `isExtended` die Felder `display_short` und `display_long` nach denselben Regeln wie alle anderen Team-Endpoints ausliefern. Zugriffsregeln und Auswahl der Teams bleiben unverändert.

#### Scenario: Mitglied in einer von zwei C-Jugenden
- **WHEN** ein Spieler im Kader der zweiten von zwei männlichen C-Jugenden der aktiven Saison `GET /api/teams/my` aufruft
- **THEN** enthält das Team-Item `display_long = "C-Jugend männlich 2"` und `display_short = "mC2"`

#### Scenario: Nicht angemeldet
- **WHEN** `GET /api/teams/my` ohne gültiges Token aufgerufen wird
- **THEN** antwortet der Server mit HTTP 401

### Requirement: „Mein Team" zeigt eingeklappt und aufgeklappt denselben Namen

Die Seite „Mein Team" SHALL den Kartentitel jedes Teams aus dessen Langform rendern — unabhängig davon, ob die Karte eingeklappt ist oder der Kader bereits geladen wurde. Der gespeicherte `teams.name` SHALL nur verwendet werden, wenn keine Langform vorliegt.

#### Scenario: Eingeklappte Karte
- **WHEN** ein Nutzer mit Zugang zu beiden männlichen C-Jugenden „Mein Team" öffnet, ohne eine Karte aufzuklappen
- **THEN** tragen die Karten die Titel „C-Jugend männlich 1" und „C-Jugend männlich 2"

#### Scenario: Aufklappen ändert den Titel nicht
- **WHEN** der Nutzer die Karte „C-Jugend männlich 1" aufklappt bzw. über `/mein-team?team=<id>` direkt öffnet
- **THEN** bleibt der Titel „C-Jugend männlich 1"

### Requirement: Kader-Verwaltung folgt demselben Namensschema

Oberflächen, die den Namen einer Mannschaft aus Kader-Feldern (Altersklasse, Geschlecht, Mannschaftsnummer) selbst zusammensetzen — Kader-Verwaltung inklusive Lösch-Bestätigung, Kader-Kopieren und automatische Zuordnung — SHALL dasselbe Schema wie die Langform verwenden: Nummer am Ende, angehängt genau dann, wenn in der angezeigten Saison mehr als ein Kader derselben Altersklasse und desselben Geschlechts existiert.

#### Scenario: Kader-Karte bei zwei Mannschaften
- **WHEN** die Kader-Verwaltung einer Saison mit zwei männlichen C-Jugenden angezeigt wird
- **THEN** heißen die Karten „C-Jugend männlich 1" und „C-Jugend männlich 2"

#### Scenario: Lösch-Bestätigung der ersten Mannschaft
- **WHEN** der Nutzer die Löschung der Mannschaft 1 von zwei männlichen C-Jugenden anstößt
- **THEN** nennt die Bestätigung „C-Jugend männlich 1" (nicht „C-Jugend männlich")

#### Scenario: Kader kopieren bei zwei Mannschaften
- **WHEN** der Kopier-Dialog Kader einer Saison mit zwei männlichen C-Jugenden auflistet
- **THEN** sind die beiden Einträge als „C-Jugend männlich 1" und „C-Jugend männlich 2" unterscheidbar
