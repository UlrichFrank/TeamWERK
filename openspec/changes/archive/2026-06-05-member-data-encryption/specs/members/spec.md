## MODIFIED Requirements

### Requirement: Player profile management
The system SHALL allow admins and trainer to create and maintain player profiles. A player profile contains: first name, last name, pass number, jersey number, position, and member status. The fields `date_of_birth`, `street`, `zip`, `city`, `iban`, and `account_holder` are no longer part of the standard player profile response; they are stored encrypted and accessed via the separate sensitive-data endpoint.

#### Scenario: Admin creates player profile
- **WHEN** an admin submits a new player profile with required fields (first name, last name)
- **THEN** the system creates the profile with status `aktiv` by default; sensitive fields are not part of the creation payload

#### Scenario: Teamleiter creates player in own team
- **WHEN** a `trainer` creates a player profile
- **THEN** the player is automatically assigned to the trainer's team

#### Scenario: Duplicate pass number rejected
- **WHEN** a profile is saved with a pass number that already exists in the system
- **THEN** the system returns a validation error identifying the conflict

#### Scenario: Sensitive fields absent from standard profile response
- **WHEN** any role calls `GET /api/members/{id}`
- **THEN** the response does NOT include `date_of_birth`, `street`, `zip`, `city`, `iban`, or `account_holder`

---

## ADDED Requirements

### Requirement: Sensitive member data endpoint
The system SHALL expose a dedicated endpoint for reading and writing encrypted sensitive member data.

#### Scenario: Vorstand reads sensitive data
- **WHEN** a Vorstand member calls `GET /api/members/{id}/sensitive`
- **THEN** the server returns `{ ciphertext, dek_enc_vorstand, member_salt?, dek_enc_member? }` with HTTP 200, or HTTP 204 if no sensitive data exists yet

#### Scenario: Member reads own sensitive data
- **WHEN** a `spieler` or `elternteil` calls `GET /api/members/{id}/sensitive` for their own linked profile
- **THEN** the server returns `{ ciphertext, dek_enc_member, member_salt }` (omitting `dek_enc_vorstand`)

#### Scenario: Non-Vorstand reads other member's data
- **WHEN** a non-Vorstand user calls `GET /api/members/{id}/sensitive` for a member not linked to their account
- **THEN** the server returns HTTP 403

#### Scenario: Vorstand writes sensitive data
- **WHEN** a Vorstand member calls `PUT /api/members/{id}/sensitive` with `{ ciphertext, dek_enc_vorstand, dek_enc_member?, member_salt? }`
- **THEN** the server upserts the row in `member_sensitive` and returns HTTP 200

---

### Requirement: Member list export (encrypted)
The system SHALL provide an export endpoint that returns all member data including encrypted sensitive blobs, for client-side decryption and CSV generation.

#### Scenario: Encrypted export for Vorstand
- **WHEN** a Vorstand member calls `GET /api/members/export-encrypted`
- **THEN** the server returns a JSON array of all members, each including plaintext profile fields and the encrypted sensitive blob (`ciphertext`, `dek_enc_vorstand`) — no decryption on the server

## REMOVED Requirements

### Requirement: Member list export
**Reason:** Replaced by the encrypted export endpoint. The previous server-side CSV export at `GET /api/members/export` included sensitive plaintext fields (`date_of_birth`, `iban`, `street`, etc.) and cannot safely continue once those fields are encrypted server-side.
**Migration:** Use `GET /api/members/export-encrypted` and perform client-side CSV generation in the browser via the Vorstand vault.
