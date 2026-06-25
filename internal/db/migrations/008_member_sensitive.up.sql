-- Zero-Knowledge-Vault für Bank-/SEPA-PII (clientseitige Envelope-Verschlüsselung).
-- Pro Mitglied: AES-GCM-Blob + der mit dem geteilten Gruppen-Schlüssel gewrappte Data-Key.
-- Es gibt bewusst keinen Eigentümer-Wrap (nur die Finance-Gruppe liest).
CREATE TABLE member_sensitive (
    member_id        INTEGER PRIMARY KEY REFERENCES members(id) ON DELETE CASCADE,
    ciphertext       TEXT NOT NULL,   -- base64(IV ‖ AES-GCM(payload, DEK))
    dek_enc_vorstand TEXT NOT NULL    -- base64(AES-KW(DEK, vorstand_key))
);

-- Tresor (Modell B — asymmetrisches Gruppen-Keypair). Gespeichert werden nur:
--   - der öffentliche Schlüssel (nicht geheim, erlaubt jedem das Schreiben),
--   - der mit PBKDF2(passphrase) verschlüsselte private Schlüssel (zum Lesen),
--   - Salt + Key-Check zur Passphrase-Verifikation. Die Passphrase selbst nie.
ALTER TABLE clubs ADD COLUMN group_public_key TEXT;       -- SPKI base64 (öffentlich)
ALTER TABLE clubs ADD COLUMN group_private_key_enc TEXT;  -- base64(IV ‖ AES-GCM(PKCS8, KEK))
ALTER TABLE clubs ADD COLUMN vorstand_kdf_salt TEXT;      -- base64, PBKDF2-Salt
ALTER TABLE clubs ADD COLUMN vorstand_key_check TEXT;     -- base64(IV ‖ AES-GCM("ok", KEK))

-- Vereins-SEPA-Stammdaten als clientseitiger Envelope (glaeubiger_id/iban/bic/kontoinhaber
-- in einem JSON-Blob, an den Gruppen-Public-Key verschlüsselt). Ersetzt die alten v1-Felder.
ALTER TABLE clubs ADD COLUMN sepa_ciphertext TEXT;        -- base64(IV ‖ AES-GCM(JSON, DEK))
ALTER TABLE clubs ADD COLUMN sepa_dek_enc TEXT;           -- RSA-OAEP(DEK, group_public_key)
