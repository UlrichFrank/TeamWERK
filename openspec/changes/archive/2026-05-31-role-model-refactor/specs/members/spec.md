## MODIFIED Requirements

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

### Requirement: Parent/child linking
The system SHALL allow linking standard user accounts (acting as parents/guardians) to player profiles. A parent user can be linked to one or more player profiles via `family_links`. Parent users have no linked member record of their own.

#### Scenario: Admin links parent to player
- **WHEN** an admin links a standard user account to a player profile via family_links
- **THEN** the parent can view that player's data and act on their behalf (Zu-/Absagen, Dienste)

#### Scenario: Parent sees only linked children
- **WHEN** a user with `is_parent: true` views the member area
- **THEN** only their linked children's profiles are visible

#### Scenario: Player account linked to own profile
- **WHEN** a standard user (age ≥ 14) with the `spieler` club function is assigned a user account
- **THEN** they can view and partially edit their own profile
