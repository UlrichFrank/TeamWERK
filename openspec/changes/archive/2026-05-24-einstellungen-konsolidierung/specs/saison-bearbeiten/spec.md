## ADDED Requirements

### Requirement: PUT /api/admin/seasons/{id}
Neuer Endpoint zum Bearbeiten einer bestehenden Saison (Name, Start- und Enddatum).

**Request:** `PUT /api/admin/seasons/{id}` — body: `{ name, start_date, end_date }`  
**Auth:** admin oder vorstand  
**Response:** 200 OK mit aktualisiertem Season-Objekt

#### Scenario: Saison-Daten aktualisieren
- **WHEN** ein Admin `PUT /api/admin/seasons/1` mit gültigen Daten aufruft
- **THEN** werden Name und Datumsfelder in der DB aktualisiert

#### Scenario: Aktive Saison bearbeiten erlaubt
- **WHEN** die Saison `is_active = true` ist
- **THEN** darf sie trotzdem bearbeitet werden (kein HTTP-Fehler)

#### Scenario: Nicht gefundene Saison
- **WHEN** eine nicht existierende ID übergeben wird
- **THEN** antwortet die API mit 404

#### Scenario: Fehlende Felder
- **WHEN** `name` oder `start_date` oder `end_date` fehlen
- **THEN** antwortet die API mit 400
