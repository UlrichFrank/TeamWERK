-- SEPA-Stammdaten des Vereins
ALTER TABLE clubs ADD COLUMN glaeubiger_id TEXT;
ALTER TABLE clubs ADD COLUMN iban          TEXT;
ALTER TABLE clubs ADD COLUMN bic           TEXT;
ALTER TABLE clubs ADD COLUMN kontoinhaber  TEXT;

-- Beitragsmatrix mit Historie (Beträge in Cent)
CREATE TABLE beitrags_saetze (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    kategorie   TEXT NOT NULL CHECK (kategorie IN (
        'aktiv_ohne',
        'aktiv_mit',
        'passiv'
    )),
    betrag_eur  INTEGER NOT NULL,
    valid_from  DATE NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_beitrags_saetze_kat_valid ON beitrags_saetze(kategorie, valid_from);

-- Seed aus Beitragsordnung (Anlage 1, beschlossen 22.04.2026)
INSERT OR IGNORE INTO beitrags_saetze (kategorie, betrag_eur, valid_from) VALUES
    ('aktiv_ohne', 22600, '2026-07-01'),
    ('aktiv_mit',   9600, '2026-07-01'),
    ('passiv',      6000, '2027-01-01');
