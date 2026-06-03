-- SQLite does not support DROP COLUMN in older versions; recreate table without title
CREATE TABLE training_sessions_bak AS SELECT id, series_id, team_id, season_id, date, start_time, end_time, location, note, status, cancel_reason, created_at FROM training_sessions;
DROP TABLE training_sessions;
ALTER TABLE training_sessions_bak RENAME TO training_sessions;
