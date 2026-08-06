DROP TABLE IF EXISTS h4a_staffel_team_map;

DROP INDEX IF EXISTS idx_games_external_id;
ALTER TABLE games DROP COLUMN external_id;

DROP INDEX IF EXISTS idx_venues_hall_number;
ALTER TABLE venues DROP COLUMN hall_number;
