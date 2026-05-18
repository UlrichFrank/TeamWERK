CREATE TABLE users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    password    TEXT    NOT NULL,
    role        TEXT    NOT NULL DEFAULT 'elternteil' CHECK (role IN ('admin','trainer','elternteil','spieler')),
    team_id     INTEGER REFERENCES teams(id) ON DELETE SET NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE password_reset_tokens (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT     NOT NULL UNIQUE,
    expires_at  DATETIME NOT NULL,
    used_at     DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE membership_requests (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    email       TEXT     NOT NULL,
    team_id     INTEGER  NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    status      TEXT     NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    handled_by  INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    handled_at  DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
