-- Rückbau: match_report-Tabellen droppen, users/invitation_tokens auf altes
-- role-Enum zurückrollen. Zeilen mit role='presseteam' müssen vorher manuell
-- migriert werden (auf 'standard' oder 'admin' setzen), sonst wirft der
-- INSERT SELECT unten am CHECK-Constraint einen Fehler — bewusst, damit
-- der Rückbau nicht stillschweigend Daten verwirft.

DROP INDEX IF EXISTS idx_match_report_images_report;
DROP TABLE IF EXISTS match_report_images;

DROP INDEX IF EXISTS idx_match_reports_slot;
DROP INDEX IF EXISTS idx_match_reports_state;
DROP INDEX IF EXISTS idx_match_reports_author;
DROP TABLE IF EXISTS match_reports;

-- invitation_tokens: role-CHECK zurück auf ('admin','standard')
CREATE TABLE invitation_tokens_old (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    email      TEXT     NOT NULL,
    team_id    INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    role       TEXT     NOT NULL DEFAULT 'standard'
               CHECK (role IN ('admin','standard')),
    token      TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment    TEXT,
    first_name TEXT     NOT NULL DEFAULT '',
    last_name  TEXT     NOT NULL DEFAULT '',
    member_id  INTEGER  REFERENCES members(id) ON DELETE SET NULL
);

INSERT INTO invitation_tokens_old (id, email, team_id, role, token, expires_at,
    used_at, created_at, comment, first_name, last_name, member_id)
SELECT id, email, team_id, role, token, expires_at, used_at, created_at,
    comment, first_name, last_name, member_id
FROM invitation_tokens;

DROP TABLE invitation_tokens;
ALTER TABLE invitation_tokens_old RENAME TO invitation_tokens;

-- users: role-CHECK zurück auf ('admin','standard')
CREATE TABLE users_old (
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
    can_login          INTEGER  NOT NULL DEFAULT 1,
    maps_provider      TEXT     NOT NULL DEFAULT 'auto'
                       CHECK (maps_provider IN ('auto','google','apple')),
    login_name         TEXT,
    recovery_email     TEXT,
    failed_login_count INTEGER  NOT NULL DEFAULT 0,
    locked_until       TEXT
);

INSERT INTO users_old (id, email, password, role, team_id, created_at,
    updated_at, first_name, last_name, street, zip, city, photo_path,
    duty_reminder_days, last_login_at, can_login, maps_provider, login_name,
    recovery_email, failed_login_count, locked_until)
SELECT id, email, password, role, team_id, created_at, updated_at,
    first_name, last_name, street, zip, city, photo_path, duty_reminder_days,
    last_login_at, can_login, maps_provider, login_name, recovery_email,
    failed_login_count, locked_until
FROM users;

DROP INDEX IF EXISTS users_login_name_unique;
DROP INDEX IF EXISTS users_email_login_unique;
DROP TABLE users;
ALTER TABLE users_old RENAME TO users;

CREATE UNIQUE INDEX users_email_login_unique ON users(email)
WHERE can_login = 1 AND email IS NOT NULL;
CREATE UNIQUE INDEX users_login_name_unique ON users(LOWER(login_name))
WHERE login_name IS NOT NULL;
