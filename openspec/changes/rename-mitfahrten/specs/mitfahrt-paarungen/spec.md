## MODIFIED Requirements

### Requirement: Paarungen im Board anzeigen
Bestätigte und offene Paarungen MUST für alle authentifizierten Nutzer im Board sichtbar sein.

#### Scenario: Paarungen in der List-Antwort
- **WHEN** `GET /api/mitfahrten` aufgerufen wird
- **THEN** enthält die Antwort pro Spiel ein `paarungen`-Array mit Bieter-Name, Sucher-Name, Anzahl (suche.plaetze) und Status (`pending` / `confirmed`)

#### Scenario: Rejected Paarungen ausgeblendet
- **WHEN** eine Paarung den Status `rejected` hat
- **THEN** erscheint sie nicht im `paarungen`-Array
