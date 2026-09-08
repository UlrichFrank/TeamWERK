-- 058_trainings_kader_owner: Der Kader ist der Besitzer eines Trainings.
--
-- training_sessions.kader_id und training_series.kader_id werden zum
-- alleinigen Besitzer (NOT NULL). team_id bleibt erhalten, wird aber nullable
-- und damit zur Projektion: gesetzt = gehört einer Mannschaft, NULL = gehört
-- einer Übungsgruppe (proposal.md — Invariante 1). Diese NULL trägt den
-- gesamten Ausschluss der Übungsgruppen aus den team-gebundenen Auswertungen
-- und darf nie durch einen "Reparatur"-Backfill gefüllt werden.
--
-- Backfill über (team_id, season_id): kader führt je Saison höchstens eine
-- Zeile pro Team, die Zuordnung ist damit eindeutig. Ein Datensatz ohne
-- passenden Kader lässt die Migration am NOT NULL abbrechen — gewollt (laut
-- scheitern statt still NULL). Zählabfrage vor dem Deploy: tasks.md 6.1.
--
-- ON DELETE RESTRICT, nicht CASCADE: teams werden nie gelöscht, nur über
-- is_active stillgelegt — Kader sind löschbar. Ein CASCADE nähme beim
-- Aufräumen eines leeren Altkaders die Trainingshistorie (Anwesenheit, RSVP)
-- still mit.
PRAGMA legacy_alter_table=ON;

-- training_series --------------------------------------------------------
CREATE TABLE training_series_new (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id     INTEGER  NOT NULL REFERENCES kader(id) ON DELETE RESTRICT,
    team_id      INTEGER  REFERENCES teams(id) ON DELETE CASCADE,
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

INSERT INTO training_series_new (id, kader_id, team_id, season_id, name,
    location, day_of_week, start_time, end_time, valid_from, valid_until, note,
    created_by, created_at, rsvp_default_players, rsvp_default_extended,
    rsvp_require_reason, venue_id)
SELECT tsr.id,
    (SELECT k.id FROM kader k
      WHERE k.team_id = tsr.team_id AND k.season_id = tsr.season_id),
    tsr.team_id, tsr.season_id, tsr.name, tsr.location, tsr.day_of_week,
    tsr.start_time, tsr.end_time, tsr.valid_from, tsr.valid_until, tsr.note,
    tsr.created_by, tsr.created_at, tsr.rsvp_default_players,
    tsr.rsvp_default_extended, tsr.rsvp_require_reason, tsr.venue_id
FROM training_series tsr;

DROP TABLE training_series;
ALTER TABLE training_series_new RENAME TO training_series;

CREATE INDEX idx_training_series_team ON training_series(team_id);
CREATE INDEX idx_training_series_kader ON training_series(kader_id);

-- training_sessions ------------------------------------------------------
CREATE TABLE training_sessions_new (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    series_id     INTEGER  REFERENCES training_series(id) ON DELETE SET NULL,
    kader_id      INTEGER  NOT NULL REFERENCES kader(id) ON DELETE RESTRICT,
    team_id       INTEGER  REFERENCES teams(id) ON DELETE CASCADE,
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

INSERT INTO training_sessions_new (id, series_id, kader_id, team_id, season_id,
    date, start_time, end_time, location, note, status, cancel_reason,
    created_at, title, rsvp_default_players, rsvp_default_extended,
    rsvp_require_reason, venue_id, attendance_tracked, attendance_excluded)
SELECT ts.id, ts.series_id,
    (SELECT k.id FROM kader k
      WHERE k.team_id = ts.team_id AND k.season_id = ts.season_id),
    ts.team_id, ts.season_id, ts.date, ts.start_time, ts.end_time, ts.location,
    ts.note, ts.status, ts.cancel_reason, ts.created_at, ts.title,
    ts.rsvp_default_players, ts.rsvp_default_extended, ts.rsvp_require_reason,
    ts.venue_id, ts.attendance_tracked, ts.attendance_excluded
FROM training_sessions ts;

DROP TABLE training_sessions;
ALTER TABLE training_sessions_new RENAME TO training_sessions;

CREATE INDEX idx_training_sessions_team_date ON training_sessions(team_id, date);
CREATE INDEX idx_training_sessions_series ON training_sessions(series_id);
CREATE INDEX idx_training_sessions_kader_date ON training_sessions(kader_id, date);
