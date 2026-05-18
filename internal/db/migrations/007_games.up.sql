-- game_templates: one global template (is_active=1)
CREATE TABLE game_templates (
    id                    INTEGER  PRIMARY KEY AUTOINCREMENT,
    name                  TEXT     NOT NULL DEFAULT 'Heimspiel Standard',
    game_duration_minutes INTEGER  NOT NULL DEFAULT 90,
    is_active             INTEGER  NOT NULL DEFAULT 1,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- game_template_items: one row per slot type in the template
CREATE TABLE game_template_items (
    id             INTEGER  PRIMARY KEY AUTOINCREMENT,
    template_id    INTEGER  NOT NULL REFERENCES game_templates(id) ON DELETE CASCADE,
    duty_type_id   INTEGER  NOT NULL REFERENCES duty_types(id)     ON DELETE RESTRICT,
    anchor         TEXT     NOT NULL DEFAULT 'start' CHECK (anchor IN ('start','end')),
    offset_minutes INTEGER  NOT NULL DEFAULT 0,
    slots_count    INTEGER  NOT NULL DEFAULT 1,
    role_desc      TEXT,
    sort_order     INTEGER  NOT NULL DEFAULT 0,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- games: one row per home game
CREATE TABLE games (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    team_id     INTEGER  NOT NULL REFERENCES teams(id)   ON DELETE RESTRICT,
    season_id   INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    opponent    TEXT     NOT NULL,
    date        DATE     NOT NULL,
    time        TEXT     NOT NULL DEFAULT '00:00',
    is_home     INTEGER  NOT NULL DEFAULT 1,
    source      TEXT     NOT NULL DEFAULT 'manual',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- extend duty_slots: add event_time + game_id via rebuild (SQLite has no ADD CONSTRAINT)
PRAGMA foreign_keys=OFF;

CREATE TABLE duty_slots_new (
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
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO duty_slots_new
    SELECT id, event_name, event_date, NULL, duty_type_id, role_desc,
           slots_total, slots_filled, team_id, season_id, NULL, created_at
    FROM duty_slots;

DROP TABLE duty_slots;
ALTER TABLE duty_slots_new RENAME TO duty_slots;

PRAGMA foreign_keys=ON;
