## ADDED Requirements

### Requirement: Mehrtägige Events mit end_date

Events (heim, auswärts, generisch) SHALL optional ein `end_date` haben, das ein Enddatum (inklusive) für das Event festlegt. Wenn `end_date` gesetzt ist und nach `date` liegt, erstreckt sich das Event über mehrere Tage.

#### Scenario: Event ohne end_date (Standardfall)
- **WHEN** ein Event ohne `end_date` angelegt wird
- **THEN** wird es wie bisher als eintägiges Event behandelt

#### Scenario: Mehrtägiges Event anlegen
- **WHEN** `POST /api/kalender` mit `end_date` aufgerufen wird und `end_date >= date`
- **THEN** wird das Event mit `end_date` gespeichert und HTTP 201 zurückgegeben

#### Scenario: end_date vor date wird abgelehnt
- **WHEN** `POST /api/kalender` oder `PUT /api/kalender/{id}` mit `end_date < date` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: end_date in GET-Response enthalten
- **WHEN** `GET /api/kalender` oder `GET /api/kalender/{id}` aufgerufen wird
- **THEN** enthält jedes Event mit gesetztem `end_date` das Feld `end_date` in der Response (ISO-Datum-String)
- **THEN** Events ohne `end_date` liefern `end_date: null`

#### Scenario: Mehrtägiges Event bearbeiten
- **WHEN** `PUT /api/kalender/{id}` mit neuem `end_date` aufgerufen wird
- **THEN** wird `end_date` aktualisiert
- **WHEN** `PUT /api/kalender/{id}` mit `end_date: null` aufgerufen wird
- **THEN** wird `end_date` auf NULL gesetzt (Event wird wieder eintägig)
