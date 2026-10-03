DROP TABLE IF EXISTS pending_game_meeting_push;
ALTER TABLE games DROP COLUMN meet_place;
ALTER TABLE games DROP COLUMN meet_offset_minutes;
