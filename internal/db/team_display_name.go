package db

// TeamDisplayName returns a SQL expression that computes the canonical team display
// name using the active season's kader. teamAlias is the SQL alias for the teams table.
// Returns NULL (→ use COALESCE with t.name) if the team has no kader entry.
//
// Display name format:
//   - Single team of this age_class+gender: "<age_class> <gender_label>"
//   - Multiple teams:                       "<age_class> <team_number> <gender_label>"
func TeamDisplayName(teamAlias string) string {
	a := teamAlias
	return `(
		SELECT CASE
			WHEN (SELECT COUNT(*) FROM kader k_cnt
			      WHERE k_cnt.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
			        AND k_cnt.age_class = k_dn.age_class AND k_cnt.gender = k_dn.gender) > 1
			THEN k_dn.age_class || ' ' || CAST(k_dn.team_number AS TEXT) || ' ' ||
			     CASE k_dn.gender WHEN 'm' THEN 'männlich' WHEN 'f' THEN 'weiblich' ELSE 'gemischt' END
			ELSE k_dn.age_class || ' ' ||
			     CASE k_dn.gender WHEN 'm' THEN 'männlich' WHEN 'f' THEN 'weiblich' ELSE 'gemischt' END
		END
		FROM kader k_dn
		WHERE k_dn.team_id = ` + a + `.id
		  AND k_dn.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
		LIMIT 1
	)`
}
