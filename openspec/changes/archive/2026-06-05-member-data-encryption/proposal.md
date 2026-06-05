## Why

Personenbezogene Mitgliedsdaten (IBAN, Adresse, Geburtsdatum) liegen derzeit im Klartext in der SQLite-Datenbank. Ein übersehener Bug — fehlende Auth-Middleware, zu breites SELECT, versehentliches Logging — würde diese Daten ungeschützt exponieren. Client-seitige Verschlüsselung stellt sicher, dass der Server ausschließlich Ciphertext speichert und verarbeitet, sodass selbst ein erfolgreicher Leak keinen Zugang zu den Rohdaten bietet.

## What Changes

- Sensible Felder (`date_of_birth`, `street`, `zip`, `city`, `iban`, `account_holder`) werden aus der `members`-Tabelle entfernt und verschlüsselt in einer neuen Tabelle `member_sensitive` gespeichert
- Verschlüsselung und Entschlüsselung finden ausschließlich im Browser statt (WebCrypto API — kein npm, kein Server-Decrypt)
- Vorstand-Mitglieder entsperren den Tresor mit einer geteilten Passphrase; ihr Key wird per PBKDF2 im Browser abgeleitet und in `sessionStorage` gehalten
- Mitglieder mit verknüpftem User-Account können ihre eigenen Daten via Login-Passwort entschlüsseln (DEK wird beim Schreiben zusätzlich mit dem member_key gewrappt)
- Der CSV-Export läuft vollständig im Browser: der neue Endpoint gibt Ciphertext zurück, der Client entschlüsselt und generiert die Datei lokal
- Passphrase-Rotation für den Vorstand: Browser-Workflow re-wrapped alle DEKs ohne Server-Kenntnis des Klartexts
- **BREAKING**: `date_of_birth`, `street`, `zip`, `city`, `iban`, `account_holder` verschwinden aus allen bestehenden API-Responses; Clients, die diese Felder direkt auslesen, müssen auf die neuen verschlüsselten Endpunkte migrieren

## Capabilities

### New Capabilities

- `member-encryption`: Envelope-Verschlüsselung sensibler Mitgliedsdaten (AES-GCM + AES-KW + PBKDF2) mit Dual-Key-Zugriff (Vorstand-Gruppenkey + optionaler Member-Key)
- `vorstand-vault`: Vorstand-Tresor-UI — Passphrase-Dialog, sessionStorage-Key-Caching, Inaktivitäts-Timer, Rotations-Workflow

### Modified Capabilities

- `members`: Sensitive Felder werden aus den bestehenden CRUD-Responses entfernt; Lesen/Schreiben dieser Felder erfordert jetzt den verschlüsselten Pfad

## Impact

**Backend (Go):**
- Neue DB-Migration: Tabelle `member_sensitive`, Spalten `vorstand_kdf_salt` + `vorstand_key_check` in `clubs`
- Sensitive Felder aus `members`-SELECT-Abfragen und `Member`-Struct entfernt
- Neue Endpunkte: `GET/PUT /api/members/{id}/sensitive`, `GET /api/members/export-encrypted`, `PUT /api/admin/rotate-encryption`, `GET /api/admin/encryption-config`
- Passwort-Änderungs-Flow (`PUT /api/auth/change-password`) muss DEK_enc_member neu wrappen

**Frontend (React/TypeScript):**
- Neue Crypto-Utility (`lib/crypto.ts`) mit WebCrypto-Wrappern
- Vorstand-Tresor-Komponente (Passphrase-Dialog, sessionStorage-Management)
- Mitglieder-Detailseite: sensible Felder werden asynchron entschlüsselt angezeigt
- Export-Seite: client-seitiger CSV-Generator ersetzt den Server-Export

**Datenbank:**
- Einmalige Datenmigration: bestehende Klartext-Felder müssen vom Vorstand initial verschlüsselt werden (kein automatisches Server-side-Migrate möglich)
