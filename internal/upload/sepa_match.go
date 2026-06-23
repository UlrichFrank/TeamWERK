package upload

import (
	"context"
	"database/sql"
	"strings"
)

// normalizeName collapses a name (or PDF basename) to a comparison-safe form:
// lowercase, German umlaut substitution, and stripped of whitespace/punctuation.
func normalizeName(s string) string {
	s = strings.ToLower(s)

	replacer := strings.NewReplacer(
		"ä", "ae",
		"ö", "oe",
		"ü", "ue",
		"ß", "ss",
	)
	s = replacer.Replace(s)

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '-', '_', '.', '\'', '`', '’':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// matchMemberByFilename returns all member IDs whose first_name+last_name (in
// either order) normalize to the same string as basename. Length 0 = no match,
// 1 = unique match, >1 = ambiguous.
func matchMemberByFilename(ctx context.Context, db *sql.DB, basename string) ([]int, error) {
	target := normalizeName(basename)
	if target == "" {
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, `SELECT id, first_name, last_name FROM members`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []int
	for rows.Next() {
		var id int
		var first, last string
		if err := rows.Scan(&id, &first, &last); err != nil {
			return nil, err
		}
		fn := normalizeName(first + last)
		ln := normalizeName(last + first)
		if fn == target || ln == target {
			matches = append(matches, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}
