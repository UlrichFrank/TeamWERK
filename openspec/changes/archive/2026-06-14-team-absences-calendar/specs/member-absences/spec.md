## MODIFIED Requirements

### Requirement: Profil-Toggle „Abwesenheiten für Trainer sichtbar" liest und speichert korrekt
Das System SHALL den Wert von `absences_public` aus der Datenbank korrekt über `GET /api/profile/me` zurückgeben, sodass der Toggle in `ProfileMiscTab` den gespeicherten Zustand anzeigt. `PUT /api/profile/absence-visibility` speichert den Wert weiterhin korrekt.

#### Scenario: Toggle zeigt gespeicherten Wert
- **WHEN** ein Nutzer `absences_public = 1` gesetzt hat und `GET /api/profile/me` aufruft
- **THEN** enthält `own_member.absences_public` den Wert `1` (oder `true`) und der Toggle wird als aktiv angezeigt

#### Scenario: Toggle zeigt inaktiv nach Deaktivierung
- **WHEN** ein Nutzer `PUT /api/profile/absence-visibility` mit `{"public": false}` aufruft und danach `GET /api/profile/me` aufruft
- **THEN** enthält `own_member.absences_public` den Wert `0` (oder `false`) und der Toggle wird als inaktiv angezeigt
