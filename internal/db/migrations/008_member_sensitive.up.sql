-- Zero-Knowledge-Vault für Bank-/SEPA-PII (clientseitige Envelope-Verschlüsselung).
-- Pro Mitglied: AES-GCM-Blob + der mit dem geteilten Gruppen-Schlüssel gewrappte Data-Key.
-- Es gibt bewusst keinen Eigentümer-Wrap (nur die Finance-Gruppe liest).
CREATE TABLE member_sensitive (
    member_id        INTEGER PRIMARY KEY REFERENCES members(id) ON DELETE CASCADE,
    ciphertext       TEXT NOT NULL,   -- base64(IV ‖ AES-GCM(payload, DEK))
    dek_enc_vorstand TEXT NOT NULL    -- base64(AES-KW(DEK, vorstand_key))
);

-- Tresor-Hilfswerte: Salt + Key-Check. Die Passphrase selbst wird nie gespeichert.
ALTER TABLE clubs ADD COLUMN vorstand_kdf_salt TEXT;   -- base64, PBKDF2-Salt
ALTER TABLE clubs ADD COLUMN vorstand_key_check TEXT;  -- base64(IV ‖ AES-GCM("ok", vorstand_key))
