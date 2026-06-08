## MODIFIED Requirements

### Requirement: User login with JWT
The system SHALL authenticate users via e-mail and password and issue JWT tokens. Login SHALL only succeed for accounts with `can_login = 1`. Proxy accounts (`can_login = 0`) MUST be excluded from the login query.

#### Scenario: Successful login
- **WHEN** a user submits valid credentials
- **THEN** the system returns an Access Token (15-minute lifetime) in the response body and sets a Refresh Token (7-day lifetime) as an HttpOnly cookie

#### Scenario: Invalid credentials
- **WHEN** a user submits an unknown e-mail or wrong password
- **THEN** the system returns HTTP 401 with a generic error message (no disclosure of which field is wrong)

#### Scenario: Login attempt with proxy account e-mail
- **WHEN** a user submits the e-mail address of a proxy account (`can_login = 0`)
- **THEN** the system returns HTTP 401 with the same generic error message (no enumeration of account type)

#### Scenario: Access Token used for API requests
- **WHEN** a client sends a request with a valid Access Token in the `Authorization: Bearer` header
- **THEN** the system processes the request and returns the result

#### Scenario: Expired Access Token
- **WHEN** a client sends a request with an expired Access Token
- **THEN** the system returns HTTP 401

### Requirement: Password reset
The system SHALL allow users to reset a forgotten password via e-mail. Password reset SHALL only send reset links for accounts with `can_login = 1`.

#### Scenario: Reset request
- **WHEN** a user submits their e-mail address on the password-reset form
- **THEN** the system sends a reset link (valid 1 hour) if a `can_login = 1` account with that address exists — the response is identical whether the address exists or not (no enumeration)

#### Scenario: Reset request for proxy account e-mail
- **WHEN** a user submits an e-mail address that belongs only to a proxy account (`can_login = 0`)
- **THEN** the system does NOT send a reset link, but the response is identical to the "address not found" case (no enumeration)

#### Scenario: Password reset completion
- **WHEN** a user opens a valid reset link and submits a new password
- **THEN** the system updates the password hash, invalidates the token, and invalidates all existing Refresh Tokens for that user
