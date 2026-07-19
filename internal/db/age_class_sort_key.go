package db

// AgeClassSortKey returns a SQL expression that yields the canonical ordering
// key for an age-class column. col is the SQL reference to the age-class column
// (e.g. "k.age_class" or "t.age_class").
//
// The key prefixes the raw age-class string with a category block so that
// ORDER BY groups the two kinds of kader in a fixed order without touching the
// A–D-Jugend ordering:
//
//	Block '0' = non-training-group classes (*-Jugend): sorted alphabetically →
//	            A-Jugend, B-Jugend, C-Jugend, D-Jugend stay as before.
//	Block '1' = training-group categories: ordered by
//	            training_group_categories.sort_order (Perspektivkader before
//	            Förderkader), NOT alphabetically (which would put "Förderkader"
//	            before "Perspektivkader"). sort_order is the single source of
//	            truth for this ordering.
//
// Mirrors design.md Entscheidung 7 and the TS twin compareAgeClass in
// web/src/lib/teamName.ts. Use as the primary ORDER BY term, keeping
// gender/team_number as secondary criteria:
//
//	ORDER BY <AgeClassSortKey("k.age_class")>, k.gender, k.team_number
func AgeClassSortKey(col string) string {
	return `(CASE WHEN ` + col + ` IN (SELECT name FROM training_group_categories)
	              THEN '1' || printf('%04d', (SELECT sort_order FROM training_group_categories WHERE name = ` + col + `))
	              ELSE '0' END) || ` + col
}
