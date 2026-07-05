-- Spielbericht-Publisher: users.role um 'presseteam' erweitern (hierarchisch:
-- admin ⊇ presseteam ⊇ standard), invitation_tokens.role synchron dazu,
-- und die neuen Tabellen match_reports + match_report_images anlegen.
--
-- SQLite kennt kein ALTER CHECK → Tabellen-Rebuild (12-Schritt-Recipe, wie in
-- 011_event_notes). FK-Enforcement ist während der Migration auf
-- Connection-Ebene deaktiviert (siehe internal/db/db.go Migrate()), daher
-- kein PRAGMA nötig.

-- users: role-CHECK erweitern via Rebuild.
CREATE TABLE users_new (
    id                 INTEGER  PRIMARY KEY AUTOINCREMENT,
    email              TEXT,
    password           TEXT     NOT NULL DEFAULT '',
    role               TEXT     NOT NULL DEFAULT 'standard'
                       CHECK (role IN ('admin','standard','presseteam')),
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
    can_login          INTEGER  NOT NULL DEFAULT 1,
    maps_provider      TEXT     NOT NULL DEFAULT 'auto'
                       CHECK (maps_provider IN ('auto','google','apple')),
    login_name         TEXT,
    recovery_email     TEXT,
    failed_login_count INTEGER  NOT NULL DEFAULT 0,
    locked_until       TEXT
);

INSERT INTO users_new (id, email, password, role, team_id, created_at,
    updated_at, first_name, last_name, street, zip, city, photo_path,
    duty_reminder_days, last_login_at, can_login, maps_provider, login_name,
    recovery_email, failed_login_count, locked_until)
SELECT id, email, password, role, team_id, created_at, updated_at,
    first_name, last_name, street, zip, city, photo_path, duty_reminder_days,
    last_login_at, can_login, maps_provider, login_name, recovery_email,
    failed_login_count, locked_until
FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

CREATE UNIQUE INDEX users_email_login_unique ON users(email)
WHERE can_login = 1 AND email IS NOT NULL;
CREATE UNIQUE INDEX users_login_name_unique ON users(LOWER(login_name))
WHERE login_name IS NOT NULL;

-- invitation_tokens: role-CHECK erweitern via Rebuild.
CREATE TABLE invitation_tokens_new (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    email      TEXT     NOT NULL,
    team_id    INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    role       TEXT     NOT NULL DEFAULT 'standard'
               CHECK (role IN ('admin','standard','presseteam')),
    token      TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment    TEXT,
    first_name TEXT     NOT NULL DEFAULT '',
    last_name  TEXT     NOT NULL DEFAULT '',
    member_id  INTEGER  REFERENCES members(id) ON DELETE SET NULL
);

INSERT INTO invitation_tokens_new (id, email, team_id, role, token, expires_at,
    used_at, created_at, comment, first_name, last_name, member_id)
SELECT id, email, team_id, role, token, expires_at, used_at, created_at,
    comment, first_name, last_name, member_id
FROM invitation_tokens;

DROP TABLE invitation_tokens;
ALTER TABLE invitation_tokens_new RENAME TO invitation_tokens;

-- match_reports: State-Machine + Metadaten. Ein Draft pro Spiel (UNIQUE game_id).
-- duty_slot_id nullable (falls Slot gelöscht wurde, Bericht bleibt).
-- typo3_page_uid + published_url werden nach 2xx vom Publisher gefüllt.
CREATE TABLE match_reports (
    id                INTEGER  PRIMARY KEY AUTOINCREMENT,
    game_id           INTEGER  NOT NULL UNIQUE REFERENCES games(id) ON DELETE CASCADE,
    duty_slot_id      INTEGER  REFERENCES duty_slots(id) ON DELETE SET NULL,
    author_user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    state             TEXT     NOT NULL DEFAULT 'draft'
                      CHECK (state IN ('draft','publishing','published','publish_failed')),
    home_goals        INTEGER,
    away_goals        INTEGER,
    home_goals_ht     INTEGER,
    away_goals_ht     INTEGER,
    tournament        INTEGER  NOT NULL DEFAULT 0,
    abstract          TEXT     NOT NULL DEFAULT '' CHECK (length(abstract) <= 500),
    body_md           TEXT     NOT NULL DEFAULT '',
    published_url     TEXT,
    typo3_page_uid    INTEGER,
    error_message     TEXT,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at      DATETIME
);

CREATE INDEX idx_match_reports_author ON match_reports(author_user_id);
CREATE INDEX idx_match_reports_state  ON match_reports(state);
CREATE INDEX idx_match_reports_slot   ON match_reports(duty_slot_id);

-- match_report_images: Reihenfolge über position, Cleanup nach state='published'.
-- storage_path relativ zu ./storage/match-report-images/{report_id}/…
CREATE TABLE match_report_images (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    report_id     INTEGER  NOT NULL REFERENCES match_reports(id) ON DELETE CASCADE,
    position      INTEGER  NOT NULL,
    caption       TEXT     NOT NULL DEFAULT '',
    storage_path  TEXT     NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (report_id, position)
);

CREATE INDEX idx_match_report_images_report ON match_report_images(report_id);
