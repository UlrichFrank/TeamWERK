-- 039 down: Remove 'anwaerter' from members.status CHECK constraint
-- Existing anwaerter rows are set to 'aktiv' before removing the status value.
PRAGMA foreign_keys = OFF;

DROP VIEW IF EXISTS user_accessible_teams;
DROP VIEW IF EXISTS player_memberships;
DROP VIEW IF EXISTS trainer_memberships;

CREATE TABLE members_new (
    id                      INTEGER  PRIMARY KEY AUTOINCREMENT,
    first_name              TEXT     NOT NULL,
    last_name               TEXT     NOT NULL,
    date_of_birth           DATE,
    pass_number             TEXT     UNIQUE,
    jersey_number           INTEGER,
    position                TEXT,
    status                  TEXT     NOT NULL DEFAULT 'aktiv'
                            CHECK (status IN ('aktiv','verletzt','pausiert','ausgetreten','passiv','honorar')),
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
    welcome_email_sent_at   TEXT,
    home_club               TEXT,
    phones_visible          INTEGER  NOT NULL DEFAULT 0,
    address_visible         INTEGER  NOT NULL DEFAULT 0,
    email_visible           INTEGER  NOT NULL DEFAULT 0,
    absences_public         INTEGER  NOT NULL DEFAULT 0,
    beitragsfrei            INTEGER  NOT NULL DEFAULT 0,
    zweitspielrecht         INTEGER  NOT NULL DEFAULT 0
);

INSERT INTO members_new
    SELECT id, first_name, last_name, date_of_birth, pass_number, jersey_number, position,
           CASE WHEN status = 'anwaerter' THEN 'aktiv' ELSE status END,
           user_id, created_at, updated_at, member_number, gender,
           street, zip, city, join_date, iban, photo_path, photo_visible,
           dsgvo_verarbeitung, dsgvo_verarbeitung_date,
           dsgvo_weitergabe, dsgvo_weitergabe_date,
           sepa_mandat, sepa_mandat_date, sepa_mandat_path,
           account_holder, welcome_email_sent_at, home_club,
           phones_visible, address_visible, email_visible,
           absences_public, beitragsfrei, zweitspielrecht
    FROM members;

DROP TABLE members;
ALTER TABLE members_new RENAME TO members;

CREATE UNIQUE INDEX idx_members_member_number ON members(member_number) WHERE member_number IS NOT NULL;

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
WHERE k.team_id IS NOT NULL
UNION ALL
SELECT m.user_id, k.team_id, k.season_id
FROM kader_extended_members kem
JOIN kader k ON k.id = kem.kader_id
JOIN members m ON m.id = kem.member_id
WHERE k.team_id IS NOT NULL;

CREATE VIEW player_memberships AS
SELECT km.id, km.member_id, k.team_id, k.season_id
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL;

CREATE VIEW trainer_memberships AS
SELECT kt.kader_id * 100000 + kt.member_id AS id, kt.member_id, k.team_id, k.season_id
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
WHERE k.team_id IS NOT NULL;

PRAGMA foreign_keys = ON;
