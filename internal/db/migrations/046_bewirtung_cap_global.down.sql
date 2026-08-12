-- Rückweg: der vereinsweite Cap wird auf jedes rotations-aktive Item zurück-
-- geschrieben (alle Items bekommen denselben Wert — die frühere Möglichkeit
-- abweichender Caps pro Vorlage lässt sich nicht rekonstruieren und war der
-- Grund für den Umzug).

ALTER TABLE game_template_items ADD COLUMN rotation_max_per_team INTEGER;

UPDATE game_template_items
SET rotation_max_per_team = CAST(
    COALESCE((SELECT value FROM system_settings WHERE key = 'bewirtung_max_per_team'), '1') AS INTEGER
)
WHERE rotation_enabled = 1;

ALTER TABLE game_template_items DROP COLUMN rotation_enabled;

DELETE FROM system_settings WHERE key = 'bewirtung_max_per_team';
