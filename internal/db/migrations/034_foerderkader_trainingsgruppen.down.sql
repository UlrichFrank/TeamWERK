-- Reverse of 034.
-- ACHTUNG: der members-Rebuild auf die alte CHECK schlägt fehl, wenn noch
-- Zeilen mit status='foerderkind' existieren. Vor dem Down solche Zeilen
-- bereinigen/umsetzen (bewusst kein stilles DELETE hier).
-- legacy_alter_table=ON: siehe Kommentar in der .up.sql (Views auf members).
PRAGMA legacy_alter_table=ON;

DROP TABLE IF EXISTS training_group_categories;

CREATE TABLE members_new (
    id                      INTEGER  PRIMARY KEY AUTOINCREMENT,
    first_name              TEXT     NOT NULL,
    last_name               TEXT     NOT NULL,
    date_of_birth           DATE,
    pass_number             TEXT     UNIQUE,
    jersey_number           INTEGER,
    position                TEXT,
    status                  TEXT     NOT NULL DEFAULT 'aktiv'
                            CHECK (status IN ('aktiv','verletzt','pausiert','ausgetreten','passiv','honorar','anwaerter')),
    user_id                 INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    member_number           TEXT,
    gender                  TEXT     NOT NULL DEFAULT 'u' CHECK (gender IN ('m','f','u')),
    street                  TEXT,
    zip                     TEXT,
    city                    TEXT,
    join_date               DATE,
    dsgvo_verarbeitung      INTEGER  NOT NULL DEFAULT 0,
    dsgvo_verarbeitung_date DATE,
    dsgvo_weitergabe        INTEGER  NOT NULL DEFAULT 0,
    dsgvo_weitergabe_date   DATE,
    sepa_mandat             INTEGER  NOT NULL DEFAULT 0,
    sepa_mandat_date        DATE,
    sepa_mandat_path        TEXT,
    welcome_email_sent_at   TEXT,
    home_club               TEXT,
    phones_visible          INTEGER  NOT NULL DEFAULT 0,
    address_visible         INTEGER  NOT NULL DEFAULT 0,
    email_visible           INTEGER  NOT NULL DEFAULT 0,
    absences_public         INTEGER  NOT NULL DEFAULT 0,
    beitragsfrei            INTEGER  NOT NULL DEFAULT 0,
    zweitspielrecht         INTEGER  NOT NULL DEFAULT 0,
    home_club_id            INTEGER  REFERENCES stammvereine(id),
    cross_team_visible      INTEGER  NOT NULL DEFAULT 0,
    beitragsfrei_grund      TEXT,
    sepa_mandat_dek_enc     TEXT,
    exit_date               DATE,
    foto_veroeffentlichung  INTEGER  NOT NULL DEFAULT 0,
    foto_veroeffentlichung_date DATE
);

INSERT INTO members_new (
    id, first_name, last_name, date_of_birth, pass_number, jersey_number,
    position, status, user_id, created_at, updated_at, member_number, gender,
    street, zip, city, join_date, dsgvo_verarbeitung, dsgvo_verarbeitung_date,
    dsgvo_weitergabe, dsgvo_weitergabe_date, sepa_mandat, sepa_mandat_date,
    sepa_mandat_path, welcome_email_sent_at, home_club, phones_visible,
    address_visible, email_visible, absences_public, beitragsfrei,
    zweitspielrecht, home_club_id, cross_team_visible, beitragsfrei_grund,
    sepa_mandat_dek_enc, exit_date, foto_veroeffentlichung,
    foto_veroeffentlichung_date
)
SELECT
    id, first_name, last_name, date_of_birth, pass_number, jersey_number,
    position, status, user_id, created_at, updated_at, member_number, gender,
    street, zip, city, join_date, dsgvo_verarbeitung, dsgvo_verarbeitung_date,
    dsgvo_weitergabe, dsgvo_weitergabe_date, sepa_mandat, sepa_mandat_date,
    sepa_mandat_path, welcome_email_sent_at, home_club, phones_visible,
    address_visible, email_visible, absences_public, beitragsfrei,
    zweitspielrecht, home_club_id, cross_team_visible, beitragsfrei_grund,
    sepa_mandat_dek_enc, exit_date, foto_veroeffentlichung,
    foto_veroeffentlichung_date
FROM members;

DROP TABLE members;
ALTER TABLE members_new RENAME TO members;
CREATE UNIQUE INDEX idx_members_member_number ON members(member_number) WHERE member_number IS NOT NULL;
