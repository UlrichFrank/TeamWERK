# auth Specification

## Purpose

Diese Spezifikation beschreibt die Capability `auth`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Role-based access control
The system SHALL enforce access based on the user's system role and club functions embedded in the JWT claims.

System roles are persisted in `users.role` and accept exactly two values: `admin` (full platform access, bypasses all `RequireClubFunction` checks) and `standard` (default; all access decisions delegated to club functions and ownership). The endpoint `PUT /api/admin/users/{id}/role` MUST reject any other value with HTTP 400.

Club functions (`spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`, `kassierer`, `sportliche_leitung`) and parent status (`is_parent`) are additional JWT claims that gate domain-specific features, not system access. Club functions and `is_parent` are NEVER stored as `users.role` values.

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

#### Scenario: UpdateUserRole rejects legacy role names
- **WHEN** an admin sends `PUT /api/admin/users/{id}/role` with body `{"role":"trainer"}` (or `"vorstand"`, `"spieler"`, `"elternteil"`, `"sportliche_leitung"`)
- **THEN** the system returns HTTP 400 with body `"invalid role"` and does not modify `users.role`

#### Scenario: UpdateUserRole accepts standard role
- **WHEN** an admin sends `PUT /api/admin/users/{id}/role` with body `{"role":"standard"}`
- **THEN** the system updates `users.role` to `standard` and returns HTTP 204
