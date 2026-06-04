-- 015: Qualifikations-Kader
-- Ermöglicht parallele Qualifikationskader neben regulären Kadern

ALTER TABLE kader ADD COLUMN type TEXT NOT NULL DEFAULT 'regular'
    CHECK(type IN ('regular','qualification'));
ALTER TABLE kader ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1;

DROP INDEX kader_unique;

-- Max. 1 aktiver regulärer Kader pro (season, age_class, gender, team_number)
CREATE UNIQUE INDEX kader_unique_active_regular
    ON kader(season_id, age_class, gender, team_number)
    WHERE type='regular' AND is_active=1;

-- Max. 1 aktiver Qualifikationskader pro (season, age_class, gender)
CREATE UNIQUE INDEX kader_unique_active_quali
    ON kader(season_id, age_class, gender)
    WHERE type='qualification' AND is_active=1;
