-- Rückbau in FK-sicherer Reihenfolge (rein additive Migration, kein Bestandsdatenverlust).
DROP TABLE IF EXISTS kader_kassenwarte;
DROP TABLE IF EXISTS team_cashbook_entries;
DROP TABLE IF EXISTS penalty_settings;
