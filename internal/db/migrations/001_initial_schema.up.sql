-- Complete database schema
PRAGMA foreign_keys = ON;

CREATE TABLE refresh_tokens (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT     NOT NULL UNIQUE,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE invitation_tokens (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    email       TEXT     NOT NULL,
    team_id     INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    role        TEXT     NOT NULL DEFAULT 'elternteil',
    token       TEXT     NOT NULL UNIQUE,
    expires_at  DATETIME NOT NULL,
    used_at     DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment     TEXT
);

CREATE TABLE password_reset_tokens (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT     NOT NULL UNIQUE,
    expires_at  DATETIME NOT NULL,
    used_at     DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE clubs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    logo_url    TEXT,
    address     TEXT,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE seasons (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL UNIQUE,
    start_date  DATE     NOT NULL,
    end_date    DATE     NOT NULL,
    is_active   INTEGER  NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE teams (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    age_class   TEXT    NOT NULL,
    gender      TEXT    NOT NULL CHECK (gender IN ('m','f','mixed')),
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE team_trainers (
    team_id     INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    password    TEXT    NOT NULL,
    role        TEXT    NOT NULL DEFAULT 'elternteil' CHECK (role IN ('admin','vorstand','trainer','elternteil','spieler')),
    team_id     INTEGER REFERENCES teams(id) ON DELETE SET NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    street      TEXT,
    zip         TEXT,
    city        TEXT,
    photo_path  TEXT
);

CREATE TABLE members (
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
    account_holder          TEXT
);

CREATE UNIQUE INDEX idx_members_member_number ON members(member_number) WHERE member_number IS NOT NULL;

CREATE TABLE team_memberships (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    member_id   INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    team_id     INTEGER NOT NULL REFERENCES teams(id)   ON DELETE CASCADE,
    season_id   INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    is_primary  INTEGER NOT NULL DEFAULT 0,
    UNIQUE (member_id, team_id, season_id)
);

CREATE TABLE family_links (
    parent_user_id  INTEGER NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    member_id       INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    PRIMARY KEY (parent_user_id, member_id)
);

CREATE TABLE vehicle_info (
    user_id     INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    seats       INTEGER NOT NULL DEFAULT 0,
    notes       TEXT,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE duty_types (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    name                     TEXT    NOT NULL,
    hours_value              REAL    NOT NULL DEFAULT 1.0,
    cash_substitute          REAL,
    created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    default_anchor           TEXT    NOT NULL DEFAULT 'start',
    default_offset_minutes   INTEGER NOT NULL DEFAULT 0,
    same_day_behavior        TEXT    NOT NULL DEFAULT 'normal' CHECK (same_day_behavior IN ('normal', 'skip', 'reduced')),
    same_day_variant_id      INTEGER REFERENCES duty_types(id),
    adjacent_day_behavior    TEXT    NOT NULL DEFAULT 'normal' CHECK (adjacent_day_behavior IN ('normal', 'skip', 'reduced')),
    adjacent_day_variant_id  INTEGER REFERENCES duty_types(id)
);

CREATE TABLE duty_slots (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    event_name   TEXT     NOT NULL,
    event_date   DATE     NOT NULL,
    event_time   TEXT,
    duty_type_id INTEGER  NOT NULL REFERENCES duty_types(id) ON DELETE RESTRICT,
    role_desc    TEXT,
    slots_total  INTEGER  NOT NULL DEFAULT 1,
    slots_filled INTEGER  NOT NULL DEFAULT 0,
    team_id      INTEGER  REFERENCES teams(id)   ON DELETE SET NULL,
    season_id    INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    game_id      INTEGER  REFERENCES games(id)   ON DELETE SET NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE duty_assignments (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    duty_slot_id    INTEGER  NOT NULL REFERENCES duty_slots(id)  ON DELETE CASCADE,
    user_id         INTEGER  NOT NULL REFERENCES users(id)        ON DELETE CASCADE,
    status          TEXT     NOT NULL DEFAULT 'assigned' CHECK (status IN ('assigned','fulfilled','cash_substitute')),
    cash_amount     REAL,
    fulfilled_at    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (duty_slot_id, user_id)
);

CREATE TABLE duty_accounts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    season_id   INTEGER NOT NULL REFERENCES seasons(id)  ON DELETE CASCADE,
    soll        REAL    NOT NULL DEFAULT 0,
    ist         REAL    NOT NULL DEFAULT 0,
    UNIQUE (user_id, season_id)
);

CREATE TABLE duty_season_targets (
    season_id       INTEGER NOT NULL REFERENCES seasons(id)    ON DELETE CASCADE,
    duty_type_id    INTEGER NOT NULL REFERENCES duty_types(id) ON DELETE CASCADE,
    target_hours    REAL    NOT NULL DEFAULT 0,
    PRIMARY KEY (season_id, duty_type_id)
);

CREATE TABLE membership_requests (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    email       TEXT     NOT NULL,
    team_id     INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    status      TEXT     NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    handled_by  INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    handled_at  DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment     TEXT
);

CREATE TABLE games (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    team_id     INTEGER  NOT NULL REFERENCES teams(id)   ON DELETE RESTRICT,
    season_id   INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    opponent    TEXT     NOT NULL,
    date        DATE     NOT NULL,
    time        TEXT     NOT NULL DEFAULT '00:00',
    is_home     INTEGER  NOT NULL DEFAULT 1,
    source      TEXT     NOT NULL DEFAULT 'manual',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE email_change_tokens (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT     NOT NULL UNIQUE,
    new_email   TEXT     NOT NULL,
    expires_at  DATETIME NOT NULL,
    used_at     DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE game_templates (
    id                    INTEGER  PRIMARY KEY AUTOINCREMENT,
    name                  TEXT     NOT NULL DEFAULT 'Heimspiel Standard',
    game_duration_minutes INTEGER  NOT NULL DEFAULT 90,
    is_active             INTEGER  NOT NULL DEFAULT 0,
    template_type         TEXT     NOT NULL DEFAULT 'generisch' CHECK(template_type IN ('heim','auswärts','generisch')),
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE game_template_items (
    id             INTEGER  PRIMARY KEY AUTOINCREMENT,
    template_id    INTEGER  NOT NULL REFERENCES game_templates(id) ON DELETE CASCADE,
    duty_type_id   INTEGER  NOT NULL REFERENCES duty_types(id)     ON DELETE RESTRICT,
    anchor         TEXT     NOT NULL DEFAULT 'start' CHECK (anchor IN ('start','end')),
    offset_minutes INTEGER  NOT NULL DEFAULT 0,
    slots_count    INTEGER  NOT NULL DEFAULT 1,
    role_desc      TEXT,
    sort_order     INTEGER  NOT NULL DEFAULT 0,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE member_change_drafts (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    member_id          INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    field_name         VARCHAR(50) NOT NULL,
    old_value          TEXT NOT NULL,
    new_value          TEXT NOT NULL,
    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by_user_id INTEGER REFERENCES users(id),
    UNIQUE(member_id, field_name)
);

CREATE INDEX idx_member_change_drafts_member_id ON member_change_drafts(member_id);

CREATE TABLE kader (
    id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id            INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    age_class            TEXT     NOT NULL,
    gender               TEXT     NOT NULL CHECK (gender IN ('m', 'f', 'mixed')),
    team_number          INTEGER  NOT NULL DEFAULT 1,
    dedicated_birth_year INTEGER,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX kader_unique ON kader(season_id, age_class, gender, team_number);
CREATE INDEX idx_kader_season ON kader(season_id);

CREATE TABLE kader_members (
    id        INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id  INTEGER  NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    member_id INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kader_id, member_id)
);

CREATE INDEX idx_kader_members_kader ON kader_members(kader_id);
CREATE INDEX idx_kader_members_member ON kader_members(member_id);

CREATE TABLE kader_trainers (
    kader_id  INTEGER NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    PRIMARY KEY (kader_id, member_id)
);

CREATE TABLE user_phones (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label       TEXT    NOT NULL,
    number      TEXT    NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_visibility (
    user_id         INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    phones_visible  INTEGER NOT NULL DEFAULT 0,
    address_visible INTEGER NOT NULL DEFAULT 0,
    photo_visible   INTEGER NOT NULL DEFAULT 0
);
