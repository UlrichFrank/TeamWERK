-- SQLite does not support DROP COLUMN on older versions; recreate without end_time
CREATE TABLE games_new AS SELECT id, season_id, opponent, date, time, is_home, source, event_type, template_id, created_at FROM games;
DROP TABLE games;
ALTER TABLE games_new RENAME TO games;
