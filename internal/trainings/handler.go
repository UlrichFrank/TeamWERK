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
	"unicode/utf8"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
)

type Handler struct {
	db  *sql.DB
	cfg *appconfig.Config
	hub *hub.EventHub
	now func() time.Time
}

func NewHandler(db *sql.DB, cfg *appconfig.Config, h *hub.EventHub) *Handler {
	return &Handler{db: db, cfg: cfg, hub: h, now: time.Now}
}

// SetNow overrides the clock used for cutoff checks. Intended for tests.
func (h *Handler) SetNow(now func() time.Time) { h.now = now }

// TrainingRSVPCutoff: bis dahin (vor Session-Beginn) sind RSVP-Änderungen
// für Spieler/Eltern erlaubt. Trainer/Vorstand/Admin können auch danach pflegen.
const TrainingRSVPCutoff = 2 * time.Hour

var berlinTZ = mustLoadBerlin()

func mustLoadBerlin() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic("trainings: cannot load Europe/Berlin timezone: " + err.Error())
	}
	return loc
}

// trainingLocksAt liefert den UTC-Zeitpunkt, ab dem RSVP-Änderungen
// für reguläre Mitglieder gesperrt sind. dateISO ist `YYYY-MM-DD`,
// startTimeHHMM ist `HH:MM` (Sekunden werden toleriert) in Europe/Berlin.
func trainingLocksAt(dateISO, startTimeHHMM string) (time.Time, error) {
	t, err := parseBerlinDateTime(dateISO, startTimeHHMM)
	if err != nil {
		return time.Time{}, err
	}
	return t.Add(-TrainingRSVPCutoff).UTC(), nil
}

