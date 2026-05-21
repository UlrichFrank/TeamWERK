-- Add template_type to game_templates (table rebuild needed for CHECK constraint in SQLite)
PRAGMA foreign_keys=OFF;

CREATE TABLE game_templates_new (
    id                    INTEGER  PRIMARY KEY AUTOINCREMENT,
    name                  TEXT     NOT NULL DEFAULT 'Heimspiel Standard',
    game_duration_minutes INTEGER  NOT NULL DEFAULT 90,
    is_active             INTEGER  NOT NULL DEFAULT 0,
    template_type         TEXT     NOT NULL DEFAULT 'generisch'
                          CHECK(template_type IN ('heim','auswärts','generisch')),
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO game_templates_new (id, name, game_duration_minutes, is_active, template_type, created_at)
SELECT id, name, game_duration_minutes, is_active, 'generisch', created_at
FROM game_templates;

DROP TABLE game_templates;
ALTER TABLE game_templates_new RENAME TO game_templates;

PRAGMA foreign_keys=ON;
