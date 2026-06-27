-- event-notes: games.note (neu) + CHECK(length(note)<=200) auf games.note und
-- training_sessions.note + Debounce-Queue pending_event_notes_push.
--
-- SQLite kennt kein ADD CONSTRAINT → Tabellen-Rebuild (12-Schritt-Recipe). Die
-- FK-Enforcement ist während der Migration auf Connection-Ebene deaktiviert
-- (siehe internal/db/db.go Migrate()), daher hier kein PRAGMA nötig.

-- games: Spalte note hinzufügen + CHECK ≤ 200 per Rebuild.
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
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rsvp_opt_out        INTEGER NOT NULL DEFAULT 0,
    rsvp_require_reason INTEGER NOT NULL DEFAULT 1,
    venue_id    INTEGER  REFERENCES venues(id) ON DELETE SET NULL,
    end_date    DATE,
    note        TEXT     NOT NULL DEFAULT '' CHECK (length(note) <= 200)
);

INSERT INTO games_new (id, season_id, opponent, date, time, is_home, source,
    event_type, template_id, end_time, created_at, rsvp_opt_out,
    rsvp_require_reason, venue_id, end_date)
SELECT id, season_id, opponent, date, time, is_home, source, event_type,
    template_id, end_time, created_at, rsvp_opt_out, rsvp_require_reason,
    venue_id, end_date
FROM games;

DROP TABLE games;
ALTER TABLE games_new RENAME TO games;

-- training_sessions: bestehende Spalte note bekommt CHECK ≤ 200 per Rebuild.
CREATE TABLE training_sessions_new (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    series_id     INTEGER  REFERENCES training_series(id) ON DELETE SET NULL,
    team_id       INTEGER  NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    season_id     INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    date          DATE     NOT NULL,
    start_time    TEXT     NOT NULL,
    end_time      TEXT     NOT NULL,
    location      TEXT     NOT NULL DEFAULT '',
    note          TEXT     NOT NULL DEFAULT '' CHECK (length(note) <= 200),
    status        TEXT     NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','cancelled')),
    cancel_reason TEXT     NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    title         TEXT     NOT NULL DEFAULT '',
    rsvp_opt_out        INTEGER NOT NULL DEFAULT 0,
    rsvp_require_reason INTEGER NOT NULL DEFAULT 1,
    venue_id      INTEGER  REFERENCES venues(id) ON DELETE SET NULL
);

INSERT INTO training_sessions_new (id, series_id, team_id, season_id, date,
    start_time, end_time, location, note, status, cancel_reason, created_at,
    title, rsvp_opt_out, rsvp_require_reason, venue_id)
SELECT id, series_id, team_id, season_id, date, start_time, end_time, location,
    note, status, cancel_reason, created_at, title, rsvp_opt_out,
    rsvp_require_reason, venue_id
FROM training_sessions;

DROP TABLE training_sessions;
ALTER TABLE training_sessions_new RENAME TO training_sessions;

CREATE INDEX idx_training_sessions_team_date ON training_sessions(team_id, date);
CREATE INDEX idx_training_sessions_series ON training_sessions(series_id);

-- Debounce-Queue: max. ein pending Push pro Termin (PK ref_type+ref_id).
CREATE TABLE pending_event_notes_push (
    ref_type     TEXT     NOT NULL CHECK (ref_type IN ('training','game')),
    ref_id       INTEGER  NOT NULL,
    note_text    TEXT     NOT NULL,
    notify_after DATETIME NOT NULL,
    updated_by   INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (ref_type, ref_id)
);
CREATE INDEX idx_pending_event_notes_due ON pending_event_notes_push(notify_after);
