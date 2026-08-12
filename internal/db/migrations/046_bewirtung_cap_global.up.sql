-- Der Cap "Max. Kuchen pro Mannschaft" wandert vom Vorlagen-Item in die
-- vereinsweiten Einstellungen (bewirtung-cap-global). Am Item bleibt nur noch
-- der boolesche Schalter, der die Rotation aktiviert.
--
-- Reihenfolge ist zwingend: der Startwert des Settings und der Schalter werden
-- BEIDE aus der alten Spalte abgeleitet, die erst danach fallen darf.

-- Startwert = größter bereits konfigurierter Item-Cap, sonst der fachliche
-- Default 1. CAST AS TEXT, weil system_settings.value ein String-Feld ist und
-- die Konsumenten (settings.GetBewirtungMaxPerTeam) in einen String scannen.
INSERT OR IGNORE INTO system_settings (key, value)
VALUES (
    'bewirtung_max_per_team',
    CAST(COALESCE((SELECT MAX(rotation_max_per_team) FROM game_template_items), 1) AS TEXT)
);

ALTER TABLE game_template_items ADD COLUMN rotation_enabled INTEGER NOT NULL DEFAULT 0;

UPDATE game_template_items SET rotation_enabled = 1 WHERE rotation_max_per_team IS NOT NULL;

ALTER TABLE game_template_items DROP COLUMN rotation_max_per_team;
