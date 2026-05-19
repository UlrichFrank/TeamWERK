DROP TABLE IF EXISTS game_template_items;
DROP TABLE IF EXISTS game_templates;
DROP TABLE IF EXISTS games;

-- SQLite does not support DROP COLUMN; recreate duty_slots without event_time and game_id
CREATE TABLE duty_slots_backup AS
    SELECT id, event_name, event_date, duty_type_id, role_desc,
           slots_total, slots_filled, team_id, season_id, created_at
    FROM duty_slots;
DROP TABLE duty_slots;
ALTER TABLE duty_slots_backup RENAME TO duty_slots;
