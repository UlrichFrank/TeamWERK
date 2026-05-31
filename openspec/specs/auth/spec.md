## ADDED Requirements

### Requirement: Membership request by prospective user
The system SHALL allow any person to submit a membership request for a specific team without being logged in. The request contains: first name, last name, e-mail address, and desired team. An admin, trainer of that team, or Vorstand member must approve or reject the request.

#### Scenario: User submits membership request
- **WHEN** a visitor submits a membership request form with name, e-mail, and team selection
- **THEN** the system stores the request with status `pending` and notifies the team's trainer and all admins by e-mail

#### Scenario: Trainer approves request
- **WHEN** a logged-in `trainer` or `admin` approves a pending request for their team
- **THEN** the system generates a registration token (48-hour expiry), sends a registration link to the applicant's e-mail, and marks the request as `approved`

#### Scenario: Request rejected
- **WHEN** a `trainer` or `admin` rejects a pending request
- **THEN** the system marks it `rejected` and sends a rejection notification to the applicant

#### Scenario: Expired registration link after approval
- **WHEN** an applicant opens a registration link older than 48 hours
- **THEN** the system shows an error and prompts them to contact the admin for a new link

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

### Requirement: User login with JWT
The system SHALL authenticate users via e-mail and password and issue JWT tokens.

#### Scenario: Successful login
- **WHEN** a user submits valid credentials
- **THEN** the system returns an Access Token (15-minute lifetime) in the response body and sets a Refresh Token (7-day lifetime) as an HttpOnly cookie

#### Scenario: Invalid credentials
- **WHEN** a user submits an unknown e-mail or wrong password
- **THEN** the system returns HTTP 401 with a generic error message (no disclosure of which field is wrong)

#### Scenario: Access Token used for API requests
- **WHEN** a client sends a request with a valid Access Token in the `Authorization: Bearer` header
- **THEN** the system processes the request and returns the result

#### Scenario: Expired Access Token
- **WHEN** a client sends a request with an expired Access Token
- **THEN** the system returns HTTP 401

### Requirement: Token refresh
The system SHALL allow silent renewal of an expired Access Token using the Refresh Token.

#### Scenario: Valid refresh
- **WHEN** a client calls `POST /api/auth/refresh` with a valid HttpOnly Refresh Token cookie
- **THEN** the system issues a new Access Token and rotates the Refresh Token

#### Scenario: Expired or missing Refresh Token
- **WHEN** a client calls `POST /api/auth/refresh` without a valid Refresh Token cookie
- **THEN** the system returns HTTP 401 and the client MUST redirect to the login page

### Requirement: Logout
The system SHALL allow users to explicitly terminate their session.

#### Scenario: Logout clears Refresh Token
- **WHEN** a user calls `POST /api/auth/logout`
- **THEN** the system clears the Refresh Token cookie and invalidates the stored token hash

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

### Requirement: Password reset
The system SHALL allow users to reset a forgotten password via e-mail.

#### Scenario: Reset request
- **WHEN** a user submits their e-mail address on the password-reset form
- **THEN** the system sends a reset link (valid 1 hour) if the address exists — the response is identical whether the address exists or not (no enumeration)

#### Scenario: Password reset completion
- **WHEN** a user opens a valid reset link and submits a new password
- **THEN** the system updates the password hash, invalidates the token, and invalidates all existing Refresh Tokens for that user
