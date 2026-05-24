package carpooling

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
)

type Handler struct {
	db  *sql.DB
	cfg *appconfig.Config
}

func NewHandler(db *sql.DB, cfg *appconfig.Config) *Handler { return &Handler{db: db, cfg: cfg} }

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

// GET /api/mitfahrgelegenheiten
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT DISTINCT g.id, g.date, g.opponent, t.name, g.event_type
		FROM games g
		JOIN game_teams gt ON g.id = gt.game_id
		JOIN teams t ON t.id = gt.team_id
		WHERE DATE(g.date) >= DATE('now')
		ORDER BY g.date ASC`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []GameEntry
	for rows.Next() {
		var g GameEntry
		rows.Scan(&g.ID, &g.Date, &g.Opponent, &g.Team, &g.EventType)
		games = append(games, g)
	}

	gamesList := make([]CarpoolResponse, 0, len(games))
	for _, g := range games {
		biete, suche := h.queryEntries(r, g.ID, userID)
		gamesList = append(gamesList, CarpoolResponse{
			Game:  g,
			Biete: biete,
			Suche: suche,
		})
	}

	var vehicleSeats *int
	h.db.QueryRowContext(r.Context(), `SELECT seats FROM vehicle_info WHERE user_id = ?`, userID).Scan(&vehicleSeats)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListResponse{Games: gamesList, VehicleSeats: vehicleSeats})
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

	// notify users with the opposite typ
	oppositeTyp := "suche"
	if body.Typ == "suche" {
		oppositeTyp = "biete"
	}
	senderTyp := body.Typ
	gameID := body.GameID
	go func() {
		name := h.userName(userID)
		h.notifyOpposite(gameID, userID, name, senderTyp, oppositeTyp)
	}()
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

	// fetch typ and game_id before deleting (needed for notification)
	var typ string
	var gameID int
	h.db.QueryRowContext(r.Context(),
		`SELECT typ, game_id FROM mitfahrgelegenheiten WHERE id = ? AND user_id = ?`, id, userID).
		Scan(&typ, &gameID)

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

	// only notify when a "biete" offer is withdrawn
	if typ == "biete" && gameID != 0 {
		go func() {
			name := h.userName(userID)
			h.notifyWithdrawn(gameID, userID, name)
		}()
	}
}

// notifyOpposite sends a push to all users with oppositeTyp for the same game (excluding self).
func (h *Handler) notifyOpposite(gameID, senderID int, senderName, senderTyp, targetTyp string) {
	userIDs := h.usersWithTyp(gameID, targetTyp, senderID)
	if len(userIDs) == 0 {
		return
	}
	opponent, date := h.gameInfo(gameID)
	var title, body string
	if senderTyp == "biete" {
		title = "Mitfahrgelegenheit"
		body = fmt.Sprintf("%s bietet Plätze an — %s, %s", senderName, opponent, date)
	} else {
		title = "Mitfahrgelegenheit"
		body = fmt.Sprintf("%s sucht noch einen Platz — %s, %s", senderName, opponent, date)
	}
	notifications.SendToUsers(h.db, h.cfg, userIDs, title, body, "/mitfahrgelegenheiten")
}

// notifyWithdrawn sends a push to all "suche" users when a "biete" entry is deleted.
func (h *Handler) notifyWithdrawn(gameID, senderID int, senderName string) {
	userIDs := h.usersWithTyp(gameID, "suche", senderID)
	if len(userIDs) == 0 {
		return
	}
	opponent, date := h.gameInfo(gameID)
	body := fmt.Sprintf("%s hat sein Angebot zurückgezogen — %s, %s", senderName, opponent, date)
	notifications.SendToUsers(h.db, h.cfg, userIDs, "Mitfahrgelegenheit", body, "/mitfahrgelegenheiten")
}

// usersWithTyp returns user IDs with the given typ for a game, excluding excludeID.
func (h *Handler) usersWithTyp(gameID int, typ string, excludeID int) []int {
	rows, err := h.db.Query(
		`SELECT user_id FROM mitfahrgelegenheiten WHERE game_id = ? AND typ = ? AND user_id != ?`,
		gameID, typ, excludeID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

// gameInfo returns opponent and formatted date for a game.
func (h *Handler) gameInfo(gameID int) (opponent, date string) {
	h.db.QueryRow(
		`SELECT opponent, date FROM games WHERE id = ?`, gameID).
		Scan(&opponent, &date)
	if len(date) >= 10 {
		date = date[:10]
	}
	return opponent, date
}

// userName returns the display name for a user.
func (h *Handler) userName(userID int) string {
	var firstName, lastName string
	h.db.QueryRow(`SELECT first_name, last_name FROM users WHERE id = ?`, userID).
		Scan(&firstName, &lastName)
	if firstName == "" {
		return lastName
	}
	return firstName + " " + lastName
}
