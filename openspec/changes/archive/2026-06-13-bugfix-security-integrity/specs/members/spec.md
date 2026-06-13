## MODIFIED Requirements

### Requirement: Parent/child linking
The system SHALL allow linking standard user accounts (acting as parents/guardians) to player member profiles. A linked parent user SHALL be able to view the child member's contact information. The API MUST return the correct parent user data when queried for a member's parents.

#### Scenario: Parent linked to member — data returned correctly
- **WHEN** `GET /api/members/{id}/parents` is called for a member with linked parent users
- **THEN** the response contains each parent's `id`, full name (`first_name || ' ' || last_name`), and `email`
- **THEN** the response MUST NOT be empty due to a non-existent `name` column

#### Scenario: Member with no linked parents
- **WHEN** `GET /api/members/{id}/parents` is called for a member with no family links
- **THEN** the response contains an empty array

### Requirement: CSV-Import für Mitglieder
Das System SHALL einen CSV-Import für Mitgliedsdaten unterstützen. Geburtsdaten in zweistelligem Jahresformat (DD.MM.YY) MÜSSEN korrekt auf das 20. oder 21. Jahrhundert abgebildet werden.

#### Scenario: Importierte Geburtstage — aktuelle Jugendliche
- **WHEN** ein CSV-Import ein Geburtsdatum mit zweistelligem Jahr `< 30` enthält (z.B. `01.03.25`)
- **THEN** wird das Geburtsjahr als `2025` gespeichert

#### Scenario: Importierte Geburtstage — ältere Mitglieder
- **WHEN** ein CSV-Import ein Geburtsdatum mit zweistelligem Jahr `>= 68` enthält (z.B. `15.07.72`)
- **THEN** wird das Geburtsjahr als `1972` gespeichert

#### Scenario: Importierte Geburtstage — Grenzbereich 2030er-Jahrgang
- **WHEN** ein CSV-Import ein Geburtsdatum mit zweistelligem Jahr im Bereich `30`–`67` enthält (z.B. `10.05.30`)
- **THEN** wird das Geburtsjahr als `2030` gespeichert (nicht als `1930`)

#### Scenario: Vierstelliges Jahr bleibt unverändert
- **WHEN** ein CSV-Import ein Geburtsdatum mit vierstelligem Jahr enthält (z.B. `10.05.2030`)
- **THEN** wird das Geburtsjahr unverändert als `2030` gespeichert
