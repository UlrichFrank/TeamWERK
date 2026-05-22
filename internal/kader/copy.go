package kader

import (
	"context"
	"database/sql"
	"fmt"
)

// CopyAssignment describes how to populate one Kader in the target season.
type CopyAssignment struct {
	AgeClass     string `json:"age_class"`
	Gender       string `json:"gender"`
	MemberSource string `json:"member_source"` // empty | same-age-previous | age-before-previous | auto-assign
}

type createdKader struct {
	ID           int    `json:"id"`
	AgeClass     string `json:"age_class"`
	Gender       string `json:"gender"`
	MemberCount  int    `json:"member_count"`
}

// ageClassBelow returns the next younger age class (one level down).
// A-Jugend→B-Jugend, B-Jugend→C-Jugend, C-Jugend→D-Jugend, D-Jugend→""
func ageClassBelow(ac string) string {
	switch ac {
	case "A-Jugend":
		return "B-Jugend"
	case "B-Jugend":
		return "C-Jugend"
	case "C-Jugend":
		return "D-Jugend"
	default:
		return ""
	}
}

func copyKader(ctx context.Context, db *sql.DB, fromSeasonID, toSeasonID, targetStartYear int, assignments []CopyAssignment) ([]createdKader, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Load source kader: map[ageClass|gender] → kader_id
	srcRows, err := tx.QueryContext(ctx,
		`SELECT id, age_class, gender FROM kader WHERE season_id=?`, fromSeasonID)
	if err != nil {
		return nil, err
	}
	type srcKader struct{ id int; ageClass, gender string }
	sourceMap := map[string]int{} // "A-Jugend|m" → kader_id
	var srcList []srcKader
	for srcRows.Next() {
		var k srcKader
		srcRows.Scan(&k.id, &k.ageClass, &k.gender)
		sourceMap[k.ageClass+"|"+k.gender] = k.id
		srcList = append(srcList, k)
	}
	srcRows.Close()

	// If no explicit assignments, copy all source kader with same-age-previous default
	if len(assignments) == 0 {
		for _, k := range srcList {
			assignments = append(assignments, CopyAssignment{
				AgeClass:     k.ageClass,
				Gender:       k.gender,
				MemberSource: "same-age-previous",
			})
		}
	}

	// Compute target bracket for age classes
	targetBrackets := ComputeAgeBrackets(targetStartYear)

	var created []createdKader
	for _, a := range assignments {
		teamID, err := ensureTeam(ctx, tx, a.AgeClass, a.Gender, 1)
		if err != nil {
			return nil, fmt.Errorf("ensure team %s/%s: %w", a.AgeClass, a.Gender, err)
		}
		// Insert kader for target season with team_number=1, dedicated_birth_year=NULL (mixed mode)
		res, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO kader (season_id, age_class, gender, team_number, team_id) VALUES (?,?,?,1,?)`,
			toSeasonID, a.AgeClass, a.Gender, teamID)
		if err != nil {
			return nil, fmt.Errorf("insert kader %s/%s: %w", a.AgeClass, a.Gender, err)
		}
		newKaderID, _ := res.LastInsertId()
		if newKaderID == 0 {
			// Already exists — fetch its id and ensure team_id is set
			tx.QueryRowContext(ctx,
				`SELECT id FROM kader WHERE season_id=? AND age_class=? AND gender=? AND team_number=1`,
				toSeasonID, a.AgeClass, a.Gender).Scan(&newKaderID)
			tx.ExecContext(ctx,
				`UPDATE kader SET team_id=? WHERE id=? AND team_id IS NULL`,
				teamID, newKaderID)
		}

		memberCount := 0
		bracket, hasBracket := targetBrackets[a.AgeClass]

		switch a.MemberSource {
		case "", "smart-copy":
			// Smart-copy: combine same-age and one-level-below sources, filtered by target bracket
			if hasBracket {
				// Source 1: same age class
				srcID, ok := sourceMap[a.AgeClass+"|"+a.Gender]
				if ok {
					count, err := copyMembersFromKader(ctx, tx, srcID, int(newKaderID), bracket[0], bracket[1])
					if err != nil {
						return nil, err
					}
					memberCount = count
				}

				// Source 2: one level below (younger class, where older cohort comes from)
				youngerClass := ageClassBelow(a.AgeClass)
				if youngerClass != "" {
					srcID, ok := sourceMap[youngerClass+"|"+a.Gender]
					if ok {
						count, err := copyMembersFromKader(ctx, tx, srcID, int(newKaderID), bracket[0], bracket[1])
						if err != nil {
							return nil, err
						}
						memberCount = count
					}
				}
			}
		case "same-age-previous":
			// Legacy: same age only, filtered by target bracket
			if hasBracket {
				srcID, ok := sourceMap[a.AgeClass+"|"+a.Gender]
				if ok {
					count, err := copyMembersFromKader(ctx, tx, srcID, int(newKaderID), bracket[0], bracket[1])
					if err != nil {
						return nil, err
					}
					memberCount = count
				}
			}
		case "age-before-previous":
			// Legacy: one level below only, filtered by target bracket
			if hasBracket {
				olderClass := ageClassBelow(a.AgeClass)
				if olderClass != "" {
					srcID, ok := sourceMap[olderClass+"|"+a.Gender]
					if ok {
						count, err := copyMembersFromKader(ctx, tx, srcID, int(newKaderID), bracket[0], bracket[1])
						if err != nil {
							return nil, err
						}
						memberCount = count
					}
				}
			}
		case "auto-assign":
			memberCount, err = autoAssignMembers(ctx, tx, int(newKaderID), a.AgeClass, a.Gender, targetStartYear, nil)
			if err != nil {
				return nil, err
			}
		// "empty" or unrecognized: no members
		}

		created = append(created, createdKader{
			ID:          int(newKaderID),
			AgeClass:    a.AgeClass,
			Gender:      a.Gender,
			MemberCount: memberCount,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

func copyMembersFromKader(ctx context.Context, tx *sql.Tx, fromKaderID, toKaderID, bracketMin, bracketMax int) (int, error) {
	_, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO kader_members (kader_id, member_id)
		 SELECT ?, km.member_id FROM kader_members km
		 JOIN members m ON m.id = km.member_id
		 WHERE km.kader_id=? AND CAST(strftime('%Y', m.date_of_birth) AS INTEGER) BETWEEN ? AND ?`,
		toKaderID, fromKaderID, bracketMin, bracketMax)
	if err != nil {
		return 0, err
	}
	var count int
	tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM kader_members WHERE kader_id=?`, toKaderID).Scan(&count)
	return count, nil
}

func autoAssignMembers(ctx context.Context, tx *sql.Tx, kaderID int, ageClass, gender string, seasonStartYear int, dedicatedBirthYear *int) (int, error) {
	var yearFilter string
	var args []any
	args = append(args, kaderID)

	if dedicatedBirthYear != nil {
		yearFilter = " AND CAST(strftime('%Y', m.date_of_birth) AS INTEGER) = ?"
		args = append(args, *dedicatedBirthYear)
	} else {
		brackets := ComputeAgeBrackets(seasonStartYear)
		bracket, ok := brackets[ageClass]
		if !ok {
			return 0, nil
		}
		yearFilter = " AND CAST(strftime('%Y', m.date_of_birth) AS INTEGER) BETWEEN ? AND ?"
		args = append(args, bracket[0], bracket[1])
	}

	genderFilter := ""
	if gender != "mixed" {
		genderFilter = " AND (m.gender=? OR m.gender='u')"
		args = append(args, gender)
	}

	_, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO kader_members (kader_id, member_id)
		 SELECT ?, m.id FROM members m
		 WHERE m.status != 'ausgetreten'`+yearFilter+genderFilter,
		args...)
	if err != nil {
		return 0, err
	}

	var count int
	tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM kader_members WHERE kader_id=?`, kaderID).Scan(&count)
	return count, nil
}
