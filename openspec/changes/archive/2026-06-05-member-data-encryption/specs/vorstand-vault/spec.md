## ADDED Requirements

### Requirement: Vault passphrase entry before sensitive data access
The system SHALL require Vorstand members to enter the vault passphrase before any sensitive member data is displayed or editable. The passphrase is never transmitted to the server.

#### Scenario: First access triggers passphrase dialog
- **WHEN** a Vorstand member navigates to a member detail page and the vault is not yet unlocked
- **THEN** a modal dialog prompts for the vault passphrase before sensitive fields are shown

#### Scenario: Wrong passphrase rejected client-side
- **WHEN** a Vorstand member enters an incorrect passphrase
- **THEN** the browser derives the key, attempts to decrypt `vorstand_key_check`, fails, and displays "Falsche Passphrase" — no server request is made

#### Scenario: Correct passphrase unlocks vault
- **WHEN** the derived key successfully decrypts `vorstand_key_check` to the value "ok"
- **THEN** the key is stored in `sessionStorage['vk']` and sensitive fields are decrypted and displayed

---

### Requirement: Vault session expiry
The vault key in `sessionStorage` SHALL expire after 30 minutes of inactivity and on tab/window close.

#### Scenario: Inactivity timeout
- **WHEN** 30 minutes pass without user interaction after the vault was unlocked
- **THEN** `sessionStorage['vk']` is deleted and sensitive fields are hidden; re-entry of passphrase is required

#### Scenario: Tab close clears key
- **WHEN** the browser tab is closed or the session ends
- **THEN** `sessionStorage` is cleared automatically by the browser, removing the vault key

---

### Requirement: Vault initial setup
Before any sensitive data can be written, the Vorstand SHALL perform a one-time setup to establish the group passphrase. The setup stores a salt and a key-check value on the server; the passphrase itself is never stored.

#### Scenario: Setup generates salt and key-check value
- **WHEN** a Vorstand member completes the setup form at `/admin/tresor-einrichten`
- **THEN** the browser generates a random 32-byte salt, derives the key via PBKDF2, encrypts the string "ok" with AES-GCM, and posts `{ vorstand_kdf_salt, vorstand_key_check }` to `PUT /api/admin/encryption-config`

#### Scenario: Setup can only run once
- **WHEN** a salt and key-check value already exist in the database
- **THEN** the setup endpoint returns HTTP 409 and a message directing to the rotation workflow instead

---

### Requirement: Passphrase rotation
The Vorstand SHALL be able to rotate the group passphrase. Rotation re-wraps all member DEKs in the browser; the server never receives either the old or the new passphrase.

#### Scenario: Rotation requires unlocked vault
- **WHEN** a Vorstand member opens the rotation workflow
- **THEN** the vault must already be unlocked (old key present in sessionStorage); otherwise the passphrase dialog is shown first

#### Scenario: Rotation re-wraps all DEKs
- **WHEN** the new passphrase is confirmed and rotation is submitted
- **THEN** the browser derives the new key, fetches all `{ member_id, dek_enc_vorstand }` records, re-wraps each DEK, and posts the batch plus new salt and key-check value to `PUT /api/admin/rotate-encryption`

#### Scenario: New key replaces old key in session
- **WHEN** rotation completes successfully
- **THEN** `sessionStorage['vk']` is updated to the new key so the session remains active

---

### Requirement: Initial data migration
After vault setup, the Vorstand SHALL be able to migrate existing plaintext sensitive data from the old database columns into the encrypted `member_sensitive` table via a browser-driven migration workflow.

#### Scenario: Migration workflow encrypts and posts per member
- **WHEN** a Vorstand member runs the migration at `/admin/tresor-migration`
- **THEN** the browser fetches members with legacy plaintext fields still present, encrypts each locally, posts to `PUT /api/members/{id}/sensitive`, and shows a progress counter

#### Scenario: Migration is idempotent
- **WHEN** a member already has an entry in `member_sensitive`
- **THEN** that member is skipped during migration; no data is overwritten

---

### Requirement: Client-side CSV export
The Vorstand SHALL be able to download a CSV of all member data including decrypted sensitive fields. Decryption occurs in the browser; the server returns only ciphertext.

#### Scenario: Export decrypts all records in browser
- **WHEN** a Vorstand member with an unlocked vault triggers the export
- **THEN** the browser fetches all member records including encrypted sensitive blobs, decrypts each, assembles the CSV rows, and triggers a file download — no plaintext is sent to or from the server

#### Scenario: Export blocked if vault not unlocked
- **WHEN** a Vorstand member triggers the export without an unlocked vault
- **THEN** the passphrase dialog is shown before the export proceeds
