package db

// TeamDisplayShort returns a SQL expression that computes the canonical team
// short name used across listing UIs (Kalender-Tile, Termine, Mitfahrten, …).
// teamAlias is the SQL alias for the teams table.
// Returns NULL (→ use COALESCE with t.name) if the team has no kader entry
// in the active season.
//
// Format mirrors web/src/lib/teamName.ts → buildTeamShortNames:
//
//	gender → m/w/g (m→m, f→w, anything else→g)
//	+ first character of age_class
//	+ team_number iff multiple teams share age_class+gender in active season
//
// Examples: "mA", "mA1", "wB2", "gE".
func TeamDisplayShort(teamAlias string) string {
	a := teamAlias
	return `(
		SELECT
			(CASE k_dn.gender WHEN 'm' THEN 'm' WHEN 'f' THEN 'w' ELSE 'g' END)
			|| SUBSTR(k_dn.age_class, 1, 1)
			|| CASE
				WHEN (SELECT COUNT(*) FROM kader k_cnt
				      WHERE k_cnt.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
				        AND k_cnt.age_class = k_dn.age_class
				        AND k_cnt.gender = k_dn.gender) > 1
				THEN CAST(k_dn.team_number AS TEXT)
				ELSE ''
			END
		FROM kader k_dn
		WHERE k_dn.team_id = ` + a + `.id
		  AND k_dn.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
		LIMIT 1
	)`
}
