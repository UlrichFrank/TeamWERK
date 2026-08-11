ALTER TABLE game_template_items ADD COLUMN rotation_max_per_team INTEGER;
INSERT OR IGNORE INTO system_settings (key, value) VALUES ('bewirtung_verhaeltnis', '1');
