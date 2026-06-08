PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;

-- Remove proxy accounts (can_login = 0) before restoring NOT NULL constraint on email
DELETE FROM users WHERE can_login = 0;

DROP INDEX IF EXISTS users_email_login_unique;

ALTER TABLE users RENAME TO users_old;

CREATE TABLE users (
    id                 INTEGER  PRIMARY KEY AUTOINCREMENT,
    email              TEXT     NOT NULL UNIQUE,
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
    last_login_at      DATETIME
);

INSERT INTO users (id, email, password, role, team_id, created_at, updated_at,
                   first_name, last_name, street, zip, city, photo_path,
                   duty_reminder_days, last_login_at)
SELECT id, email, password, role, team_id, created_at, updated_at,
       first_name, last_name, street, zip, city, photo_path,
       duty_reminder_days, last_login_at
FROM users_old;

DROP TABLE users_old;

PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;
