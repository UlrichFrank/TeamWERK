## ADDED Requirements

### Requirement: Admin kann alle Nutzer auflisten
Das System SHALL eine Tabelle aller registrierten Nutzer anzeigen, wenn ein Admin die Nutzerverwaltungsseite öffnet. Jede Zeile SHALL Name, E-Mail-Adresse, Rolle und Team des Nutzers enthalten.

#### Scenario: Nutzerliste wird geladen
- **WHEN** ein Admin die Seite `/admin/users` aufruft
- **THEN** lädt das System alle Nutzer via `GET /api/admin/users` und zeigt sie in einer Tabelle mit den Spalten Name, E-Mail, Rolle und Team an

#### Scenario: Leere Nutzerliste
- **WHEN** keine Nutzer in der Datenbank vorhanden sind
- **THEN** zeigt das System eine leere Tabelle ohne Fehlermeldung an

#### Scenario: Nutzer ohne Team-Zugehörigkeit
- **WHEN** ein Nutzer keinem Team zugeordnet ist
- **THEN** bleibt die Team-Spalte leer (kein Fehler)

### Requirement: API liefert Team-Name im Nutzer-Endpunkt
`GET /api/admin/users` SHALL für jeden Nutzer zusätzlich das Feld `team_name` (String, leer wenn kein Team) zurückgeben.

#### Scenario: Nutzer mit Team
- **WHEN** `GET /api/admin/users` aufgerufen wird und ein Nutzer einem Team zugeordnet ist
- **THEN** enthält das JSON-Objekt des Nutzers ein Feld `team_name` mit dem Namen des Teams

#### Scenario: Nutzer ohne Team
- **WHEN** `GET /api/admin/users` aufgerufen wird und ein Nutzer keinem Team zugeordnet ist
- **THEN** enthält das JSON-Objekt des Nutzers ein Feld `team_name` mit leerem String

### Requirement: Nur Admins dürfen die Nutzerliste abrufen
`GET /api/admin/users` SHALL nur für Nutzer mit der Rolle `admin` zugänglich sein.

#### Scenario: Nicht-Admin ruft Endpunkt ab
- **WHEN** ein Nutzer mit Rolle `trainer`, `elternteil` oder `spieler` `GET /api/admin/users` aufruft
- **THEN** antwortet das Backend mit HTTP 403
