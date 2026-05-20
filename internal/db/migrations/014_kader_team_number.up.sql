-- Save kader_members data before rebuilding kader (FK dependency)
CREATE TABLE kader_members_backup AS SELECT * FROM kader_members;
DROP TABLE kader_members;

-- Rebuild kader with team_number + dedicated_birth_year, dropping old UNIQUE(season,class,gender)
CREATE TABLE kader_new (
    id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id            INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    age_class            TEXT     NOT NULL,
    gender               TEXT     NOT NULL CHECK (gender IN ('m', 'f', 'mixed')),
    team_number          INTEGER  NOT NULL DEFAULT 1,
    dedicated_birth_year INTEGER,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO kader_new (id, season_id, age_class, gender, team_number, dedicated_birth_year, created_at, updated_at)
SELECT id, season_id, age_class, gender, 1, NULL, created_at, updated_at FROM kader;

DROP TABLE kader;
ALTER TABLE kader_new RENAME TO kader;

-- Recreate kader_members with FK references to the new kader table
CREATE TABLE kader_members (
    id        INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id  INTEGER  NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    member_id INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kader_id, member_id)
);

INSERT INTO kader_members SELECT * FROM kader_members_backup;
DROP TABLE kader_members_backup;

CREATE UNIQUE INDEX kader_unique ON kader(season_id, age_class, gender, team_number);
CREATE INDEX idx_kader_season ON kader(season_id);
CREATE INDEX idx_kader_members_kader ON kader_members(kader_id);
CREATE INDEX idx_kader_members_member ON kader_members(member_id);
