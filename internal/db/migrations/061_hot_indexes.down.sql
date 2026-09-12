-- Rückbau der Hot-Path-Indizes aus 061_hot_indexes.up.sql. Reines Entfernen
-- von Indizes — keine Datentabelle betroffen, kein Datenverlust.
DROP INDEX IF EXISTS idx_duty_slots_game;
DROP INDEX IF EXISTS idx_duty_slots_date_season;
DROP INDEX IF EXISTS idx_family_links_member;
DROP INDEX IF EXISTS idx_duty_assignments_user;
