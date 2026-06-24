## REMOVED Requirements

### Requirement: Vault passphrase entry before sensitive data access
**Reason**: Der browserseitige Vorstands-Tresor entfällt; A1 entschlüsselt serverseitig mit app-gehaltenem Schlüssel, Zugriff via `policy.CanDecryptBankData`.
**Migration**: Kein Passphrase-Eintrag mehr nötig; berechtigte Rollen sehen Klartext direkt über die regulären Endpoints.

### Requirement: Vault session expiry
**Reason**: Ohne Tresor gibt es keine Tresor-Session.
**Migration**: Entfällt; reguläre JWT-Session-Mechanik bleibt unverändert.

### Requirement: Vault initial setup
**Reason**: Kein Gruppen-Passphrase-Setup mehr.
**Migration**: Ersetzt durch einmalige Schlüsselerzeugung `teamwerk gen-encryption-key` + Ablage in `/etc/teamwerk/env`.

### Requirement: Passphrase rotation
**Reason**: Keine Tresor-Passphrase.
**Migration**: Schlüsselrotation serverseitig über versionierten Ciphertext-Prefix und Re-Encrypt-Subcommand.

### Requirement: Initial data migration
**Reason**: Ersetzt durch das serverseitige, idempotente Subcommand `encrypt-pii`.
**Migration**: `teamwerk encrypt-pii` verschlüsselt den Bestand in-place (siehe `bank-data-at-rest-encryption`).

### Requirement: Client-side CSV export
**Reason**: Der Server ist in A1 nicht blind; Export/SEPA-XML bleibt serverseitig und muss nicht in den Browser portiert werden.
**Migration**: SEPA-Export (`POST /api/fee-run/export`) bleibt serverseitig; er entschlüsselt die benötigten Felder zur Laufzeit hinter der bestehenden Authz.
