-- RSVP-Voreinstellung pro Rolle: ersetzt den Boolean rsvp_opt_out durch zwei
-- unabhängige Enum-Spalten für Stammkader-Spieler und Erweiterten Kader.
-- Werte: 'confirmed' | 'declined' | 'none'.
--
-- SQLite kennt kein ADD/DROP COLUMN mit CHECK-Constraint → Tabellen-Rebuild.
-- Backfill konservativ: opt_out=1 → players='confirmed', sonst 'none';
-- extended überall 'none' (aktuelles Verhalten für Erweiterten Kader).

-- games ------------------------------------------------------------------
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
    rsvp_default_players  TEXT NOT NULL DEFAULT 'none'
                CHECK (rsvp_default_players IN ('confirmed','declined','none')),
    rsvp_default_extended TEXT NOT NULL DEFAULT 'none'
                CHECK (rsvp_default_extended IN ('confirmed','declined','none')),
    rsvp_require_reason INTEGER NOT NULL DEFAULT 1,
    venue_id    INTEGER  REFERENCES venues(id) ON DELETE SET NULL,
    end_date    DATE,
    note        TEXT     NOT NULL DEFAULT '' CHECK (length(note) <= 200)
);

INSERT INTO games_new (id, season_id, opponent, date, time, is_home, source,
    event_type, template_id, end_time, created_at,
    rsvp_default_players, rsvp_default_extended,
    rsvp_require_reason, venue_id, end_date, note)
SELECT id, season_id, opponent, date, time, is_home, source, event_type,
    template_id, end_time, created_at,
    CASE WHEN rsvp_opt_out = 1 THEN 'confirmed' ELSE 'none' END,
    'none',
    rsvp_require_reason, venue_id, end_date, note
FROM games;

DROP TABLE games;
ALTER TABLE games_new RENAME TO games;

-- training_sessions ------------------------------------------------------
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
    rsvp_default_players  TEXT NOT NULL DEFAULT 'none'
                CHECK (rsvp_default_players IN ('confirmed','declined','none')),
    rsvp_default_extended TEXT NOT NULL DEFAULT 'none'
                CHECK (rsvp_default_extended IN ('confirmed','declined','none')),
    rsvp_require_reason INTEGER NOT NULL DEFAULT 1,
    venue_id      INTEGER  REFERENCES venues(id) ON DELETE SET NULL
);

INSERT INTO training_sessions_new (id, series_id, team_id, season_id, date,
    start_time, end_time, location, note, status, cancel_reason, created_at,
    title,
    rsvp_default_players, rsvp_default_extended,
    rsvp_require_reason, venue_id)
SELECT id, series_id, team_id, season_id, date, start_time, end_time, location,
    note, status, cancel_reason, created_at, title,
    CASE WHEN rsvp_opt_out = 1 THEN 'confirmed' ELSE 'none' END,
    'none',
    rsvp_require_reason, venue_id
FROM training_sessions;

DROP TABLE training_sessions;
ALTER TABLE training_sessions_new RENAME TO training_sessions;

CREATE INDEX idx_training_sessions_team_date ON training_sessions(team_id, date);
CREATE INDEX idx_training_sessions_series ON training_sessions(series_id);

-- training_series --------------------------------------------------------
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
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rsvp_default_players  TEXT NOT NULL DEFAULT 'none'
                CHECK (rsvp_default_players IN ('confirmed','declined','none')),
    rsvp_default_extended TEXT NOT NULL DEFAULT 'none'
                CHECK (rsvp_default_extended IN ('confirmed','declined','none')),
    rsvp_require_reason INTEGER NOT NULL DEFAULT 1,
    venue_id     INTEGER  REFERENCES venues(id) ON DELETE SET NULL
);

INSERT INTO training_series_new (id, team_id, season_id, name, location,
    day_of_week, start_time, end_time, valid_from, valid_until, note,
    created_by, created_at,
    rsvp_default_players, rsvp_default_extended,
    rsvp_require_reason, venue_id)
SELECT id, team_id, season_id, name, location, day_of_week, start_time,
    end_time, valid_from, valid_until, note, created_by, created_at,
    CASE WHEN rsvp_opt_out = 1 THEN 'confirmed' ELSE 'none' END,
    'none',
    rsvp_require_reason, venue_id
FROM training_series;

DROP TABLE training_series;
ALTER TABLE training_series_new RENAME TO training_series;

CREATE INDEX idx_training_series_team ON training_series(team_id);
