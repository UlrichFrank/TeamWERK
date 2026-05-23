DROP TABLE team_memberships;

CREATE VIEW team_memberships AS
SELECT km.id, km.member_id, k.team_id, k.season_id, 0 AS is_primary
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL

UNION

SELECT kt.kader_id * 100000 + kt.member_id, kt.member_id, k.team_id, k.season_id, 0
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
WHERE k.team_id IS NOT NULL;
