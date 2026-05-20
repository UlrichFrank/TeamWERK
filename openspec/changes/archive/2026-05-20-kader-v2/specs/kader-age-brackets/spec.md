## MODIFIED Requirements

### Requirement: Jahrgangskalkulation je Saison

Das System SHALL für jede Altersklasse den korrekten Geburtsjahrgang-Bereich berechnen,
basierend auf dem Startjahr der Saison und den DHB-Altersklassen-Definitionen.

Basisjahre für Saison 2025/26 (Startjahr 2025):

| Altersklasse | Jüngerer Jg. | Älterer Jg. |
|---|---|---|
| A-Jugend | 2008 | 2007 |
| B-Jugend | 2010 | 2009 |
| C-Jugend | 2012 | 2011 |
| D-Jugend | 2014 | 2013 |

Für jede weitere Saison MUSS der Offset +1 auf beide Jahrgänge addiert werden.
Die Altersklassen DÜRFEN SICH NICHT überlappen (kein Jahrgang in zwei Klassen gleichzeitig).

#### Scenario: Korrekte Jahrgänge für 2025/26

- **WHEN** `ComputeAgeBrackets(2025)` aufgerufen wird
- **THEN** gibt A-Jugend [2007, 2008] zurück, B-Jugend [2009, 2010], C-Jugend [2011, 2012], D-Jugend [2013, 2014]

#### Scenario: Offset bei Folgesaison

- **WHEN** `ComputeAgeBrackets(2026)` aufgerufen wird
- **THEN** gibt A-Jugend [2008, 2009] zurück, B-Jugend [2010, 2011], C-Jugend [2012, 2013], D-Jugend [2014, 2015]

#### Scenario: Keine Überschneidung zwischen Altersklassen

- **WHEN** die Brackets für eine Saison berechnet werden
- **THEN** existiert kein Geburtsjahr das in mehr als einer Altersklasse liegt

#### Scenario: Spieler-Jahrgang in B-Jugend nach Saisonwechsel

- **WHEN** ein Spieler Jahrgang 2011 ist und die Saison 2026/27 aktiv ist
- **THEN** gehört er zur B-Jugend (Bracket [2010, 2011]), nicht mehr zur C-Jugend

### Requirement: Response enthält berechnete Jahrgänge

`GET /api/admin/kader` und `GET /api/admin/kader/{id}` SOLLEN ein Feld `birth_years: []int`
zurückgeben, das die für diesen Kader relevanten Geburtsjahre enthält.

Bei `dedicated_birth_year = NULL`: beide Jahrgänge des Brackets (z.B. [2011, 2012]).
Bei `dedicated_birth_year = 2011`: nur [2011].

#### Scenario: Gemischter Kader gibt beide Jahrgänge zurück

- **WHEN** ein Kader C-Jugend mit `dedicated_birth_year = NULL` abgerufen wird (Saison 2025/26)
- **THEN** enthält `birth_years` den Wert [2011, 2012]

#### Scenario: Dedizierter Kader gibt einen Jahrgang zurück

- **WHEN** ein Kader C-Jugend 1 mit `dedicated_birth_year = 2011` abgerufen wird
- **THEN** enthält `birth_years` den Wert [2011]
