package carpooling

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
)

type Handler struct {
	db  *sql.DB
	cfg *appconfig.Config
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, cfg *appconfig.Config, h *hub.EventHub) *Handler {
	return &Handler{db: db, cfg: cfg, hub: h}
}

type GameEntry struct {
	ID        int    `json:"id"`
	Date      string `json:"date"`
	Time      string `json:"time"`
	Opponent  string `json:"opponent"`
	Team      string `json:"team"`
	TeamIDs   []int  `json:"teamIds"`
	EventType string `json:"eventType"`
}

type CarpoolEntry struct {
	ID         int     `json:"id"`
	UserID     int     `json:"userId"`
	UserName   string  `json:"userName"`
	PhotoURL   *string `json:"photoUrl,omitempty"`
	Plaetze    *int    `json:"plaetze,omitempty"`
	Treffpunkt string  `json:"treffpunkt,omitempty"`
	Notiz      string  `json:"notiz,omitempty"`
	IsOwn      bool    `json:"isOwn"`
}

type PaarungEntry struct {
	ID            int     `json:"id"`
	BieteID       int     `json:"bieteId"`
	SucheID       int     `json:"sucheId"`
	BieteName     string  `json:"bieteName"`
	SucheName     string  `json:"sucheName"`
	BietePhotoURL *string `json:"bietePhotoUrl,omitempty"`
	SuchePhotoURL *string `json:"suchePhotoUrl,omitempty"`
	BieteUserID   int     `json:"bieteUserId"`
	SucheUserID   int     `json:"sucheUserId"`
	Anzahl        int     `json:"anzahl"`
	Status        string  `json:"status"`
	InitiertVon   string  `json:"initiertVon"`
	BieteIsOwn    bool    `json:"bieteIsOwn"`
	SucheIsOwn    bool    `json:"sucheIsOwn"`
}

type CarpoolResponse struct {
	Game      GameEntry      `json:"game"`
	Biete     []CarpoolEntry `json:"biete"`
	Suche     []CarpoolEntry `json:"suche"`
	Paarungen []PaarungEntry `json:"paarungen"`
}

type ChildUser struct {
	UserID int    `json:"userId"`
	Name   string `json:"name"`
}

type ListResponse struct {
	Games        []CarpoolResponse `json:"games"`
	VehicleSeats *int              `json:"vehicleSeats"`
	Children     []ChildUser       `json:"children"`
}

