CREATE TABLE age_class_game_rules (
    age_class          TEXT PRIMARY KEY CHECK(age_class IN ('A','B','C','D')),
    half_duration_minutes INTEGER NOT NULL CHECK(half_duration_minutes > 0),
    break_minutes         INTEGER NOT NULL CHECK(break_minutes > 0)
);

INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES
    ('A', 30, 15),
    ('B', 25, 10),
    ('C', 25, 10),
    ('D', 20, 10);

ALTER TABLE game_templates RENAME COLUMN game_duration_minutes TO duration_minutes;
