ALTER TABLE duty_types ADD COLUMN audiences TEXT;
ALTER TABLE duty_types DROP COLUMN audience;
ALTER TABLE game_template_items ADD COLUMN audiences TEXT;
ALTER TABLE game_template_items DROP COLUMN audience;
ALTER TABLE duty_slots ADD COLUMN audiences TEXT;
ALTER TABLE duty_slots DROP COLUMN audience;
