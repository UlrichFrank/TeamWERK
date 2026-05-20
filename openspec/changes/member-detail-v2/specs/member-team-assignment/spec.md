## MODIFIED Requirements

### Requirement: Mannschafts-Zuweisung über Kaderplanung

Die Zuweisung eines Mitglieds zu einer Mannschaft MUSS ausschließlich über die Kaderplanung
(`/admin/kader`) erfolgen. Die Mitglieder-Detailseite DARF keine Mannschafts-Zuweisung mehr anbieten.

Der Backend-Endpoint `POST /api/members/{id}/team-assignment` DARF erhalten bleiben
(für interne Nutzung), wird aber nicht mehr aus der Mitglieder-UI aufgerufen.

#### Scenario: Keine Mannschafts-Sektion auf Mitglieder-Detailseite

- **WHEN** ein Admin die Detailseite eines Mitglieds aufruft
- **THEN** ist kein Bereich „Mannschaft zuweisen" sichtbar

#### Scenario: Team-Zuweisung über Kader möglich

- **WHEN** ein Admin die Kaderplanung unter `/admin/kader` aufruft
- **THEN** kann er Mitglieder zu Kadern (und damit Mannschaften) zuweisen
