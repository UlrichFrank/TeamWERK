CREATE TABLE member_sensitive (
    member_id        INTEGER PRIMARY KEY REFERENCES members(id) ON DELETE CASCADE,
    ciphertext       TEXT NOT NULL,
    dek_enc_vorstand TEXT NOT NULL,
    dek_enc_member   TEXT,
    member_salt      TEXT
);

ALTER TABLE clubs ADD COLUMN vorstand_kdf_salt TEXT;
ALTER TABLE clubs ADD COLUMN vorstand_key_check TEXT;
