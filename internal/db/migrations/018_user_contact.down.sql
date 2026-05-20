DROP TABLE IF EXISTS user_visibility;
DROP TABLE IF EXISTS user_phones;

PRAGMA foreign_keys=OFF;

CREATE TABLE users_new (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    password    TEXT    NOT NULL,
    role        TEXT    NOT NULL DEFAULT 'elternteil' CHECK (role IN ('admin','vorstand','trainer','elternteil','spieler')),
    team_id     INTEGER REFERENCES teams(id) ON DELETE SET NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO users_new
    SELECT id, email, name, password, role, team_id, created_at, updated_at
    FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

PRAGMA foreign_keys=ON;