func parseBerlinDateTime(dateISO, hhmm string) (time.Time, error) {
	// SQLite DATE columns are returned as RFC3339 ("2026-06-15T00:00:00Z");
	// keep only the YYYY-MM-DD prefix. Tolerate "HH:MM:SS" similarly.
	if len(dateISO) > 10 {
		dateISO = dateISO[:10]
	}
	if len(hhmm) > 5 {
		hhmm = hhmm[:5]
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", dateISO+" "+hhmm, berlinTZ)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// teamMembersAndParents returns user IDs of all active kader members (and their parents) for a team.
func (h *Handler) teamMembersAndParents(teamID int) []int {
	rows, err := h.db.Query(
		`SELECT DISTINCT u.id FROM users u
		 JOIN members m ON m.user_id = u.id
		 JOIN player_memberships pm ON pm.member_id = m.id
		 JOIN seasons s ON s.id = pm.season_id AND s.is_active = 1
		 WHERE pm.team_id = ?
		 UNION
		 SELECT DISTINCT fl.parent_user_id FROM family_links fl
		 JOIN members m ON m.id = fl.member_id
		 JOIN player_memberships pm ON pm.member_id = m.id
		 JOIN seasons s ON s.id = pm.season_id AND s.is_active = 1
		 WHERE pm.team_id = ?
		 UNION
		 SELECT DISTINCT m.user_id FROM members m
		 JOIN kader_extended_members kem ON kem.member_id = m.id
		 JOIN kader k ON k.id = kem.kader_id
		 JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
		 WHERE k.team_id = ? AND m.user_id IS NOT NULL`, teamID, teamID, teamID)
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

// hasTeamAccess returns true if the user is admin, vorstand, sportliche_leitung,
// or a kader trainer of teamID.
func (h *Handler) hasTeamAccess(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims.Role == "admin" || claims.HasFunction("vorstand") || claims.HasFunction("sportliche_leitung") {
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
func insertSessions(ctx context.Context, tx *sql.Tx, seriesID int, teamID, seasonID int, startTime, endTime string, venueID *int, note string, rsvpOptOut, rsvpRequireReason int, dates []time.Time) error {
	var venueIDVal interface{}
	if venueID != nil {
		venueIDVal = *venueID
	}
	for _, d := range dates {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO training_sessions (series_id, team_id, season_id, date, start_time, end_time, venue_id, note, rsvp_opt_out, rsvp_require_reason)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			seriesID, teamID, seasonID, d.Format("2006-01-02"), startTime, endTime, venueIDVal, note, rsvpOptOut, rsvpRequireReason)
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
		SELECT s.id, s.team_id, s.season_id, s.name, s.day_of_week,
		       s.start_time, s.end_time, s.valid_from, s.valid_until, s.note,
		       COALESCE(`+appdb.TeamDisplayShort("t")+`, t.name) as team_name,
		       COUNT(ts.id) as session_count,
		       s.rsvp_opt_out, s.rsvp_require_reason,
		       v.id, v.name, v.street, v.city, v.postal_code, v.note
		FROM training_series s
		JOIN teams t ON t.id = s.team_id
		LEFT JOIN training_sessions ts ON ts.series_id = s.id
		LEFT JOIN venues v ON v.id = s.venue_id
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

	type venueRef struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Street     string `json:"street"`
		City       string `json:"city"`
		PostalCode string `json:"postal_code"`
		Note       string `json:"note"`
	}
	type seriesItem struct {
		ID                int       `json:"id"`
		TeamID            int       `json:"team_id"`
		SeasonID          int       `json:"season_id"`
		Name              string    `json:"name"`
		DayOfWeek         int       `json:"day_of_week"`
		StartTime         string    `json:"start_time"`
		EndTime           string    `json:"end_time"`
		ValidFrom         string    `json:"valid_from"`
		ValidUntil        string    `json:"valid_until"`
		Note              string    `json:"note"`
		TeamName          string    `json:"team_name"`
		SessionCount      int       `json:"session_count"`
		RsvpOptOut        int       `json:"rsvp_opt_out"`
		RsvpRequireReason int       `json:"rsvp_require_reason"`
		Venue             *venueRef `json:"venue,omitempty"`
	}
	result := []seriesItem{}
	for rows.Next() {
		var s seriesItem
		var vID sql.NullInt64
		var vName, vStreet, vCity, vPostal, vNote sql.NullString
		rows.Scan(&s.ID, &s.TeamID, &s.SeasonID, &s.Name, &s.DayOfWeek,
			&s.StartTime, &s.EndTime, &s.ValidFrom, &s.ValidUntil, &s.Note,
			&s.TeamName, &s.SessionCount, &s.RsvpOptOut, &s.RsvpRequireReason,
			&vID, &vName, &vStreet, &vCity, &vPostal, &vNote)
		if vID.Valid {
			s.Venue = &venueRef{
				ID: int(vID.Int64), Name: vName.String, Street: vStreet.String,
				City: vCity.String, PostalCode: vPostal.String, Note: vNote.String,
			}
		}
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
		VenueID           *int   `json:"venue_id"`
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

	var venueIDVal interface{}
	if req.VenueID != nil {
		venueIDVal = *req.VenueID
	}
	res, err := tx.ExecContext(r.Context(),
		`INSERT INTO training_series (team_id, season_id, name, venue_id, day_of_week, start_time, end_time, valid_from, valid_until, note, created_by, rsvp_opt_out, rsvp_require_reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.TeamID, req.SeasonID, req.Name, venueIDVal, req.DayOfWeek,
		req.StartTime, req.EndTime, req.ValidFrom, req.ValidUntil, req.Note, claims.UserID,
		req.RsvpOptOut, req.RsvpRequireReason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateSeries insert series: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	seriesID, _ := res.LastInsertId()

	dates := generateSessionDates(from, until, req.DayOfWeek)
	if err := insertSessions(r.Context(), tx, int(seriesID), req.TeamID, req.SeasonID, req.StartTime, req.EndTime, req.VenueID, req.Note, req.RsvpOptOut, req.RsvpRequireReason, dates); err != nil {
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
		VenueID           *int   `json:"venue_id"`
		DayOfWeek         int    `json:"day_of_week"`
		StartTime         string `json:"start_time"`
		EndTime           string `json:"end_time"`
		ValidFrom         string `json:"valid_from"`
		ValidUntil        string `json:"valid_until"`
		Note              string `json:"note"`
		Scope             string `json:"scope"`     // "all" or "this_and_following"
		FromDate          string `json:"from_date"` // required when scope="this_and_following"
		RsvpOptOut        *int   `json:"rsvp_opt_out,omitempty"`
		RsvpRequireReason *int   `json:"rsvp_require_reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Scope != "all" && req.Scope != "this_and_following" {
		http.Error(w, "scope must be 'all' or 'this_and_following'", http.StatusBadRequest)
		return
	}

	var teamID, seasonID, curOptOut, curReqReason int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT team_id, season_id, rsvp_opt_out, rsvp_require_reason FROM training_series WHERE id = ?`, seriesID).
		Scan(&teamID, &seasonID, &curOptOut, &curReqReason)
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

	// Partial-Update: fehlende RSVP-Felder behalten den aktuellen DB-Wert.
	rsvpOptOut := curOptOut
	if req.RsvpOptOut != nil {
		rsvpOptOut = *req.RsvpOptOut
	}
	rsvpRequireReason := curReqReason
	if req.RsvpRequireReason != nil {
		rsvpRequireReason = *req.RsvpRequireReason
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

	var venueIDVal interface{}
	if req.VenueID != nil {
		venueIDVal = *req.VenueID
	}
	_, err = tx.ExecContext(r.Context(),
		`UPDATE training_series SET name=?, venue_id=?, day_of_week=?, start_time=?, end_time=?, valid_from=?, valid_until=?, note=?, rsvp_opt_out=?, rsvp_require_reason=? WHERE id=?`,
		req.Name, venueIDVal, req.DayOfWeek, req.StartTime, req.EndTime, req.ValidFrom, req.ValidUntil, req.Note, rsvpOptOut, rsvpRequireReason, seriesID)
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
	if err := insertSessions(r.Context(), tx, seriesID, teamID, seasonID, req.StartTime, req.EndTime, req.VenueID, req.Note, rsvpOptOut, rsvpRequireReason, dates); err != nil {
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
	// event-notes: pending Push-Rows der betroffenen Sessions vorab aufräumen.
	cleanupPending := func(extra string, args ...any) {
		h.db.ExecContext(r.Context(),
			`DELETE FROM pending_event_notes_push WHERE ref_type='training' AND ref_id IN (
				SELECT id FROM training_sessions WHERE series_id = ?`+extra+`)`, args...)
	}
	var execErr error
	if scope == "all" {
		cleanupPending("", seriesID)
		_, execErr = h.db.ExecContext(r.Context(),
			`DELETE FROM training_sessions WHERE series_id = ?`, seriesID)
	} else if scope == "this_and_following" && fromDate != "" {
		cleanupPending(" AND date >= ?", seriesID, fromDate)
		_, execErr = h.db.ExecContext(r.Context(),
			`DELETE FROM training_sessions WHERE series_id = ? AND date >= ?`, seriesID, fromDate)
	} else {
		today := time.Now().Format("2006-01-02")
		cleanupPending(" AND date >= ?", seriesID, today)
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
	notify.Send(h.db, h.cfg, h.teamMembersAndParents(teamID),
		"trainings", "Trainingsserie gelöscht", "Eine Trainingsserie wurde beendet", "/termine")
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
	if _, err = h.db.ExecContext(r.Context(),
		`DELETE FROM pending_event_notes_push WHERE ref_type='training' AND ref_id=?`, sessionID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err = h.db.ExecContext(r.Context(), `DELETE FROM training_sessions WHERE id = ?`, sessionID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("trainings")
	notify.Send(h.db, h.cfg, h.teamMembersAndParents(teamID),
		"trainings", "Training abgesagt", "Eine Trainingseinheit wurde abgesagt", "/termine")
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/trainings/{id}/note — setzt das Hinweisfeld eines Trainings.
// Berechtigung: Trainer des Teams / Vorstand / sportliche_leitung / Admin.
// Atomar mit der Debounce-Queue (5-min-Push); leerer Text entfernt die pending-Row.
func (h *Handler) UpdateTrainingNote(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(req.Note) > 200 {
		http.Error(w, "note too long", http.StatusBadRequest)
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

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(r.Context(),
		`UPDATE training_sessions SET note = ? WHERE id = ?`, req.Note, sessionID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(req.Note) == "" {
		if _, err = tx.ExecContext(r.Context(),
			`DELETE FROM pending_event_notes_push WHERE ref_type='training' AND ref_id=?`,
			sessionID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		if _, err = tx.ExecContext(r.Context(), `
			INSERT INTO pending_event_notes_push (ref_type, ref_id, note_text, notify_after, updated_by)
			VALUES ('training', ?, ?, datetime('now', '+5 minutes'), ?)
			ON CONFLICT(ref_type, ref_id) DO UPDATE SET
				note_text    = excluded.note_text,
				notify_after = excluded.notify_after,
				updated_by   = excluded.updated_by`,
			sessionID, req.Note, claims.UserID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("event-note")
	w.WriteHeader(http.StatusOK)
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
		VenueID           *int   `json:"venue_id"`
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
	var venueIDVal interface{}
	if req.VenueID != nil {
		venueIDVal = *req.VenueID
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO training_sessions (team_id, season_id, title, date, start_time, end_time, venue_id, note, rsvp_opt_out, rsvp_require_reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.TeamID, req.SeasonID, req.Title, req.Date, req.StartTime, req.EndTime, venueIDVal, req.Note, req.RsvpOptOut, req.RsvpRequireReason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateSession: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	h.db.ExecContext(r.Context(), `
		INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at, absence_id)
		SELECT ?, km.member_id, m.user_id, 'declined', a.type, datetime('now'), a.id
		FROM member_absences a
		JOIN members m ON m.id = a.member_id
		JOIN kader_members km ON km.member_id = a.member_id
		JOIN kader k ON k.id = km.kader_id AND k.team_id = ? AND k.season_id = ?
		WHERE ? BETWEEN a.start_date AND a.end_date
		ON CONFLICT(training_id, member_id) DO NOTHING`,
		id, req.TeamID, req.SeasonID, req.Date)

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
		Title             string  `json:"title"`
		Date              string  `json:"date"`
		StartTime         string  `json:"start_time"`
		EndTime           string  `json:"end_time"`
		VenueID           *int    `json:"venue_id"`
		Note              *string `json:"note"`
		Status            string  `json:"status"`
		CancelReason      string  `json:"cancel_reason"`
		RsvpOptOut        *int    `json:"rsvp_opt_out,omitempty"`
		RsvpRequireReason *int    `json:"rsvp_require_reason,omitempty"`
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
	var venueIDVal interface{}
	if req.VenueID != nil {
		venueIDVal = *req.VenueID
	}
	_, err = h.db.ExecContext(r.Context(),
		`UPDATE training_sessions SET title=?, date=?, start_time=?, end_time=?, venue_id=?, status=?, cancel_reason=? WHERE id=?`,
		req.Title, req.Date, req.StartTime, req.EndTime, venueIDVal, status, req.CancelReason, sessionID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// note ist Tri-State: fehlt das Feld, bleibt der Hinweis unverändert (er wird
	// über PUT /api/trainings/{id}/note + Debounce-Queue gepflegt, nicht hier).
	if req.Note != nil {
		if _, err = h.db.ExecContext(r.Context(),
			`UPDATE training_sessions SET note=? WHERE id=?`, *req.Note, sessionID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	// Partial-Update: rsvp_opt_out / rsvp_require_reason nur setzen, wenn im Request enthalten.
	if req.RsvpOptOut != nil || req.RsvpRequireReason != nil {
		setParts := []string{}
		setArgs := []interface{}{}
		if req.RsvpOptOut != nil {
			setParts = append(setParts, "rsvp_opt_out=?")
			setArgs = append(setArgs, *req.RsvpOptOut)
		}
		if req.RsvpRequireReason != nil {
			setParts = append(setParts, "rsvp_require_reason=?")
			setArgs = append(setArgs, *req.RsvpRequireReason)
		}
		setArgs = append(setArgs, sessionID)
		if _, err = h.db.ExecContext(r.Context(),
			`UPDATE training_sessions SET `+strings.Join(setParts, ", ")+` WHERE id=?`, setArgs...); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	h.hub.Broadcast("trainings")
	notify.Send(h.db, h.cfg, h.teamMembersAndParents(teamID),
		"trainings", "Training geändert", "Eine Trainingseinheit wurde aktualisiert", fmt.Sprintf("/termine?focus=training-%d", sessionID))
	w.WriteHeader(http.StatusNoContent)
}

type childRSVP struct {
	MemberID int     `json:"member_id"`
	Name     string  `json:"name"`
	RSVP     *string `json:"rsvp"`
}

type sessionVenueRef struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Note       string `json:"note"`
}

type sessionListItem struct {
	ID                int              `json:"id"`
	SeriesID          *int             `json:"series_id,omitempty"`
	TeamID            int              `json:"team_id"`
	TeamName          string           `json:"team_name"`
	SeasonID          int              `json:"season_id"`
	Title             string           `json:"title"`
	Date              string           `json:"date"`
	StartTime         string           `json:"start_time"`
	EndTime           string           `json:"end_time"`
	Venue             *sessionVenueRef `json:"venue,omitempty"`
	Note              string           `json:"note"`
	Status            string           `json:"status"`
	CancelReason      string           `json:"cancel_reason,omitempty"`
	ConfirmedCount    int              `json:"confirmed_count"`
	DeclinedCount     int              `json:"declined_count"`
	MaybeCount        int              `json:"maybe_count"`
	MyRSVP            *string          `json:"my_rsvp"`
	MyRSVPLocked      bool             `json:"my_rsvp_locked"`
	ChildrenRSVP      []childRSVP      `json:"children_rsvp,omitempty"`
	RsvpOptOut        int              `json:"rsvp_opt_out"`
	RsvpRequireReason int              `json:"rsvp_require_reason"`
	RsvpLocksAt       string           `json:"rsvp_locks_at,omitempty"`
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
	if claims.Role == "admin" || claims.HasFunction("vorstand") || claims.HasFunction("sportliche_leitung") {
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
			conds = append(conds, `ts.team_id IN (
				SELECT DISTINCT tm.team_id FROM player_memberships tm
				JOIN members m ON m.id = tm.member_id
				JOIN family_links fl ON fl.member_id = m.id
				WHERE fl.parent_user_id = ?
				UNION
				SELECT DISTINCT k.team_id FROM kader_extended_members kem
				JOIN kader k ON k.id = kem.kader_id
				JOIN family_links fl ON fl.member_id = kem.member_id
				WHERE fl.parent_user_id = ?)`)
			teamArgs = append(teamArgs, claims.UserID, claims.UserID)
		}
		conds = append(conds, `ts.team_id IN (
				SELECT DISTINCT tm.team_id FROM player_memberships tm JOIN members m ON m.id = tm.member_id WHERE m.user_id = ?
				UNION
				SELECT DISTINCT k.team_id FROM kader_extended_members kem
				JOIN kader k ON k.id = kem.kader_id
				JOIN members m2 ON m2.id = kem.member_id WHERE m2.user_id = ?)`)
		teamArgs = append(teamArgs, claims.UserID, claims.UserID)
		teamSQL = "(" + strings.Join(conds, " OR ") + ")"
	}

	// Order must match the ?-markers in the query:
	// 1. member_id (my_rsvp subquery), 2. member_id (my_rsvp_locked subquery), 3. teamArgs (WHERE), 4. from, 5. to, 6. optional team filter
	args := append([]any{memberID, memberID}, teamArgs...)
	args = append(args, from, to)
	optTeamFilter := ""
	if teamFilter != "" {
		optTeamFilter = "AND ts.team_id = ?"
		args = append(args, teamFilter)
	}

	query := fmt.Sprintf(`
		SELECT ts.id, ts.series_id, ts.team_id, COALESCE(`+appdb.TeamDisplayShort("t")+`, t.name, ''), ts.season_id, ts.title, ts.date, ts.start_time, ts.end_time,
		       ts.note, ts.status, ts.cancel_reason,
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
		       (SELECT absence_id IS NOT NULL FROM training_responses WHERE training_id = ts.id AND member_id = ? LIMIT 1),
		       ts.rsvp_opt_out, ts.rsvp_require_reason,
		       v.id, v.name, v.street, v.city, v.postal_code, v.note
		FROM training_sessions ts
		LEFT JOIN teams t ON t.id = ts.team_id
		LEFT JOIN training_responses tr ON tr.training_id = ts.id
		LEFT JOIN venues v ON v.id = ts.venue_id
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
		var myRSVPLocked sql.NullInt64
		var vID sql.NullInt64
		var vName, vStreet, vCity, vPostal, vNote sql.NullString
		err := rows.Scan(
			&s.ID, &seriesID, &s.TeamID, &s.TeamName, &s.SeasonID, &s.Title, &s.Date, &s.StartTime, &s.EndTime,
			&s.Note, &s.Status, &s.CancelReason,
			&s.ConfirmedCount, &s.DeclinedCount, &s.MaybeCount, &myRSVP, &myRSVPLocked,
			&s.RsvpOptOut, &s.RsvpRequireReason,
			&vID, &vName, &vStreet, &vCity, &vPostal, &vNote)
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
		s.MyRSVPLocked = myRSVPLocked.Valid && myRSVPLocked.Int64 == 1
		if vID.Valid {
			s.Venue = &sessionVenueRef{
				ID: int(vID.Int64), Name: vName.String, Street: vStreet.String,
				City: vCity.String, PostalCode: vPostal.String, Note: vNote.String,
			}
		}
		if locksAt, err := trainingLocksAt(s.Date, s.StartTime); err == nil {
			s.RsvpLocksAt = locksAt.Format(time.RFC3339)
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
	var vID sql.NullInt64
	var vName, vStreet, vCity, vPostal, vNote sql.NullString
	err = h.db.QueryRowContext(r.Context(), `
		SELECT ts.id, ts.series_id, ts.team_id, COALESCE(`+appdb.TeamDisplayName("t")+`, t.name, ''), ts.season_id, ts.title, ts.date, ts.start_time, ts.end_time,
		       ts.note, ts.status, ts.cancel_reason,
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
		       ts.rsvp_opt_out, ts.rsvp_require_reason,
		       v.id, v.name, v.street, v.city, v.postal_code, v.note
		FROM training_sessions ts
		LEFT JOIN teams t ON t.id = ts.team_id
		LEFT JOIN venues v ON v.id = ts.venue_id
		WHERE ts.id = ?`, memberID, sessionID).Scan(
		&s.ID, &seriesID, &s.TeamID, &s.TeamName, &s.SeasonID, &s.Title, &s.Date, &s.StartTime, &s.EndTime,
		&s.Note, &s.Status, &s.CancelReason,
		&s.ConfirmedCount, &s.DeclinedCount, &s.MaybeCount, &myRSVP,
		&s.RsvpOptOut, &s.RsvpRequireReason,
		&vID, &vName, &vStreet, &vCity, &vPostal, &vNote)
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
	if vID.Valid {
		s.Venue = &sessionVenueRef{
			ID: int(vID.Int64), Name: vName.String, Street: vStreet.String,
			City: vCity.String, PostalCode: vPostal.String, Note: vNote.String,
		}
	}
	if locksAt, err := trainingLocksAt(s.Date, s.StartTime); err == nil {
		s.RsvpLocksAt = locksAt.Format(time.RFC3339)
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

	var existingAbsenceID sql.NullInt64
	h.db.QueryRowContext(r.Context(),
		`SELECT absence_id FROM training_responses WHERE training_id = ? AND member_id = ?`,
		sessionID, memberID).Scan(&existingAbsenceID)
	if existingAbsenceID.Valid {
		http.Error(w, "response is locked by an absence", http.StatusForbidden)
		return
	}

	if !claims.CanOverrideRSVPCutoff() {
		var sessDate, sessStart string
		if err := h.db.QueryRowContext(r.Context(),
			`SELECT date(date), substr(start_time,1,5) FROM training_sessions WHERE id = ?`,
			sessionID).Scan(&sessDate, &sessStart); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		locksAt, err := trainingLocksAt(sessDate, sessStart)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if h.now().After(locksAt) {
			writeRSVPLocked(w, "Training kann nur bis 2 Stunden vor Beginn umgesagt werden.", locksAt)
			return
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

func writeRSVPLocked(w http.ResponseWriter, message string, locksAt time.Time) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":    "rsvp_locked",
		"message":  message,
		"locks_at": locksAt.UTC().Format(time.RFC3339),
	})
}

type attendanceItem struct {
	MemberID   int     `json:"member_id"`
	MemberName string  `json:"member_name"`
	IsExtended bool    `json:"is_extended"`
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
		SELECT member_id, member_name, is_extended, rsvp_status, reason, present
		FROM (
			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       0 AS is_extended,
			       tr.status AS rsvp_status,
			       tr.reason AS reason,
			       ta.present AS present
			FROM members m
			JOIN player_memberships pm ON pm.member_id = m.id AND pm.team_id = ? AND pm.season_id = ?
			LEFT JOIN training_responses tr ON tr.training_id = ? AND tr.member_id = m.id
			LEFT JOIN training_attendances ta ON ta.training_id = ? AND ta.member_id = m.id

			UNION

			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       1 AS is_extended,
			       tr.status AS rsvp_status,
			       tr.reason AS reason,
			       ta.present AS present
			FROM members m
			JOIN kader_extended_members kem ON kem.member_id = m.id
			JOIN kader k ON k.id = kem.kader_id AND k.team_id = ? AND k.season_id = ?
			LEFT JOIN training_responses tr ON tr.training_id = ? AND tr.member_id = m.id
			LEFT JOIN training_attendances ta ON ta.training_id = ? AND ta.member_id = m.id
			WHERE NOT EXISTS (
				SELECT 1 FROM player_memberships pm WHERE pm.member_id = m.id AND pm.team_id = ? AND pm.season_id = ?
			)
		)
		ORDER BY member_name`, teamID, seasonID, sessionID, sessionID, teamID, seasonID, sessionID, sessionID, teamID, seasonID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAttendances: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []attendanceItem{}
	for rows.Next() {
		var item attendanceItem
		var isExtended int
		var rsvp, reason sql.NullString
		var present sql.NullInt64
		rows.Scan(&item.MemberID, &item.MemberName, &isExtended, &rsvp, &reason, &present)
		item.IsExtended = isExtended == 1
		if rsvp.Valid {
			item.RSVPStatus = &rsvp.String
		} else if rsvpOptOut == 1 && !item.IsExtended {
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

	var teamID int
	var isPastOrToday bool
	err = h.db.QueryRowContext(r.Context(),
		`SELECT team_id, date(date) <= date('now') FROM training_sessions WHERE id = ?`, sessionID).Scan(&teamID, &isPastOrToday)
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

	if !isPastOrToday {
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
// Includes children who are in the regular (kader_members) OR extended
// (kader_extended_members) squad of the session's team. Extended-only children
// are not auto-confirmed under rsvp_opt_out — they must always respond explicitly.
func (h *Handler) attachChildrenRSVPToSessions(ctx context.Context, parentUserID int, items []sessionListItem) error {
	placeholders := make([]string, len(items))
	sessionIDs := make([]any, len(items))
	for i, s := range items {
		placeholders[i] = "?"
		sessionIDs[i] = s.ID
	}
	ph := strings.Join(placeholders, ",")
	// Two branches: regular squad (is_extended=0) and extended squad (is_extended=1).
	// The extended branch excludes members already counted as regular for the same
	// team/season so a child in both squads appears exactly once (regular wins).
	rows, err := h.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT ts.id, m.id, m.first_name || ' ' || m.last_name, tr.status, ts.rsvp_opt_out, 0 AS is_extended
		FROM training_sessions ts
		JOIN kader k ON k.team_id = ts.team_id AND k.season_id = ts.season_id
		JOIN kader_members km ON km.kader_id = k.id
		JOIN members m ON m.id = km.member_id
		JOIN family_links fl ON fl.member_id = m.id AND fl.parent_user_id = ?
		LEFT JOIN training_responses tr ON tr.training_id = ts.id AND tr.member_id = m.id
		WHERE ts.id IN (%s)

		UNION

		SELECT ts.id, m.id, m.first_name || ' ' || m.last_name, tr.status, ts.rsvp_opt_out, 1 AS is_extended
		FROM training_sessions ts
		JOIN kader k ON k.team_id = ts.team_id AND k.season_id = ts.season_id
		JOIN kader_extended_members kem ON kem.kader_id = k.id
		JOIN members m ON m.id = kem.member_id
		JOIN family_links fl ON fl.member_id = m.id AND fl.parent_user_id = ?
		LEFT JOIN training_responses tr ON tr.training_id = ts.id AND tr.member_id = m.id
		WHERE ts.id IN (%s)
		  AND NOT EXISTS (
			SELECT 1 FROM kader_members km2
			JOIN kader k2 ON k2.id = km2.kader_id
			WHERE km2.member_id = m.id AND k2.team_id = ts.team_id AND k2.season_id = ts.season_id
		  )

		ORDER BY 3`, ph, ph),
		append(append(append([]any{parentUserID}, sessionIDs...), parentUserID), sessionIDs...)...)
	if err != nil {
		return err
	}
	defer rows.Close()

	bySession := map[int][]childRSVP{}
	for rows.Next() {
		var sid int
		var c childRSVP
		var rsvp sql.NullString
		var rsvpOptOut, isExtended int
		rows.Scan(&sid, &c.MemberID, &c.Name, &rsvp, &rsvpOptOut, &isExtended)
		if rsvp.Valid {
			s := rsvp.String
			c.RSVP = &s
		} else if rsvpOptOut == 1 && isExtended == 0 {
			confirmed := "confirmed"
			c.RSVP = &confirmed
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
