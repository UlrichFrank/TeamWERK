-- Rollback von 057: kader zurück auf NOT NULL age_class/gender, ohne kind/name.
--
-- Der Rebuild scheitert bewusst am NOT NULL, solange kind='practice'-Zeilen
-- existieren — ein stiller Datenverlust (Übungsgruppen samt Mitgliedern und
-- Trainern) wäre die Alternative. Vor dem Down müssen Übungsgruppen gelöscht
-- werden; deren Trainings blockieren das ihrerseits (ON DELETE RESTRICT).
PRAGMA legacy_alter_table=ON;

CREATE TABLE kader_old (
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

INSERT INTO kader_old (id, season_id, age_class, gender, team_number, team_id,
    dedicated_birth_year, games_per_season, created_at, updated_at)
SELECT id, season_id, age_class, gender, team_number, team_id,
    dedicated_birth_year, games_per_season, created_at, updated_at
FROM kader;

DROP TABLE kader;
ALTER TABLE kader_old RENAME TO kader;

CREATE UNIQUE INDEX kader_unique ON kader(season_id, age_class, gender, team_number);
CREATE INDEX idx_kader_season ON kader(season_id);
