package absences

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

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
	CanEdit    bool   `json:"can_edit"`
	Type       string `json:"type"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Note       string `json:"note"`
	CreatedAt  string `json:"created_at"`
	CreatedBy  int    `json:"created_by"`
	IsOwn      bool   `json:"is_own"`
}

type previewEvent struct {
	EventType string `json:"event_type"` // "training" or "game"
	EventID   int    `json:"event_id"`
	Name      string `json:"name"`
	Date      string `json:"date"`
	Pending   bool   `json:"pending"` // true = no prior response, will be newly declined
}

type previewKey struct {
	eventType string
	eventID   int
}

// memberIDForUser returns the member ID linked to the given user, or 0.
func (h *Handler) memberIDForUser(ctx context.Context, userID int) int {
	var id int
	h.db.QueryRowContext(ctx, `SELECT id FROM members WHERE user_id = ?`, userID).Scan(&id)
	return id
}

// resolveMemberID returns the member_id the caller may act on, or 0 + error message.
func (h *Handler) resolveMemberID(ctx context.Context, claims *auth.Claims, requestedMemberID int) (int, string) {
	if requestedMemberID != 0 {
		if claims.Role == "admin" {
			return requestedMemberID, ""
		}
		var count int
		h.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM family_links WHERE parent_user_id = ? AND member_id = ?`,
			claims.UserID, requestedMemberID).Scan(&count)
		if count > 0 {
			return requestedMemberID, ""
		}
		return 0, "forbidden"
	}
	if claims.IsParent {
		return 0, "member_id required for elternteil"
	}
	mid := h.memberIDForUser(ctx, claims.UserID)
	if mid == 0 {
		return 0, "account is not linked to a member record"
	}
	return mid, ""
}

// parseMemberIDs normalizes the requested member IDs from either a CSV
// "member_ids" param/body field or a legacy single "member_id". Returns
// nil when neither is set, signaling "fall back to the caller's own member".
func parseMemberIDs(csv, single string) []int {
	if csv != "" {
		parts := strings.Split(csv, ",")
		ids := make([]int, 0, len(parts))
		for _, p := range parts {
			if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n > 0 {
				ids = append(ids, n)
			}
		}
		return ids
	}
	if single != "" {
		if n, err := strconv.Atoi(single); err == nil && n > 0 {
			return []int{n}
		}
	}
	return nil
}

// resolveMemberIDs runs resolveMemberID for each requested ID and aggregates
// the results. If the requested list is empty, it falls back to the caller's
// own member exactly like resolveMemberID(0). On any failure it short-circuits
// with the appropriate HTTP status and message.
func (h *Handler) resolveMemberIDs(ctx context.Context, claims *auth.Claims, requested []int) ([]int, int, string) {
	if len(requested) == 0 {
		mid, errMsg := h.resolveMemberID(ctx, claims, 0)
		if errMsg != "" {
			if errMsg == "forbidden" {
				return nil, http.StatusForbidden, "forbidden"
			}
			return nil, http.StatusBadRequest, errMsg
		}
		return []int{mid}, 0, ""
	}
	resolved := make([]int, 0, len(requested))
	seen := map[int]bool{}
	for _, mid := range requested {
		if seen[mid] {
			continue // ignore duplicates in the input
		}
		seen[mid] = true
		r, errMsg := h.resolveMemberID(ctx, claims, mid)
		if errMsg != "" {
			if errMsg == "forbidden" {
				return nil, http.StatusForbidden, "forbidden"
			}
			return nil, http.StatusBadRequest, errMsg
		}
		resolved = append(resolved, r)
	}
	return resolved, 0, ""
}

