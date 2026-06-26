# me-capabilities Specification

## Purpose

Diese Spezifikation beschreibt die Capability `me-capabilities`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

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
- **THEN** enthält `nav` keinen Eintrag mit `route: "/mitglieder"`

#### Scenario: Vorstand sieht Mitglieder-Nav-Item
- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/me` aufruft
- **THEN** enthält `nav` einen Eintrag mit `route: "/mitglieder"`

---

### Requirement: /api/me-Response-Schema

Das System SHALL `GET /api/me` im folgenden Schema antworten:

```json
{
  "user": { "id": 1, "email": "…", "role": "standard" },
  "capabilities": ["manage_games", "manage_members"],
  "nav": [
    { "label": "Dashboard", "route": "/" },
    { "label": "Mitglieder", "route": "/mitglieder" }
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

### Requirement: Capability-Vokabular

Das System SHALL die folgenden Capability-Strings über `GET /api/me` ausliefern. Sie werden
zentral in `policy.Capabilities(claims)` berechnet; das Frontend MUSS Feature-/Button-Sichtbarkeit
ausschließlich daraus (bzw. aus per-Item `can.*`) ableiten.

| Capability | Personas (zzgl. `admin`) |
|---|---|
| `manage_members` | `vorstand` |
| `manage_games` | `vorstand`, `trainer`, `sportliche_leitung` |
| `manage_duties` | `vorstand`, `trainer`, `sportliche_leitung` |
| `manage_kader` | `vorstand`, `trainer`, `sportliche_leitung` |
| `manage_users`, `manage_seasons`, `manage_club`, `manage_duty_types` | `vorstand` |
| `manage_trainings` | `trainer`, `sportliche_leitung` |
| `fulfill_duties` | `trainer`, `sportliche_leitung` |
| `broadcast_messages` | `vorstand`, `trainer`, `sportliche_leitung` |
| `broadcast_all` | `vorstand` |
| `manage_documents` | — (nur `admin`) |
| `moderate_chat` | — (nur `admin`) |
| `impersonate` | — (nur `admin`) |

Relationship-Marker (`is_parent`) und eigene Vereinsfunktionen für eigene Profil-Features
(z.B. `spieler` für Dienst-Erinnerungen) bleiben über die JWT-Claims abbildbar und sind KEINE
Capabilities.

#### Scenario: Trainer erhält manage_trainings, aber nicht broadcast_all
- **WHEN** ein User mit Vereinsfunktion `trainer` `GET /api/me` aufruft
- **THEN** enthält `capabilities` den Wert `"manage_trainings"` und NICHT `"broadcast_all"`

#### Scenario: Reiner Vorstand erhält broadcast_all, aber nicht manage_trainings
- **WHEN** ein User mit Vereinsfunktion `vorstand` (ohne `trainer`/`sportliche_leitung`) `GET /api/me` aufruft
- **THEN** enthält `capabilities` den Wert `"broadcast_all"` und NICHT `"manage_trainings"`

---

### Requirement: Capabilities werden bei jedem /api/me-Call neu berechnet

Das System SHALL `capabilities` und `nav` aus den aktuellen JWT-Claims berechnen, nicht aus
einem Cache. Damit ist nach einem Token-Refresh der nächste `/api/me`-Call immer aktuell,
ohne dass ein neues Login erforderlich ist.

#### Scenario: Rollenänderung spiegelt sich nach Token-Refresh wider
- **WHEN** einem User eine neue Vereinsfunktion zugewiesen wird und er einen Token-Refresh durchführt
- **THEN** enthält der nächste `/api/me`-Aufruf die aktualisierte Capability-Liste
