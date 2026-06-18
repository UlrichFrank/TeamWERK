## ADDED Requirements

### Requirement: GET /api/me liefert Capabilities und Nav-Items

Das System SHALL `GET /api/me` um die Felder `capabilities` und `nav` erweitern.
Das Frontend MUSS Nav-Items und Feature-Sichtbarkeit ausschließlich aus diesem Endpoint
beziehen und DARF NICHT `navModules[i].items[j].roles` oder lokale `const isXxx`-Konstrukte
für diesen Zweck verwenden.

#### Scenario: Vorstand erhält manage_members-Capability
- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/me` aufruft
- **THEN** enthält `capabilities` den Wert `"manage_members"`

#### Scenario: Spieler erhält keine manage_members-Capability
- **WHEN** ein User mit Vereinsfunktion `spieler` `GET /api/me` aufruft
- **THEN** enthält `capabilities` NICHT den Wert `"manage_members"`

#### Scenario: Nav enthält nur für den Nutzer sichtbare Items
- **WHEN** ein User mit Vereinsfunktion `spieler` `GET /api/me` aufruft
- **THEN** enthält `nav` keinen Eintrag mit `route: "/members"`

#### Scenario: Vorstand sieht Mitglieder-Nav-Item
- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/me` aufruft
- **THEN** enthält `nav` einen Eintrag mit `route: "/members"`

---

### Requirement: /api/me-Response-Schema

Das System SHALL `GET /api/me` im folgenden Schema antworten:

```json
{
  "user": { "id": 1, "email": "…", "name": "…", "role": "standard" },
  "capabilities": ["create_game", "manage_members"],
  "nav": [
    { "label": "Dashboard", "route": "/dashboard" },
    { "label": "Mitglieder", "route": "/members" }
  ]
}
```

`nav`-Items sind in der Reihenfolge sortiert, in der sie in der Sidebar erscheinen sollen.

#### Scenario: Response enthält alle drei Top-Level-Felder
- **WHEN** ein eingeloggter User `GET /api/me` aufruft
- **THEN** enthält die Response die Felder `user`, `capabilities` und `nav`

#### Scenario: Unauthenticated Request wird abgelehnt
- **WHEN** `GET /api/me` ohne gültigen Access-Token aufgerufen wird
- **THEN** antwortet der Server mit HTTP 401

---

### Requirement: Capabilities werden bei jedem /api/me-Call neu berechnet

Das System SHALL `capabilities` und `nav` aus den aktuellen JWT-Claims berechnen, nicht aus
einem Cache. Damit ist nach einem Token-Refresh der nächste `/api/me`-Call immer aktuell,
ohne dass ein neues Login erforderlich ist.

#### Scenario: Rollenänderung spiegelt sich nach Token-Refresh wider
- **WHEN** einem User eine neue Vereinsfunktion zugewiesen wird und er einen Token-Refresh durchführt
- **THEN** enthält der nächste `/api/me`-Aufruf die aktualisierte Capability-Liste
