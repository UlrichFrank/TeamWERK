ALTER TABLE duty_types ADD COLUMN audience TEXT CHECK(audience IN ('spieler','trainer','vorstand','vorstand_beisitzer','eltern'));
ALTER TABLE game_template_items ADD COLUMN audience TEXT CHECK(audience IN ('spieler','trainer','vorstand','vorstand_beisitzer','eltern'));
ALTER TABLE duty_slots ADD COLUMN audience TEXT CHECK(audience IN ('spieler','trainer','vorstand','vorstand_beisitzer','eltern'));
