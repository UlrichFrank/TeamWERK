-- 057_uebungsgruppen: Übungsgruppe als Kader-Variante ohne teams-Zwilling.
--
-- kader bekommt zwei Spalten (kind, name), age_class und gender werden
-- nullable. Eine Übungsgruppe (kind='practice') trägt einen Namen und weder
-- Altersklasse, Geschlecht, Jahrgang noch team_id; ein CHECK erzwingt das als
-- Alles-oder-nichts (design.md — Entscheidung 6). Die Abwesenheit der
-- teams-Zeile ist der Mechanismus, über den alle team-gebundenen Flächen für
-- Übungsgruppen unerreichbar bleiben.
--
-- SQLite kennt kein ALTER … DROP NOT NULL / ADD CHECK → Tabellen-Rebuild
-- (Muster wie Migration 018/034). migrate setzt beim Up PRAGMA
-- foreign_keys=OFF (internal/db/db.go), die eingehenden FKs sind damit
-- unkritisch. legacy_alter_table=ON verhindert, dass SQLite beim DROP/RENAME
-- die vier auf kader verweisenden Views (player_memberships, team_memberships,
-- trainer_memberships, user_accessible_teams) validiert und dadurch scheitert.
PRAGMA legacy_alter_table=ON;

CREATE TABLE kader_new (
    id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id            INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    age_class            TEXT,
    gender               TEXT     CHECK (gender IN ('m','f','mixed')),
    team_number          INTEGER  NOT NULL DEFAULT 1,
    team_id              INTEGER  REFERENCES teams(id),
    dedicated_birth_year INTEGER,
    games_per_season     INTEGER  NOT NULL DEFAULT 0,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    kind                 TEXT     NOT NULL DEFAULT 'team'
                         CHECK (kind IN ('team','practice')),
    name                 TEXT,
    -- Alles-oder-nichts je Variante. Der bestehende
    -- CHECK (gender IN ('m','f','mixed')) bleibt unverändert: in SQLite wertet
    -- NULL IN (…) zu NULL aus, und ein CHECK gilt bei NULL als erfüllt.
    CHECK (
         (kind = 'team'     AND age_class IS NOT NULL AND gender IS NOT NULL
                            AND name IS NULL)
      OR (kind = 'practice' AND age_class IS NULL AND gender IS NULL
                            AND name IS NOT NULL AND team_id IS NULL
                            AND dedicated_birth_year IS NULL)
    )
);

INSERT INTO kader_new (id, season_id, age_class, gender, team_number, team_id,
    dedicated_birth_year, games_per_season, created_at, updated_at, kind, name)
SELECT id, season_id, age_class, gender, team_number, team_id,
    dedicated_birth_year, games_per_season, created_at, updated_at, 'team', NULL
FROM kader;

DROP TABLE kader;
ALTER TABLE kader_new RENAME TO kader;

CREATE UNIQUE INDEX kader_unique ON kader(season_id, age_class, gender, team_number);
CREATE INDEX idx_kader_season ON kader(season_id);

-- Namenseindeutigkeit je Saison. Der bestehende UNIQUE(season_id, age_class,
-- gender, team_number) braucht keine Anpassung: SQLite behandelt NULLs in
-- UNIQUE-Indizes als distinkt, (season, NULL, NULL, 1) kollidiert also nie.
CREATE UNIQUE INDEX idx_kader_practice_name
    ON kader(season_id, name) WHERE kind = 'practice';
