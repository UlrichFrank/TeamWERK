CREATE TABLE venues (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    name          TEXT     NOT NULL,
    street        TEXT     NOT NULL,
    city          TEXT     NOT NULL,
    postal_code   TEXT     NOT NULL,
    country       TEXT     NOT NULL DEFAULT 'DE',
    note          TEXT     NOT NULL DEFAULT '',
    is_home_venue INTEGER  NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE games ADD COLUMN venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL;

ALTER TABLE training_series ADD COLUMN venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL;

ALTER TABLE training_sessions ADD COLUMN venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL;

CREATE INDEX idx_venues_is_home ON venues(is_home_venue);
