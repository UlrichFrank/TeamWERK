-- Rückbau in FK-sicherer Reihenfolge (rein additive Migration, kein Bestandsdatenverlust).
DROP TABLE IF EXISTS team_penalties;
DROP TABLE IF EXISTS kader_strafenwarte;
DROP TABLE IF EXISTS penalty_types;
DROP TABLE IF EXISTS member_responsibilities;
DROP TABLE IF EXISTS responsibility_types;
