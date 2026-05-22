-- Link kader entries to the teams table so Spielplan can derive teams from Kader.

ALTER TABLE kader ADD COLUMN team_id INTEGER REFERENCES teams(id);

-- Create canonical teams for every distinct (age_class, gender, team_number) in kader.
INSERT INTO teams (name, age_class, gender, is_active)
SELECT DISTINCT
    CASE WHEN k.team_number > 1
        THEN k.age_class || ' ' || CASE k.gender WHEN 'm' THEN 'männlich' WHEN 'f' THEN 'weiblich' ELSE 'gemischt' END || ' ' || CAST(k.team_number AS TEXT)
        ELSE k.age_class || ' ' || CASE k.gender WHEN 'm' THEN 'männlich' WHEN 'f' THEN 'weiblich' ELSE 'gemischt' END
    END,
    k.age_class,
    k.gender,
    1
FROM kader k;

-- Back-fill team_id on existing kader rows.
UPDATE kader SET team_id = (
    SELECT t.id FROM teams t
    WHERE t.age_class = kader.age_class
      AND t.gender    = kader.gender
      AND t.name      = CASE WHEN kader.team_number > 1
          THEN kader.age_class || ' ' || CASE kader.gender WHEN 'm' THEN 'männlich' WHEN 'f' THEN 'weiblich' ELSE 'gemischt' END || ' ' || CAST(kader.team_number AS TEXT)
          ELSE kader.age_class || ' ' || CASE kader.gender WHEN 'm' THEN 'männlich' WHEN 'f' THEN 'weiblich' ELSE 'gemischt' END
      END
    LIMIT 1
);