// GET /api/absences/preview
// Accepts either ?member_id=N (single) or ?member_ids=1,2,3 (CSV).
// With multiple members, returned events are the deduplicated union per
// (event_type, event_id) so a shared training/game appears once.
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	if from == "" || to == "" {
		http.Error(w, "from and to required", http.StatusBadRequest)
		return
	}

	requested := parseMemberIDs(q.Get("member_ids"), q.Get("member_id"))
	resolvedIDs, errStatus, errMsg := h.resolveMemberIDs(r.Context(), claims, requested)
	if errMsg != "" {
		http.Error(w, errMsg, errStatus)
		return
	}

	seen := map[previewKey]previewEvent{}

	for _, memberID := range resolvedIDs {
		if err := h.collectPreviewEvents(r.Context(), memberID, from, to, seen); err != nil {
			fmt.Fprintf(os.Stderr, "absences preview: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	events := make([]previewEvent, 0, len(seen))
	for _, ev := range seen {
		events = append(events, ev)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Date != events[j].Date {
			return events[i].Date < events[j].Date
		}
		if events[i].EventType != events[j].EventType {
			return events[i].EventType < events[j].EventType
		}
		return events[i].EventID < events[j].EventID
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// collectPreviewEvents fills seen with affected events for a single member.
// A "confirmed" entry overrides a "pending" one for the same key.
func (h *Handler) collectPreviewEvents(ctx context.Context, memberID int, from, to string, seen map[previewKey]previewEvent) error {
	// confirmed training responses
	tRows, err := h.db.QueryContext(ctx, `
		SELECT ts.id, COALESCE(ts.title, ''), ts.date
		FROM training_sessions ts
		JOIN training_responses tr ON tr.training_id = ts.id AND tr.member_id = ? AND tr.status = 'confirmed'
		WHERE ts.date BETWEEN ? AND ?`, memberID, from, to)
	if err != nil {
		return err
	}
	for tRows.Next() {
		var ev previewEvent
		ev.EventType = "training"
		tRows.Scan(&ev.EventID, &ev.Name, &ev.Date)
		if ev.Name == "" {
			ev.Name = "Training"
		}
		ev.Date = ev.Date[:10]
		seen[previewKey{ev.EventType, ev.EventID}] = ev // confirmed wins over pending
	}
	tRows.Close()

	// pending training sessions: member is in kader but has no response yet
	pRows, err := h.db.QueryContext(ctx, `
		SELECT ts.id, COALESCE(ts.title, ''), ts.date
		FROM training_sessions ts
		JOIN kader_members km ON km.member_id = ?
		JOIN kader k ON k.id = km.kader_id AND k.team_id = ts.team_id
		WHERE ts.date BETWEEN ? AND ?
		  AND NOT EXISTS (
		    SELECT 1 FROM training_responses tr
		    WHERE tr.training_id = ts.id AND tr.member_id = ?
		  )`, memberID, from, to, memberID)
	if err != nil {
		return err
	}
	for pRows.Next() {
		var ev previewEvent
		ev.EventType = "training"
		ev.Pending = true
		pRows.Scan(&ev.EventID, &ev.Name, &ev.Date)
		if ev.Name == "" {
			ev.Name = "Training"
		}
		ev.Date = ev.Date[:10]
		k := previewKey{ev.EventType, ev.EventID}
		if _, exists := seen[k]; !exists {
			seen[k] = ev
		}
	}
	pRows.Close()

	// confirmed game responses
	gRows, err := h.db.QueryContext(ctx, `
		SELECT g.id, g.opponent, g.date
		FROM games g
		JOIN game_responses gr ON gr.game_id = g.id AND gr.member_id = ? AND gr.status = 'confirmed'
		WHERE g.date BETWEEN ? AND ?`, memberID, from, to)
	if err != nil {
		return err
	}
	for gRows.Next() {
		var ev previewEvent
		ev.EventType = "game"
		gRows.Scan(&ev.EventID, &ev.Name, &ev.Date)
		ev.Date = ev.Date[:10]
		seen[previewKey{ev.EventType, ev.EventID}] = ev
	}
	gRows.Close()
	return nil
}

// POST /api/absences
// Accepts either { member_id: N } (legacy single) or { member_ids: [..] }
// (multi). With multiple members the semantics are all-or-nothing: any overlap
// or permission failure aborts the whole request and nothing is inserted.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		MemberID  int    `json:"member_id"`
		MemberIDs []int  `json:"member_ids"`
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

	requested := req.MemberIDs
	if len(requested) == 0 && req.MemberID > 0 {
		requested = []int{req.MemberID}
	}
	resolvedIDs, errStatus, errMsg := h.resolveMemberIDs(r.Context(), claims, requested)
	if errMsg != "" {
		http.Error(w, errMsg, errStatus)
		return
	}

	// Phase 1 — collect all overlap conflicts before touching the DB.
	type conflictEntry struct {
		MemberID   int    `json:"member_id"`
		MemberName string `json:"member_name"`
	}
	var conflicts []conflictEntry
	for _, mid := range resolvedIDs {
		var n int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM member_absences
			 WHERE member_id = ? AND type = ? AND start_date <= ? AND end_date >= ?`,
			mid, req.Type, req.EndDate, req.StartDate).Scan(&n)
		if n > 0 {
			var first, last string
			h.db.QueryRowContext(r.Context(),
				`SELECT first_name, last_name FROM members WHERE id = ?`, mid).
				Scan(&first, &last)
			conflicts = append(conflicts, conflictEntry{
				MemberID:   mid,
				MemberName: strings.TrimSpace(first + " " + last),
			})
		}
	}
	if len(conflicts) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		// Single-member legacy callers expect the old shape; multi-member callers
		// rely on the conflicts list to identify which child blocked the request.
		body := map[string]any{"error": "overlap"}
		if len(resolvedIDs) > 1 || len(req.MemberIDs) > 0 {
			body["conflicts"] = conflicts
		}
		json.NewEncoder(w).Encode(body)
		return
	}

	// Phase 2 — insert + auto-decline per member in one transaction.
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	absenceIDs := make([]int64, 0, len(resolvedIDs))
	for _, mid := range resolvedIDs {
		res, err := tx.ExecContext(r.Context(),
			`INSERT INTO member_absences (member_id, type, start_date, end_date, note, created_by)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			mid, req.Type, req.StartDate, req.EndDate, req.Note, claims.UserID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "absences create: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		absenceID, _ := res.LastInsertId()
		absenceIDs = append(absenceIDs, absenceID)

		if _, err = tx.ExecContext(r.Context(), `
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
			mid, claims.UserID, req.Type, absenceID,
			mid, req.StartDate, req.EndDate); err != nil {
			fmt.Fprintf(os.Stderr, "absences auto-decline trainings: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if _, err = tx.ExecContext(r.Context(), `
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
			mid, claims.UserID, req.Type, absenceID,
			mid, req.StartDate, req.EndDate); err != nil {
			fmt.Fprintf(os.Stderr, "absences auto-decline games: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("absences")
	h.hub.Broadcast("trainings")
	h.hub.Broadcast("games")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	// Legacy single-member callers (or self-as-spieler) still get { id }.
	// Multi-member callers get { absence_ids: [...] } in input order.
	if len(req.MemberIDs) > 0 {
		json.NewEncoder(w).Encode(map[string]any{"absence_ids": absenceIDs})
	} else {
		json.NewEncoder(w).Encode(map[string]any{"id": absenceIDs[0]})
	}
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

// GET /api/absences/calendar?from=&to=[&show_team=true][&team_id=X]
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		http.Error(w, "from and to required", http.StatusBadRequest)
		return
	}
	showTeam := r.URL.Query().Get("show_team") == "true"
	teamIDStr := r.URL.Query().Get("team_id")

	// Own + children absences
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT a.id, a.member_id, m.first_name || ' ' || m.last_name,
		       a.type, a.start_date, a.end_date, a.note, a.created_by,
		       (a.created_by = ? OR m.user_id = ? OR EXISTS (
		         SELECT 1 FROM family_links fl WHERE fl.parent_user_id = ? AND fl.member_id = a.member_id
		       )) AS can_edit
		FROM member_absences a
		JOIN members m ON m.id = a.member_id
		WHERE a.start_date <= ? AND a.end_date >= ?
		  AND a.member_id IN (
		    SELECT id FROM members WHERE user_id = ?
		    UNION
		    SELECT fl.member_id FROM family_links fl WHERE fl.parent_user_id = ?
		  )
		ORDER BY a.start_date`, claims.UserID, claims.UserID, claims.UserID, to, from, claims.UserID, claims.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences calendar: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []absence{}
	ownIDs := map[int]struct{}{}
	for rows.Next() {
		var a absence
		var canEdit int
		rows.Scan(&a.ID, &a.MemberID, &a.MemberName, &a.Type, &a.StartDate, &a.EndDate, &a.Note, &a.CreatedBy, &canEdit)
		a.CanEdit = canEdit != 0 || claims.Role == "admin"
		a.StartDate = a.StartDate[:10]
		a.EndDate = a.EndDate[:10]
		a.IsOwn = true
		ownIDs[a.ID] = struct{}{}
		result = append(result, a)
	}

	// Team absences — only for authorized roles
	canSeeTeam := showTeam && (claims.Role == "admin" || claims.Role == "trainer" ||
		claims.HasFunction("sportvorstand") || claims.HasFunction("vorstand") || claims.IsTrainerLike())
	if canSeeTeam {
		var teamRows *sql.Rows
		var teamErr error
		if teamIDStr != "" {
			teamRows, teamErr = h.db.QueryContext(r.Context(), `
				SELECT a.id, a.member_id, m.first_name || ' ' || m.last_name,
				       a.type, a.start_date, a.end_date, a.note, a.created_by
				FROM member_absences a
				JOIN members m ON m.id = a.member_id
				WHERE a.start_date <= ? AND a.end_date >= ?
				  AND m.absences_public = 1
				  AND a.member_id IN (
				    SELECT tm.member_id FROM team_memberships tm
				    JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
				    WHERE tm.team_id IN (SELECT team_id FROM user_accessible_teams WHERE user_id = ?)
				      AND tm.team_id = ?
				  )
				  AND a.member_id NOT IN (
				    SELECT id FROM members WHERE user_id = ?
				    UNION
				    SELECT fl.member_id FROM family_links fl WHERE fl.parent_user_id = ?
				  )
				ORDER BY a.start_date`, to, from, claims.UserID, teamIDStr, claims.UserID, claims.UserID)
		} else {
			teamRows, teamErr = h.db.QueryContext(r.Context(), `
				SELECT a.id, a.member_id, m.first_name || ' ' || m.last_name,
				       a.type, a.start_date, a.end_date, a.note, a.created_by
				FROM member_absences a
				JOIN members m ON m.id = a.member_id
				WHERE a.start_date <= ? AND a.end_date >= ?
				  AND m.absences_public = 1
				  AND a.member_id IN (
				    SELECT tm.member_id FROM team_memberships tm
				    JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
				    WHERE tm.team_id IN (SELECT team_id FROM user_accessible_teams WHERE user_id = ?)
				  )
				  AND a.member_id NOT IN (
				    SELECT id FROM members WHERE user_id = ?
				    UNION
				    SELECT fl.member_id FROM family_links fl WHERE fl.parent_user_id = ?
				  )
				ORDER BY a.start_date`, to, from, claims.UserID, claims.UserID, claims.UserID)
		}
		if teamErr != nil {
			fmt.Fprintf(os.Stderr, "absences calendar team: %v\n", teamErr)
		} else {
			defer teamRows.Close()
			for teamRows.Next() {
				var a absence
				teamRows.Scan(&a.ID, &a.MemberID, &a.MemberName, &a.Type, &a.StartDate, &a.EndDate, &a.Note, &a.CreatedBy)
				if _, alreadyOwn := ownIDs[a.ID]; alreadyOwn {
					continue
				}
				a.StartDate = a.StartDate[:10]
				a.EndDate = a.EndDate[:10]
				a.IsOwn = false
				result = append(result, a)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PUT /api/absences/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var existing absence
	err = h.db.QueryRowContext(r.Context(),
		`SELECT id, member_id, created_by FROM member_absences WHERE id = ?`, id).
		Scan(&existing.ID, &existing.MemberID, &existing.CreatedBy)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if claims.Role != "admin" && existing.CreatedBy != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
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

	var overlapCount int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM member_absences
		 WHERE member_id = ? AND type = ? AND id != ?
		   AND start_date <= ? AND end_date >= ?`,
		existing.MemberID, req.Type, id, req.EndDate, req.StartDate).Scan(&overlapCount)
	if overlapCount > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "overlap"})
		return
	}

	_, err = h.db.ExecContext(r.Context(),
		`UPDATE member_absences SET type = ?, start_date = ?, end_date = ?, note = ? WHERE id = ?`,
		req.Type, req.StartDate, req.EndDate, req.Note, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences update: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Remove old auto-declines for this absence, then re-apply for new period
	h.db.ExecContext(r.Context(),
		`DELETE FROM training_responses WHERE absence_id = ?`, id)
	h.db.ExecContext(r.Context(),
		`DELETE FROM game_responses WHERE absence_id = ?`, id)

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
		existing.MemberID, claims.UserID, req.Type, id,
		existing.MemberID, req.StartDate, req.EndDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences update auto-decline trainings: %v\n", err)
	}

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
		existing.MemberID, claims.UserID, req.Type, id,
		existing.MemberID, req.StartDate, req.EndDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "absences update auto-decline games: %v\n", err)
	}

	h.hub.Broadcast("absences")
	h.hub.Broadcast("trainings")
	h.hub.Broadcast("games")
	w.WriteHeader(http.StatusNoContent)
}
