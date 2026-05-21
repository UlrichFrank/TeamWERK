ALTER TABLE duty_types ADD COLUMN same_day_behavior TEXT NOT NULL DEFAULT 'normal' CHECK (same_day_behavior IN ('normal', 'skip', 'reduced'));
ALTER TABLE duty_types ADD COLUMN same_day_variant_id INTEGER REFERENCES duty_types(id);
ALTER TABLE duty_types ADD COLUMN adjacent_day_behavior TEXT NOT NULL DEFAULT 'normal' CHECK (adjacent_day_behavior IN ('normal', 'skip', 'reduced'));
ALTER TABLE duty_types ADD COLUMN adjacent_day_variant_id INTEGER REFERENCES duty_types(id);
