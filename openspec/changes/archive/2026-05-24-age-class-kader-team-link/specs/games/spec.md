## MODIFIED Requirements

### Requirement: Spieldauer aus Altersklassen-Regel
Für Events vom Typ `heim` und `auswärts` SHALL das Backend die Spieldauer direkt aus `age_class_game_rules` lesen, indem es `teams.age_class` als FK-Wert ohne Transformation verwendet. Der bisherige Workaround (Extraktion des ersten Buchstabens mit `[:1]`) entfällt, da `teams.age_class` und `age_class_game_rules.age_class` jetzt denselben Langform-Wert tragen.

#### Scenario: Spieldauer für Heim-Event mit B-Jugend-Team
- **WHEN** Slots für ein Heim-Event mit einem Team mit `age_class = "B-Jugend"` generiert werden
- **THEN** verwendet das Backend die Regel für `"B-Jugend"` (2 × 25 + 10 = 60 min)

#### Scenario: Heim-Event mit Team ohne Altersklasse
- **WHEN** Slots für ein Heim-Event mit einem Team ohne `age_class` (NULL) generiert werden
- **THEN** antwortet der Server mit HTTP 422 und der Meldung "Team hat keine Altersklasse"

#### Scenario: Heim-Event mit unbekannter Altersklasse
- **WHEN** `teams.age_class` einen Wert enthält, der nicht in `age_class_game_rules` existiert
- **THEN** antwortet der Server mit HTTP 422 (durch FK-Constraint auf DB-Ebene nicht erreichbar für reguläre Pfade)
