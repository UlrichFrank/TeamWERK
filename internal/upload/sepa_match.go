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

// matchMemberByFilename returns all member IDs whose first_name+last_name
// normalizes to basename. Beyond the exact full-name match, the first token of
// first_name is also tried as a fallback so that members with a stored second
// given name (e.g. "Luca Marco" Buric) still match a "BuricLuca.pdf". Length
// 0 = no match, 1 = unique match, >1 = ambiguous.
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
		if memberMatchesTarget(first, last, target) {
			matches = append(matches, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}

// memberMatchesTarget returns true if the given member's name (in any of four
// normalized variants — full and first-token-only, each in both orders) equals
// the already-normalized target.
func memberMatchesTarget(first, last, target string) bool {
	if normalizeName(first+last) == target || normalizeName(last+first) == target {
		return true
	}
	firstToken := firstWord(first)
	if firstToken == "" || firstToken == first {
		return false
	}
	return normalizeName(firstToken+last) == target || normalizeName(last+firstToken) == target
}

func firstWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
