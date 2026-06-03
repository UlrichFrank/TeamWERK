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
);

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
);

CREATE TABLE training_responses (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    training_id   INTEGER  NOT NULL REFERENCES training_sessions(id) ON DELETE CASCADE,
    member_id     INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    responded_by  INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status        TEXT     NOT NULL CHECK (status IN ('confirmed','declined','maybe')),
    reason        TEXT     NOT NULL DEFAULT '',
    responded_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
