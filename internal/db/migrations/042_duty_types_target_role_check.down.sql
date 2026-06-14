-- Rollback: alten CHECK wiederherstellen. 'sportliche_leitung','vorstand_beisitzer','kassierer'
-- werden zurück auf 'vorstand' gemappt, da der alte CHECK sie nicht erlaubt.
PRAGMA foreign_keys = OFF;

CREATE TABLE duty_types_new (
    id                      INTEGER  PRIMARY KEY AUTOINCREMENT,
    name                    TEXT     NOT NULL,
    hours_value             REAL     NOT NULL DEFAULT 1.0,
    cash_substitute         REAL,
    default_anchor          TEXT     NOT NULL DEFAULT 'start',
    default_offset_minutes  INTEGER  NOT NULL DEFAULT 0,
    target_role             TEXT     NOT NULL DEFAULT 'elternteil'
                            CHECK(target_role IN ('spieler','elternteil','trainer','admin','vorstand')),
    consecutive_behavior    TEXT     NOT NULL DEFAULT 'normal'
                            CHECK(consecutive_behavior IN ('normal','skip','reduced')),
    consecutive_variant_id  INTEGER  REFERENCES duty_types(id),
    same_day_behavior       TEXT     NOT NULL DEFAULT 'normal'
                            CHECK(same_day_behavior IN ('normal','skip','reduced')),
    same_day_variant_id     INTEGER  REFERENCES duty_types(id),
    adjacent_day_behavior   TEXT     NOT NULL DEFAULT 'normal'
                            CHECK(adjacent_day_behavior IN ('normal','skip','reduced')),
    adjacent_day_variant_id INTEGER  REFERENCES duty_types(id),
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    audiences               TEXT
);

INSERT INTO duty_types_new (
    id, name, hours_value, cash_substitute, default_anchor, default_offset_minutes,
    target_role,
    consecutive_behavior, consecutive_variant_id,
    same_day_behavior, same_day_variant_id,
    adjacent_day_behavior, adjacent_day_variant_id,
    created_at, audiences
)
SELECT
    id, name, hours_value, cash_substitute, default_anchor, default_offset_minutes,
    CASE target_role
        WHEN 'sportliche_leitung' THEN 'vorstand'
        WHEN 'vorstand_beisitzer' THEN 'vorstand'
        WHEN 'kassierer' THEN 'vorstand'
        ELSE target_role
    END,
    consecutive_behavior, consecutive_variant_id,
    same_day_behavior, same_day_variant_id,
    adjacent_day_behavior, adjacent_day_variant_id,
    created_at, audiences
FROM duty_types;

DROP TABLE duty_types;
ALTER TABLE duty_types_new RENAME TO duty_types;

PRAGMA foreign_keys = ON;
