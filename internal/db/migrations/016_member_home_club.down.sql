-- 016 down: remove home_club
-- SQLite does not support DROP COLUMN before 3.35.0; recreate table if needed
ALTER TABLE members DROP COLUMN home_club;
