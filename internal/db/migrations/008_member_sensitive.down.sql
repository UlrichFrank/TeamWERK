ALTER TABLE members DROP COLUMN sepa_mandat_dek_enc;
ALTER TABLE clubs DROP COLUMN sepa_dek_enc;
ALTER TABLE clubs DROP COLUMN sepa_ciphertext;
ALTER TABLE clubs DROP COLUMN vorstand_key_check;
ALTER TABLE clubs DROP COLUMN vorstand_kdf_salt;
ALTER TABLE clubs DROP COLUMN group_private_key_enc;
ALTER TABLE clubs DROP COLUMN group_public_key;
DROP TABLE member_sensitive;
