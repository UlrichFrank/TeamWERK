-- 015 down: Remove rsvp_opt_out and rsvp_require_reason columns
-- SQLite does not support DROP COLUMN before 3.35, so we recreate each table.

PRAGMA foreign_keys = OFF;

-- training_series
CREATE TABLE training_series_new (
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
INSERT INTO training_series_new SELECT id, team_id, season_id, name, location, day_of_week, start_time, end_time, valid_from, valid_until, note, created_by, created_at FROM training_series;
DROP TABLE training_series;
ALTER TABLE training_series_new RENAME TO training_series;

-- training_sessions
CREATE TABLE training_sessions_new (
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
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    title         TEXT     NOT NULL DEFAULT ''
);
INSERT INTO training_sessions_new SELECT id, series_id, team_id, season_id, date, start_time, end_time, location, note, status, cancel_reason, created_at, title FROM training_sessions;
DROP TABLE training_sessions;
ALTER TABLE training_sessions_new RENAME TO training_sessions;

-- games
CREATE TABLE games_new (
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
);
INSERT INTO games_new SELECT id, season_id, opponent, date, time, is_home, source, event_type, template_id, end_time, created_at FROM games;
DROP TABLE games;
ALTER TABLE games_new RENAME TO games;

PRAGMA foreign_keys = ON;
