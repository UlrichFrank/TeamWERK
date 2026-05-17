PRAGMA foreign_keys=OFF;

-- membership_requests: team_id nullable machen
CREATE TABLE membership_requests_new (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    email       TEXT     NOT NULL,
    team_id     INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    status      TEXT     NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    handled_by  INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    handled_at  DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO membership_requests_new SELECT * FROM membership_requests;
DROP TABLE membership_requests;
ALTER TABLE membership_requests_new RENAME TO membership_requests;

-- members: Status 'passiv' hinzufügen
CREATE TABLE members_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    first_name      TEXT    NOT NULL,
    last_name       TEXT    NOT NULL,
    date_of_birth   DATE,
    pass_number     TEXT    UNIQUE,
    jersey_number   INTEGER,
    position        TEXT,
    status          TEXT    NOT NULL DEFAULT 'aktiv' CHECK (status IN ('aktiv','verletzt','pausiert','ausgetreten','passiv')),
    user_id         INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO members_new SELECT * FROM members;
DROP TABLE members;
ALTER TABLE members_new RENAME TO members;

PRAGMA foreign_keys=ON;
