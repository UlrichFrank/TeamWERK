-- Rebuild games with team_id (from game_teams)
CREATE TABLE games_new (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    team_id     INTEGER  NOT NULL DEFAULT 0,
    season_id   INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    opponent    TEXT     NOT NULL,
    date        DATE     NOT NULL,
    time        TEXT     NOT NULL DEFAULT '00:00',
    is_home     INTEGER  NOT NULL DEFAULT 1,
    source      TEXT     NOT NULL DEFAULT 'manual',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO games_new (id, team_id, season_id, opponent, date, time, is_home, source, created_at)
SELECT g.id,
       COALESCE(gt.team_id, 0),
       g.season_id,
       g.opponent,
       g.date,
       g.time,
       g.is_home,
       g.source,
       g.created_at
FROM games g
LEFT JOIN game_teams gt ON g.id = gt.game_id;

DROP TABLE game_teams;
DROP TABLE games;
ALTER TABLE games_new RENAME TO games;
