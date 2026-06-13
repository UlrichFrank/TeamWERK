## MODIFIED Requirements

### Requirement: User login with JWT
The system SHALL authenticate users via e-mail and password and issue JWT tokens. Login SHALL only succeed for accounts with `can_login = 1`. Proxy accounts (`can_login = 0`) MUST be excluded from the login query. The system MUST perform a constant-time password comparison regardless of whether the e-mail address exists, to prevent timing-based user enumeration.

#### Scenario: Successful login
- **WHEN** a user submits valid credentials
- **THEN** the system returns an Access Token (15-minute lifetime) in the response body and sets a Refresh Token (7-day lifetime) as an HttpOnly cookie

#### Scenario: Invalid credentials
- **WHEN** a user submits an unknown e-mail or wrong password
- **THEN** the system returns HTTP 401 with a generic error message (no disclosure of which field is wrong)
- **THEN** the response time SHALL be indistinguishable from a failed login with a known e-mail (bcrypt dummy comparison MUST be performed even when the e-mail is not found)

#### Scenario: Login attempt with proxy account e-mail
- **WHEN** a user submits the e-mail address of a proxy account (`can_login = 0`)
- **THEN** the system returns HTTP 401 with the same generic error message (no enumeration of account type)

#### Scenario: Access Token used for API requests
- **WHEN** a client sends a request with a valid Access Token in the `Authorization: Bearer` header
- **THEN** the system processes the request and returns the result

#### Scenario: Expired Access Token
- **WHEN** a client sends a request with an expired Access Token
- **THEN** the system returns HTTP 401

### Requirement: Token refresh
The system SHALL allow silent renewal of an expired Access Token using the Refresh Token. The rotation of the Refresh Token MUST be atomic: the old token MUST NOT be invalidated unless a new token has been successfully stored.

#### Scenario: Valid refresh
- **WHEN** a client calls `POST /api/auth/refresh` with a valid HttpOnly Refresh Token cookie
- **THEN** the system atomically deletes the old Refresh Token and inserts a new one within a single database transaction
- **THEN** the system issues a new Access Token and sets the new Refresh Token as an HttpOnly cookie

#### Scenario: Concurrent refresh with the same token
- **WHEN** two concurrent requests arrive at `POST /api/auth/refresh` with the same Refresh Token
- **THEN** exactly one SHALL succeed and receive a new token pair; the other SHALL receive HTTP 401

#### Scenario: Database failure during token rotation
- **WHEN** the database write fails during Refresh Token rotation (e.g., transient lock)
- **THEN** the old Refresh Token remains valid and the client can retry
- **THEN** the system MUST NOT leave the user in a permanently logged-out state

#### Scenario: Expired or missing Refresh Token
- **WHEN** a client calls `POST /api/auth/refresh` without a valid Refresh Token cookie
- **THEN** the system returns HTTP 401 and the client MUST redirect to the login page

### Requirement: User registration via invitation token
The system SHALL allow invited users to complete registration by submitting name and password. Password hashing MUST be completed successfully before any user record is written to the database.

#### Scenario: Successful registration
- **WHEN** an invitee submits a valid invitation token, name, and password
- **THEN** the system hashes the password, stores the user with the hashed password, and returns HTTP 201

#### Scenario: Internal error during password hashing
- **WHEN** password hashing fails due to a system error (e.g., out of memory)
- **THEN** the system MUST return HTTP 500 and MUST NOT create a user record with an empty or invalid password hash
