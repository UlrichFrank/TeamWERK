## ADDED Requirements

### Requirement: age_class_game_rules ist kanonische Werteliste
Die Tabelle `age_class_game_rules` SHALL die einzige authoritative Quelle für gültige Altersklassen-Bezeichner sein. Die Primary Keys sind Langformen: 'A-Jugend', 'B-Jugend', 'C-Jugend', 'D-Jugend'.

#### Scenario: Langform-Keys in der Tabelle
- **WHEN** `GET /api/admin/age-class-rules` aufgerufen wird
- **THEN** enthält die Antwort Einträge mit `age_class`-Werten in Langform (z.B. `"A-Jugend"`)

#### Scenario: Backend validiert nur bekannte Klassen
- **WHEN** `PUT /api/admin/age-class-rules/B-Jugend` mit validen Werten aufgerufen wird
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Unbekannte Klasse wird abgelehnt
- **WHEN** `PUT /api/admin/age-class-rules/X-Klasse` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 422

### Requirement: teams.age_class ist FK auf age_class_game_rules
`teams.age_class` SHALL nur Werte enthalten, die in `age_class_game_rules.age_class` existieren (oder NULL für Teams ohne Jugendklasse). Die DB-Ebene erzwingt dies über einen FK-Constraint.

#### Scenario: Team mit gültiger Altersklasse anlegen
- **WHEN** `POST /api/admin/teams` mit `age_class: "B-Jugend"` aufgerufen wird
- **THEN** wird das Team angelegt und antwortet mit HTTP 201

#### Scenario: Team ohne Altersklasse anlegen (Erwachsenenteam)
- **WHEN** `POST /api/admin/teams` mit `age_class: null` oder ohne `age_class` aufgerufen wird
- **THEN** wird das Team angelegt mit `age_class = NULL`

#### Scenario: Team mit ungültiger Altersklasse anlegen
- **WHEN** `POST /api/admin/teams` mit `age_class: "X-Klasse"` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 422

### Requirement: Admin-UI für Teams nutzt Dropdown für age_class
Die Teams-Verwaltungs-UI SHALL ein `<select>`-Element für `age_class` anzeigen, das die verfügbaren Klassen aus `GET /api/admin/age-class-rules` lädt, sowie eine Leer-Option für Teams ohne Jugendklasse.

#### Scenario: Dropdown zeigt verfügbare Klassen
- **WHEN** ein Admin die Team-Bearbeitung öffnet
- **THEN** werden die Optionen A-Jugend, B-Jugend, C-Jugend, D-Jugend angezeigt (plus Leer-Option)

#### Scenario: Aktuelle Klasse ist vorselektiert
- **WHEN** ein Team mit `age_class = "B-Jugend"` bearbeitet wird
- **THEN** ist "B-Jugend" im Dropdown vorausgewählt

### Requirement: Altersklassen-Regeln-UI zeigt Langform ohne manuelles Suffix
`AdminAgeClassRulesPage` SHALL den Wert aus `age_class` direkt als Zeilenbeschriftung verwenden, ohne `-Jugend` anzuhängen.

#### Scenario: Klassen-Label in der UI
- **WHEN** die Altersklassen-Regeln-Seite geöffnet wird
- **THEN** zeigt die Klassen-Spalte "A-Jugend", "B-Jugend" usw. (kein "A-Jugend-Jugend")
