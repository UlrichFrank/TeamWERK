-- 002 down: Restore original 5-value role system and club_function column
PRAGMA foreign_keys = OFF;

-- 1a. Drop views that reference members
DROP VIEW IF EXISTS user_accessible_teams;

-- 1b. Restore members table with club_function column (take highest-priority function per member)
CREATE TABLE members_old (
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
    club_function           TEXT     CHECK(club_function IN ('spieler','trainer','vorstand','vorstand_beisitzer')),
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
INSERT INTO members_old
    SELECT m.id, m.first_name, m.last_name, m.date_of_birth, m.pass_number, m.jersey_number, m.position,
           m.status, m.user_id, m.created_at, m.updated_at, m.member_number, m.gender,
           (SELECT mcf.function FROM member_club_functions mcf WHERE mcf.member_id = m.id
            ORDER BY CASE mcf.function WHEN 'trainer' THEN 1 WHEN 'spieler' THEN 2
                     WHEN 'vorstand' THEN 3 ELSE 4 END LIMIT 1),
           m.street, m.zip, m.city, m.join_date, m.iban, m.photo_path, m.photo_visible,
           m.dsgvo_verarbeitung, m.dsgvo_verarbeitung_date,
           m.dsgvo_weitergabe, m.dsgvo_weitergabe_date,
           m.sepa_mandat, m.sepa_mandat_date, m.sepa_mandat_path,
           m.account_holder, m.welcome_email_sent_at
    FROM members m;
DROP TABLE members;
ALTER TABLE members_old RENAME TO members;
CREATE UNIQUE INDEX idx_members_member_number ON members(member_number) WHERE member_number IS NOT NULL;

-- 2. Drop junction table
DROP TABLE member_club_functions;

-- 3. Restore invitation_tokens with original role CHECK
CREATE TABLE invitation_tokens_old (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    email      TEXT     NOT NULL,
    team_id    INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    role       TEXT     NOT NULL DEFAULT 'elternteil'
               CHECK (role IN ('admin','vorstand','trainer','elternteil','spieler')),
    token      TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment    TEXT
);
INSERT INTO invitation_tokens_old
    SELECT id, email, team_id,
           CASE WHEN role = 'admin' THEN 'admin' ELSE 'elternteil' END,
           token, expires_at, used_at, created_at, comment
    FROM invitation_tokens;
DROP TABLE invitation_tokens;
ALTER TABLE invitation_tokens_old RENAME TO invitation_tokens;

-- 4. Restore users table with original 5-value role CHECK
CREATE TABLE users_old (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    email      TEXT     NOT NULL UNIQUE,
    password   TEXT     NOT NULL,
    role       TEXT     NOT NULL DEFAULT 'elternteil'
               CHECK (role IN ('admin','vorstand','trainer','elternteil','spieler')),
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
INSERT INTO users_old
    SELECT id, email, password,
           CASE WHEN role = 'admin' THEN 'admin' ELSE 'elternteil' END,
           team_id, created_at, updated_at, first_name, last_name, street, zip, city, photo_path
    FROM users;
DROP TABLE users;
ALTER TABLE users_old RENAME TO users;

-- 5. Recreate views dropped in step 1a
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
