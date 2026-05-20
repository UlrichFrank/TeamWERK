PRAGMA foreign_keys=OFF;

CREATE TABLE members_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    first_name      TEXT    NOT NULL,
    last_name       TEXT    NOT NULL,
    date_of_birth   DATE,
    member_number   TEXT,
    pass_number     TEXT    UNIQUE,
    jersey_number   INTEGER,
    position        TEXT,
    gender          TEXT    NOT NULL DEFAULT 'u' CHECK (gender IN ('m', 'f', 'u')),
    status          TEXT    NOT NULL DEFAULT 'aktiv' CHECK (status IN ('aktiv','verletzt','pausiert','ausgetreten')),
    club_function   TEXT    CHECK(club_function IN ('trainer','vorstand','vorstand_beisitzer')),
    user_id         INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO members_new
    SELECT id, first_name, last_name, date_of_birth, member_number, pass_number,
           jersey_number, position, gender, status, club_function,
           user_id, created_at, updated_at
    FROM members;

DROP TABLE members;
ALTER TABLE members_new RENAME TO members;

CREATE UNIQUE INDEX IF NOT EXISTS idx_members_member_number
    ON members(member_number) WHERE member_number IS NOT NULL;

PRAGMA foreign_keys=ON;
