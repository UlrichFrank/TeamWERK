package teams

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

type Team struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	DisplayShort string `json:"display_short"`
	DisplayLong  string `json:"display_long"`
	IsExtended   bool   `json:"isExtended"`
}

type TrainerEntry struct {
	UserID int    `json:"userId"`
	Name   string `json:"name"`
}

type PlayerEntry struct {
	UserID           int                        `json:"userId"`
	MemberID         int                        `json:"memberId"`
	Name             string                     `json:"name"`
	JerseyNumber     *int                       `json:"jerseyNumber"`
	Responsibilities []ResponsibilityAssignment `json:"responsibilities"`
}

type ParentEntry struct {
	UserID   int      `json:"userId"`
	Name     string   `json:"name"`
	Children []string `json:"children"`
}

type RosterResponse struct {
	Team            Team           `json:"team"`
	Trainers        []TrainerEntry `json:"trainers"`
	Players         []PlayerEntry  `json:"players"`
	Parents         []ParentEntry  `json:"parents"`
	ExtendedPlayers []PlayerEntry  `json:"extended_players"`
	ExtendedParents []ParentEntry  `json:"extended_parents"`
	CanManage       bool           `json:"canManage"`
}

// GET /api/teams/:id/roster
func (h *Handler) GetRoster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)
	userID := claims.UserID

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	// Check access: user must have access to this team in the active season
	var hasAccess int
	err = h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_accessible_teams uat
		JOIN seasons s ON s.id = uat.season_id
		WHERE uat.user_id = ? AND uat.team_id = ? AND s.is_active = 1`,
		userID, teamID,
	).Scan(&hasAccess)
	if err != nil || hasAccess == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Get team name + display variants
	var team Team
	team.ID = teamID
	if err := h.db.QueryRowContext(ctx,
		`SELECT t.name,
		        COALESCE(`+appdb.TeamDisplayShort("t")+`, t.name) AS display_short,
		        COALESCE(`+appdb.TeamDisplayName("t")+`, t.name) AS display_long
		 FROM teams t WHERE t.id = ?`, teamID).Scan(&team.Name, &team.DisplayShort, &team.DisplayLong); err != nil {
		http.Error(w, "team not found", http.StatusNotFound)
		return
	}

	resp := RosterResponse{
		Team:            team,
		Trainers:        []TrainerEntry{},
		Players:         []PlayerEntry{},
		Parents:         []ParentEntry{},
		ExtendedPlayers: []PlayerEntry{},
		ExtendedParents: []ParentEntry{},
	}

	// canManage: der anfragende Nutzer ist Trainer/Admin dieses Teams (aktive Saison).
	if canManage, mErr := h.isTrainerOfTeam(ctx, claims, teamID); mErr == nil {
		resp.CanManage = canManage
	}

	// Active season ID
	var seasonID int
	h.db.QueryRowContext(ctx, `SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`).Scan(&seasonID)

	// Trainers: kader_trainers → members → users
	trainerRows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT COALESCE(u.id, 0), m.first_name || ' ' || m.last_name
		FROM kader_trainers kt
		JOIN kader k ON k.id = kt.kader_id
		JOIN members m ON m.id = kt.member_id
		LEFT JOIN users u ON u.id = m.user_id
		WHERE k.team_id = ? AND k.season_id = ?
		ORDER BY m.first_name, m.last_name`, teamID, seasonID)
	if err == nil {
		defer trainerRows.Close()
		for trainerRows.Next() {
			var t TrainerEntry
			trainerRows.Scan(&t.UserID, &t.Name)
			resp.Trainers = append(resp.Trainers, t)
		}
	}

	// Players: kader_members → members → users
	playerRows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT m.id, COALESCE(u.id, 0), m.first_name || ' ' || m.last_name,
		       m.jersey_number
		FROM kader_members km
		JOIN kader k ON k.id = km.kader_id
		JOIN members m ON m.id = km.member_id
		LEFT JOIN users u ON u.id = m.user_id
		WHERE k.team_id = ? AND k.season_id = ?
		ORDER BY m.first_name, m.last_name`, teamID, seasonID)
	if err == nil {
		defer playerRows.Close()
		for playerRows.Next() {
			var p PlayerEntry
			var jerseyNum sql.NullInt64
			playerRows.Scan(&p.MemberID, &p.UserID, &p.Name, &jerseyNum)
			if jerseyNum.Valid {
				n := int(jerseyNum.Int64)
				p.JerseyNumber = &n
			}
			p.Responsibilities = h.responsibilitiesFor(ctx, teamID, seasonID, p.MemberID)
			resp.Players = append(resp.Players, p)
		}
	}

	// Extended players: kader_extended_members → members → users
	extRows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT m.id, COALESCE(u.id, 0), m.first_name || ' ' || m.last_name,
		       m.jersey_number
		FROM kader_extended_members kem
		JOIN kader k ON k.id = kem.kader_id
		JOIN members m ON m.id = kem.member_id
		LEFT JOIN users u ON u.id = m.user_id
		WHERE k.team_id = ? AND k.season_id = ?
		ORDER BY m.first_name, m.last_name`, teamID, seasonID)
	if err == nil {
		defer extRows.Close()
		for extRows.Next() {
			var p PlayerEntry
			var jerseyNum sql.NullInt64
			extRows.Scan(&p.MemberID, &p.UserID, &p.Name, &jerseyNum)
			if jerseyNum.Valid {
				n := int(jerseyNum.Int64)
				p.JerseyNumber = &n
			}
			p.Responsibilities = h.responsibilitiesFor(ctx, teamID, seasonID, p.MemberID)
			resp.ExtendedPlayers = append(resp.ExtendedPlayers, p)
		}
	}

	// Parents: family_links where a child is in the regular OR extended kader for
	// this team+season. hasRegular = the parent has at least one child in the
	// regular kader; such parents belong to resp.Parents. Parents whose children
	// are exclusively in the extended kader go to resp.ExtendedParents (mirrors
	// the Players/ExtendedPlayers split: regular membership wins).
	parentRows, err := h.db.QueryContext(ctx, `
		SELECT u.id, u.first_name || ' ' || u.last_name,
		       MAX(src.regular) AS has_regular
		FROM (
			SELECT fl.parent_user_id, 1 AS regular
			FROM family_links fl
			JOIN kader_members km ON km.member_id = fl.member_id
			JOIN kader k ON k.id = km.kader_id
			WHERE k.team_id = ? AND k.season_id = ?
			UNION ALL
			SELECT fl.parent_user_id, 0 AS regular
			FROM family_links fl
			JOIN kader_extended_members kem ON kem.member_id = fl.member_id
			JOIN kader k ON k.id = kem.kader_id
			WHERE k.team_id = ? AND k.season_id = ?
		) src
		JOIN users u ON u.id = src.parent_user_id
		GROUP BY u.id, u.first_name, u.last_name
		ORDER BY u.first_name, u.last_name`, teamID, seasonID, teamID, seasonID)
	if err == nil {
		defer parentRows.Close()
		for parentRows.Next() {
			var p ParentEntry
			var hasRegular int
			p.Children = []string{}
			parentRows.Scan(&p.UserID, &p.Name, &hasRegular)

			// Get children names in this team (regular + extended kader)
			childRows, err := h.db.QueryContext(ctx, `
				SELECT DISTINCT m.first_name || ' ' || m.last_name, m.first_name
				FROM family_links fl
				JOIN members m ON m.id = fl.member_id
				WHERE fl.parent_user_id = ?
				  AND EXISTS (
				      SELECT 1 FROM kader_members km
				      JOIN kader k ON k.id = km.kader_id
				      WHERE km.member_id = m.id AND k.team_id = ? AND k.season_id = ?
				      UNION ALL
				      SELECT 1 FROM kader_extended_members kem
				      JOIN kader k ON k.id = kem.kader_id
				      WHERE kem.member_id = m.id AND k.team_id = ? AND k.season_id = ?
				  )
				ORDER BY m.first_name`, p.UserID, teamID, seasonID, teamID, seasonID)
			if err == nil {
				for childRows.Next() {
					var childName, firstName string
					childRows.Scan(&childName, &firstName)
					p.Children = append(p.Children, childName)
				}
				childRows.Close()
			}
			if hasRegular == 1 {
				resp.Parents = append(resp.Parents, p)
			} else {
				resp.ExtendedParents = append(resp.ExtendedParents, p)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /api/teams/my — returns teams the user has access to in the active season,
// with isExtended=true when access comes exclusively via kader_extended_members.
func (h *Handler) ListMyTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)
	userID := claims.UserID

	// Stammkader, Trainer, Eltern → isExtended=false
	// Erweiterter Kader (ohne Stammzugang) → isExtended=true
	// UNION deduplicates: if a team appears in both selects, is_extended=0 wins.
	rows, err := h.db.QueryContext(ctx, `
		SELECT t.id, t.name, 0 AS is_extended
		FROM user_accessible_teams uat
		JOIN teams t ON t.id = uat.team_id
		JOIN seasons s ON s.id = uat.season_id
		WHERE uat.user_id = ? AND s.is_active = 1
		  AND (
		    EXISTS (SELECT 1 FROM kader_members km JOIN kader k ON k.id = km.kader_id
		            JOIN members m ON m.id = km.member_id
		            WHERE m.user_id = ? AND k.team_id = t.id AND k.season_id = s.id)
		    OR EXISTS (SELECT 1 FROM kader_trainers kt JOIN kader k ON k.id = kt.kader_id
		              JOIN members m ON m.id = kt.member_id
		              WHERE m.user_id = ? AND k.team_id = t.id AND k.season_id = s.id)
		    OR EXISTS (SELECT 1 FROM family_links fl JOIN kader_members km2 ON km2.member_id = fl.member_id
		              JOIN kader k ON k.id = km2.kader_id
		              WHERE fl.parent_user_id = ? AND k.team_id = t.id AND k.season_id = s.id)
		    OR EXISTS (SELECT 1 FROM family_links fl JOIN kader_extended_members kem ON kem.member_id = fl.member_id
		              JOIN kader k ON k.id = kem.kader_id
		              WHERE fl.parent_user_id = ? AND k.team_id = t.id AND k.season_id = s.id)
		  )
		UNION
		SELECT t.id, t.name, 1 AS is_extended
		FROM kader_extended_members kem
		JOIN kader k ON k.id = kem.kader_id
		JOIN teams t ON t.id = k.team_id
		JOIN seasons s ON s.id = k.season_id
		JOIN members m ON m.id = kem.member_id
		WHERE m.user_id = ? AND s.is_active = 1
		  AND NOT EXISTS (
		    SELECT 1 FROM kader_members km JOIN kader k2 ON k2.id = km.kader_id
		    JOIN members m2 ON m2.id = km.member_id
		    WHERE m2.user_id = ? AND k2.team_id = t.id AND k2.season_id = s.id
		  )
		ORDER BY t.name`,
		userID, userID, userID, userID, userID, userID, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	teams := []Team{}
	for rows.Next() {
		var t Team
		var isExtended int
		rows.Scan(&t.ID, &t.Name, &isExtended)
		t.IsExtended = isExtended == 1
		teams = append(teams, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}
