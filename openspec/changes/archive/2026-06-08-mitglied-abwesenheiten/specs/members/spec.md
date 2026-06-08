## ADDED Requirements

### Requirement: Abwesenheits-Sichtbarkeit am Member-Profil
Jeder Member SHALL ein Feld `absences_public` (Boolean, default `false`) besitzen, das steuert, ob seine Abwesenheiten für Trainer im Kalender sichtbar sind. Das Feld MUSS über das eigene Profil gesetzt werden können.

#### Scenario: Standard-Sichtbarkeit ist privat
- **WHEN** ein neuer Member angelegt wird
- **THEN** ist `absences_public` standardmäßig `false`

#### Scenario: Spieler aktiviert Sichtbarkeit
- **WHEN** ein Spieler `PUT /api/profile/absence-visibility` mit `{"public": true}` aufruft
- **THEN** wird `absences_public` auf `true` gesetzt und ist sofort wirksam für Kalenderabfragen
