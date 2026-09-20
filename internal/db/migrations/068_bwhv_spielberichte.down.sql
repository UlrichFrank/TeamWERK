-- Rücknahme in Abhängigkeitsreihenfolge (Blätter zuerst).
DROP INDEX IF EXISTS idx_bwhv_events_player;
DROP TABLE IF EXISTS bwhv_events;

DROP INDEX IF EXISTS idx_bwhv_player_games_player;
DROP TABLE IF EXISTS bwhv_player_games;

DROP INDEX IF EXISTS idx_bwhv_players_member;
DROP TABLE IF EXISTS bwhv_players;

DROP TABLE IF EXISTS bwhv_reports;

DROP INDEX IF EXISTS idx_bwhv_games_game_id;
DROP INDEX IF EXISTS idx_bwhv_games_staffel_date;
DROP TABLE IF EXISTS bwhv_games;

DROP TABLE IF EXISTS bwhv_staffeln;

ALTER TABLE kader DROP COLUMN staffel;
