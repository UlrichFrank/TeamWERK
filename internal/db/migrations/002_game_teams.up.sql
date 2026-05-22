-- Add event_type to games
ALTER TABLE games ADD COLUMN event_type TEXT NOT NULL DEFAULT 'heim' CHECK (event_type IN ('heim', 'auswärts', 'generisch'));

-- Derive event_type from existing is_home values
UPDATE games SET event_type = CASE WHEN is_home = 1 THEN 'heim' ELSE 'auswärts' END;

-- Create game_teams junction table
CREATE TABLE game_teams (
    game_id  INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    team_id  INTEGER NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
    PRIMARY KEY (game_id, team_id)
);

-- Migrate existing games.team_id to game_teams (only where team still exists)
INSERT INTO game_teams (game_id, team_id)
SELECT g.id, g.team_id FROM games g
INNER JOIN teams t ON t.id = g.team_id
WHERE g.team_id IS NOT NULL;

-- Rebuild games table without team_id
CREATE TABLE games_new (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id   INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    opponent    TEXT     NOT NULL,
    date        DATE     NOT NULL,
    time        TEXT     NOT NULL DEFAULT '00:00',
    is_home     INTEGER  NOT NULL DEFAULT 1,
    source      TEXT     NOT NULL DEFAULT 'manual',
    event_type  TEXT     NOT NULL DEFAULT 'heim' CHECK (event_type IN ('heim', 'auswärts', 'generisch')),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO games_new (id, season_id, opponent, date, time, is_home, source, event_type, created_at)
SELECT id, season_id, opponent, date, time, is_home, source, event_type, created_at FROM games;

DROP TABLE games;
ALTER TABLE games_new RENAME TO games;
