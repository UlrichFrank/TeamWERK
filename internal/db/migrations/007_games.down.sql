PRAGMA foreign_keys=OFF;

-- Rebuild duty_slots removing event_time and game_id
CREATE TABLE duty_slots_old (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    event_name   TEXT     NOT NULL,
    event_date   DATE     NOT NULL,
    duty_type_id INTEGER  NOT NULL REFERENCES duty_types(id) ON DELETE RESTRICT,
    role_desc    TEXT,
    slots_total  INTEGER  NOT NULL DEFAULT 1,
    slots_filled INTEGER  NOT NULL DEFAULT 0,
    team_id      INTEGER  REFERENCES teams(id)   ON DELETE SET NULL,
    season_id    INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO duty_slots_old
    SELECT id, event_name, event_date, duty_type_id, role_desc,
           slots_total, slots_filled, team_id, season_id, created_at
    FROM duty_slots;

DROP TABLE duty_slots;
ALTER TABLE duty_slots_old RENAME TO duty_slots;

DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS game_template_items;
DROP TABLE IF EXISTS game_templates;

PRAGMA foreign_keys=ON;
