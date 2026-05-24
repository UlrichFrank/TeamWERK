-- Rebuild age_class_game_rules with full German names as PKs.
-- SQLite can't ALTER a CHECK constraint, so we recreate the table.
-- Note: No other table currently has a FK to age_class_game_rules,
-- so this can be done within a normal transaction.

CREATE TABLE age_class_game_rules_new (
    age_class              TEXT    PRIMARY KEY CHECK(age_class IN ('A-Jugend','B-Jugend','C-Jugend','D-Jugend')),
    half_duration_minutes  INTEGER NOT NULL CHECK(half_duration_minutes > 0),
    break_minutes          INTEGER NOT NULL CHECK(break_minutes > 0)
);

-- Migrate existing short-key rows to long-key rows (from migration 010).
INSERT INTO age_class_game_rules_new (age_class, half_duration_minutes, break_minutes)
SELECT
    age_class || '-Jugend',
    half_duration_minutes,
    break_minutes
FROM age_class_game_rules
WHERE age_class IN ('A','B','C','D');

-- Insert defaults for any class not yet present (idempotent).
INSERT OR IGNORE INTO age_class_game_rules_new (age_class, half_duration_minutes, break_minutes) VALUES
    ('A-Jugend', 30, 15),
    ('B-Jugend', 25, 10),
    ('C-Jugend', 25, 10),
    ('D-Jugend', 20, 10);

DROP TABLE age_class_game_rules;
ALTER TABLE age_class_game_rules_new RENAME TO age_class_game_rules;
