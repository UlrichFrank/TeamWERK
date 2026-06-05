CREATE VIEW player_memberships AS
SELECT km.id, km.member_id, k.team_id, k.season_id
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL;

CREATE VIEW trainer_memberships AS
SELECT kt.kader_id * 100000 + kt.member_id AS id, kt.member_id, k.team_id, k.season_id
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
WHERE k.team_id IS NOT NULL;
