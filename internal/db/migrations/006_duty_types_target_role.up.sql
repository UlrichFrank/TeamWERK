ALTER TABLE duty_types ADD COLUMN target_role TEXT NOT NULL DEFAULT 'elternteil' CHECK(target_role IN ('spieler','elternteil','trainer','admin','vorstand'));
