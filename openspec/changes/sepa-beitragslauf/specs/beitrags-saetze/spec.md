## ADDED Requirements

### Requirement: Beitragsmatrix verwalten
Nutzer mit Vereinsfunktion `vorstand` oder `kassierer` (sowie System-Rolle `admin`) SOLLEN die Beitragsmatrix pflegen können. Es gibt genau drei Kategorien: `aktiv_ohne`, `aktiv_mit` und `passiv`. Via `GET /api/fee-rates` MUST alle Sätze inkl. Historie zurückgegeben werden, sortiert nach `kategorie, valid_from DESC`. Via `POST /api/fee-rates` SHALL ein neuer Satz mit `kategorie`, `betrag_cent` (> 0) und `valid_from` angelegt werden. Der gültige Beitrag einer Kategorie zu einem Stichtag ist der Satz mit dem größten `valid_from`, das `<= Stichtag` ist.

#### Scenario: Historie bleibt erhalten
- **WHEN** für eine Kategorie zwei Sätze mit unterschiedlichem `valid_from` angelegt wurden und `GET /api/fee-rates` aufgerufen wird
- **THEN** liefert die Antwort beide Sätze, nach `valid_from` absteigend sortiert

#### Scenario: Neue valid_from-Version
- **WHEN** ein Vorstand einen Satz mit identischer `kategorie` und identischem `valid_from` wie ein bestehender Satz anlegt
- **THEN** wird er angelegt (kein 409); die UI macht die Dublette sichtbar

#### Scenario: Ungültige Kategorie
- **WHEN** ein `POST /api/fee-rates` mit einer Kategorie außerhalb {`aktiv_ohne`, `aktiv_mit`, `passiv`} erfolgt
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Zugriff ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `GET /api/fee-rates` aufruft
- **THEN** antwortet der Server mit HTTP 403
