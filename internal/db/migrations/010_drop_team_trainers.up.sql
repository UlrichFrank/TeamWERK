-- 010: Drop legacy team_trainers table.
-- Trainer-team assignments are managed exclusively via kader_trainers.
DROP TABLE IF EXISTS team_trainers;
