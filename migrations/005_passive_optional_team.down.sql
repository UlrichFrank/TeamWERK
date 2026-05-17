PRAGMA foreign_keys=OFF;

CREATE TABLE membership_requests_old (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    email       TEXT     NOT NULL,
    team_id     INTEGER  NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    status      TEXT     NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    handled_by  INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    handled_at  DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO membership_requests_old SELECT * FROM membership_requests WHERE team_id IS NOT NULL;
DROP TABLE membership_requests;
ALTER TABLE membership_requests_old RENAME TO membership_requests;

CREATE TABLE members_old (
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
INSERT INTO members_old SELECT * FROM members WHERE status != 'passiv';
DROP TABLE members;
ALTER TABLE members_old RENAME TO members;

PRAGMA foreign_keys=ON;
