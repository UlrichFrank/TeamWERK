PRAGMA foreign_keys = OFF;

CREATE TABLE invitation_tokens_tmp (
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
);
INSERT INTO invitation_tokens_tmp
    SELECT id, email, team_id, role, token, expires_at, used_at, created_at, comment
    FROM invitation_tokens;
DROP TABLE invitation_tokens;
ALTER TABLE invitation_tokens_tmp RENAME TO invitation_tokens;

PRAGMA foreign_keys = ON;
