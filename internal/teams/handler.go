package teams

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

type Team struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TrainerEntry struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type PlayerEntry struct {
	Name         string `json:"name"`
	JerseyNumber *int   `json:"jerseyNumber"`
	Status       string `json:"status"`
	Email        string `json:"email"`
}

type ParentEntry struct {
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Children []string `json:"children"`
}

type RosterResponse struct {
	Team     Team          `json:"team"`
	Trainers []TrainerEntry `json:"trainers"`
	Players  []PlayerEntry  `json:"players"`
	Parents  []ParentEntry  `json:"parents"`
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

	// Get team name
	var team Team
	team.ID = teamID
	if err := h.db.QueryRowContext(ctx, `SELECT name FROM teams WHERE id = ?`, teamID).Scan(&team.Name); err != nil {
		http.Error(w, "team not found", http.StatusNotFound)
		return
	}

	resp := RosterResponse{
		Team:     team,
		Trainers: []TrainerEntry{},
		Players:  []PlayerEntry{},
		Parents:  []ParentEntry{},
	}

	// Active season ID
	var seasonID int
	h.db.QueryRowContext(ctx, `SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`).Scan(&seasonID)

	// Trainers: kader_trainers → members → users
	trainerRows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT m.first_name || ' ' || m.last_name, COALESCE(u.email, '')
		FROM kader_trainers kt
		JOIN kader k ON k.id = kt.kader_id
		JOIN members m ON m.id = kt.member_id
		LEFT JOIN users u ON u.id = m.user_id
		WHERE k.team_id = ? AND k.season_id = ?
		ORDER BY m.last_name, m.first_name`, teamID, seasonID)
	if err == nil {
		defer trainerRows.Close()
		for trainerRows.Next() {
			var t TrainerEntry
			trainerRows.Scan(&t.Name, &t.Email)
			resp.Trainers = append(resp.Trainers, t)
		}
	}

	// Players: kader_members → members → users
	playerRows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT m.first_name || ' ' || m.last_name,
		       m.jersey_number,
		       m.status,
		       COALESCE(u.email, '')
		FROM kader_members km
		JOIN kader k ON k.id = km.kader_id
		JOIN members m ON m.id = km.member_id
		LEFT JOIN users u ON u.id = m.user_id
		WHERE k.team_id = ? AND k.season_id = ?
		ORDER BY m.last_name, m.first_name`, teamID, seasonID)
	if err == nil {
		defer playerRows.Close()
		for playerRows.Next() {
			var p PlayerEntry
			var jerseyNum sql.NullInt64
			playerRows.Scan(&p.Name, &jerseyNum, &p.Status, &p.Email)
			if jerseyNum.Valid {
				n := int(jerseyNum.Int64)
				p.JerseyNumber = &n
			}
			resp.Players = append(resp.Players, p)
		}
	}

	// Parents: family_links where member is in kader for this team+season
	parentRows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT u.id, u.first_name || ' ' || u.last_name, u.email
		FROM family_links fl
		JOIN kader_members km ON km.member_id = fl.member_id
		JOIN kader k ON k.id = km.kader_id
		JOIN users u ON u.id = fl.parent_user_id
		WHERE k.team_id = ? AND k.season_id = ?
		ORDER BY u.last_name, u.first_name`, teamID, seasonID)
	if err == nil {
		defer parentRows.Close()
		for parentRows.Next() {
			var parentUserID int
			var p ParentEntry
			p.Children = []string{}
			parentRows.Scan(&parentUserID, &p.Name, &p.Email)

			// Get children names in this team
			childRows, err := h.db.QueryContext(ctx, `
				SELECT m.first_name || ' ' || m.last_name
				FROM family_links fl
				JOIN members m ON m.id = fl.member_id
				JOIN kader_members km ON km.member_id = m.id
				JOIN kader k ON k.id = km.kader_id
				WHERE fl.parent_user_id = ? AND k.team_id = ? AND k.season_id = ?
				ORDER BY m.first_name`, parentUserID, teamID, seasonID)
			if err == nil {
				for childRows.Next() {
					var childName string
					childRows.Scan(&childName)
					p.Children = append(p.Children, childName)
				}
				childRows.Close()
			}
			resp.Parents = append(resp.Parents, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /api/teams — returns teams the user has access to in the active season
func (h *Handler) ListMyTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	rows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT t.id, t.name
		FROM user_accessible_teams uat
		JOIN teams t ON t.id = uat.team_id
		JOIN seasons s ON s.id = uat.season_id
		WHERE uat.user_id = ? AND s.is_active = 1
		ORDER BY t.name`, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	teams := []Team{}
	for rows.Next() {
		var t Team
		rows.Scan(&t.ID, &t.Name)
		teams = append(teams, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}
