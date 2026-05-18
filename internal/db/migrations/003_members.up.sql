CREATE TABLE members (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    first_name      TEXT    NOT NULL,
    last_name       TEXT    NOT NULL,
    date_of_birth   DATE,
    pass_number     TEXT    UNIQUE,
    jersey_number   INTEGER,
    position        TEXT,
    status          TEXT    NOT NULL DEFAULT 'aktiv' CHECK (status IN ('aktiv','verletzt','pausiert','ausgetreten')),
    user_id         INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE team_memberships (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    member_id   INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    team_id     INTEGER NOT NULL REFERENCES teams(id)   ON DELETE CASCADE,
    season_id   INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    is_primary  INTEGER NOT NULL DEFAULT 0,
    UNIQUE (member_id, team_id, season_id)
);

CREATE TABLE family_links (
    parent_user_id  INTEGER NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    member_id       INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    PRIMARY KEY (parent_user_id, member_id)
);

CREATE TABLE vehicle_info (
    user_id     INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    seats       INTEGER NOT NULL DEFAULT 0,
    notes       TEXT,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
