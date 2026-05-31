## MODIFIED Requirements

### Requirement: Direct invitation by admin
The system SHALL allow admins to directly invite a person by e-mail address with a system role of `admin` or `standard`. No Vereinsfunktion is set at invitation time; this is configured separately via the member record. Only admins may send invitations with target role `admin`.

#### Scenario: Admin sends direct invitation as standard user
- **WHEN** an `admin` submits an e-mail address and target role `standard` via the invitation form
- **THEN** the system generates a registration token (48-hour expiry) and sends a registration link to that address

#### Scenario: Admin invites another admin
- **WHEN** an `admin` submits an e-mail address and target role `admin` via the invitation form
- **THEN** the system generates a registration token (48-hour expiry) and sends a registration link

#### Scenario: Non-admin attempts to invite an admin
- **WHEN** a `standard` user attempts to invite someone with target role `admin`
- **THEN** the system returns HTTP 403

#### Scenario: Invited user registers
- **WHEN** an invitee opens a valid registration link and submits name and password
- **THEN** the system creates a user account with the pre-configured role (`admin` or `standard`), invalidates the token, and redirects to the login page

#### Scenario: Expired direct invitation link
- **WHEN** an invitee opens a registration link older than 48 hours
- **THEN** the system returns an error message and prompts to contact the admin

### Requirement: Role-based access control
The system SHALL enforce access based on the user's system role and club functions embedded in the JWT claims.

System roles: `admin` (full platform access), `standard` (regular user). Club functions (`spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`) and parent status (`is_parent`) are additional JWT claims that gate domain-specific features, not system access.

#### Scenario: Admin accesses admin-only route
- **WHEN** an `admin` user calls an admin-protected endpoint
- **THEN** the system processes the request normally

#### Scenario: Standard user accesses admin-only route
- **WHEN** a `standard` user calls an admin-protected endpoint
- **THEN** the system returns HTTP 403

#### Scenario: Trainer-function user accesses trainer-gated feature
- **WHEN** a `standard` user whose JWT contains `club_functions: ["trainer"]` calls a trainer-gated endpoint
- **THEN** the system processes the request normally

#### Scenario: User without trainer function accesses trainer-gated feature
- **WHEN** a `standard` user whose JWT does not contain `trainer` in `club_functions` calls a trainer-gated endpoint
- **THEN** the system returns HTTP 403

#### Scenario: Role or function change requires re-login
- **WHEN** an admin changes a user's system role or a member's club functions
- **THEN** the change takes effect only after the affected user's next login or token refresh (existing JWT claims are not updated mid-session)
