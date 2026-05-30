CREATE VIEW user_accessible_teams AS
-- spieler: direkt im Kader eingetragen
SELECT m.user_id, k.team_id, k.season_id
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
JOIN members m ON m.id = km.member_id
WHERE k.team_id IS NOT NULL

UNION ALL

-- elternteil: über family_links mit Kind verbunden
SELECT fl.parent_user_id AS user_id, k.team_id, k.season_id
FROM family_links fl
JOIN kader_members km ON km.member_id = fl.member_id
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL

UNION ALL

-- trainer: über kader_trainers eingetragen
SELECT m.user_id, k.team_id, k.season_id
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
JOIN members m ON m.id = kt.member_id
WHERE k.team_id IS NOT NULL;
