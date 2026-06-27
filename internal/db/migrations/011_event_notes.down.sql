-- Rollback event-notes: pending-Queue weg, CHECKs entfernen, games.note entfernen.

DROP INDEX IF EXISTS idx_pending_event_notes_due;
DROP TABLE IF EXISTS pending_event_notes_push;

-- games: note-Spalte + CHECK wieder entfernen (Rebuild ohne note).
CREATE TABLE games_old (
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
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rsvp_opt_out        INTEGER NOT NULL DEFAULT 0,
    rsvp_require_reason INTEGER NOT NULL DEFAULT 1,
    venue_id    INTEGER  REFERENCES venues(id) ON DELETE SET NULL,
    end_date    DATE
);

INSERT INTO games_old (id, season_id, opponent, date, time, is_home, source,
    event_type, template_id, end_time, created_at, rsvp_opt_out,
    rsvp_require_reason, venue_id, end_date)
SELECT id, season_id, opponent, date, time, is_home, source, event_type,
    template_id, end_time, created_at, rsvp_opt_out, rsvp_require_reason,
    venue_id, end_date
FROM games;

DROP TABLE games;
ALTER TABLE games_old RENAME TO games;

-- training_sessions: CHECK auf note wieder entfernen (Rebuild ohne CHECK).
CREATE TABLE training_sessions_old (
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
    title         TEXT     NOT NULL DEFAULT '',
    rsvp_opt_out        INTEGER NOT NULL DEFAULT 0,
    rsvp_require_reason INTEGER NOT NULL DEFAULT 1,
    venue_id      INTEGER  REFERENCES venues(id) ON DELETE SET NULL
);

INSERT INTO training_sessions_old (id, series_id, team_id, season_id, date,
    start_time, end_time, location, note, status, cancel_reason, created_at,
    title, rsvp_opt_out, rsvp_require_reason, venue_id)
SELECT id, series_id, team_id, season_id, date, start_time, end_time, location,
    note, status, cancel_reason, created_at, title, rsvp_opt_out,
    rsvp_require_reason, venue_id
FROM training_sessions;

DROP TABLE training_sessions;
ALTER TABLE training_sessions_old RENAME TO training_sessions;

CREATE INDEX idx_training_sessions_team_date ON training_sessions(team_id, date);
CREATE INDEX idx_training_sessions_series ON training_sessions(series_id);
