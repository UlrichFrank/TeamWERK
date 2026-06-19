-- Personal iCal feed tokens (one per user).
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
