-- Opt-In pro Member: erlaubt Anzeige des Members in Teilnehmerlisten von
-- Multi-Team-Events für Mitglieder fremder Teams. Default 0 (privat).
ALTER TABLE members ADD COLUMN cross_team_visible INTEGER NOT NULL DEFAULT 0;
