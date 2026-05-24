DROP TABLE IF EXISTS member_sensitive;

ALTER TABLE clubs DROP COLUMN vorstand_kdf_salt;
ALTER TABLE clubs DROP COLUMN vorstand_key_check;
