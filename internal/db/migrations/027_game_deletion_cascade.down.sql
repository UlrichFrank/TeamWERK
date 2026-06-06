PRAGMA foreign_keys = OFF;

CREATE TABLE duty_slots_old (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    event_name   TEXT     NOT NULL,
    event_date   DATE     NOT NULL,
    event_time   TEXT,
    duty_type_id INTEGER  NOT NULL REFERENCES duty_types(id) ON DELETE RESTRICT,
    role_desc    TEXT,
    slots_total  INTEGER  NOT NULL DEFAULT 1,
    slots_filled INTEGER  NOT NULL DEFAULT 0,
    team_id      INTEGER  REFERENCES teams(id)   ON DELETE SET NULL,
    season_id    INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    game_id      INTEGER  REFERENCES games(id)   ON DELETE SET NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    audiences    TEXT
);

INSERT INTO duty_slots_old SELECT * FROM duty_slots;

DROP TABLE duty_slots;
ALTER TABLE duty_slots_old RENAME TO duty_slots;

PRAGMA foreign_keys = ON;
