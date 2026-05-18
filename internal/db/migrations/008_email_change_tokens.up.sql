CREATE TABLE email_change_tokens (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT     NOT NULL UNIQUE,
    new_email   TEXT     NOT NULL,
    expires_at  DATETIME NOT NULL,
    used_at     DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
