package trainings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

// hasTeamAccess returns true if the user is admin or a kader trainer of teamID.
func (h *Handler) hasTeamAccess(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims.Role == "admin" {
		return true, nil
	}
	var count int
	err := h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM kader_trainers kt
		 JOIN kader k ON k.id = kt.kader_id
		 JOIN members m ON m.id = kt.member_id
		 WHERE m.user_id = ? AND k.team_id = ?`,
		claims.UserID, teamID).Scan(&count)
	return count > 0, err
}

// memberIDForUser returns the member_id for a user, or 0 if not found.
func (h *Handler) memberIDForUser(ctx context.Context, userID int) (int, error) {
	var memberID int
	err := h.db.QueryRowContext(ctx,
		`SELECT id FROM members WHERE user_id = ?`, userID).Scan(&memberID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return memberID, err
}

// parentHasChild returns true if parentUserID has a family_links entry for memberID.
func (h *Handler) parentHasChild(ctx context.Context, parentUserID, memberID int) (bool, error) {
	var count int
	err := h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id = ? AND member_id = ?`,
		parentUserID, memberID).Scan(&count)
	return count > 0, err
}

// generateSessionDates returns all dates in [from, until] matching dayOfWeek (0=Mon…6=Sun).
func generateSessionDates(from, until time.Time, dayOfWeek int) []time.Time {
	// Convert our 0=Monday scheme to Go's time.Weekday (Sunday=0, Monday=1…)
	target := time.Weekday((dayOfWeek + 1) % 7)
	var dates []time.Time
	cur := from
	for !cur.After(until) {
		if cur.Weekday() == target {
			dates = append(dates, cur)
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return dates
}

// insertSessions bulk-inserts training_sessions within an existing transaction.
func insertSessions(ctx context.Context, tx *sql.Tx, seriesID int, teamID, seasonID int, startTime, endTime, location, note string, rsvpOptOut, rsvpRequireReason int, dates []time.Time) error {
	for _, d := range dates {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO training_sessions (series_id, team_id, season_id, date, start_time, end_time, location, note, rsvp_opt_out, rsvp_require_reason)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			seriesID, teamID, seasonID, d.Format("2006-01-02"), startTime, endTime, location, note, rsvpOptOut, rsvpRequireReason)
		if err != nil {
			return err
		}
	}
	return nil
}

// GET /api/training-series
func (h *Handler) ListSeries(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	var whereSQL string
	var args []any
	if claims.Role == "admin" {
		whereSQL = "1=1"
	} else {
		whereSQL = `s.team_id IN (
			SELECT DISTINCT k.team_id FROM kader k
			JOIN kader_trainers kt ON kt.kader_id = k.id
			JOIN members m ON m.id = kt.member_id
			WHERE m.user_id = ?)`
		args = append(args, claims.UserID)
	}
	if tid := r.URL.Query().Get("team_id"); tid != "" {
		whereSQL += " AND s.team_id = ?"
		args = append(args, tid)
	}

	query := fmt.Sprintf(`
		SELECT s.id, s.team_id, s.season_id, s.name, s.location, s.day_of_week,
		       s.start_time, s.end_time, s.valid_from, s.valid_until, s.note,
		       t.name as team_name,
		       COUNT(ts.id) as session_count,
		       s.rsvp_opt_out, s.rsvp_require_reason
		FROM training_series s
		JOIN teams t ON t.id = s.team_id
		LEFT JOIN training_sessions ts ON ts.series_id = s.id
		WHERE %s
		GROUP BY s.id
		ORDER BY s.valid_from DESC`, whereSQL)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListSeries: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type seriesItem struct {
		ID                int    `json:"id"`
		TeamID            int    `json:"team_id"`
		SeasonID          int    `json:"season_id"`
		Name              string `json:"name"`
		Location          string `json:"location"`
		DayOfWeek         int    `json:"day_of_week"`
		StartTime         string `json:"start_time"`
		EndTime           string `json:"end_time"`
		ValidFrom         string `json:"valid_from"`
		ValidUntil        string `json:"valid_until"`
		Note              string `json:"note"`
		TeamName          string `json:"team_name"`
		SessionCount      int    `json:"session_count"`
		RsvpOptOut        int    `json:"rsvp_opt_out"`
		RsvpRequireReason int    `json:"rsvp_require_reason"`
	}
	result := []seriesItem{}
	for rows.Next() {
		var s seriesItem
		rows.Scan(&s.ID, &s.TeamID, &s.SeasonID, &s.Name, &s.Location, &s.DayOfWeek,
			&s.StartTime, &s.EndTime, &s.ValidFrom, &s.ValidUntil, &s.Note,
			&s.TeamName, &s.SessionCount, &s.RsvpOptOut, &s.RsvpRequireReason)
		result = append(result, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/training-series
func (h *Handler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		TeamID            int    `json:"team_id"`
		SeasonID          int    `json:"season_id"`
		Name              string `json:"name"`
		Location          string `json:"location"`
		DayOfWeek         int    `json:"day_of_week"`
		StartTime         string `json:"start_time"`
		EndTime           string `json:"end_time"`
		ValidFrom         string `json:"valid_from"`
		ValidUntil        string `json:"valid_until"`
		Note              string `json:"note"`
		RsvpOptOut        int    `json:"rsvp_opt_out"`
		RsvpRequireReason int    `json:"rsvp_require_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ok, err := h.hasTeamAccess(r.Context(), claims, req.TeamID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateSeries team check: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	from, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		http.Error(w, "invalid valid_from date", http.StatusBadRequest)
		return
	}
	until, err := time.Parse("2006-01-02", req.ValidUntil)
	if err != nil {
		http.Error(w, "invalid valid_until date", http.StatusBadRequest)
		return
	}
	if req.DayOfWeek < 0 || req.DayOfWeek > 6 {
		http.Error(w, "day_of_week must be 0-6", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(r.Context(),
		`INSERT INTO training_series (team_id, season_id, name, location, day_of_week, start_time, end_time, valid_from, valid_until, note, created_by, rsvp_opt_out, rsvp_require_reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.TeamID, req.SeasonID, req.Name, req.Location, req.DayOfWeek,
		req.StartTime, req.EndTime, req.ValidFrom, req.ValidUntil, req.Note, claims.UserID,
		req.RsvpOptOut, req.RsvpRequireReason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateSeries insert series: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	seriesID, _ := res.LastInsertId()

	dates := generateSessionDates(from, until, req.DayOfWeek)
	if err := insertSessions(r.Context(), tx, int(seriesID), req.TeamID, req.SeasonID, req.StartTime, req.EndTime, req.Location, req.Note, req.RsvpOptOut, req.RsvpRequireReason, dates); err != nil {
		fmt.Fprintf(os.Stderr, "CreateSeries insert sessions: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("trainings")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":               seriesID,
		"sessions_created": len(dates),
	})
}

// PUT /api/training-series/{id}
func (h *Handler) UpdateSeries(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	seriesID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name              string `json:"name"`
		Location          string `json:"location"`
		DayOfWeek         int    `json:"day_of_week"`
		StartTime         string `json:"start_time"`
		EndTime           string `json:"end_time"`
		ValidFrom         string `json:"valid_from"`
		ValidUntil        string `json:"valid_until"`
		Note              string `json:"note"`
		Scope             string `json:"scope"`     // "all" or "this_and_following"
		FromDate          string `json:"from_date"` // required when scope="this_and_following"
		RsvpOptOut        int    `json:"rsvp_opt_out"`
		RsvpRequireReason int    `json:"rsvp_require_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Scope != "all" && req.Scope != "this_and_following" {
		http.Error(w, "scope must be 'all' or 'this_and_following'", http.StatusBadRequest)
		return
	}

	var teamID, seasonID int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT team_id, season_id FROM training_series WHERE id = ?`, seriesID).Scan(&teamID, &seasonID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ok, err := h.hasTeamAccess(r.Context(), claims, teamID)
	if err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	until, err := time.Parse("2006-01-02", req.ValidUntil)
	if err != nil {
		http.Error(w, "invalid valid_until", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(r.Context(),
		`UPDATE training_series SET name=?, location=?, day_of_week=?, start_time=?, end_time=?, valid_from=?, valid_until=?, note=?, rsvp_opt_out=?, rsvp_require_reason=? WHERE id=?`,
		req.Name, req.Location, req.DayOfWeek, req.StartTime, req.EndTime, req.ValidFrom, req.ValidUntil, req.Note, req.RsvpOptOut, req.RsvpRequireReason, seriesID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var genFrom time.Time
	if req.Scope == "all" {
		_, err = tx.ExecContext(r.Context(),
			`DELETE FROM training_sessions WHERE series_id = ?`, seriesID)
		genFrom, _ = time.Parse("2006-01-02", req.ValidFrom)
	} else {
		if req.FromDate == "" {
			http.Error(w, "from_date required for this_and_following scope", http.StatusBadRequest)
			return
		}
		genFrom, err = time.Parse("2006-01-02", req.FromDate)
		if err != nil {
			http.Error(w, "invalid from_date", http.StatusBadRequest)
			return
		}
		_, err = tx.ExecContext(r.Context(),
			`DELETE FROM training_sessions WHERE series_id = ? AND date >= ?`,
			seriesID, req.FromDate)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dates := generateSessionDates(genFrom, until, req.DayOfWeek)
	if err := insertSessions(r.Context(), tx, seriesID, teamID, seasonID, req.StartTime, req.EndTime, req.Location, req.Note, req.RsvpOptOut, req.RsvpRequireReason, dates); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("trainings")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"sessions_created": len(dates)})
}

// DELETE /api/training-series/{id}
func (h *Handler) DeleteSeries(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	seriesID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var teamID int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT team_id FROM training_series WHERE id = ?`, seriesID).Scan(&teamID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ok, err := h.hasTeamAccess(r.Context(), claims, teamID)
	if err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	scope := r.URL.Query().Get("scope")
	fromDate := r.URL.Query().Get("from")
	var execErr error
	if scope == "all" {
		_, execErr = h.db.ExecContext(r.Context(),
			`DELETE FROM training_sessions WHERE series_id = ?`, seriesID)
	} else if scope == "this_and_following" && fromDate != "" {
		_, execErr = h.db.ExecContext(r.Context(),
			`DELETE FROM training_sessions WHERE series_id = ? AND date >= ?`, seriesID, fromDate)
	} else {
		today := time.Now().Format("2006-01-02")
		_, execErr = h.db.ExecContext(r.Context(),
			`DELETE FROM training_sessions WHERE series_id = ? AND date >= ?`, seriesID, today)
	}
	if execErr != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err = h.db.ExecContext(r.Context(), `DELETE FROM training_series WHERE id = ?`, seriesID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("trainings")
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/training-sessions/{id}
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var teamID int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT team_id FROM training_sessions WHERE id = ?`, sessionID).Scan(&teamID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ok, err := h.hasTeamAccess(r.Context(), claims, teamID)
	if err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err = h.db.ExecContext(r.Context(), `DELETE FROM training_sessions WHERE id = ?`, sessionID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("trainings")
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/training-sessions
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		TeamID            int    `json:"team_id"`
		SeasonID          int    `json:"season_id"`
		Title             string `json:"title"`
		Date              string `json:"date"`
		StartTime         string `json:"start_time"`
		EndTime           string `json:"end_time"`
		Location          string `json:"location"`
		Note              string `json:"note"`
		RsvpOptOut        int    `json:"rsvp_opt_out"`
		RsvpRequireReason int    `json:"rsvp_require_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ok, err := h.hasTeamAccess(r.Context(), claims, req.TeamID)
	if err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO training_sessions (team_id, season_id, title, date, start_time, end_time, location, note, rsvp_opt_out, rsvp_require_reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.TeamID, req.SeasonID, req.Title, req.Date, req.StartTime, req.EndTime, req.Location, req.Note, req.RsvpOptOut, req.RsvpRequireReason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateSession: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	h.hub.Broadcast("trainings")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// PUT /api/training-sessions/{id}
func (h *Handler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var teamID int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT team_id FROM training_sessions WHERE id = ?`, sessionID).Scan(&teamID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ok, err := h.hasTeamAccess(r.Context(), claims, teamID)
	if err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Title        string `json:"title"`
		Date         string `json:"date"`
		StartTime    string `json:"start_time"`
		EndTime      string `json:"end_time"`
		Location     string `json:"location"`
		Note         string `json:"note"`
		Status       string `json:"status"`
		CancelReason string `json:"cancel_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Status != "" && req.Status != "active" && req.Status != "cancelled" {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	_, err = h.db.ExecContext(r.Context(),
		`UPDATE training_sessions SET title=?, date=?, start_time=?, end_time=?, location=?, note=?, status=?, cancel_reason=? WHERE id=?`,
		req.Title, req.Date, req.StartTime, req.EndTime, req.Location, req.Note, status, req.CancelReason, sessionID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("trainings")
	w.WriteHeader(http.StatusNoContent)
}

type childRSVP struct {
	MemberID int     `json:"member_id"`
	Name     string  `json:"name"`
	RSVP     *string `json:"rsvp"`
}

type sessionListItem struct {
	ID                int         `json:"id"`
	SeriesID          *int        `json:"series_id,omitempty"`
	TeamID            int         `json:"team_id"`
	TeamName          string      `json:"team_name"`
	SeasonID          int         `json:"season_id"`
	Title             string      `json:"title"`
	Date              string      `json:"date"`
	StartTime         string      `json:"start_time"`
	EndTime           string      `json:"end_time"`
	Location          string      `json:"location"`
	Note              string      `json:"note"`
	Status            string      `json:"status"`
	CancelReason      string      `json:"cancel_reason,omitempty"`
	ConfirmedCount    int         `json:"confirmed_count"`
	DeclinedCount     int         `json:"declined_count"`
	MaybeCount        int         `json:"maybe_count"`
	MyRSVP            *string     `json:"my_rsvp"`
	ChildrenRSVP      []childRSVP `json:"children_rsvp,omitempty"`
	RsvpOptOut        int         `json:"rsvp_opt_out"`
	RsvpRequireReason int         `json:"rsvp_require_reason"`
}

// GET /api/training-sessions
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	q := r.URL.Query()
	teamFilter := q.Get("team_id")
	from := q.Get("from")
	to := q.Get("to")
	if from == "" {
		from = time.Now().Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().AddDate(0, 3, 0).Format("2006-01-02")
	}

	memberID, err := h.memberIDForUser(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var teamSQL string
	var teamArgs []any
	if claims.Role == "admin" {
		teamSQL = "1=1"
	} else {
		var conds []string
		if claims.HasFunction("trainer") {
			conds = append(conds, `ts.team_id IN (
				SELECT DISTINCT k.team_id FROM kader k
				JOIN kader_trainers kt ON kt.kader_id = k.id
				JOIN members m ON m.id = kt.member_id
				WHERE m.user_id = ?)`)
			teamArgs = append(teamArgs, claims.UserID)
		}
		if claims.IsParent {
			conds = append(conds, "ts.team_id IN (SELECT DISTINCT tm.team_id FROM player_memberships tm JOIN members m ON m.id = tm.member_id JOIN family_links fl ON fl.member_id = m.id WHERE fl.parent_user_id = ?)")
			teamArgs = append(teamArgs, claims.UserID)
		}
		conds = append(conds, "ts.team_id IN (SELECT DISTINCT tm.team_id FROM player_memberships tm JOIN members m ON m.id = tm.member_id WHERE m.user_id = ?)")
		teamArgs = append(teamArgs, claims.UserID)
		teamSQL = "(" + strings.Join(conds, " OR ") + ")"
	}

	// Order must match the ?-markers in the query:
	// 1. member_id (SELECT subquery), 2. teamArgs (WHERE teamSQL), 3. from, 4. to, 5. optional team filter
	args := append([]any{memberID}, teamArgs...)
	args = append(args, from, to)
	optTeamFilter := ""
	if teamFilter != "" {
		optTeamFilter = "AND ts.team_id = ?"
		args = append(args, teamFilter)
	}

	query := fmt.Sprintf(`
		SELECT ts.id, ts.series_id, ts.team_id, COALESCE(t.name, ''), ts.season_id, ts.title, ts.date, ts.start_time, ts.end_time,
		       ts.location, ts.note, ts.status, ts.cancel_reason,
		       CASE WHEN ts.rsvp_opt_out = 1
		            THEN COALESCE(SUM(CASE WHEN tr.status='confirmed' THEN 1 ELSE 0 END), 0) + (
		                   SELECT COUNT(*) FROM player_memberships tm2
		                   JOIN members m2 ON m2.id = tm2.member_id
		                   WHERE tm2.team_id = ts.team_id
		                   AND NOT EXISTS (SELECT 1 FROM training_responses tr2 WHERE tr2.training_id = ts.id AND tr2.member_id = tm2.member_id)
		                 )
		            ELSE COALESCE(SUM(CASE WHEN tr.status='confirmed' THEN 1 ELSE 0 END), 0)
		       END,
		       COALESCE(SUM(CASE WHEN tr.status='declined'  THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN tr.status='maybe'     THEN 1 ELSE 0 END), 0),
		       (SELECT status FROM training_responses WHERE training_id = ts.id AND member_id = ?),
		       ts.rsvp_opt_out, ts.rsvp_require_reason
		FROM training_sessions ts
		LEFT JOIN teams t ON t.id = ts.team_id
		LEFT JOIN training_responses tr ON tr.training_id = ts.id
		WHERE %s AND ts.date >= ? AND ts.date <= ? %s
		GROUP BY ts.id
		ORDER BY ts.date, ts.start_time`, teamSQL, optTeamFilter)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListSessions query: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []sessionListItem{}
	for rows.Next() {
		var s sessionListItem
		var seriesID sql.NullInt64
		var myRSVP sql.NullString
		err := rows.Scan(
			&s.ID, &seriesID, &s.TeamID, &s.TeamName, &s.SeasonID, &s.Title, &s.Date, &s.StartTime, &s.EndTime,
			&s.Location, &s.Note, &s.Status, &s.CancelReason,
			&s.ConfirmedCount, &s.DeclinedCount, &s.MaybeCount, &myRSVP,
			&s.RsvpOptOut, &s.RsvpRequireReason)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ListSessions scan: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if seriesID.Valid {
			id := int(seriesID.Int64)
			s.SeriesID = &id
		}
		if myRSVP.Valid {
			s.MyRSVP = &myRSVP.String
		} else if s.RsvpOptOut == 1 {
			confirmed := "confirmed"
			s.MyRSVP = &confirmed
		}
		result = append(result, s)
	}

	if claims.IsParent && len(result) > 0 {
		if err := h.attachChildrenRSVPToSessions(r.Context(), claims.UserID, result); err != nil {
			fmt.Fprintf(os.Stderr, "ListSessions children_rsvp: %v\n", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type sessionResponse struct {
	MemberID    int     `json:"member_id"`
	MemberName  string  `json:"member_name"`
	Status      string  `json:"status"`
	Reason      *string `json:"reason"`
	RespondedBy int     `json:"responded_by"`
	RespondedAt string  `json:"responded_at"`
}

type sessionDetail struct {
	sessionListItem
	Responses []sessionResponse `json:"responses"`
}

// GET /api/training-sessions/{id}
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	memberID, err := h.memberIDForUser(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var s sessionListItem
	var seriesID sql.NullInt64
	var myRSVP sql.NullString
	err = h.db.QueryRowContext(r.Context(), `
		SELECT ts.id, ts.series_id, ts.team_id, ts.season_id, ts.title, ts.date, ts.start_time, ts.end_time,
		       ts.location, ts.note, ts.status, ts.cancel_reason,
		       CASE WHEN ts.rsvp_opt_out = 1
		            THEN COALESCE((SELECT COUNT(*) FROM training_responses WHERE training_id=ts.id AND status='confirmed'),0)
		                 + (SELECT COUNT(*) FROM player_memberships tm2
		                    WHERE tm2.team_id = ts.team_id
		                    AND NOT EXISTS (SELECT 1 FROM training_responses tr2 WHERE tr2.training_id=ts.id AND tr2.member_id=tm2.member_id))
		            ELSE COALESCE((SELECT COUNT(*) FROM training_responses WHERE training_id=ts.id AND status='confirmed'),0)
		       END,
		       COALESCE((SELECT COUNT(*) FROM training_responses WHERE training_id=ts.id AND status='declined'),0),
		       COALESCE((SELECT COUNT(*) FROM training_responses WHERE training_id=ts.id AND status='maybe'),0),
		       (SELECT status FROM training_responses WHERE training_id=ts.id AND member_id=?),
		       ts.rsvp_opt_out, ts.rsvp_require_reason
		FROM training_sessions ts WHERE ts.id = ?`, memberID, sessionID).Scan(
		&s.ID, &seriesID, &s.TeamID, &s.SeasonID, &s.Title, &s.Date, &s.StartTime, &s.EndTime,
		&s.Location, &s.Note, &s.Status, &s.CancelReason,
		&s.ConfirmedCount, &s.DeclinedCount, &s.MaybeCount, &myRSVP,
		&s.RsvpOptOut, &s.RsvpRequireReason)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSession: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if seriesID.Valid {
		id := int(seriesID.Int64)
		s.SeriesID = &id
	}
	if myRSVP.Valid {
		s.MyRSVP = &myRSVP.String
	} else if s.RsvpOptOut == 1 {
		confirmed := "confirmed"
		s.MyRSVP = &confirmed
	}

	// Load responses
	isTrainerLike := claims.Role == "admin" || claims.HasFunction("trainer")
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT tr.member_id,
		       m.first_name || ' ' || m.last_name,
		       tr.status, tr.reason, tr.responded_by, tr.responded_at
		FROM training_responses tr
		JOIN members m ON m.id = tr.member_id
		WHERE tr.training_id = ?
		ORDER BY m.last_name, m.first_name`, sessionID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Pre-fetch children of parent for privacy check (one query)
	childMemberIDs := map[int]bool{}
	if claims.IsParent {
		childRows, err := h.db.QueryContext(r.Context(),
			`SELECT member_id FROM family_links WHERE parent_user_id = ?`, claims.UserID)
		if err == nil {
			defer childRows.Close()
			for childRows.Next() {
				var cid int
				childRows.Scan(&cid)
				childMemberIDs[cid] = true
			}
		}
	}

	responses := []sessionResponse{}
	for rows.Next() {
		var resp sessionResponse
		var reason string
		rows.Scan(&resp.MemberID, &resp.MemberName, &resp.Status, &reason, &resp.RespondedBy, &resp.RespondedAt)

		canSeeReason := isTrainerLike ||
			(memberID > 0 && resp.MemberID == memberID) ||
			childMemberIDs[resp.MemberID]
		if canSeeReason && reason != "" {
			resp.Reason = &reason
		}
		responses = append(responses, resp)
	}

	detail := sessionDetail{sessionListItem: s, Responses: responses}
	if detail.Responses == nil {
		detail.Responses = []sessionResponse{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// POST /api/training-sessions/{id}/respond
func (h *Handler) Respond(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		MemberID int    `json:"member_id"` // required for elternteil; ignored for spieler
		Status   string `json:"status"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Status != "confirmed" && req.Status != "declined" && req.Status != "maybe" {
		http.Error(w, "status must be confirmed, declined, or maybe", http.StatusBadRequest)
		return
	}

	var memberID int
	switch claims.Role {
	case "spieler":
		memberID, err = h.memberIDForUser(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if memberID == 0 {
			http.Error(w, "your account is not linked to a member record", http.StatusUnprocessableEntity)
			return
		}
	case "elternteil":
		if req.MemberID == 0 {
			http.Error(w, "member_id required for elternteil", http.StatusBadRequest)
			return
		}
		ok, err := h.parentHasChild(r.Context(), claims.UserID, req.MemberID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		memberID = req.MemberID
	default:
		// trainer/admin: can respond for any member if member_id is provided
		if req.MemberID == 0 {
			// try own member record
			memberID, err = h.memberIDForUser(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if memberID == 0 {
				http.Error(w, "member_id required", http.StatusBadRequest)
				return
			}
		} else {
			memberID = req.MemberID
		}
	}

	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(training_id, member_id) DO UPDATE SET
		  responded_by = excluded.responded_by,
		  status       = excluded.status,
		  reason       = excluded.reason,
		  responded_at = CURRENT_TIMESTAMP`,
		sessionID, memberID, claims.UserID, req.Status, req.Reason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Respond upsert: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("trainings")
	w.WriteHeader(http.StatusNoContent)
}

type attendanceItem struct {
	MemberID   int     `json:"member_id"`
	MemberName string  `json:"member_name"`
	RSVPStatus *string `json:"rsvp_status"`
	Reason     *string `json:"reason"`
	Present    *bool   `json:"present"`
}

// GET /api/training-sessions/{id}/attendances
func (h *Handler) GetAttendances(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var teamID, seasonID, rsvpOptOut int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT team_id, season_id, rsvp_opt_out FROM training_sessions WHERE id = ?`, sessionID).Scan(&teamID, &seasonID, &rsvpOptOut)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	isTrainerLike, err := h.hasTeamAccess(r.Context(), claims, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !isTrainerLike {
		var count int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM user_accessible_teams WHERE user_id = ? AND team_id = ? AND season_id = ?`,
			claims.UserID, teamID, seasonID).Scan(&count)
		if count == 0 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT m.id, m.first_name || ' ' || m.last_name,
		       tr.status, tr.reason, ta.present
		FROM members m
		JOIN player_memberships tm ON tm.member_id = m.id AND tm.team_id = ? AND tm.season_id = ?
		LEFT JOIN training_responses tr ON tr.training_id = ? AND tr.member_id = m.id
		LEFT JOIN training_attendances ta ON ta.training_id = ? AND ta.member_id = m.id
		GROUP BY m.id
		ORDER BY m.last_name, m.first_name`, teamID, seasonID, sessionID, sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAttendances: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []attendanceItem{}
	for rows.Next() {
		var item attendanceItem
		var rsvp, reason sql.NullString
		var present sql.NullInt64
		rows.Scan(&item.MemberID, &item.MemberName, &rsvp, &reason, &present)
		if rsvp.Valid {
			item.RSVPStatus = &rsvp.String
		} else if rsvpOptOut == 1 {
			confirmed := "confirmed"
			item.RSVPStatus = &confirmed
		}
		canSeeReason := isTrainerLike
		if canSeeReason && reason.Valid && reason.String != "" {
			item.Reason = &reason.String
		}
		if present.Valid {
			b := present.Int64 == 1
			item.Present = &b
		}
		result = append(result, item)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/training-sessions/{id}/attendances
func (h *Handler) SaveAttendances(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var sessionDate string
	var teamID int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT date, team_id FROM training_sessions WHERE id = ?`, sessionID).Scan(&sessionDate, &teamID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ok, err := h.hasTeamAccess(r.Context(), claims, teamID)
	if err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	today := time.Now().Format("2006-01-02")
	if sessionDate > today {
		http.Error(w, "attendance can only be recorded for past or current sessions", http.StatusUnprocessableEntity)
		return
	}

	var entries []struct {
		MemberID int  `json:"member_id"`
		Present  bool `json:"present"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, e := range entries {
		present := 0
		if e.Present {
			present = 1
		}
		_, err := tx.ExecContext(r.Context(), `
			INSERT INTO training_attendances (training_id, member_id, present, noted_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(training_id, member_id) DO UPDATE SET present=excluded.present, noted_at=CURRENT_TIMESTAMP`,
			sessionID, e.MemberID, present)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SaveAttendances upsert: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("trainings")
	w.WriteHeader(http.StatusNoContent)
}

// attachChildrenRSVPToSessions fills ChildrenRSVP on each item for parent users.
// Only includes children who are kader members of the session's team.
func (h *Handler) attachChildrenRSVPToSessions(ctx context.Context, parentUserID int, items []sessionListItem) error {
	placeholders := make([]string, len(items))
	sessionIDs := make([]any, len(items))
	for i, s := range items {
		placeholders[i] = "?"
		sessionIDs[i] = s.ID
	}
	// Single query: for each session, return only children who are in the kader of that session's team/season
	rows, err := h.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT ts.id, m.id, m.first_name || ' ' || m.last_name, tr.status
		FROM training_sessions ts
		JOIN kader k ON k.team_id = ts.team_id AND k.season_id = ts.season_id
		JOIN kader_members km ON km.kader_id = k.id
		JOIN members m ON m.id = km.member_id
		JOIN family_links fl ON fl.member_id = m.id AND fl.parent_user_id = ?
		LEFT JOIN training_responses tr ON tr.training_id = ts.id AND tr.member_id = m.id
		WHERE ts.id IN (%s)
		ORDER BY m.last_name, m.first_name`,
		strings.Join(placeholders, ",")),
		append([]any{parentUserID}, sessionIDs...)...)
	if err != nil {
		return err
	}
	defer rows.Close()

	bySession := map[int][]childRSVP{}
	for rows.Next() {
		var sid int
		var c childRSVP
		var rsvp sql.NullString
		rows.Scan(&sid, &c.MemberID, &c.Name, &rsvp)
		if rsvp.Valid {
			s := rsvp.String
			c.RSVP = &s
		}
		bySession[sid] = append(bySession[sid], c)
	}

	for i := range items {
		if children, ok := bySession[items[i].ID]; ok {
			items[i].ChildrenRSVP = children
		} else {
			items[i].ChildrenRSVP = []childRSVP{}
		}
	}
	return nil
}
