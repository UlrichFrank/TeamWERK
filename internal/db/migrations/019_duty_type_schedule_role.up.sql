ALTER TABLE duty_types ADD COLUMN consecutive_behavior TEXT NOT NULL DEFAULT 'normal' CHECK (consecutive_behavior IN ('normal', 'skip', 'reduced'));
ALTER TABLE duty_types ADD COLUMN consecutive_variant_id INTEGER REFERENCES duty_types(id);
