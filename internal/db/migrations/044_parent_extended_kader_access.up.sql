-- 044: Eltern von Anwärtern (kader_extended_members) sehen das Team ihres Kindes.
-- Bisher fehlte ein Branch in user_accessible_teams für family_links → kader_extended_members.
DROP VIEW IF EXISTS user_accessible_teams;

CREATE VIEW user_accessible_teams AS
SELECT m.user_id, k.team_id, k.season_id
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
JOIN members m ON m.id = km.member_id
WHERE k.team_id IS NOT NULL
UNION ALL
SELECT fl.parent_user_id AS user_id, k.team_id, k.season_id
FROM family_links fl
JOIN kader_members km ON km.member_id = fl.member_id
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL
UNION ALL
SELECT m.user_id, k.team_id, k.season_id
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
JOIN members m ON m.id = kt.member_id
WHERE k.team_id IS NOT NULL
UNION ALL
SELECT m.user_id, k.team_id, k.season_id
FROM kader_extended_members kem
JOIN kader k ON k.id = kem.kader_id
JOIN members m ON m.id = kem.member_id
WHERE k.team_id IS NOT NULL
UNION ALL
SELECT fl.parent_user_id AS user_id, k.team_id, k.season_id
FROM family_links fl
JOIN kader_extended_members kem ON kem.member_id = fl.member_id
JOIN kader k ON k.id = kem.kader_id
WHERE k.team_id IS NOT NULL;
