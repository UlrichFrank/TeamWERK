-- Save kader_members, then drop kader_members (FK dependency)
CREATE TABLE kader_members_backup AS SELECT * FROM kader_members;
DROP TABLE kader_members;

-- Rebuild kader without team_number and dedicated_birth_year; only keep team_number=1 entries
CREATE TABLE kader_new (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id  INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    age_class  TEXT     NOT NULL,
    gender     TEXT     NOT NULL CHECK (gender IN ('m', 'f', 'mixed')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (season_id, age_class, gender)
);

INSERT INTO kader_new (id, season_id, age_class, gender, created_at, updated_at)
SELECT id, season_id, age_class, gender, created_at, updated_at FROM kader WHERE team_number = 1;

DROP TABLE kader;
ALTER TABLE kader_new RENAME TO kader;

-- Restore kader_members for remaining (team_number=1) kader entries
CREATE TABLE kader_members (
    id        INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id  INTEGER  NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    member_id INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kader_id, member_id)
);

INSERT INTO kader_members
SELECT b.* FROM kader_members_backup b
WHERE EXISTS (SELECT 1 FROM kader k WHERE k.id = b.kader_id);

DROP TABLE kader_members_backup;

CREATE INDEX idx_kader_season ON kader(season_id);
CREATE INDEX idx_kader_members_kader ON kader_members(kader_id);
CREATE INDEX idx_kader_members_member ON kader_members(member_id);
