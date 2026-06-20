-- Stammvereine als verwaltbare Entität (ersetzt die hardcodierte
-- Mitgliedsvereine[]-Liste in internal/beitragslauf/compute.go).
CREATE TABLE stammvereine (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    aktiv       INTEGER NOT NULL DEFAULT 1,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed der 8 bestehenden Mitgliedsvereine.
INSERT OR IGNORE INTO stammvereine (name, sort_order) VALUES
    ('SKG Gablenberg 1884',             1),
    ('SKG Stuttgart Max-Eyth-See 1898', 2),
    ('SportKultur Stuttgart',           3),
    ('Spvgg 1897 Cannstatt',            4),
    ('TB Gaisburg 1886',                5),
    ('TB Untertürkheim 1888',           6),
    ('TSV Stuttgart-Münster 1875/99',   7),
    ('TV Cannstatt 1846',               8);

-- FK auf stammvereine. Alle Bestands-Mitglieder erhalten NULL;
-- der Backfill home_club -> home_club_id ist ein separater, vorab
-- reviewbarer Schritt (deploy/stammverein-mapping-*.sql), NICHT Teil
-- dieser automatisch laufenden Migration.
ALTER TABLE members ADD COLUMN home_club_id INTEGER REFERENCES stammvereine(id);