// GET /api/mitfahrgelegenheiten
// Optional query param: ?team_id=X  (ignored for admin/vorstand)
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	restricted := claims.Role != "admin" && !claims.HasFunction("vorstand")

	var seasonID int
	if restricted {
		if err := h.db.QueryRowContext(r.Context(),
			`SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`,
		).Scan(&seasonID); err != nil {
			// No active season → empty list
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ListResponse{Games: []CarpoolResponse{}})
			return
		}
	}

	// Optional team filter (validated against accessible teams for non-admins)
	var teamFilter int
	if v := r.URL.Query().Get("team_id"); v != "" {
		fmt.Sscanf(v, "%d", &teamFilter)
	}

	var (
		query string
		args  []any
	)
	const selectCols = `SELECT g.id, g.date, COALESCE(g.time, '') AS time, g.opponent,
				       GROUP_CONCAT(t.name, ', ') AS team_names,
				       GROUP_CONCAT(t.id) AS team_ids,
				       g.event_type`
	if restricted {
		if teamFilter > 0 {
			query = selectCols + `
				FROM games g
				JOIN game_teams gt ON g.id = gt.game_id
				JOIN teams t ON t.id = gt.team_id
				WHERE DATE(g.date) >= DATE('now')
				  AND gt.team_id IN (
				    SELECT team_id FROM user_accessible_teams
				    WHERE user_id = ? AND season_id = ?
				  )
				  AND gt.team_id = ?
				GROUP BY g.id
				ORDER BY g.date ASC, g.time ASC`
			args = []any{userID, seasonID, teamFilter}
		} else {
			query = selectCols + `
				FROM games g
				JOIN game_teams gt ON g.id = gt.game_id
				JOIN teams t ON t.id = gt.team_id
				WHERE DATE(g.date) >= DATE('now')
				  AND gt.team_id IN (
				    SELECT team_id FROM user_accessible_teams
				    WHERE user_id = ? AND season_id = ?
				  )
				GROUP BY g.id
				ORDER BY g.date ASC, g.time ASC`
			args = []any{userID, seasonID}
		}
	} else {
		if teamFilter > 0 {
			query = selectCols + `
				FROM games g
				JOIN game_teams gt ON g.id = gt.game_id
				JOIN teams t ON t.id = gt.team_id
				WHERE DATE(g.date) >= DATE('now')
				  AND gt.team_id = ?
				GROUP BY g.id
				ORDER BY g.date ASC, g.time ASC`
			args = []any{teamFilter}
		} else {
			query = selectCols + `
				FROM games g
				JOIN game_teams gt ON g.id = gt.game_id
				JOIN teams t ON t.id = gt.team_id
				WHERE DATE(g.date) >= DATE('now')
				GROUP BY g.id
				ORDER BY g.date ASC, g.time ASC`
		}
	}

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []GameEntry
	for rows.Next() {
		var g GameEntry
		var teamIDsCSV sql.NullString
		rows.Scan(&g.ID, &g.Date, &g.Time, &g.Opponent, &g.Team, &teamIDsCSV, &g.EventType)
		g.TeamIDs = parseTeamIDs(teamIDsCSV)
		games = append(games, g)
	}

	children := h.childUsers(r.Context(), userID)
	childIDSet := makeChildIDSet(children)

	gamesList := make([]CarpoolResponse, 0, len(games))
	for _, g := range games {
		biete, suche := h.queryEntries(r, g.ID, userID, childIDSet)
		paarungen := h.queryPaarungen(r, g.ID, userID, childIDSet)
		gamesList = append(gamesList, CarpoolResponse{
			Game:      g,
			Biete:     biete,
			Suche:     suche,
			Paarungen: paarungen,
		})
	}

	var vehicleSeats *int
	h.db.QueryRowContext(r.Context(), `SELECT seats FROM vehicle_info WHERE user_id = ?`, userID).Scan(&vehicleSeats)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListResponse{Games: gamesList, VehicleSeats: vehicleSeats, Children: children})
}

// parseTeamIDs converts a comma-separated string of team IDs (from GROUP_CONCAT)
// into a sorted []int. Returns an empty slice if csv is NULL or unparseable.
func parseTeamIDs(csv sql.NullString) []int {
	if !csv.Valid || csv.String == "" {
		return []int{}
	}
	parts := strings.Split(csv.String, ",")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			ids = append(ids, n)
		}
	}
	sort.Ints(ids)
	return ids
}

func (h *Handler) queryEntries(r *http.Request, gameID, currentUserID int, childIDSet map[int]bool) ([]CarpoolEntry, []CarpoolEntry) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT m.id, u.first_name || ' ' || u.last_name, m.typ, m.plaetze, COALESCE(m.treffpunkt,''), COALESCE(m.notiz,''), m.user_id,
		       CASE WHEN COALESCE(uv.photo_visible,0)=1 AND COALESCE(u.photo_path,'') != '' THEN '/api/uploads/' || u.photo_path END
		FROM mitfahrgelegenheiten m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN user_visibility uv ON uv.user_id = m.user_id
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
		var photoURL sql.NullString
		rows.Scan(&e.ID, &e.UserName, &typ, &plaetze, &e.Treffpunkt, &e.Notiz, &ownerID, &photoURL)
		if plaetze.Valid {
			n := int(plaetze.Int64)
			e.Plaetze = &n
		}
		if photoURL.Valid && photoURL.String != "" {
			e.PhotoURL = &photoURL.String
		}
		e.UserID = ownerID
		e.IsOwn = ownerID == currentUserID || childIDSet[ownerID]
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

