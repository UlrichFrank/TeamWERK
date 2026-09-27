package db

// TeamDisplayName returns a SQL expression that computes the canonical team display
// name using the active season's kader. teamAlias is the SQL alias for the teams table.
// Returns NULL (→ use COALESCE with t.name) if the team has no kader entry.
//
// Display name format:
//   - Single team of this age_class+gender: "<age_class> <gender_label>"
//   - Multiple teams:                       "<age_class> <gender_label> <team_number>"
//
// Die Nummer steht am Ende ("C-Jugend männlich 1") — dieselbe Reihenfolge wie
// der gespeicherte teams.name ("C-Jugend männlich 2"), damit eingeklappte und
// aufgeklappte Anzeigen nicht auseinanderlaufen.
func TeamDisplayName(teamAlias string) string {
	a := teamAlias
	return `(
		SELECT CASE
			WHEN (SELECT COUNT(*) FROM kader k_cnt
			      WHERE k_cnt.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
			        AND k_cnt.age_class = k_dn.age_class AND k_cnt.gender = k_dn.gender) > 1
			THEN k_dn.age_class || ' ' ||
			     CASE k_dn.gender WHEN 'm' THEN 'männlich' WHEN 'f' THEN 'weiblich' ELSE 'gemischt' END ||
			     ' ' || CAST(k_dn.team_number AS TEXT)
			ELSE k_dn.age_class || ' ' ||
			     CASE k_dn.gender WHEN 'm' THEN 'männlich' WHEN 'f' THEN 'weiblich' ELSE 'gemischt' END
		END
		FROM kader k_dn
		WHERE k_dn.team_id = ` + a + `.id
		  AND k_dn.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
		LIMIT 1
	)`
}

// TeamLongName ist der Anzeigename eines Teams für jede Ausgabe an Menschen
// (API-Felder, Push-Texte, Exporte, Kalender): die Langform aus
// TeamDisplayName, mit dem gespeicherten teams.name als Fallback für Teams ohne
// Kader in der aktiven Saison. Der gespeicherte Name selbst ist
// Identitätsschlüssel (kader.ensureTeam, H4A-Abgleich) und kein Anzeigewert —
// bei Mannschaft 1 fehlt ihm die Nummer, sobald es eine zweite gibt.
func TeamLongName(teamAlias string) string {
	return `COALESCE(` + TeamDisplayName(teamAlias) + `, ` + teamAlias + `.name)`
}
