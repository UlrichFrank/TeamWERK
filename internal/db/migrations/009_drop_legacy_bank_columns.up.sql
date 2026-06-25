-- Zero-Knowledge-Abschluss: Die alten serverseitig verschlüsselten ("v1:") Bank-/SEPA-Spalten
-- werden nach der Bestandsmigration (clientseitige Envelopes in member_sensitive bzw.
-- clubs.sepa_ciphertext/sepa_dek_enc) entfernt. Vorbedingung: Migration vollständig
-- durchgeführt (keine v1:-Werte mehr) und FIELD_ENCRYPTION_KEY aus der Umgebung entfernt.
ALTER TABLE members DROP COLUMN iban;
ALTER TABLE members DROP COLUMN account_holder;

ALTER TABLE clubs DROP COLUMN glaeubiger_id;
ALTER TABLE clubs DROP COLUMN iban;
ALTER TABLE clubs DROP COLUMN bic;
ALTER TABLE clubs DROP COLUMN kontoinhaber;
