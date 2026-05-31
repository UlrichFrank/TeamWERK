### Requirement: Multi-valued club function per member
The system SHALL support assigning zero, one, or more club functions to a member record. Valid functions: `spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`. Functions are stored in a junction table `member_club_functions(member_id, function)` and are independent of the user account's system role.

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

### Requirement: Club functions and parent status propagated in JWT
The system SHALL include the linked member's club functions and the user's parent status in the JWT Access Token at login and refresh time.

#### Scenario: Login for member-linked user
- **WHEN** a standard user with a linked member record logs in
- **THEN** the JWT contains `club_functions` as an array of that member's current functions and `is_parent: false` (assuming no family_links)

#### Scenario: Login for parent user
- **WHEN** a standard user with one or more `family_links` entries but no linked member logs in
- **THEN** the JWT contains `club_functions: []` and `is_parent: true`

#### Scenario: Login for user with no member and no family_links
- **WHEN** a standard user with neither a linked member nor family_links logs in
- **THEN** the JWT contains `club_functions: []` and `is_parent: false`

#### Scenario: Function change takes effect after next login
- **WHEN** an admin changes a member's club functions
- **THEN** the change is reflected in the JWT only after the affected user's next login or token refresh

### Requirement: Duty obligation priority for members with multiple functions
The system SHALL apply duty obligations based on the member's highest-priority function when multiple functions are present. Priority order: `trainer` > `spieler` > `elternteil` (parent status).

#### Scenario: Trainer-Spieler owes trainer duty obligations
- **WHEN** a member has both `spieler` and `trainer` functions and the system calculates their duty Soll
- **THEN** the trainer duty targets apply and spieler targets are ignored for that user

#### Scenario: Spieler without trainer function owes spieler duty obligations
- **WHEN** a member has only the `spieler` function
- **THEN** the spieler duty targets apply

#### Scenario: Parent user with no member function owes elternteil obligations
- **WHEN** a user has `is_parent: true` and `club_functions: []`
- **THEN** the elternteil duty calculation (based on linked children's kader memberships) applies
