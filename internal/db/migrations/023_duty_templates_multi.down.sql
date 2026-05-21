-- Remove template_type from game_templates (table rebuild)
PRAGMA foreign_keys=OFF;

CREATE TABLE game_templates_old (
    id                    INTEGER  PRIMARY KEY AUTOINCREMENT,
    name                  TEXT     NOT NULL DEFAULT 'Heimspiel Standard',
    game_duration_minutes INTEGER  NOT NULL DEFAULT 90,
    is_active             INTEGER  NOT NULL DEFAULT 0,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO game_templates_old (id, name, game_duration_minutes, is_active, created_at)
SELECT id, name, game_duration_minutes, is_active, created_at
FROM game_templates;

DROP TABLE game_templates;
ALTER TABLE game_templates_old RENAME TO game_templates;

PRAGMA foreign_keys=ON;
