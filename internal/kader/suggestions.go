package kader

import (
	"context"
	"database/sql"
)

type Suggestion struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	BirthYear      int    `json:"birth_year"`
	Gender         string `json:"gender"`
	Reason         string `json:"reason"`
	AlreadyInKader bool   `json:"already_in_kader"`
}

func suggestMembers(ctx context.Context, db *sql.DB, kaderID int, ageClass, gender string, seasonStartYear int, search string, filterByBracket bool) ([]Suggestion, error) {
	brackets := ComputeAgeBrackets(seasonStartYear)
	bracket := brackets[ageClass]

	// Build gender filter: mixed kader accepts all genders
	var genderFilter string
	var args []any

	query := `SELECT m.id,
	                 m.first_name || ' ' || m.last_name,
	                 COALESCE(CAST(strftime('%Y', m.date_of_birth) AS INTEGER), 0),
	                 m.gender,
	                 EXISTS(SELECT 1 FROM kader_members km WHERE km.kader_id=? AND km.member_id=m.id) AS in_kader
	          FROM members m
	          WHERE m.status != 'ausgetreten'`
	args = append(args, kaderID)

	if gender != "mixed" {
		// Include members matching the gender or with unspecified gender ('u')
		genderFilter = " AND (m.gender=? OR m.gender='u')"
		args = append(args, gender)
	}
	query += genderFilter

	if search != "" {
		query += ` AND (m.first_name || ' ' || m.last_name) LIKE ?`
		args = append(args, "%"+search+"%")
	}

	if filterByBracket {
		query += ` AND CAST(strftime('%Y', m.date_of_birth) AS INTEGER) BETWEEN ? AND ?`
		args = append(args, bracket[0], bracket[1])
	}

	query += ` ORDER BY m.last_name, m.first_name LIMIT 20`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []Suggestion{}
	for rows.Next() {
		var s Suggestion
		var inKader int
		rows.Scan(&s.ID, &s.Name, &s.BirthYear, &s.Gender, &inKader)
		s.AlreadyInKader = inKader == 1
		if filterByBracket {
			s.Reason = "Passender Jahrgang " + ageClass
		}
		result = append(result, s)
	}
	return result, nil
}
