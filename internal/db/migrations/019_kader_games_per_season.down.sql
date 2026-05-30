PRAGMA foreign_keys=OFF;

CREATE TABLE kader_new (
    id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id            INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    age_class            TEXT     NOT NULL,
    gender               TEXT     NOT NULL CHECK (gender IN ('m', 'f', 'mixed')),
    team_number          INTEGER  NOT NULL DEFAULT 1,
    dedicated_birth_year INTEGER,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    team_id              INTEGER  REFERENCES teams(id)
);

INSERT INTO kader_new SELECT
    id, season_id, age_class, gender, team_number, dedicated_birth_year,
    created_at, updated_at, team_id
FROM kader;

DROP TABLE kader;
ALTER TABLE kader_new RENAME TO kader;

CREATE UNIQUE INDEX kader_unique ON kader(season_id, age_class, gender, team_number);
CREATE INDEX idx_kader_season ON kader(season_id);

PRAGMA foreign_keys=ON;
