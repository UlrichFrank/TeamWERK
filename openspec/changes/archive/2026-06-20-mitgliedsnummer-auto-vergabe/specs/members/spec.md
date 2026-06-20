## MODIFIED Requirements

### Requirement: Player profile management
The system SHALL allow admins and users with the `trainer` club function to create and maintain player profiles. A player profile contains: first name, last name, date of birth, pass number, jersey number, position, and member status. The membership number (`member_number`) is NOT part of the editable base fields: it is system-assigned on creation and read-only for non-admins (see capability `mitgliedsnummer-verwaltung`). Club functions are stored as a set (zero or more) in `member_club_functions` and are managed separately from the profile's base fields.

#### Scenario: Admin creates player profile
- **WHEN** an admin submits a new player profile with required fields (first name, last name, date of birth)
- **THEN** the system creates the profile with status `aktiv` by default, an empty club function set, and an automatically assigned membership number (highest numeric + 1)

#### Scenario: Admin assigns club functions to member
- **WHEN** an admin submits a set of club functions (e.g., `["spieler", "trainer"]`) for an existing member
- **THEN** the system replaces the member's current function set with the submitted set

#### Scenario: Teamleiter creates player in own team
- **WHEN** a user with `trainer` club function creates a player profile
- **THEN** the player is automatically assigned to the trainer's team

#### Scenario: Duplicate pass number rejected
- **WHEN** a profile is saved with a pass number that already exists in the system
- **THEN** the system returns a validation error identifying the conflict

#### Scenario: Membership number is not taken from the create request
- **WHEN** a create request includes an explicit `member_number`
- **THEN** the system ignores it and assigns the next free number automatically
