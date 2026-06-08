package absences

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler {
	return &Handler{db: db, hub: h}
}

type absence struct {
	ID         int    `json:"id"`
	MemberID   int    `json:"member_id"`
	MemberName string `json:"member_name"`
	Type       string `json:"type"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Note       string `json:"note"`
	CreatedAt  string `json:"created_at"`
}

type previewEvent struct {
	EventType string `json:"event_type"` // "training" or "game"
	EventID   int    `json:"event_id"`
	Name      string `json:"name"`
	Date      string `json:"date"`
}

// memberIDForUser returns the member ID linked to the given user, or 0.
func (h *Handler) memberIDForUser(ctx context.Context, userID int) int {
	var id int
	h.db.QueryRowContext(ctx, `SELECT id FROM members WHERE user_id = ?`, userID).Scan(&id)
	return id
}

// resolveMemberID returns the member_id the caller may act on, or 0 + error message.
func (h *Handler) resolveMemberID(ctx context.Context, claims *auth.Claims, requestedMemberID int) (int, string) {
	switch claims.Role {
	case "spieler":
		mid := h.memberIDForUser(ctx, claims.UserID)
		if mid == 0 {
			return 0, "account is not linked to a member record"
		}
		return mid, ""
	case "elternteil":
		if requestedMemberID == 0 {
			return 0, "member_id required for elternteil"
		}
		var count int
		h.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM family_links WHERE parent_user_id = ? AND member_id = ?`,
			claims.UserID, requestedMemberID).Scan(&count)
		if count == 0 {
			return 0, "forbidden"
		}
		return requestedMemberID, ""
	default:
		return 0, "only spieler and elternteil may manage absences"
	}
}

