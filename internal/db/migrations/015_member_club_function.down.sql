PRAGMA foreign_keys=OFF;

CREATE TABLE members_new (
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

INSERT INTO members_new
    SELECT id, first_name, last_name, date_of_birth, pass_number, jersey_number,
           position, status, user_id, created_at, updated_at
    FROM members;

DROP TABLE members;
ALTER TABLE members_new RENAME TO members;

PRAGMA foreign_keys=ON;
