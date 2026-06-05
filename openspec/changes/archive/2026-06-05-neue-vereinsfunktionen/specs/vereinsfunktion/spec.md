## MODIFIED Requirements

### Requirement: Multi-valued club function per member
The system SHALL support assigning zero, one, or more club functions to a member record. Valid functions: `spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`, `kassierer`, `sportliche_leitung`. Functions are stored in a junction table `member_club_functions(member_id, function)` and are independent of the user account's system role.

#### Scenario: Member assigned multiple functions
- **WHEN** an admin assigns both `spieler` and `trainer` functions to a member
- **THEN** both functions are stored and returned in subsequent reads of that member's profile

#### Scenario: Member with no function
- **WHEN** a member record is created without specifying any club function
- **THEN** the system stores an empty function set (no rows in `member_club_functions` for that member)

#### Scenario: Function removed
- **WHEN** an admin removes a function from a member who holds multiple functions
- **THEN** only the specified function is removed; remaining functions are unaffected

#### Scenario: Member list filtered by function
- **WHEN** `GET /api/members?club_function=trainer` is called
- **THEN** only members who have `trainer` in their function set are returned

#### Scenario: kassierer function assignable
- **WHEN** an admin assigns the `kassierer` function to a member
- **THEN** the function is stored and returned in subsequent reads of that member's profile

#### Scenario: sportliche_leitung function assignable
- **WHEN** an admin assigns the `sportliche_leitung` function to a member
- **THEN** the function is stored and returned in subsequent reads of that member's profile

## ADDED Requirements

### Requirement: sportliche_leitung has trainer-equivalent route access
The system SHALL grant users with the `sportliche_leitung` club function access to all routes gated by `RequireClubFunction("trainer")`, including Kader management, Spielplan, and Dienste.

#### Scenario: sportliche_leitung accesses Kader route
- **WHEN** a user with `sportliche_leitung` in their `club_functions` requests a trainer-gated route
- **THEN** the system responds with 200 (not 403)

#### Scenario: sportliche_leitung denied Mitglieder route
- **WHEN** a user with only `sportliche_leitung` (no `trainer`) requests a Mitglieder-only route
- **THEN** the system responds with 403

### Requirement: sportliche_leitung sees all Kader teams without filter
The system SHALL return all active Kader teams for users with `sportliche_leitung`, bypassing the per-trainer team restriction that applies to users with only the `trainer` function.

#### Scenario: trainer sees only own teams
- **WHEN** a user with `trainer` (and not `sportliche_leitung`) calls `GET /api/teams`
- **THEN** only teams where that user is registered as a Kader trainer are returned

#### Scenario: sportliche_leitung sees all teams
- **WHEN** a user with `sportliche_leitung` calls `GET /api/teams`
- **THEN** all active Kader teams are returned regardless of trainer assignment

#### Scenario: sportliche_leitung game filter includes all teams
- **WHEN** a user with `sportliche_leitung` requests the Spielplan calendar
- **THEN** games from all teams are included (no team-based WHERE restriction)
