-- 001_initial: konsolidiertes Schema (Stand 2026-06-20, Migrationen 001-049
-- der vorherigen Historie zusammengefasst). Nur Schema, keine Seeds: frische
-- Installationen pflegen Stammvereine und Beitragssätze entweder im Admin-UI
-- oder über separate Seed-Schritte; Tests seeden über internal/testutil/db.go.
--
-- Hinweis zur Reihenfolge: members.home_club_id referenziert stammvereine,
-- aber CREATE TABLE wird hier in der vom sqlite-Dump erzeugten Reihenfolge
-- ausgeführt. Das funktioniert, weil migrate während des Up PRAGMA
-- foreign_keys=OFF setzt (siehe internal/db/db.go).

CREATE TABLE clubs (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL,
    logo_url   TEXT,
    address    TEXT,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, glaeubiger_id TEXT, iban          TEXT, bic           TEXT, kontoinhaber  TEXT);
CREATE TABLE seasons (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL UNIQUE,
    start_date DATE     NOT NULL,
    end_date   DATE     NOT NULL,
    is_active  INTEGER  NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE teams (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL,
    age_class  TEXT     NOT NULL,
    gender     TEXT     NOT NULL CHECK (gender IN ('m','f','mixed')),
    is_active  INTEGER  NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE age_class_game_rules (
    age_class             TEXT    PRIMARY KEY
                          CHECK(age_class IN ('A-Jugend','B-Jugend','C-Jugend','D-Jugend')),
    half_duration_minutes INTEGER NOT NULL CHECK(half_duration_minutes > 0),
    break_minutes         INTEGER NOT NULL CHECK(break_minutes > 0)
);
CREATE TABLE refresh_tokens (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE password_reset_tokens (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE email_change_tokens (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT     NOT NULL UNIQUE,
    new_email  TEXT     NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE user_phones (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label      TEXT    NOT NULL,
    number     TEXT    NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE user_visibility (
    user_id         INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    phones_visible  INTEGER NOT NULL DEFAULT 0,
    address_visible INTEGER NOT NULL DEFAULT 0,
    photo_visible   INTEGER NOT NULL DEFAULT 0
, email_visible INTEGER NOT NULL DEFAULT 0, whatsapp_visible INTEGER NOT NULL DEFAULT 0);
CREATE TABLE push_subscriptions (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT     NOT NULL UNIQUE,
    p256dh     TEXT     NOT NULL,
    auth       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);
CREATE TABLE family_links (
    parent_user_id INTEGER NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    member_id      INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    PRIMARY KEY (parent_user_id, member_id)
);
CREATE TABLE member_change_drafts (
    id                 INTEGER    PRIMARY KEY AUTOINCREMENT,
    member_id          INTEGER    NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    field_name         VARCHAR(50) NOT NULL,
    old_value          TEXT       NOT NULL,
    new_value          TEXT       NOT NULL,
    created_at         TIMESTAMP  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by_user_id INTEGER    REFERENCES users(id),
    UNIQUE(member_id, field_name)
);
CREATE INDEX idx_member_change_drafts_member_id ON member_change_drafts(member_id);
CREATE TABLE membership_requests (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    email      TEXT     NOT NULL,
    team_id    INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    status     TEXT     NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending','approved','rejected')),
    handled_by INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    handled_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment    TEXT,
    first_name TEXT     NOT NULL DEFAULT '',
    last_name  TEXT     NOT NULL DEFAULT ''
);
CREATE TABLE vehicle_info (
    user_id    INTEGER  PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    seats      INTEGER  NOT NULL DEFAULT 0,
    notes      TEXT,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE kader (
    id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id            INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    age_class            TEXT     NOT NULL,
    gender               TEXT     NOT NULL CHECK (gender IN ('m','f','mixed')),
    team_number          INTEGER  NOT NULL DEFAULT 1,
    team_id              INTEGER  REFERENCES teams(id),
    dedicated_birth_year INTEGER,
    games_per_season     INTEGER  NOT NULL DEFAULT 0,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX kader_unique ON kader(season_id, age_class, gender, team_number);
CREATE INDEX idx_kader_season ON kader(season_id);
CREATE TABLE kader_members (
    id        INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id  INTEGER  NOT NULL REFERENCES kader(id)   ON DELETE CASCADE,
    member_id INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kader_id, member_id)
);
CREATE INDEX idx_kader_members_kader ON kader_members(kader_id);
CREATE INDEX idx_kader_members_member ON kader_members(member_id);
CREATE TABLE kader_trainers (
    kader_id  INTEGER NOT NULL REFERENCES kader(id)   ON DELETE CASCADE,
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    PRIMARY KEY (kader_id, member_id)
);
CREATE TABLE duty_assignments (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    duty_slot_id INTEGER  NOT NULL REFERENCES duty_slots(id) ON DELETE CASCADE,
    user_id      INTEGER  NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
    status       TEXT     NOT NULL DEFAULT 'assigned'
                 CHECK (status IN ('assigned','fulfilled','cash_substitute')),
    cash_amount  REAL,
    fulfilled_at DATETIME,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (duty_slot_id, user_id)
);
CREATE TABLE duty_accounts (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id   INTEGER NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    season_id INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    soll      REAL    NOT NULL DEFAULT 0,
    ist       REAL    NOT NULL DEFAULT 0,
    UNIQUE (user_id, season_id)
);
CREATE TABLE duty_season_targets (
    season_id    INTEGER NOT NULL REFERENCES seasons(id)    ON DELETE CASCADE,
    duty_type_id INTEGER NOT NULL REFERENCES duty_types(id) ON DELETE CASCADE,
    target_hours REAL    NOT NULL DEFAULT 0,
    PRIMARY KEY (season_id, duty_type_id)
);
CREATE TABLE game_templates (
    id               INTEGER  PRIMARY KEY AUTOINCREMENT,
    name             TEXT     NOT NULL DEFAULT 'Heimspiel Standard',
    duration_minutes INTEGER  NOT NULL DEFAULT 90,
    is_active        INTEGER  NOT NULL DEFAULT 0,
    template_type    TEXT     NOT NULL DEFAULT 'generisch'
                     CHECK(template_type IN ('heim','auswärts','generisch')),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE game_template_items (
    id             INTEGER  PRIMARY KEY AUTOINCREMENT,
    template_id    INTEGER  NOT NULL REFERENCES game_templates(id) ON DELETE CASCADE,
    duty_type_id   INTEGER  NOT NULL REFERENCES duty_types(id)     ON DELETE RESTRICT,
    anchor         TEXT     NOT NULL DEFAULT 'start' CHECK (anchor IN ('start','end')),
    offset_minutes INTEGER  NOT NULL DEFAULT 0,
    slots_count    INTEGER  NOT NULL DEFAULT 1,
    sort_order     INTEGER  NOT NULL DEFAULT 0,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, audiences TEXT);
CREATE TABLE games (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id   INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    opponent    TEXT     NOT NULL,
    date        DATE     NOT NULL,
    time        TEXT     NOT NULL DEFAULT '00:00',
    is_home     INTEGER  NOT NULL DEFAULT 1,
    source      TEXT     NOT NULL DEFAULT 'manual',
    event_type  TEXT     NOT NULL DEFAULT 'heim'
                CHECK (event_type IN ('heim','auswärts','generisch')),
    template_id INTEGER  REFERENCES game_templates(id),
    end_time    TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, rsvp_opt_out INTEGER NOT NULL DEFAULT 0, rsvp_require_reason INTEGER NOT NULL DEFAULT 1, venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL, end_date DATE);
CREATE TABLE game_teams (
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
    PRIMARY KEY (game_id, team_id)
);
CREATE TABLE mitfahrgelegenheiten (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    game_id    INTEGER  NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    typ        TEXT     NOT NULL CHECK(typ IN ('biete','suche')),
    plaetze    INTEGER,
    treffpunkt TEXT,
    notiz      TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_mitfahr_biete_unique
    ON mitfahrgelegenheiten(game_id, user_id) WHERE typ = 'biete';
CREATE TABLE mitfahrt_paarungen (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    biete_id      INTEGER  NOT NULL REFERENCES mitfahrgelegenheiten(id) ON DELETE CASCADE,
    suche_id      INTEGER  NOT NULL REFERENCES mitfahrgelegenheiten(id) ON DELETE CASCADE,
    initiiert_von TEXT     NOT NULL CHECK(initiiert_von IN ('biete','suche')),
    status        TEXT     NOT NULL DEFAULT 'pending'
                  CHECK(status IN ('pending','confirmed','rejected')),
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(biete_id, suche_id)
);
CREATE TABLE carpooling_events (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    game_id    INTEGER  NOT NULL REFERENCES games(id)  ON DELETE CASCADE,
    type       TEXT     NOT NULL CHECK(type IN (
                   'biete_created','suche_created',
                   'pairing_requested','pairing_confirmed','pairing_rejected','pairing_cancelled',
                   'biete_deleted','suche_deleted'
               )),
    actor_name TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE VIEW team_memberships AS
SELECT km.id, km.member_id, k.team_id, k.season_id, 0 AS is_primary
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL
UNION
SELECT kt.kader_id * 100000 + kt.member_id, kt.member_id, k.team_id, k.season_id, 0
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
WHERE k.team_id IS NOT NULL
/* team_memberships(id,member_id,team_id,season_id,is_primary) */;
CREATE TABLE IF NOT EXISTS "invitation_tokens" (
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
, first_name TEXT NOT NULL DEFAULT '', last_name  TEXT NOT NULL DEFAULT '', member_id INTEGER REFERENCES members(id) ON DELETE SET NULL);
CREATE TABLE duty_reminder_log (
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_date DATE     NOT NULL,
    sent_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, event_date)
);
CREATE TABLE file_folders (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    parent_id  INTEGER REFERENCES file_folders(id) ON DELETE CASCADE,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE TABLE folder_permissions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id      INTEGER NOT NULL REFERENCES file_folders(id) ON DELETE CASCADE,
    principal_type TEXT    NOT NULL CHECK (principal_type IN ('everyone','role','club_function','user')),
    principal_ref  TEXT,
    can_read       INTEGER NOT NULL DEFAULT 0,
    can_write      INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE files (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id     INTEGER NOT NULL REFERENCES file_folders(id) ON DELETE CASCADE,
    original_name TEXT    NOT NULL,
    disk_name     TEXT    NOT NULL UNIQUE,
    size          INTEGER NOT NULL,
    mime_type     TEXT    NOT NULL DEFAULT '',
    uploaded_by   INTEGER NOT NULL REFERENCES users(id),
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE TABLE training_series (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    team_id      INTEGER  NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    season_id    INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    name         TEXT     NOT NULL,
    location     TEXT     NOT NULL DEFAULT '',
    day_of_week  INTEGER  NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time   TEXT     NOT NULL,
    end_time     TEXT     NOT NULL,
    valid_from   DATE     NOT NULL,
    valid_until  DATE     NOT NULL,
    note         TEXT     NOT NULL DEFAULT '',
    created_by   INTEGER  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, rsvp_opt_out INTEGER NOT NULL DEFAULT 0, rsvp_require_reason INTEGER NOT NULL DEFAULT 1, venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL);
CREATE TABLE training_sessions (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    series_id     INTEGER  REFERENCES training_series(id) ON DELETE SET NULL,
    team_id       INTEGER  NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    season_id     INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    date          DATE     NOT NULL,
    start_time    TEXT     NOT NULL,
    end_time      TEXT     NOT NULL,
    location      TEXT     NOT NULL DEFAULT '',
    note          TEXT     NOT NULL DEFAULT '',
    status        TEXT     NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','cancelled')),
    cancel_reason TEXT     NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, title TEXT NOT NULL DEFAULT '', rsvp_opt_out INTEGER NOT NULL DEFAULT 0, rsvp_require_reason INTEGER NOT NULL DEFAULT 1, venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL);
CREATE TABLE training_responses (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    training_id   INTEGER  NOT NULL REFERENCES training_sessions(id) ON DELETE CASCADE,
    member_id     INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    responded_by  INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status        TEXT     NOT NULL CHECK (status IN ('confirmed','declined','maybe')),
    reason        TEXT     NOT NULL DEFAULT '',
    responded_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, absence_id INTEGER REFERENCES member_absences(id) ON DELETE CASCADE,
    UNIQUE (training_id, member_id)
);
CREATE TABLE training_attendances (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    training_id INTEGER  NOT NULL REFERENCES training_sessions(id) ON DELETE CASCADE,
    member_id   INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    present     INTEGER  NOT NULL CHECK (present IN (0, 1)),
    noted_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (training_id, member_id)
);
CREATE INDEX idx_training_sessions_team_date ON training_sessions(team_id, date);
CREATE INDEX idx_training_sessions_series ON training_sessions(series_id);
CREATE INDEX idx_training_responses_training ON training_responses(training_id);
CREATE INDEX idx_training_responses_member ON training_responses(member_id);
CREATE INDEX idx_training_attendances_training ON training_attendances(training_id);
CREATE INDEX idx_training_series_team ON training_series(team_id);
CREATE INDEX idx_training_responses_responded_by ON training_responses(responded_by);
CREATE TABLE game_responses (
    game_id      INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    member_id    INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    responded_by INTEGER NOT NULL REFERENCES users(id),
    status       TEXT    NOT NULL CHECK(status IN ('confirmed','declined','maybe')),
    reason       TEXT    NOT NULL DEFAULT '',
    responded_at TEXT    NOT NULL DEFAULT (datetime('now')), absence_id INTEGER REFERENCES member_absences(id) ON DELETE CASCADE,
    PRIMARY KEY (game_id, member_id)
);
CREATE TABLE IF NOT EXISTS "member_club_functions" (
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    function  TEXT    NOT NULL CHECK(function IN ('spieler','trainer','vorstand','vorstand_beisitzer','kassierer','sportliche_leitung')),
    PRIMARY KEY (member_id, function)
);
CREATE TABLE kader_extended_members (
    kader_id  INTEGER NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (kader_id, member_id)
);
CREATE TABLE game_lineup (
    game_id   INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_by  INTEGER REFERENCES users(id),
    added_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (game_id, member_id)
);
CREATE TABLE member_phones (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  label TEXT NOT NULL DEFAULT '',
  number TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE venues (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    name          TEXT     NOT NULL,
    street        TEXT     NOT NULL,
    city          TEXT     NOT NULL,
    postal_code   TEXT     NOT NULL,
    country       TEXT     NOT NULL DEFAULT 'DE',
    note          TEXT     NOT NULL DEFAULT '',
    is_home_venue INTEGER  NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_venues_is_home ON venues(is_home_venue);
CREATE TABLE conversations (
    id         INTEGER PRIMARY KEY,
    type       TEXT    NOT NULL CHECK(type IN ('direct', 'group')),
    name       TEXT,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE conversation_members (
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    joined_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    left_at         DATETIME,
    PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conv_members_user ON conversation_members(user_id);
CREATE TABLE messages (
    id              INTEGER PRIMARY KEY,
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       INTEGER NOT NULL REFERENCES users(id),
    body            TEXT    NOT NULL CHECK(length(body) > 0 AND length(body) <= 2000),
    sent_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, reply_to_id INTEGER REFERENCES messages(id), edited_at DATETIME, deleted_at DATETIME, is_system BOOLEAN NOT NULL DEFAULT 0);
CREATE INDEX idx_messages_conv ON messages(conversation_id, sent_at DESC);
CREATE TABLE message_reads (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    read_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, user_id)
);
CREATE INDEX idx_message_reads_user ON message_reads(user_id);
CREATE TABLE broadcasts (
    id          INTEGER PRIMARY KEY,
    sender_id   INTEGER NOT NULL REFERENCES users(id),
    target_type TEXT    NOT NULL CHECK(target_type IN ('all', 'team', 'role')),
    target_id   INTEGER,
    target_role TEXT,
    body        TEXT    NOT NULL CHECK(length(body) > 0 AND length(body) <= 2000),
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, edited_at DATETIME);
CREATE TABLE broadcast_reads (
    broadcast_id INTEGER NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    user_id      INTEGER NOT NULL REFERENCES users(id),
    read_at      DATETIME, hidden_at DATETIME,
    PRIMARY KEY (broadcast_id, user_id)
);
CREATE INDEX idx_broadcast_reads_user ON broadcast_reads(user_id);
CREATE TABLE IF NOT EXISTS "duty_slots" (
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
    game_id      INTEGER  REFERENCES games(id)   ON DELETE CASCADE,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    audiences    TEXT
, is_custom INTEGER NOT NULL DEFAULT 0);
CREATE TABLE notification_preferences (
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category      TEXT    NOT NULL,
    push_enabled  INTEGER NOT NULL DEFAULT 1,
    email_enabled INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, category),
    CHECK (category IN ('games','trainings','duties','duty_reminders','carpooling','membership'))
);
CREATE TABLE notification_log (
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ref_type TEXT    NOT NULL,
    ref_id   INTEGER NOT NULL,
    sent_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, ref_type, ref_id)
);
CREATE TABLE member_absences (
    id         INTEGER PRIMARY KEY,
    member_id  INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    type       TEXT    NOT NULL CHECK(type IN ('vacation','injury')),
    start_date DATE    NOT NULL,
    end_date   DATE    NOT NULL,
    note       TEXT    NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_member_absences_member ON member_absences(member_id);
CREATE INDEX idx_member_absences_dates  ON member_absences(start_date, end_date);
CREATE TABLE users (
    id                 INTEGER  PRIMARY KEY AUTOINCREMENT,
    email              TEXT,
    password           TEXT     NOT NULL DEFAULT '',
    role               TEXT     NOT NULL DEFAULT 'standard'
                       CHECK (role IN ('admin','standard')),
    team_id            INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    first_name         TEXT     NOT NULL DEFAULT '',
    last_name          TEXT     NOT NULL DEFAULT '',
    street             TEXT,
    zip                TEXT,
    city               TEXT,
    photo_path         TEXT,
    duty_reminder_days INT      NULL DEFAULT NULL,
    last_login_at      DATETIME,
    can_login          INTEGER  NOT NULL DEFAULT 1
, maps_provider TEXT NOT NULL DEFAULT 'auto' CHECK(maps_provider IN ('auto','google','apple')));
CREATE UNIQUE INDEX users_email_login_unique ON users(email)
WHERE can_login = 1 AND email IS NOT NULL;
CREATE TABLE message_reactions (
  message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  user_id    INTEGER NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
  emoji      TEXT    NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (message_id, user_id, emoji)
);
CREATE INDEX idx_message_reactions_msg ON message_reactions(message_id);
CREATE TABLE IF NOT EXISTS "members" (
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
, home_club_id INTEGER REFERENCES stammvereine(id));
CREATE UNIQUE INDEX idx_members_member_number ON members(member_number) WHERE member_number IS NOT NULL;
CREATE VIEW player_memberships AS
SELECT km.id, km.member_id, k.team_id, k.season_id
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL
/* player_memberships(id,member_id,team_id,season_id) */;
CREATE VIEW trainer_memberships AS
SELECT kt.kader_id * 100000 + kt.member_id AS id, kt.member_id, k.team_id, k.season_id
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
WHERE k.team_id IS NOT NULL
/* trainer_memberships(id,member_id,team_id,season_id) */;
CREATE UNIQUE INDEX idx_mitfahr_suche_unique
    ON mitfahrgelegenheiten(game_id, user_id) WHERE typ = 'suche';
CREATE TABLE IF NOT EXISTS "duty_types" (
    id                      INTEGER  PRIMARY KEY AUTOINCREMENT,
    name                    TEXT     NOT NULL,
    hours_value             REAL     NOT NULL DEFAULT 1.0,
    cash_substitute         REAL,
    default_anchor          TEXT     NOT NULL DEFAULT 'start',
    default_offset_minutes  INTEGER  NOT NULL DEFAULT 0,
    target_role             TEXT     NOT NULL DEFAULT 'elternteil'
                            CHECK(target_role IN ('spieler','elternteil','trainer','vorstand','sportliche_leitung','vorstand_beisitzer','kassierer')),
    consecutive_behavior    TEXT     NOT NULL DEFAULT 'normal'
                            CHECK(consecutive_behavior IN ('normal','skip','reduced')),
    consecutive_variant_id  INTEGER  REFERENCES duty_types(id),
    same_day_behavior       TEXT     NOT NULL DEFAULT 'normal'
                            CHECK(same_day_behavior IN ('normal','skip','reduced')),
    same_day_variant_id     INTEGER  REFERENCES duty_types(id),
    adjacent_day_behavior   TEXT     NOT NULL DEFAULT 'normal'
                            CHECK(adjacent_day_behavior IN ('normal','skip','reduced')),
    adjacent_day_variant_id INTEGER  REFERENCES duty_types(id),
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    audiences               TEXT
);
CREATE TABLE beitrags_saetze (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    kategorie   TEXT NOT NULL CHECK (kategorie IN (
        'aktiv_ohne',
        'aktiv_mit',
        'passiv'
    )),
    betrag_eur  INTEGER NOT NULL,
    valid_from  DATE NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_beitrags_saetze_kat_valid ON beitrags_saetze(kategorie, valid_from);
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
WHERE k.team_id IS NOT NULL
UNION ALL
SELECT fl.parent_user_id AS user_id, k.team_id, k.season_id
FROM family_links fl
JOIN kader_extended_members kem ON kem.member_id = fl.member_id
JOIN kader k ON k.id = kem.kader_id
WHERE k.team_id IS NOT NULL
/* user_accessible_teams(user_id,team_id,season_id) */;
CREATE TABLE calendar_tokens (
    id                INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER  NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    token             TEXT     NOT NULL UNIQUE,
    include_heim      INTEGER  NOT NULL DEFAULT 1,
    include_auswaerts INTEGER  NOT NULL DEFAULT 1,
    include_training  INTEGER  NOT NULL DEFAULT 1,
    include_generisch INTEGER  NOT NULL DEFAULT 1,
    include_duty      INTEGER  NOT NULL DEFAULT 1,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE stammvereine (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    aktiv       INTEGER NOT NULL DEFAULT 1,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
