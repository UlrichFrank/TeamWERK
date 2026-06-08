PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;

ALTER TABLE users RENAME TO users_old;

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
);

INSERT INTO users (id, email, password, role, team_id, created_at, updated_at,
                   first_name, last_name, street, zip, city, photo_path,
                   duty_reminder_days, last_login_at, can_login)
SELECT id, email, password, role, team_id, created_at, updated_at,
       first_name, last_name, street, zip, city, photo_path,
       duty_reminder_days, last_login_at, 1
FROM users_old;

DROP TABLE users_old;

CREATE UNIQUE INDEX users_email_login_unique ON users(email)
WHERE can_login = 1 AND email IS NOT NULL;

PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;
