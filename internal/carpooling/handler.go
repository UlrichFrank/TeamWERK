package carpooling

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

type GameEntry struct {
	ID        int    `json:"id"`
	Date      string `json:"date"`
	Opponent  string `json:"opponent"`
	Team      string `json:"team"`
	EventType string `json:"eventType"`
}

type CarpoolEntry struct {
	ID         int    `json:"id"`
	UserName   string `json:"userName"`
	Plaetze    *int   `json:"plaetze,omitempty"`
	Treffpunkt string `json:"treffpunkt,omitempty"`
	Notiz      string `json:"notiz,omitempty"`
	IsOwn      bool   `json:"isOwn"`
}

type CarpoolResponse struct {
	Game  GameEntry      `json:"game"`
	Biete []CarpoolEntry `json:"biete"`
	Suche []CarpoolEntry `json:"suche"`
}

type ListResponse struct {
	Games        []CarpoolResponse `json:"games"`
	VehicleSeats *int              `json:"vehicleSeats"`
}

// teamQueryForRole returns a subquery for team_ids the user belongs to,
// plus the bind args (userID, seasonID). Returns "" if role has no team filter (admin/vorstand).
func teamQueryForRole(role string) string {
	switch role {
	case "trainer":
		return `SELECT k.team_id
			FROM kader_trainers kt
			JOIN kader k ON k.id = kt.kader_id
			JOIN members m ON m.id = kt.member_id
			WHERE m.user_id = ? AND k.season_id = ? AND k.team_id IS NOT NULL`
	case "elternteil":
		return `SELECT tm.team_id
			FROM team_memberships tm
			JOIN family_links fl ON fl.member_id = tm.member_id
			WHERE fl.parent_user_id = ? AND tm.season_id = ?`
	case "spieler":
		return `SELECT tm.team_id
			FROM team_memberships tm
			JOIN members m ON m.id = tm.member_id
			WHERE m.user_id = ? AND tm.season_id = ?`
	}
	return ""
}

// GET /api/mitfahrgelegenheiten
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID
	role := claims.Role

	var seasonID int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`,
	).Scan(&seasonID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListResponse{Games: []CarpoolResponse{}})
		return
	}

	teamSub := teamQueryForRole(role)

	var (
		queryStr string
		args     []interface{}
	)
	if teamSub == "" {
		queryStr = `SELECT DISTINCT g.id, g.date, g.opponent, g.event_type, t.name
			FROM games g
			JOIN game_teams gt ON g.id = gt.game_id
			JOIN teams t ON t.id = gt.team_id
			WHERE DATE(g.date) >= DATE('now')
			ORDER BY g.date ASC, g.event_type ASC`
	} else {
		queryStr = fmt.Sprintf(`SELECT DISTINCT g.id, g.date, g.opponent, g.event_type, t.name
			FROM games g
			JOIN game_teams gt ON g.id = gt.game_id
			JOIN teams t ON t.id = gt.team_id
			WHERE DATE(g.date) >= DATE('now')
			  AND gt.team_id IN (%s)
			ORDER BY g.date ASC, g.event_type ASC`, teamSub)
		args = []interface{}{userID, seasonID}
	}

	rows, err := h.db.QueryContext(r.Context(), queryStr, args...)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []GameEntry
	for rows.Next() {
		var g GameEntry
		rows.Scan(&g.ID, &g.Date, &g.Opponent, &g.EventType, &g.Team)
		games = append(games, g)
	}

	result := make([]CarpoolResponse, 0, len(games))
	for _, g := range games {
		biete, suche := h.queryEntries(r, g.ID, userID)
		result = append(result, CarpoolResponse{
			Game:  g,
			Biete: biete,
			Suche: suche,
		})
	}

	vehicleSeats := h.queryVehicleSeats(r, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListResponse{
		Games:        result,
		VehicleSeats: vehicleSeats,
	})
}

func (h *Handler) queryVehicleSeats(r *http.Request, userID int) *int {
	var seats int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT seats FROM vehicle_info WHERE user_id = ?`, userID,
	).Scan(&seats); err != nil {
		return nil
	}
	return &seats
}

func (h *Handler) queryEntries(r *http.Request, gameID, currentUserID int) ([]CarpoolEntry, []CarpoolEntry) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT m.id, u.first_name || ' ' || u.last_name, m.typ, m.plaetze, COALESCE(m.treffpunkt,''), COALESCE(m.notiz,''), m.user_id
		FROM mitfahrgelegenheiten m
		JOIN users u ON u.id = m.user_id
		WHERE m.game_id = ?
		ORDER BY m.created_at ASC`, gameID)
	if err != nil {
		return []CarpoolEntry{}, []CarpoolEntry{}
	}
	defer rows.Close()

	var biete, suche []CarpoolEntry
	for rows.Next() {
		var e CarpoolEntry
		var typ string
		var plaetze sql.NullInt64
		var ownerID int
		rows.Scan(&e.ID, &e.UserName, &typ, &plaetze, &e.Treffpunkt, &e.Notiz, &ownerID)
		if plaetze.Valid {
			n := int(plaetze.Int64)
			e.Plaetze = &n
		}
		e.IsOwn = ownerID == currentUserID
		if typ == "biete" {
			biete = append(biete, e)
		} else {
			suche = append(suche, e)
		}
	}
	if biete == nil {
		biete = []CarpoolEntry{}
	}
	if suche == nil {
		suche = []CarpoolEntry{}
	}
	return biete, suche
}

// POST /api/mitfahrgelegenheiten
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	var body struct {
		GameID     int    `json:"gameId"`
		Typ        string `json:"typ"`
		Plaetze    *int   `json:"plaetze"`
		Treffpunkt string `json:"treffpunkt"`
		Notiz      string `json:"notiz"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Typ != "biete" && body.Typ != "suche" {
		http.Error(w, "typ must be biete or suche", http.StatusBadRequest)
		return
	}
	if body.GameID == 0 {
		http.Error(w, "gameId required", http.StatusBadRequest)
		return
	}

	var plaetze interface{} = nil
	if body.Typ == "biete" && body.Plaetze != nil && *body.Plaetze > 0 {
		plaetze = *body.Plaetze
	}

	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt, notiz, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(game_id, user_id) DO UPDATE SET
		  typ = excluded.typ,
		  plaetze = excluded.plaetze,
		  treffpunkt = excluded.treffpunkt,
		  notiz = excluded.notiz,
		  updated_at = CURRENT_TIMESTAMP`,
		body.GameID, userID, body.Typ, plaetze, body.Treffpunkt, body.Notiz)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/mitfahrgelegenheiten/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`DELETE FROM mitfahrgelegenheiten WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
