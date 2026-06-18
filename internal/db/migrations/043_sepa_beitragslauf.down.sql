DROP INDEX IF EXISTS idx_beitrags_saetze_kat_valid;
DROP TABLE IF EXISTS beitrags_saetze;

ALTER TABLE clubs DROP COLUMN kontoinhaber;
ALTER TABLE clubs DROP COLUMN bic;
ALTER TABLE clubs DROP COLUMN iban;
ALTER TABLE clubs DROP COLUMN glaeubiger_id;
