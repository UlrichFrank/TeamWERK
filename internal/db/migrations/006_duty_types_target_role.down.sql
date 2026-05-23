-- Recreate duty_types without target_role (SQLite has no DROP COLUMN before 3.35)
PRAGMA foreign_keys = OFF;

CREATE TABLE duty_types_new (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    name                     TEXT    NOT NULL,
    hours_value              REAL    NOT NULL DEFAULT 1.0,
    cash_substitute          REAL,
    created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    default_anchor           TEXT    NOT NULL DEFAULT 'start',
    default_offset_minutes   INTEGER NOT NULL DEFAULT 0,
    same_day_behavior        TEXT    NOT NULL DEFAULT 'normal' CHECK (same_day_behavior IN ('normal', 'skip', 'reduced')),
    same_day_variant_id      INTEGER REFERENCES duty_types_new(id),
    adjacent_day_behavior    TEXT    NOT NULL DEFAULT 'normal' CHECK (adjacent_day_behavior IN ('normal', 'skip', 'reduced')),
    adjacent_day_variant_id  INTEGER REFERENCES duty_types_new(id)
);

INSERT INTO duty_types_new
    SELECT id, name, hours_value, cash_substitute, created_at,
           default_anchor, default_offset_minutes,
           same_day_behavior, same_day_variant_id,
           adjacent_day_behavior, adjacent_day_variant_id
    FROM duty_types;

DROP TABLE duty_types;
ALTER TABLE duty_types_new RENAME TO duty_types;

PRAGMA foreign_keys = ON;
