CREATE TABLE kader (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    season_id   INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    age_class   TEXT    NOT NULL,
    gender      TEXT    NOT NULL CHECK (gender IN ('m', 'f', 'mixed')),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (season_id, age_class, gender)
);

CREATE TABLE kader_members (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    kader_id    INTEGER NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    member_id   INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kader_id, member_id)
);

CREATE INDEX idx_kader_season ON kader(season_id);
CREATE INDEX idx_kader_members_kader ON kader_members(kader_id);
CREATE INDEX idx_kader_members_member ON kader_members(member_id);
