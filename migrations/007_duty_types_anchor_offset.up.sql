ALTER TABLE duty_types ADD COLUMN default_anchor          TEXT    NOT NULL DEFAULT 'start';
ALTER TABLE duty_types ADD COLUMN default_offset_minutes  INTEGER NOT NULL DEFAULT 0;
