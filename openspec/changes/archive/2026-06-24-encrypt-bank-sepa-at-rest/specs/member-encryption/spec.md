## REMOVED Requirements

### Requirement: Envelope encryption for sensitive member data
**Reason**: Ersetzt durch serverseitige A1-Verschlüsselung (`bank-data-at-rest-encryption`). Die clientseitige Envelope-Variante mit separater `member_sensitive`-Tabelle wurde nie nach `main` übernommen.
**Migration**: Bankfelder werden in-place in den bestehenden Spalten (`members.iban`, `members.account_holder`) per AES-256-GCM verschlüsselt; keine `member_sensitive`-Tabelle. Erstverschlüsselung via `teamwerk encrypt-pii`.

### Requirement: Dual-key DEK access
**Reason**: A1 nutzt einen einzigen app-gehaltenen Schlüssel; Pro-Member-DEKs mit zwei Entschlüsselungspfaden entfallen.
**Migration**: Zugriff wird über `policy.CanDecryptBankData` (Eigentümer ∨ Eltern ∨ admin/vorstand/kassierer) autorisiert statt über Wrapped-DEKs.

### Requirement: Crypto primitives
**Reason**: WebCrypto-Primitiven im Browser (PBKDF2/AES-KW) entfallen mit dem Wechsel auf serverseitige stdlib-Krypto.
**Migration**: Serverseitig AES-256-GCM aus `crypto/aes`+`crypto/cipher`, Format `"v1:"+base64(nonce‖ciphertext)`.

### Requirement: DEK re-wrap on member password change
**Reason**: Ohne passwort-abgeleitete Schlüssel gibt es kein DEK-Re-Wrap beim Passwortwechsel.
**Migration**: Entfällt ersatzlos; der app-gehaltene Schlüssel ist von Nutzerpasswörtern unabhängig.

### Requirement: Passphrase rotation re-wraps all Vorstand DEKs
**Reason**: Kein Vorstands-Gruppenschlüssel mehr.
**Migration**: Schlüsselrotation erfolgt serverseitig über versionierten Ciphertext-Prefix und ein Re-Encrypt-Subcommand (siehe `design.md`, Open Questions).
