package policy

import (
	"database/sql"
	"slices"
	"strconv"
)

// folderPath returns [folderID, parentID, grandparentID, ...] up to the root.
func folderPath(db *sql.DB, folderID int) ([]int, error) {
	path := []int{}
	current := folderID
	for {
		path = append(path, current)
		var parentID sql.NullInt64
		err := db.QueryRow(`SELECT parent_id FROM file_folders WHERE id = ?`, current).Scan(&parentID)
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		if err != nil {
			return nil, err
		}
		if !parentID.Valid {
			break
		}
		current = int(parentID.Int64)
	}
	return path, nil
}

// fetchFamilyContext returns the user IDs and club functions of members linked
// to userID via family_links, so parents inherit their children's ACL rights.
func fetchFamilyContext(db *sql.DB, userID int) (linkedUserIDs []int, linkedFunctions []string) {
	rows, err := db.Query(`
		SELECT COALESCE(m.user_id, 0), COALESCE(mcf.function, '')
		  FROM family_links fl
		  JOIN members m ON m.id = fl.member_id
		  LEFT JOIN member_club_functions mcf ON mcf.member_id = m.id
		 WHERE fl.parent_user_id = ?`, userID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	for rows.Next() {
		var uid int
		var fn string
		if err := rows.Scan(&uid, &fn); err != nil {
			continue
		}
		if uid != 0 && !slices.Contains(linkedUserIDs, uid) {
			linkedUserIDs = append(linkedUserIDs, uid)
		}
		if fn != "" && !slices.Contains(linkedFunctions, fn) {
			linkedFunctions = append(linkedFunctions, fn)
		}
	}
	return linkedUserIDs, linkedFunctions
}

// FolderAccess returns the effective read/write access for the principal on folderID.
// Nearest-ancestor-wins: the closest folder in the path with explicit permissions is
// authoritative; ancestors beyond that point are ignored.
// Parent users inherit the club_function and user-ID rights of their linked children.
func FolderAccess(db *sql.DB, p *Principal, folderID int) (canRead, canWrite bool, err error) {
	if p.Role == "admin" {
		return true, true, nil
	}

	path, err := folderPath(db, folderID)
	if err != nil {
		return false, false, err
	}

	linkedUserIDs, linkedFunctions := fetchFamilyContext(db, p.UserID)
	userIDStr := strconv.Itoa(p.UserID)

	for _, id := range path {
		rows, err := db.Query(
			`SELECT principal_type, principal_ref, can_read, can_write
			   FROM folder_permissions WHERE folder_id = ?`, id)
		if err != nil {
			return false, false, err
		}

		var hasAny bool
		var cr, cw bool

		for rows.Next() {
			hasAny = true
			var pt, pr sql.NullString
			var r, w int
			if scanErr := rows.Scan(&pt, &pr, &r, &w); scanErr != nil {
				continue
			}
			matches := false
			switch pt.String {
			case "everyone":
				matches = true
			case "role":
				matches = pr.Valid && pr.String == p.Role
			case "club_function":
				matches = pr.Valid && (slices.Contains(p.ClubFunctions, pr.String) || slices.Contains(linkedFunctions, pr.String))
			case "user":
				if pr.Valid && pr.String == userIDStr {
					matches = true
				} else if pr.Valid {
					if uid, parseErr := strconv.Atoi(pr.String); parseErr == nil {
						matches = slices.Contains(linkedUserIDs, uid)
					}
				}
			}
			if matches {
				if r == 1 {
					cr = true
				}
				if w == 1 {
					cw = true
				}
			}
		}
		rows.Close()

		if hasAny {
			return cr, cw, nil
		}
	}

	return false, false, nil
}

// CanReadFolder returns true if the principal may read the given folder.
func CanReadFolder(db *sql.DB, p *Principal, folderID int) bool {
	canRead, _, _ := FolderAccess(db, p, folderID)
	return canRead
}