// GET /api/absences/preview
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	q := r.URL.Query()
	memberIDStr := q.Get("member_id")
	from := q.Get("from")
	to := q.Get("to")
	if from == "" || to == "" {
		http.Error(w, "from and to required", http.StatusBadRequest)
		return
	}

	var memberID int
	if memberIDStr != "" {
		memberID, _ = strconv.Atoi(memberIDStr)
	}
	resolvedMemberID, errMsg := h.resolveMemberID(r.Context(), claims, memberID)
	if errMsg != "" {
		if errMsg == "forbidden" {
			http.Error(w, "forbidden", http.StatusForbidden)
		} else {
			http.Error(w, errMsg, http.StatusBadRequest)
		}
		return
	}

	events := []previewEvent{}

	// confirmed training responses in range
	tRows, err := h.db.QueryContext(r.Context(), `
		SELECT ts.id, COALESCE(ts.title, ''), ts.date
		FROM training_sessions ts
		JOIN training_responses tr ON tr.training_id = ts.id AND tr.member_id = ? AND tr.status = 'confirmed'
		WHERE ts.date BETWEEN ? AND ?`, resolvedMemberID, from, to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences preview training: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tRows.Close()
	for tRows.Next() {
		var ev previewEvent
		ev.EventType = "training"
		tRows.Scan(&ev.EventID, &ev.Name, &ev.Date)
		if ev.Name == "" {
			ev.Name = "Training"
		}
		ev.Date = ev.Date[:10]
		events = append(events, ev)
	}

	// confirmed game responses in range
	gRows, err := h.db.QueryContext(r.Context(), `
		SELECT g.id, g.opponent, g.date
		FROM games g
		JOIN game_responses gr ON gr.game_id = g.id AND gr.member_id = ? AND gr.status = 'confirmed'
		WHERE g.date BETWEEN ? AND ?`, resolvedMemberID, from, to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences preview games: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer gRows.Close()
	for gRows.Next() {
		var ev previewEvent
		ev.EventType = "game"
		gRows.Scan(&ev.EventID, &ev.Name, &ev.Date)
		ev.Date = ev.Date[:10]
		events = append(events, ev)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// POST /api/absences
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		MemberID  int    `json:"member_id"`
		Type      string `json:"type"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Note      string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Type != "vacation" && req.Type != "injury" {
		http.Error(w, "type must be vacation or injury", http.StatusBadRequest)
		return
	}
	if req.StartDate == "" || req.EndDate == "" {
		http.Error(w, "start_date and end_date required", http.StatusBadRequest)
		return
	}
	if req.StartDate > req.EndDate {
		http.Error(w, "start_date must be <= end_date", http.StatusBadRequest)
		return
	}

	memberID, errMsg := h.resolveMemberID(r.Context(), claims, req.MemberID)
	if errMsg != "" {
		if errMsg == "forbidden" {
			http.Error(w, "forbidden", http.StatusForbidden)
		} else {
			http.Error(w, errMsg, http.StatusBadRequest)
		}
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO member_absences (member_id, type, start_date, end_date, note, created_by)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		memberID, req.Type, req.StartDate, req.EndDate, req.Note, claims.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences create: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	absenceID, _ := res.LastInsertId()

	// Auto-decline training responses in range
	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at, absence_id)
		SELECT ts.id, ?, ?, 'declined', ?, datetime('now'), ?
		FROM training_sessions ts
		JOIN kader_members km ON km.member_id = ?
		JOIN kader k ON k.id = km.kader_id AND k.team_id = ts.team_id
		WHERE ts.date BETWEEN ? AND ?
		ON CONFLICT(training_id, member_id) DO UPDATE SET
		  responded_by = excluded.responded_by,
		  status       = 'declined',
		  reason       = excluded.reason,
		  responded_at = datetime('now'),
		  absence_id   = excluded.absence_id`,
		memberID, claims.UserID, req.Type, absenceID,
		memberID, req.StartDate, req.EndDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences auto-decline trainings: %v\n", err)
	}

	// Auto-decline game responses in range
	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at, absence_id)
		SELECT DISTINCT g.id, ?, ?, 'declined', ?, datetime('now'), ?
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id
		JOIN kader_members km ON km.member_id = ?
		JOIN kader k ON k.id = km.kader_id AND k.team_id = gt.team_id
		WHERE g.date BETWEEN ? AND ?
		ON CONFLICT(game_id, member_id) DO UPDATE SET
		  responded_by = excluded.responded_by,
		  status       = 'declined',
		  reason       = excluded.reason,
		  responded_at = datetime('now'),
		  absence_id   = excluded.absence_id`,
		memberID, claims.UserID, req.Type, absenceID,
		memberID, req.StartDate, req.EndDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences auto-decline games: %v\n", err)
	}

	h.hub.Broadcast("absences")
	h.hub.Broadcast("trainings")
	h.hub.Broadcast("games")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": absenceID})
}

// DELETE /api/absences/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var createdBy int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT created_by FROM member_absences WHERE id = ?`, id).Scan(&createdBy)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if claims.Role != "admin" && createdBy != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	h.db.ExecContext(r.Context(), `DELETE FROM member_absences WHERE id = ?`, id)

	h.hub.Broadcast("absences")
	h.hub.Broadcast("trainings")
	h.hub.Broadcast("games")
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/absences
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT a.id, a.member_id, m.first_name || ' ' || m.last_name,
		       a.type, a.start_date, a.end_date, a.note, a.created_at
		FROM member_absences a
		JOIN members m ON m.id = a.member_id
		WHERE a.member_id IN (
		  SELECT id FROM members WHERE user_id = ?
		  UNION
		  SELECT fl.member_id FROM family_links fl WHERE fl.parent_user_id = ?
		)
		ORDER BY a.start_date DESC`, claims.UserID, claims.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences list: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []absence{}
	for rows.Next() {
		var a absence
		rows.Scan(&a.ID, &a.MemberID, &a.MemberName, &a.Type, &a.StartDate, &a.EndDate, &a.Note, &a.CreatedAt)
		a.StartDate = a.StartDate[:10]
		a.EndDate = a.EndDate[:10]
		result = append(result, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/absences/calendar?from=&to=
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		http.Error(w, "from and to required", http.StatusBadRequest)
		return
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT a.id, a.member_id, m.first_name || ' ' || m.last_name,
		       a.type, a.start_date, a.end_date, a.note
		FROM member_absences a
		JOIN members m ON m.id = a.member_id
		WHERE a.start_date <= ? AND a.end_date >= ?
		  AND (
		    a.member_id IN (
		      SELECT id FROM members WHERE user_id = ?
		      UNION
		      SELECT fl.member_id FROM family_links fl WHERE fl.parent_user_id = ?
		    )
		    OR m.absences_public = 1
		  )
		ORDER BY a.start_date`, to, from, claims.UserID, claims.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences calendar: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []absence{}
	for rows.Next() {
		var a absence
		rows.Scan(&a.ID, &a.MemberID, &a.MemberName, &a.Type, &a.StartDate, &a.EndDate, &a.Note)
		a.StartDate = a.StartDate[:10]
		a.EndDate = a.EndDate[:10]
		result = append(result, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
