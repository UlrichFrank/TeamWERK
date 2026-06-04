-- Reverse 015: Qualifikations-Kader

DROP INDEX IF EXISTS kader_unique_active_regular;
DROP INDEX IF EXISTS kader_unique_active_quali;

-- Tabellen-Rebuild um Spalten zu entfernen (SQLite unterstützt kein DROP COLUMN für CHECK-Spalten zuverlässig)
PRAGMA foreign_keys = OFF;

CREATE TABLE kader_new (
    id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
    season_id            INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    age_class            TEXT     NOT NULL,
    gender               TEXT     NOT NULL CHECK (gender IN ('m','f','mixed')),
    team_number          INTEGER  NOT NULL DEFAULT 1,
    team_id              INTEGER  REFERENCES teams(id),
    dedicated_birth_year INTEGER,
    games_per_season     INTEGER  NOT NULL DEFAULT 0,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO kader_new SELECT id, season_id, age_class, gender, team_number, team_id,
    dedicated_birth_year, games_per_season, created_at, updated_at FROM kader;

DROP TABLE kader;
ALTER TABLE kader_new RENAME TO kader;

CREATE UNIQUE INDEX kader_unique ON kader(season_id, age_class, gender, team_number);

PRAGMA foreign_keys = ON;
