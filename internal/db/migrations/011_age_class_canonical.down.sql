-- Restore age_class_game_rules with short-key PKs.
CREATE TABLE age_class_game_rules_old (
    age_class              TEXT    PRIMARY KEY CHECK(age_class IN ('A','B','C','D')),
    half_duration_minutes  INTEGER NOT NULL CHECK(half_duration_minutes > 0),
    break_minutes          INTEGER NOT NULL CHECK(break_minutes > 0)
);

INSERT INTO age_class_game_rules_old (age_class, half_duration_minutes, break_minutes)
SELECT SUBSTR(age_class, 1, 1), half_duration_minutes, break_minutes
FROM age_class_game_rules
WHERE age_class LIKE '_-Jugend';

DROP TABLE age_class_game_rules;
ALTER TABLE age_class_game_rules_old RENAME TO age_class_game_rules;
