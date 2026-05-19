CREATE TABLE games (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    team_id     INTEGER  NOT NULL REFERENCES teams(id)   ON DELETE CASCADE,
    season_id   INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    opponent    TEXT     NOT NULL DEFAULT '',
    date        DATE     NOT NULL,
    time        TEXT     NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE game_templates (
    id                    INTEGER  PRIMARY KEY AUTOINCREMENT,
    name                  TEXT     NOT NULL DEFAULT 'Heimspiel Standard',
    game_duration_minutes INTEGER  NOT NULL DEFAULT 90,
    is_active             INTEGER  NOT NULL DEFAULT 1,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE game_template_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id     INTEGER NOT NULL REFERENCES game_templates(id) ON DELETE CASCADE,
    duty_type_id    INTEGER NOT NULL REFERENCES duty_types(id)     ON DELETE CASCADE,
    anchor          TEXT    NOT NULL DEFAULT 'start' CHECK (anchor IN ('start', 'end')),
    offset_minutes  INTEGER NOT NULL DEFAULT 0,
    slots_count     INTEGER NOT NULL DEFAULT 1,
    role_desc       TEXT    NOT NULL DEFAULT '',
    sort_order      INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE duty_slots ADD COLUMN event_time TEXT    NOT NULL DEFAULT '';
ALTER TABLE duty_slots ADD COLUMN game_id    INTEGER REFERENCES games(id) ON DELETE SET NULL;
