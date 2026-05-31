-- 002: Role model refactor
-- Separates users.role (admin|standard) from Vereinsfunktion (member club functions).
-- Introduces member_club_functions junction table for multi-valued club functions.
PRAGMA foreign_keys = OFF;

-- 1. Junction table for member club functions
CREATE TABLE member_club_functions (
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    function  TEXT    NOT NULL CHECK(function IN ('spieler','trainer','vorstand','vorstand_beisitzer')),
    PRIMARY KEY (member_id, function)
);

-- 2. Migrate existing single-valued club_function to junction table
INSERT INTO member_club_functions (member_id, function)
SELECT id, club_function FROM members WHERE club_function IS NOT NULL;

-- 3. Recreate users table with simplified role CHECK
CREATE TABLE users_new (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    email      TEXT     NOT NULL UNIQUE,
    password   TEXT     NOT NULL,
    role       TEXT     NOT NULL DEFAULT 'standard'
               CHECK (role IN ('admin','standard')),
    team_id    INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    first_name TEXT     NOT NULL DEFAULT '',
    last_name  TEXT     NOT NULL DEFAULT '',
    street     TEXT,
    zip        TEXT,
    city       TEXT,
    photo_path TEXT
);
INSERT INTO users_new
    SELECT id, email, password,
           CASE WHEN role = 'admin' THEN 'admin' ELSE 'standard' END,
           team_id, created_at, updated_at, first_name, last_name, street, zip, city, photo_path
    FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

-- 4. Recreate invitation_tokens with simplified role CHECK
CREATE TABLE invitation_tokens_new (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    email      TEXT     NOT NULL,
    team_id    INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    role       TEXT     NOT NULL DEFAULT 'standard'
               CHECK (role IN ('admin','standard')),
    token      TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment    TEXT
);
INSERT INTO invitation_tokens_new
    SELECT id, email, team_id,
           CASE WHEN role = 'admin' THEN 'admin' ELSE 'standard' END,
           token, expires_at, used_at, created_at, comment
    FROM invitation_tokens;
DROP TABLE invitation_tokens;
ALTER TABLE invitation_tokens_new RENAME TO invitation_tokens;

-- 5. Drop views that reference members (recreated after rename)
DROP VIEW IF EXISTS user_accessible_teams;

-- 6. Recreate members table without club_function column
CREATE TABLE members_new (
    id                      INTEGER  PRIMARY KEY AUTOINCREMENT,
    first_name              TEXT     NOT NULL,
    last_name               TEXT     NOT NULL,
    date_of_birth           DATE,
    pass_number             TEXT     UNIQUE,
    jersey_number           INTEGER,
    position                TEXT,
    status                  TEXT     NOT NULL DEFAULT 'aktiv'
                            CHECK (status IN ('aktiv','verletzt','pausiert','ausgetreten','passiv')),
    user_id                 INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    member_number           TEXT,
    gender                  TEXT     NOT NULL DEFAULT 'u' CHECK (gender IN ('m','f','u')),
    street                  TEXT,
    zip                     TEXT,
    city                    TEXT,
    join_date               DATE,
    iban                    TEXT,
    photo_path              TEXT,
    photo_visible           INTEGER  NOT NULL DEFAULT 0,
    dsgvo_verarbeitung      INTEGER  NOT NULL DEFAULT 0,
    dsgvo_verarbeitung_date DATE,
    dsgvo_weitergabe        INTEGER  NOT NULL DEFAULT 0,
    dsgvo_weitergabe_date   DATE,
    sepa_mandat             INTEGER  NOT NULL DEFAULT 0,
    sepa_mandat_date        DATE,
    sepa_mandat_path        TEXT,
    account_holder          TEXT,
    welcome_email_sent_at   TEXT
);
INSERT INTO members_new
    SELECT id, first_name, last_name, date_of_birth, pass_number, jersey_number, position,
           status, user_id, created_at, updated_at, member_number, gender,
           street, zip, city, join_date, iban, photo_path, photo_visible,
           dsgvo_verarbeitung, dsgvo_verarbeitung_date,
           dsgvo_weitergabe, dsgvo_weitergabe_date,
           sepa_mandat, sepa_mandat_date, sepa_mandat_path,
           account_holder, welcome_email_sent_at
    FROM members;
DROP TABLE members;
ALTER TABLE members_new RENAME TO members;
CREATE UNIQUE INDEX idx_members_member_number ON members(member_number) WHERE member_number IS NOT NULL;

-- 7. Recreate views that were dropped above
CREATE VIEW user_accessible_teams AS
SELECT m.user_id, k.team_id, k.season_id
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
JOIN members m ON m.id = km.member_id
WHERE k.team_id IS NOT NULL
UNION ALL
SELECT fl.parent_user_id AS user_id, k.team_id, k.season_id
FROM family_links fl
JOIN kader_members km ON km.member_id = fl.member_id
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL
UNION ALL
SELECT m.user_id, k.team_id, k.season_id
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
JOIN members m ON m.id = kt.member_id
WHERE k.team_id IS NOT NULL;

PRAGMA foreign_keys = ON;
