package policy

import (
	"database/sql"
	"slices"
	"strconv"
	"strings"
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

// ownsAnyOf reports whether userID created any folder in path. One query over the
// already-computed path — the caller must not loop.
func ownsAnyOf(db *sql.DB, userID int, path []int) (bool, error) {
	if len(path) == 0 {
		return false, nil
	}
	args := make([]any, 0, len(path)+1)
	args = append(args, userID)
	for _, id := range path {
		args = append(args, id)
	}
	query := `SELECT 1 FROM file_folders WHERE created_by = ? AND id IN (?` +
		strings.Repeat(",?", len(path)-1) + `) LIMIT 1`

	var one int
	err := db.QueryRow(query, args...).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// principalCtx lazily resolves the expensive parts of a principal's identity.
//
// FolderAccess runs once per folder inside listing loops, so loading family and
// team membership unconditionally would cost four extra queries per folder. Each
// getter loads on first use and caches; `everyone` and `role` entries never
// trigger a query at all.
type principalCtx struct {
	db     *sql.DB
	userID int

	familyLoaded    bool
	linkedUserIDs   []int
	linkedFunctions []string

	teamsLoaded bool
	playerTeams []int // principal_type = 'team'
	parentTeams []int // principal_type = 'team_parents'
}

// family returns the user IDs and club functions of members linked to the
// principal via family_links, so parents inherit their children's ACL rights.
func (c *principalCtx) family() (linkedUserIDs []int, linkedFunctions []string) {
	if c.familyLoaded {
		return c.linkedUserIDs, c.linkedFunctions
	}
	c.familyLoaded = true

	rows, err := c.db.Query(`
		SELECT COALESCE(m.user_id, 0), COALESCE(mcf.function, '')
		  FROM family_links fl
		  JOIN members m ON m.id = fl.member_id
		  LEFT JOIN member_club_functions mcf ON mcf.member_id = m.id
		 WHERE fl.parent_user_id = ?`, c.userID)
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
		if uid != 0 && !slices.Contains(c.linkedUserIDs, uid) {
			c.linkedUserIDs = append(c.linkedUserIDs, uid)
		}
		if fn != "" && !slices.Contains(c.linkedFunctions, fn) {
			c.linkedFunctions = append(c.linkedFunctions, fn)
		}
	}
	return c.linkedUserIDs, c.linkedFunctions
}

// playerTeamsQuery lists teams whose active-season squad contains the principal
// as player, extended-squad member or trainer.
const playerTeamsQuery = `
	SELECT DISTINCT k.team_id
	  FROM kader k
	  JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
	 WHERE k.team_id IS NOT NULL
	   AND EXISTS (
	     SELECT 1 FROM members m WHERE m.user_id = ? AND (
	          EXISTS (SELECT 1 FROM kader_members          km  WHERE km.kader_id  = k.id AND km.member_id  = m.id)
	       OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = m.id)
	       OR EXISTS (SELECT 1 FROM kader_trainers         kt  WHERE kt.kader_id  = k.id AND kt.member_id  = m.id)
	     ))`

// parentTeamsQuery lists teams whose active-season squad contains a child of the
// principal as player or extended-squad member. Trainer links do not count —
// "parents of a trainer" is not a meaningful group.
const parentTeamsQuery = `
	SELECT DISTINCT k.team_id
	  FROM family_links fl
	  JOIN kader k ON k.team_id IS NOT NULL
	  JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
	 WHERE fl.parent_user_id = ?
	   AND (   EXISTS (SELECT 1 FROM kader_members          km  WHERE km.kader_id  = k.id AND km.member_id  = fl.member_id)
	        OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = fl.member_id))`

// teams returns the team IDs matching `team` and `team_parents` ACL entries.
// Resolution happens per request against the active season — nothing is frozen at
// grant time, so squad and season changes take effect immediately. Without an
// active season both sets stay empty (fail closed).
func (c *principalCtx) teams() (playerTeams, parentTeams []int) {
	if c.teamsLoaded {
		return c.playerTeams, c.parentTeams
	}
	c.teamsLoaded = true
	c.playerTeams = c.queryTeamIDs(playerTeamsQuery)
	c.parentTeams = c.queryTeamIDs(parentTeamsQuery)
	return c.playerTeams, c.parentTeams
}

func (c *principalCtx) queryTeamIDs(query string) []int {
	rows, err := c.db.Query(query, c.userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// matchesTeamRef reports whether principalRef (a teams.id as text) is in teamIDs.
func matchesTeamRef(principalRef sql.NullString, teamIDs []int) bool {
	if !principalRef.Valid {
		return false
	}
	id, err := strconv.Atoi(principalRef.String)
	if err != nil {
		return false
	}
	return slices.Contains(teamIDs, id)
}

// FolderAccess returns the effective read/write access for the principal on folderID.
//
// Owner precedence: whoever created the folder — or any of its ancestors — always
// holds read and write on it. The check runs before the path walk and short-circuits
// it, so it stays in effect no matter where the walk would stop. Without it, the
// first ACL entry on a fresh folder cuts inheritance and locks its own creator out
// (they can no longer even open the permission dialog, which requires can_write).
// The right is absolute: it survives losing a club function and is only revoked by
// changing file_folders.created_by.
//
// Otherwise nearest-ancestor-wins: the closest folder in the path with explicit
// permissions is authoritative; ancestors beyond that point are ignored.
// Parent users inherit the club_function and user-ID rights of their linked children.
func FolderAccess(db *sql.DB, p *Principal, folderID int) (canRead, canWrite bool, err error) {
	if p.Role == "admin" {
		return true, true, nil
	}

	path, err := folderPath(db, folderID)
	if err != nil {
		return false, false, err
	}

	owns, err := ownsAnyOf(db, p.UserID, path)
	if err != nil {
		return false, false, err
	}
	if owns {
		return true, true, nil
	}

	ctx := &principalCtx{db: db, userID: p.UserID}
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
				_, linkedFunctions := ctx.family()
				matches = pr.Valid && (slices.Contains(p.ClubFunctions, pr.String) || slices.Contains(linkedFunctions, pr.String))
			case "user":
				if pr.Valid && pr.String == userIDStr {
					matches = true
				} else if pr.Valid {
					if uid, parseErr := strconv.Atoi(pr.String); parseErr == nil {
						linkedUserIDs, _ := ctx.family()
						matches = slices.Contains(linkedUserIDs, uid)
					}
				}
			case "team":
				playerTeams, _ := ctx.teams()
				matches = matchesTeamRef(pr, playerTeams)
			case "team_parents":
				_, parentTeams := ctx.teams()
				matches = matchesTeamRef(pr, parentTeams)
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
