## ADDED Requirements

### Requirement: Envelope encryption for sensitive member data
The system SHALL store the fields `date_of_birth`, `street`, `zip`, `city`, `iban`, and `account_holder` exclusively as AES-GCM-256 ciphertext in the `member_sensitive` table. The server SHALL NOT store, log, or return these fields in plaintext at any point.

#### Scenario: Vorstand writes sensitive data
- **WHEN** a Vorstand member submits sensitive data for a member via `PUT /api/members/{id}/sensitive`
- **THEN** the browser encrypts the payload with a random DEK before transmission, and the server stores only the ciphertext and wrapped DEK

#### Scenario: Server never sees plaintext
- **WHEN** `GET /api/members/{id}/sensitive` is called
- **THEN** the response contains only `ciphertext`, `dek_enc_vorstand`, `dek_enc_member` (if present), and `member_salt` — never decrypted field values

#### Scenario: Existing member data not in member_sensitive
- **WHEN** a member has no entry in `member_sensitive`
- **THEN** the sensitive endpoint returns HTTP 204 (no content), and the frontend displays empty fields pending input

---

### Requirement: Dual-key DEK access
The system SHALL support two independent decryption paths for the per-member DEK: one via the Vorstand group key, one via the member's own login-password-derived key.

#### Scenario: Vorstand decrypts any member
- **WHEN** a Vorstand member has unlocked the vault (vorstand_key in sessionStorage)
- **THEN** they can decrypt any member's sensitive data using `dek_enc_vorstand`

#### Scenario: Member decrypts own data
- **WHEN** a member with a linked user account accesses their own profile
- **THEN** they can decrypt their own sensitive data using `dek_enc_member` and a key derived from their login password

#### Scenario: Member without user account
- **WHEN** a member has no linked user account (`user_id` is NULL on the members record)
- **THEN** `dek_enc_member` and `member_salt` are NULL; only Vorstand can access the data

#### Scenario: Unlinked member's data is inaccessible to non-Vorstand
- **WHEN** a non-Vorstand user requests sensitive data for a member not linked to their account
- **THEN** the server returns HTTP 403

---

### Requirement: Crypto primitives
The system SHALL use exclusively browser-native WebCrypto primitives. No external JavaScript cryptography library SHALL be introduced.

#### Scenario: Key derivation parameters
- **WHEN** a key is derived from a passphrase or password
- **THEN** the system uses PBKDF2 with SHA-256, 600 000 iterations, and a random 32-byte salt, producing a 256-bit key

#### Scenario: Data encryption format
- **WHEN** a payload is encrypted
- **THEN** the system uses AES-GCM-256 with a random 12-byte IV prepended to the ciphertext, and the result is base64-encoded for storage

#### Scenario: DEK wrapping format
- **WHEN** a DEK is wrapped
- **THEN** the system uses AES-KW with a 256-bit wrapping key, and the result is base64-encoded

---

### Requirement: DEK re-wrap on member password change
When a member changes their login password, the system SHALL re-derive the old and new member keys in the browser and replace `dek_enc_member` with a value wrapped under the new key.

#### Scenario: Password change triggers DEK re-wrap
- **WHEN** a member submits a password change via `PUT /api/auth/change-password`
- **THEN** the request includes the old password (for old key derivation), the new password, and the re-wrapped `dek_enc_member`; the server updates both the password hash and `dek_enc_member` atomically

#### Scenario: Member has no sensitive data yet
- **WHEN** a member changes their password but has no entry in `member_sensitive`
- **THEN** no DEK re-wrap is needed; the change proceeds normally

---

### Requirement: Passphrase rotation re-wraps all Vorstand DEKs
When the Vorstand group passphrase is rotated, the browser SHALL re-wrap every member's DEK under the new Vorstand key without the server ever seeing either key.

#### Scenario: Successful rotation
- **WHEN** a Vorstand member submits a rotation request via `PUT /api/rotate-encryption`
- **THEN** the request body contains the new salt, new key-check value, and an array of `{ member_id, dek_enc_vorstand }` for all members; the server replaces all affected rows atomically

#### Scenario: Rotation is atomic
- **WHEN** a rotation request is received
- **THEN** all DEK updates are applied in a single database transaction; partial updates are not persisted
