DROP VIEW team_memberships;

CREATE TABLE team_memberships (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    member_id   INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    team_id     INTEGER NOT NULL REFERENCES teams(id)   ON DELETE CASCADE,
    season_id   INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    is_primary  INTEGER NOT NULL DEFAULT 0,
    UNIQUE (member_id, team_id, season_id)
);