func (h *Handler) queryPaarungen(r *http.Request, gameID, currentUserID int, childIDSet map[int]bool) []PaarungEntry {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT p.id, p.biete_id, p.suche_id,
		       ub.first_name || ' ' || ub.last_name,
		       us.first_name || ' ' || us.last_name,
		       COALESCE(ms.plaetze, 0),
		       p.status, p.initiiert_von,
		       mb.user_id, ms.user_id,
		       CASE WHEN COALESCE(uvb.photo_visible,0)=1 AND COALESCE(ub.photo_path,'') != '' THEN '/api/uploads/' || ub.photo_path END,
		       CASE WHEN COALESCE(uvs.photo_visible,0)=1 AND COALESCE(us.photo_path,'') != '' THEN '/api/uploads/' || us.photo_path END
		FROM mitfahrt_paarungen p
		JOIN mitfahrgelegenheiten mb ON mb.id = p.biete_id
		JOIN mitfahrgelegenheiten ms ON ms.id = p.suche_id
		JOIN users ub ON ub.id = mb.user_id
		JOIN users us ON us.id = ms.user_id
		LEFT JOIN user_visibility uvb ON uvb.user_id = mb.user_id
		LEFT JOIN user_visibility uvs ON uvs.user_id = ms.user_id
		WHERE mb.game_id = ? AND p.status != 'rejected'
		ORDER BY p.created_at ASC`, gameID)
	if err != nil {
		return []PaarungEntry{}
	}
	defer rows.Close()

	var result []PaarungEntry
	for rows.Next() {
		var p PaarungEntry
		var bieteUserID, sucheUserID int
		var bietePhotoURL, suchePhotoURL sql.NullString
		rows.Scan(&p.ID, &p.BieteID, &p.SucheID,
			&p.BieteName, &p.SucheName,
			&p.Anzahl, &p.Status, &p.InitiertVon,
			&bieteUserID, &sucheUserID,
			&bietePhotoURL, &suchePhotoURL)
		p.BieteUserID = bieteUserID
		p.SucheUserID = sucheUserID
		p.BieteIsOwn = bieteUserID == currentUserID || childIDSet[bieteUserID]
		p.SucheIsOwn = sucheUserID == currentUserID || childIDSet[sucheUserID]
		if bietePhotoURL.Valid && bietePhotoURL.String != "" {
			p.BietePhotoURL = &bietePhotoURL.String
		}
		if suchePhotoURL.Valid && suchePhotoURL.String != "" {
			p.SuchePhotoURL = &suchePhotoURL.String
		}
		result = append(result, p)
	}
	if result == nil {
		result = []PaarungEntry{}
	}
	return result
}

// POST /api/mitfahrgelegenheiten
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	var body struct {
		GameID     int    `json:"gameId"`
		Typ        string `json:"typ"`
		ForUserID  *int   `json:"forUserId,omitempty"`
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
	if ok, _ := auth.UserCanSeeGame(r.Context(), h.db, claims.UserID, body.GameID); !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if body.Typ == "suche" && (body.Plaetze == nil || *body.Plaetze < 1) {
		http.Error(w, "plaetze >= 1 required for suche", http.StatusBadRequest)
		return
	}
	if body.ForUserID != nil && *body.ForUserID != userID {
		if !h.isChildOf(r.Context(), userID, *body.ForUserID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		userID = *body.ForUserID
	}

	var err error
	isNewEntry := false
	if body.Typ == "biete" {
		var plaetze interface{} = nil
		if body.Plaetze != nil && *body.Plaetze > 0 {
			plaetze = *body.Plaetze
		}
		var existingID int
		scanErr := h.db.QueryRowContext(r.Context(),
			`SELECT id FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'biete'`,
			body.GameID, userID).Scan(&existingID)
		switch scanErr {
		case sql.ErrNoRows:
			_, err = h.db.ExecContext(r.Context(),
				`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt, notiz) VALUES (?, ?, 'biete', ?, ?, ?)`,
				body.GameID, userID, plaetze, body.Treffpunkt, body.Notiz)
			isNewEntry = true
		case nil:
			_, err = h.db.ExecContext(r.Context(),
				`UPDATE mitfahrgelegenheiten SET plaetze = ?, treffpunkt = ?, notiz = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
				plaetze, body.Treffpunkt, body.Notiz, existingID)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		var existingID int
		scanErr := h.db.QueryRowContext(r.Context(),
			`SELECT id FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'suche'`,
			body.GameID, userID).Scan(&existingID)
		switch scanErr {
		case sql.ErrNoRows:
			_, err = h.db.ExecContext(r.Context(),
				`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt, notiz) VALUES (?, ?, 'suche', ?, ?, ?)`,
				body.GameID, userID, *body.Plaetze, body.Treffpunkt, body.Notiz)
			isNewEntry = true
		case nil:
			_, err = h.db.ExecContext(r.Context(),
				`UPDATE mitfahrgelegenheiten SET plaetze = ?, treffpunkt = ?, notiz = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
				*body.Plaetze, body.Treffpunkt, body.Notiz, existingID)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	actorName := h.userName(userID)
	if isNewEntry {
		if body.Typ == "biete" {
			h.writeEvents(body.GameID, h.usersWithTyp(body.GameID, "suche", userID), "biete_created", actorName)
		} else {
			h.writeEvents(body.GameID, h.usersWithTyp(body.GameID, "biete", userID), "suche_created", actorName)
		}
	}

	h.hub.Broadcast("mitfahrgelegenheiten")
	w.WriteHeader(http.StatusNoContent)

	oppositeTyp := "suche"
	if body.Typ == "suche" {
		oppositeTyp = "biete"
	}
	senderTyp := body.Typ
	gameID := body.GameID
	go func() {
		h.notifyOpposite(gameID, userID, actorName, senderTyp, oppositeTyp)
	}()

	if isNewEntry && body.Typ == "suche" {
		go h.notifyTeamForNewSuche(gameID, userID, actorName)
	}
}

// notifyTeamForNewSuche fans a push out to the kader-derived team circle
// (parents of regular + extended kader members, plus kader_trainers) for the
// game's team(s), but only when the game is each team's next upcoming game.
// Silent no-op when no team qualifies or the recipient set is empty.
func (h *Handler) notifyTeamForNewSuche(gameID, senderID int, senderName string) {
	recipients := h.teamPushRecipients(context.Background(), gameID, senderID)
	if len(recipients) == 0 {
		return
	}
	opponent, date := h.gameInfo(gameID)
	body := fmt.Sprintf("%s sucht eine Mitfahrgelegenheit zu %s, %s", senderName, opponent, date)
	notify.Send(h.db, h.cfg, recipients, "carpooling", "Mitfahrgelegenheit", body, "/mitfahrgelegenheiten")
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

	// Ownership check before opening a transaction to avoid SQLite in-memory
	// connection-pool issues where a second connection would see an empty DB.
	var typ string
	var gameID int
	var ownerID int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT typ, game_id, user_id FROM mitfahrgelegenheiten WHERE id = ?`, id).
		Scan(&typ, &gameID, &ownerID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if ownerID != userID && !h.isChildOf(r.Context(), userID, ownerID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	actorName := h.userName(userID)

	// Event-Log: nur Paarungspartner (die direkt betroffen sind)
	// Push: alle Suche-User des Spiels (bei biete_deleted), Paarungspartner (bei suche_deleted)
	var eventUserIDs, pushUserIDs []int
	switch typ {
	case "biete":
		eventUserIDs = h.sucherWithActivePaarung(id)
		pushUserIDs = h.usersWithTyp(gameID, "suche", userID)
		for _, uid := range eventUserIDs {
			tx.ExecContext(r.Context(),
				`INSERT INTO carpooling_events (user_id, game_id, type, actor_name) VALUES (?, ?, 'biete_deleted', ?)`,
				uid, gameID, actorName)
		}
	case "suche":
		eventUserIDs = h.bieterWithActivePaarung(id)
		pushUserIDs = eventUserIDs
		for _, uid := range eventUserIDs {
			tx.ExecContext(r.Context(),
				`INSERT INTO carpooling_events (user_id, game_id, type, actor_name) VALUES (?, ?, 'suche_deleted', ?)`,
				uid, gameID, actorName)
		}
	}

	res, err := tx.ExecContext(r.Context(),
		`DELETE FROM mitfahrgelegenheiten WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("mitfahrgelegenheiten")
	w.WriteHeader(http.StatusNoContent)

	if len(pushUserIDs) > 0 {
		go func() {
			opponent, date := h.gameInfo(gameID)
			var body string
			if typ == "biete" {
				body = fmt.Sprintf("%s hat sein Fahrangebot zurückgezogen — %s, %s", actorName, opponent, date)
			} else {
				body = fmt.Sprintf("%s hat seine Mitfahrzusage zurückgezogen — %s, %s", actorName, opponent, date)
			}
			notify.Send(h.db, h.cfg, pushUserIDs,
				"carpooling", "Mitfahrgelegenheit", body, "/mitfahrgelegenheiten")
		}()
	}
}

func (h *Handler) writeEvent(gameID, userID int, eventType, actorName string) {
	h.db.Exec(
		`INSERT INTO carpooling_events (user_id, game_id, type, actor_name) VALUES (?, ?, ?, ?)`,
		userID, gameID, eventType, actorName,
	)
}

func (h *Handler) writeEvents(gameID int, userIDs []int, eventType, actorName string) {
	for _, uid := range userIDs {
		h.writeEvent(gameID, uid, eventType, actorName)
	}
}

func (h *Handler) sucherWithActivePaarung(bieteID int) []int {
	rows, err := h.db.Query(`
		SELECT ms.user_id
		FROM mitfahrt_paarungen p
		JOIN mitfahrgelegenheiten ms ON ms.id = p.suche_id
		WHERE p.biete_id = ? AND p.status IN ('pending','confirmed')`, bieteID)
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

func (h *Handler) bieterWithActivePaarung(sucheID int) []int {
	rows, err := h.db.Query(`
		SELECT mb.user_id
		FROM mitfahrt_paarungen p
		JOIN mitfahrgelegenheiten mb ON mb.id = p.biete_id
		WHERE p.suche_id = ? AND p.status IN ('pending','confirmed')`, sucheID)
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
	notify.Send(h.db, h.cfg, userIDs, "carpooling", title, body, "/mitfahrgelegenheiten")
}

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

func (h *Handler) gameInfo(gameID int) (opponent, date string) {
	h.db.QueryRow(
		`SELECT opponent, date FROM games WHERE id = ?`, gameID).
		Scan(&opponent, &date)
	if len(date) >= 10 {
		date = date[:10]
	}
	return opponent, date
}

func (h *Handler) userName(userID int) string {
	var firstName, lastName string
	h.db.QueryRow(`SELECT first_name, last_name FROM users WHERE id = ?`, userID).
		Scan(&firstName, &lastName)
	if firstName == "" {
		return lastName
	}
	return firstName + " " + lastName
}

// childUsers returns all child users linked to parentUserID via family_links.
func (h *Handler) childUsers(ctx context.Context, parentUserID int) []ChildUser {
	rows, err := h.db.QueryContext(ctx, `
		SELECT u.id, u.first_name || ' ' || u.last_name
		FROM family_links fl
		JOIN members m ON m.id = fl.member_id
		JOIN users u ON u.id = m.user_id
		WHERE fl.parent_user_id = ? AND m.user_id IS NOT NULL
		ORDER BY u.first_name, u.last_name`, parentUserID)
	if err != nil {
		return []ChildUser{}
	}
	defer rows.Close()
	var result []ChildUser
	for rows.Next() {
		var c ChildUser
		rows.Scan(&c.UserID, &c.Name)
		result = append(result, c)
	}
	if result == nil {
		return []ChildUser{}
	}
	return result
}

// isChildOf returns true if targetUserID is a child of parentUserID via family_links.
func (h *Handler) isChildOf(ctx context.Context, parentUserID, targetUserID int) bool {
	var count int
	h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM family_links fl
		JOIN members m ON m.id = fl.member_id
		WHERE fl.parent_user_id = ? AND m.user_id = ?`,
		parentUserID, targetUserID).Scan(&count)
	return count > 0
}

// makeChildIDSet builds a fast lookup map from a slice of ChildUser.
func makeChildIDSet(children []ChildUser) map[int]bool {
	set := make(map[int]bool, len(children))
	for _, c := range children {
		set[c.UserID] = true
	}
	return set
}
