# team-name-display Specification

## Purpose

Diese Spezifikation beschreibt die Capability `team-name-display`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

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
- **THEN** zeigt der Header den Langnamen des Teams (z. B. „B-Jugend männlich 2")

#### Scenario: Detail-Anzeige eines Doppelheimspiels
- **WHEN** der User die Detail-Seite eines Doppelheimspiels öffnet
- **THEN** zeigt der Header beide Langnamen komma-getrennt

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

Oberflächen, die den Namen einer Mannschaft aus Kader-Feldern (Altersklasse, Geschlecht, Mannschaftsnummer) selbst zusammensetzen — Kader-Verwaltung inklusive Lösch-Bestätigung und automatische Zuordnung — SHALL dasselbe Schema wie die Langform verwenden: Nummer am Ende, angehängt genau dann, wenn in der angezeigten Saison mehr als ein Kader derselben Altersklasse und desselben Geschlechts existiert.

Der Dialog „Aus vorheriger Saison kopieren" SHALL je Kombination aus Altersklasse und Geschlecht genau eine Zeile ohne Mannschaftsnummer zeigen, weil der Kopiervorgang je Kombination arbeitet und in der Zielsaison genau eine Mannschaft anlegt. Hatte die Quellsaison mehrere Mannschaften dieser Kombination, SHALL die Zeile darauf hinweisen, dass nur eine angelegt wird.

#### Scenario: Kader-Karte bei zwei Mannschaften
- **WHEN** die Kader-Verwaltung einer Saison mit zwei männlichen C-Jugenden angezeigt wird
- **THEN** heißen die Karten „C-Jugend männlich 1" und „C-Jugend männlich 2"

#### Scenario: Lösch-Bestätigung der ersten Mannschaft
- **WHEN** der Nutzer die Löschung der Mannschaft 1 von zwei männlichen C-Jugenden anstößt
- **THEN** nennt die Bestätigung „C-Jugend männlich 1" (nicht „C-Jugend männlich")

#### Scenario: Automatische Zuordnung bei zwei Mannschaften
- **WHEN** der Dialog zur automatischen Zuordnung die Kader einer Saison mit zwei männlichen C-Jugenden auflistet
- **THEN** sind die beiden Einträge als „C-Jugend männlich 1" und „C-Jugend männlich 2" unterscheidbar

#### Scenario: Kader kopieren bei zwei Mannschaften
- **WHEN** der Kopier-Dialog eine Quellsaison mit zwei männlichen C-Jugenden auflistet
- **THEN** erscheint „C-Jugend männlich" genau einmal, mit dem Hinweis „2 Mannschaften in der Quellsaison – es wird eine angelegt"

### Requirement: Dashboard-Kachel „Mein Team" nennt die Langform

Die Kachel „Mein Team" auf dem Dashboard SHALL jedes eigene Team mit seiner Langform (`display_long` aus `GET /api/teams/my`) nennen und nur ohne Langform auf `teams.name` zurückfallen — dieselbe Schreibweise wie die Seite „Mein Team", auf die sie verlinkt.

#### Scenario: Zwei C-Jugenden auf dem Dashboard
- **WHEN** ein Nutzer mit Zugang zu beiden männlichen C-Jugenden das Dashboard öffnet
- **THEN** nennt die Kachel „C-Jugend männlich 1" und „C-Jugend männlich 2"

### Requirement: Videos nennen die Mannschaft in der Langform

`GET /api/videos`, `GET /api/videos/{id}`, `GET /api/videos/upload-eligible-games` und die Push-Meldung „Video bereit" SHALL die Mannschaft mit ihrer Langform (Regel wie `display_long`, Fallback `teams.name`) benennen.

#### Scenario: Video einer von zwei C-Jugenden
- **WHEN** ein Video der Mannschaft 1 von zwei männlichen C-Jugenden der aktiven Saison gelistet oder im Detail abgerufen wird
- **THEN** ist `team_name` „C-Jugend männlich 1"

### Requirement: Jede Ausgabe an Menschen nennt die Mannschaft in der Langform

Jede Stelle, die einen Mannschaftsnamen in Langform an Menschen ausgibt — API-Felder (Anwesenheit, Rückmelde-Matrix, Teamliste `GET /api/teams`, Mitfahrten, Spiel-Suche, Staffeln, Ordner-Rechte, H4A-Import-Vorschau), Kalender-Feed, Push-Erinnerungen, Dienst-Export und veröffentlichte Spielberichte — SHALL die Langform nach der Regel von `display_long` verwenden (Fallback `teams.name` ohne Kader in der aktiven Saison). Der gespeicherte `teams.name` SHALL nur noch als Identitätsschlüssel (Team-Anlage, H4A-Nummernabgleich) und in der Stammdatenpflege der Teams verwendet werden. Kurzformen folgen weiterhin `display_short`.

#### Scenario: Anwesenheit einer von zwei C-Jugenden
- **WHEN** ein Trainer `GET /api/teams/{id}/attendance-stats` bzw. `GET /api/teams/{id}/rsvp-matrix` für Mannschaft 1 von zwei männlichen C-Jugenden abruft
- **THEN** ist `team_name` „C-Jugend männlich 1"

#### Scenario: Teamliste für Filter
- **WHEN** `GET /api/teams` zwei männliche C-Jugenden der aktiven Saison liefert
- **THEN** tragen sie `display_long` „C-Jugend männlich 1" bzw. „C-Jugend männlich 2"

#### Scenario: Kalender-Feed
- **WHEN** der persönliche Kalender-Feed ein Heimspiel der Mannschaft 1 von zwei männlichen C-Jugenden enthält
- **THEN** nennt der Titel die Mannschaft als „C-Jugend männlich 1"
