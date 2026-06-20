-- SQLite unterstützt DROP COLUMN seit 3.35 (modernc.org/sqlite ist neuer),
-- analog zu Migration 035.
ALTER TABLE members DROP COLUMN home_club_id;
DROP TABLE IF EXISTS stammvereine;
