-- Optionale Team-Einschränkung pro Vorlagen-Item (JSON-Array von teams.id).
-- NULL / [] = Item gilt für ALLE Teams des Spiels (bisheriges Verhalten).
ALTER TABLE game_template_items ADD COLUMN team_ids TEXT;
