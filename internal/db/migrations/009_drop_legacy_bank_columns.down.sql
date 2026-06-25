-- Legt die Legacy-Bank-/SEPA-Spalten als nullable TEXT wieder an. ACHTUNG: Der frühere
-- (server-verschlüsselte) Inhalt ist NICHT wiederherstellbar — die Spalten kommen leer zurück.
ALTER TABLE members ADD COLUMN iban TEXT;
ALTER TABLE members ADD COLUMN account_holder TEXT;

ALTER TABLE clubs ADD COLUMN glaeubiger_id TEXT;
ALTER TABLE clubs ADD COLUMN iban TEXT;
ALTER TABLE clubs ADD COLUMN bic TEXT;
ALTER TABLE clubs ADD COLUMN kontoinhaber TEXT;
