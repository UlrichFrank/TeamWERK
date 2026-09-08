-- Rollback von 058: kader_id entfällt, team_id wieder NOT NULL.
--
-- Der Rebuild scheitert bewusst am NOT NULL, solange Trainings einer
-- Übungsgruppe existieren (team_id IS NULL) — sie hätten nach dem Down keinen
-- Besitzer mehr. Solche Termine müssen vorher gelöscht werden.
--
-- training_responses, training_attendances und member_series_unavailabilities
-- überstehen den Rebuild unbeschadet: sie referenzieren über die IDs, die der
-- INSERT … SELECT unverändert überträgt, und migrate läuft mit
-- PRAGMA foreign_keys=OFF.
PRAGMA legacy_alter_table=ON;

CREATE TABLE training_series_old (
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

INSERT INTO training_series_old (id, team_id, season_id, name, location,
    day_of_week, start_time, end_time, valid_from, valid_until, note,
    created_by, created_at, rsvp_default_players, rsvp_default_extended,
    rsvp_require_reason, venue_id)
SELECT id, team_id, season_id, name, location, day_of_week, start_time,
    end_time, valid_from, valid_until, note, created_by, created_at,
    rsvp_default_players, rsvp_default_extended, rsvp_require_reason, venue_id
FROM training_series;

DROP TABLE training_series;
ALTER TABLE training_series_old RENAME TO training_series;

CREATE INDEX idx_training_series_team ON training_series(team_id);

CREATE TABLE training_sessions_old (
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
    venue_id      INTEGER  REFERENCES venues(id) ON DELETE SET NULL,
    attendance_tracked  INTEGER NOT NULL DEFAULT 0,
    attendance_excluded INTEGER NOT NULL DEFAULT 0
);

INSERT INTO training_sessions_old (id, series_id, team_id, season_id, date,
    start_time, end_time, location, note, status, cancel_reason, created_at,
    title, rsvp_default_players, rsvp_default_extended, rsvp_require_reason,
    venue_id, attendance_tracked, attendance_excluded)
SELECT id, series_id, team_id, season_id, date, start_time, end_time, location,
    note, status, cancel_reason, created_at, title, rsvp_default_players,
    rsvp_default_extended, rsvp_require_reason, venue_id, attendance_tracked,
    attendance_excluded
FROM training_sessions;

DROP TABLE training_sessions;
ALTER TABLE training_sessions_old RENAME TO training_sessions;

CREATE INDEX idx_training_sessions_team_date ON training_sessions(team_id, date);
CREATE INDEX idx_training_sessions_series ON training_sessions(series_id);
