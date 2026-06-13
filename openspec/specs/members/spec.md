## ADDED Requirements

### Requirement: Player profile management
The system SHALL allow admins and users with the `trainer` club function to create and maintain player profiles. A player profile contains: first name, last name, date of birth, pass number, jersey number, position, and member status. Club functions are stored as a set (zero or more) in `member_club_functions` and are managed separately from the profile's base fields.

#### Scenario: Admin creates player profile
- **WHEN** an admin submits a new player profile with required fields (first name, last name, date of birth)
- **THEN** the system creates the profile with status `aktiv` by default and an empty club function set

#### Scenario: Admin assigns club functions to member
- **WHEN** an admin submits a set of club functions (e.g., `["spieler", "trainer"]`) for an existing member
- **THEN** the system replaces the member's current function set with the submitted set

#### Scenario: Teamleiter creates player in own team
- **WHEN** a user with `trainer` club function creates a player profile
- **THEN** the player is automatically assigned to the trainer's team

#### Scenario: Duplicate pass number rejected
- **WHEN** a profile is saved with a pass number that already exists in the system
- **THEN** the system returns a validation error identifying the conflict

### Requirement: Member status lifecycle
The system SHALL track the member status of each player. Valid statuses: `aktiv`, `verletzt`, `pausiert`, `ausgetreten`.

#### Scenario: Status change recorded
- **WHEN** an admin or trainer updates a player's status
- **THEN** the system persists the new status and records the change timestamp

#### Scenario: Ausgetretene Mitglieder excluded from active lists
- **WHEN** any module queries active members
- **THEN** members with status `ausgetreten` are excluded from results unless explicitly requested

### Requirement: Team membership assignment
The system SHALL allow assigning a player to one or more teams, with a primary team designation.

#### Scenario: Assign player to team
- **WHEN** an admin assigns a player to a team for the active season
- **THEN** the player appears in that team's member list

#### Scenario: Multiple team membership
- **WHEN** a player is assigned to more than one team
- **THEN** the system stores all assignments and marks one as primary

#### Scenario: Teamleiter sees only own team members
- **WHEN** a `trainer` views the member list
- **THEN** only members assigned to their team(s) are shown

### Requirement: Parent/child linking
The system SHALL allow linking standard user accounts (acting as parents/guardians) to player profiles. A parent user can be linked to one or more player profiles via `family_links`. Parent users have no linked member record of their own. The API MUST return the correct parent user data when queried for a member's parents.

#### Scenario: Admin links parent to player
- **WHEN** an admin links a standard user account to a player profile via family_links
- **THEN** the parent can view that player's data and act on their behalf (Zu-/Absagen, Dienste)

#### Scenario: Parent linked to member — data returned correctly
- **WHEN** `GET /api/members/{id}/parents` is called for a member with linked parent users
- **THEN** the response contains each parent's `id`, full name (`first_name || ' ' || last_name`), and `email`
- **THEN** the response MUST NOT be empty due to a non-existent `name` column

#### Scenario: Member with no linked parents
- **WHEN** `GET /api/members/{id}/parents` is called for a member with no family links
- **THEN** the response contains an empty array

#### Scenario: Parent sees only linked children
- **WHEN** a user with `is_parent: true` views the member area
- **THEN** only their linked children's profiles are visible

#### Scenario: Player account linked to own profile
- **WHEN** a standard user (age ≥ 14) with the `spieler` club function is assigned a user account
- **THEN** they can view and partially edit their own profile

### Requirement: Vehicle information for transport planning
The system SHALL allow parents and players to store vehicle information (seats available) for use in future transport planning.

#### Scenario: Parent stores vehicle data
- **WHEN** an `elternteil` user submits vehicle type and available seats
- **THEN** the system stores the data against their user account for use in transport modules

### Requirement: Member list export
The system SHALL allow admins to export the full member list as CSV.

#### Scenario: CSV export
- **WHEN** an admin triggers the CSV export
- **THEN** the system returns a downloadable CSV file with all active member profiles and their team assignments

---

## ADDED Requirements

### Requirement: Welcome email sent timestamp on member record
The system SHALL store a nullable `welcome_email_sent_at` timestamp on every member record, set when a welcome email is successfully dispatched.

#### Scenario: Field null by default
- **WHEN** a new member is created
- **THEN** `welcome_email_sent_at` is null

#### Scenario: Field set after dispatch
- **WHEN** a welcome email is successfully sent for a member
- **THEN** `welcome_email_sent_at` is set to the dispatch timestamp and cannot be reset to null via the API

#### Scenario: Field returned in member detail
- **WHEN** `GET /api/members/{id}` is called
- **THEN** the response JSON includes `welcome_email_sent_at` as an ISO-8601 string or null

---

## ADDED Requirements

### Requirement: Abwesenheits-Sichtbarkeit am Member-Profil
Jeder Member SHALL ein Feld `absences_public` (Boolean, default `false`) besitzen, das steuert, ob seine Abwesenheiten für Trainer im Kalender sichtbar sind. Das Feld MUSS über das eigene Profil gesetzt werden können.

#### Scenario: Standard-Sichtbarkeit ist privat
- **WHEN** ein neuer Member angelegt wird
- **THEN** ist `absences_public` standardmäßig `false`

#### Scenario: Spieler aktiviert Sichtbarkeit
- **WHEN** ein Spieler `PUT /api/profile/absence-visibility` mit `{"public": true}` aufruft
- **THEN** wird `absences_public` auf `true` gesetzt und ist sofort wirksam für Kalenderabfragen

---

## MODIFIED Requirements

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
