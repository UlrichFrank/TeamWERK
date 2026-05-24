package carpooling

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

type GameEntry struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Opponent string `json:"opponent"`
	Team     string `json:"team"`
}

type CarpoolEntry struct {
	ID        int    `json:"id"`
	UserName  string `json:"userName"`
	Plaetze   *int   `json:"plaetze,omitempty"`
	Treffpunkt string `json:"treffpunkt,omitempty"`
	Notiz     string `json:"notiz,omitempty"`
	IsOwn     bool   `json:"isOwn"`
}

type CarpoolResponse struct {
	Game  GameEntry      `json:"game"`
	Biete []CarpoolEntry `json:"biete"`
	Suche []CarpoolEntry `json:"suche"`
}

// GET /api/mitfahrgelegenheiten
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT DISTINCT g.id, g.date, g.opponent, t.name
		FROM games g
		JOIN game_teams gt ON g.id = gt.game_id
		JOIN teams t ON t.id = gt.team_id
		WHERE g.is_home = 0
		  AND DATE(g.date) >= DATE('now')
		ORDER BY g.date ASC`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []GameEntry
	for rows.Next() {
		var g GameEntry
		rows.Scan(&g.ID, &g.Date, &g.Opponent, &g.Team)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) queryEntries(r *http.Request, gameID, currentUserID int) ([]CarpoolEntry, []CarpoolEntry) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT m.id, u.name, m.typ, m.plaetze, COALESCE(m.treffpunkt,''), COALESCE(m.notiz,''), m.user_id
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
