PRAGMA foreign_keys=OFF;

UPDATE members SET club_function = NULL WHERE club_function = 'spieler';

CREATE TABLE members_new (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    first_name              TEXT    NOT NULL,
    last_name               TEXT    NOT NULL,
    date_of_birth           DATE,
    pass_number             TEXT    UNIQUE,
    jersey_number           INTEGER,
    position                TEXT,
    status                  TEXT    NOT NULL DEFAULT 'aktiv' CHECK (status IN ('aktiv','verletzt','pausiert','ausgetreten','passiv')),
    user_id                 INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    member_number           TEXT,
    gender                  TEXT    NOT NULL DEFAULT 'u' CHECK (gender IN ('m', 'f', 'u')),
    club_function           TEXT    CHECK(club_function IN ('trainer','vorstand','vorstand_beisitzer')),
    street                  TEXT,
    zip                     TEXT,
    city                    TEXT,
    join_date               DATE,
    iban                    TEXT,
    photo_path              TEXT,
    photo_visible           INTEGER NOT NULL DEFAULT 0,
    dsgvo_verarbeitung      INTEGER NOT NULL DEFAULT 0,
    dsgvo_verarbeitung_date DATE,
    dsgvo_weitergabe        INTEGER NOT NULL DEFAULT 0,
    dsgvo_weitergabe_date   DATE,
    sepa_mandat             INTEGER NOT NULL DEFAULT 0,
    sepa_mandat_date        DATE,
    sepa_mandat_path        TEXT,
    account_holder          TEXT,
    welcome_email_sent_at   TEXT
);

INSERT INTO members_new SELECT
    id, first_name, last_name, date_of_birth, pass_number, jersey_number, position,
    status, user_id, created_at, updated_at, member_number, gender, club_function,
    street, zip, city, join_date, iban, photo_path, photo_visible,
    dsgvo_verarbeitung, dsgvo_verarbeitung_date, dsgvo_weitergabe, dsgvo_weitergabe_date,
    sepa_mandat, sepa_mandat_date, sepa_mandat_path, account_holder, welcome_email_sent_at
FROM members;

DROP TABLE members;
ALTER TABLE members_new RENAME TO members;

CREATE UNIQUE INDEX idx_members_member_number ON members(member_number) WHERE member_number IS NOT NULL;

PRAGMA foreign_keys=ON;
