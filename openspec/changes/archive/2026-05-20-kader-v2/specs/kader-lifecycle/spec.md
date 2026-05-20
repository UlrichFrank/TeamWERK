## ADDED Requirements

### Requirement: Einzelnen Kader anlegen

Das System MUSS das Anlegen eines einzelnen Kaders für eine Saison ermöglichen,
unabhängig von der Bulk-Initialisierung.

`POST /api/admin/kader` mit Body:
```json
{
  "season_id": 1,
  "age_class": "C-Jugend",
  "gender": "m",
  "team_number": 2,
  "dedicated_birth_year": 2012
}
```

Der Endpoint MUSS mit 409 Conflict antworten, wenn bereits ein Kader mit derselben
`(season_id, age_class, gender, team_number)`-Kombination existiert.

#### Scenario: Zweiten Team-Kader anlegen

- **WHEN** `POST /api/admin/kader` mit `age_class="C-Jugend"`, `gender="m"`, `team_number=2`, `dedicated_birth_year=2012` gesendet wird
- **THEN** antwortet der Server mit 201 Created und dem neuen Kader-Objekt inkl. `birth_years`

#### Scenario: Doppelter Kader wird abgelehnt

- **WHEN** `POST /api/admin/kader` eine `(season_id, age_class, gender, team_number)`-Kombination übergibt die bereits existiert
- **THEN** antwortet der Server mit 409 Conflict

#### Scenario: Anlegen ohne team_number nutzt Default 1

- **WHEN** `POST /api/admin/kader` ohne `team_number` gesendet wird
- **THEN** wird `team_number = 1` angenommen

### Requirement: Kader löschen

Das System MUSS das Löschen eines einzelnen Kaders ermöglichen, wenn dieser keine
Mitglieder mehr hat.

`DELETE /api/admin/kader/{id}`

Wenn noch Mitglieder zugeordnet sind, MUSS der Server mit 409 Conflict antworten
und die aktuelle Mitgliederanzahl im Response-Body zurückgeben:
```json
{"error": "Kader hat noch N Mitglieder", "member_count": N}
```

#### Scenario: Leeren Kader löschen

- **WHEN** `DELETE /api/admin/kader/{id}` für einen Kader ohne Mitglieder aufgerufen wird
- **THEN** antwortet der Server mit 204 No Content und der Kader ist gelöscht

#### Scenario: Nicht-leerer Kader kann nicht gelöscht werden

- **WHEN** `DELETE /api/admin/kader/{id}` für einen Kader mit Mitgliedern aufgerufen wird
- **THEN** antwortet der Server mit 409 Conflict und `{"error": "...", "member_count": N}`

#### Scenario: Löschen-Button in der UI deaktiviert wenn Mitglieder vorhanden

- **WHEN** ein Kader Mitglieder hat
- **THEN** ist der Löschen-Button deaktiviert oder zeigt einen Hinweis „Erst alle Mitglieder entfernen"

### Requirement: Neues-Team-Button in der UI

Die `AdminKaderPage` MUSS pro Altersklasse/Geschlecht-Gruppe einen Button „+ Mannschaft anlegen"
anzeigen, sofern noch keine zwei Teams für diese Kombination existieren.

Der Button öffnet ein Formular mit:
- `team_number` (automatisch: nächste freie Nummer)
- `dedicated_birth_year` (Dropdown: beide Jahrgänge der Altersklasse oder „gemischt")

#### Scenario: Button erscheint wenn weniger als 2 Teams

- **WHEN** es für C-Jugend (m) in der aktiven Saison nur 1 Kader gibt
- **THEN** ist der Button „+ Mannschaft anlegen" für C-Jugend (m) sichtbar

#### Scenario: Button verschwindet bei 2 Teams

- **WHEN** für C-Jugend (m) bereits team_number 1 und 2 existieren
- **THEN** ist kein weiterer „+ Mannschaft anlegen"-Button sichtbar

### Requirement: team_number in Kachel-Titel

Der Titel einer Kachel MUSS `team_number` nur dann anzeigen wenn für diese
Altersklasse/Geschlecht-Kombination mehr als ein Kader existiert.

#### Scenario: Einzelkader ohne Nummer

- **WHEN** nur ein Kader für A-Jugend (m) existiert
- **THEN** zeigt die Kachel „A-Jugend männlich" (ohne Nummer)

#### Scenario: Mehrere Kader mit Nummer

- **WHEN** zwei Kader für C-Jugend (m) existieren
- **THEN** zeigen die Kacheln „C-Jugend 1 männlich (Jg. 2011)" und „C-Jugend 2 männlich (Jg. 2012)"
