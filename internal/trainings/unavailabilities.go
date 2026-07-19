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

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// unavailInfo ist der pro Mitglied ausgewiesene Serien-Abmelde-Status.
// permanent = true, wenn kein end_date gesetzt ist (Abmeldung bis Serien-Ende).
// ID ist die msu-Zeilen-ID (für die Trainer-Aktion „wieder anmelden" aus dem
// Termin-Detail heraus; bei Überlappung die permanent-bevorzugte Zeile).
type unavailInfo struct {
	ID        int    `json:"id"`
	Reason    string `json:"reason"`
	Permanent bool   `json:"permanent"`
}

// rowQuerier deckt sowohl *sql.DB als auch *sql.Tx ab, damit die Ableitung
// innerhalb einer Transaktion (SaveAttendances) wie außerhalb funktioniert.
type rowQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// sessionUnavailabilityForMember setzt die Ableitung aus der `serien-abmeldung`-Spec
// als reinen Lookup um: Eine Session (series_id=S, date=D) ist für Member X abgemeldet,
// gdw. eine Zeile in member_series_unavailabilities existiert mit member_id=X,
// training_series_id=S und D im (offenen) Fenster [start_date, end_date].
// Einzeltermine (series_id IS NULL) sind nie betroffen. Überlappende Zeilen sind
// harmlos (LIMIT 1 — sobald eine greift, gilt die Session als abgemeldet).
func sessionUnavailabilityForMember(ctx context.Context, q rowQuerier, sessionID, memberID int) (bool, string, error) {
	var reason string
	err := q.QueryRowContext(ctx, `
		SELECT COALESCE(msu.reason, '')
		FROM training_sessions ts
		JOIN member_series_unavailabilities msu
		  ON msu.training_series_id = ts.series_id
		 AND msu.member_id = ?
		 AND (msu.start_date IS NULL OR msu.start_date <= date(ts.date))
		 AND (msu.end_date   IS NULL OR msu.end_date   >= date(ts.date))
		WHERE ts.id = ? AND ts.series_id IS NOT NULL
		ORDER BY msu.id
		LIMIT 1`, memberID, sessionID).Scan(&reason)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, reason, nil
}

// unavailableMembersForSession liefert die Batch-Variante: alle Mitglieder mit einer
// greifenden Serien-Abmeldung für die gegebene Session, gemappt auf ihren Status.
// Pro Mitglied gewinnt die erste (permanent-bevorzugte) Zeile; Einzeltermine
// (series_id IS NULL) ergeben eine leere Map.
func unavailableMembersForSession(ctx context.Context, q rowQuerier, sessionID int) (map[int]unavailInfo, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT msu.member_id, msu.id, COALESCE(msu.reason, ''), CASE WHEN msu.end_date IS NULL THEN 1 ELSE 0 END
		FROM training_sessions ts
		JOIN member_series_unavailabilities msu
		  ON msu.training_series_id = ts.series_id
		 AND (msu.start_date IS NULL OR msu.start_date <= date(ts.date))
		 AND (msu.end_date   IS NULL OR msu.end_date   >= date(ts.date))
		WHERE ts.id = ? AND ts.series_id IS NOT NULL
		ORDER BY msu.member_id, (CASE WHEN msu.end_date IS NULL THEN 1 ELSE 0 END) DESC, msu.id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]unavailInfo{}
	for rows.Next() {
		var mid, id, permanent int
		var reason string
		if err := rows.Scan(&mid, &id, &reason, &permanent); err != nil {
			return nil, err
		}
		if _, ok := out[mid]; ok {
			continue // erste Zeile pro Mitglied gewinnt (permanent bevorzugt via ORDER BY)
		}
		out[mid] = unavailInfo{ID: id, Reason: reason, Permanent: permanent == 1}
	}
	return out, rows.Err()
}

// seriesUnavailability ist die API-Repräsentation einer Abmelde-Zeile.
type seriesUnavailability struct {
	ID         int     `json:"id"`
	MemberID   int     `json:"member_id"`
	MemberName string  `json:"member_name"`
	StartDate  *string `json:"start_date"`
	EndDate    *string `json:"end_date"`
	Reason     string  `json:"reason"`
	CreatedAt  string  `json:"created_at"`
}

// seriesTeamID lädt die team_id einer Serie; liefert (0, sql.ErrNoRows) wenn es die
// Serie nicht gibt (→ 404 durch den Aufrufer).
func (h *Handler) seriesTeamID(ctx context.Context, seriesID int) (int, error) {
	var teamID int
	err := h.db.QueryRowContext(ctx,
		`SELECT team_id FROM training_series WHERE id = ?`, seriesID).Scan(&teamID)
	return teamID, err
}

// GET /api/training-series/{id}/unavailabilities
func (h *Handler) ListSeriesUnavailabilities(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	seriesID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	teamID, err := h.seriesTeamID(r.Context(), seriesID)
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

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT msu.id, msu.member_id, m.first_name || ' ' || m.last_name,
		       date(msu.start_date), date(msu.end_date), msu.reason, msu.created_at
		FROM member_series_unavailabilities msu
		JOIN members m ON m.id = msu.member_id
		WHERE msu.training_series_id = ?
		ORDER BY m.last_name, m.first_name, msu.start_date`, seriesID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListSeriesUnavailabilities: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []seriesUnavailability{}
	for rows.Next() {
		var it seriesUnavailability
		var start, end sql.NullString
		if err := rows.Scan(&it.ID, &it.MemberID, &it.MemberName, &start, &end, &it.Reason, &it.CreatedAt); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if start.Valid {
			it.StartDate = &start.String
		}
		if end.Valid {
			it.EndDate = &end.String
		}
		items = append(items, it)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// POST /api/training-series/{id}/unavailabilities
func (h *Handler) CreateSeriesUnavailability(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	seriesID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		MemberID  int     `json:"member_id"`
		StartDate *string `json:"start_date"`
		EndDate   *string `json:"end_date"`
		Reason    string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.MemberID <= 0 {
		http.Error(w, "member_id required", http.StatusBadRequest)
		return
	}
	start := normalizeDatePtr(req.StartDate)
	end := normalizeDatePtr(req.EndDate)
	if start != nil && end != nil && *start > *end {
		http.Error(w, "start_date must not be after end_date", http.StatusBadRequest)
		return
	}

	teamID, err := h.seriesTeamID(r.Context(), seriesID)
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

	res, err := h.db.ExecContext(r.Context(), `
		INSERT INTO member_series_unavailabilities
		  (member_id, training_series_id, start_date, end_date, reason, created_by)
		VALUES (?, ?, ?, ?, ?, ?)`,
		req.MemberID, seriesID, start, end, strings.TrimSpace(req.Reason), claims.UserID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "already exists", http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			http.Error(w, "unknown member", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(os.Stderr, "CreateSeriesUnavailability: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	h.broadcastTeam(r.Context(), []int{teamID}, "training-unavailability-changed")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// DELETE /api/training-series/{id}/unavailabilities/{uid}
func (h *Handler) DeleteSeriesUnavailability(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	seriesID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	uid, err := strconv.Atoi(r.PathValue("uid"))
	if err != nil {
		http.Error(w, "invalid uid", http.StatusBadRequest)
		return
	}
	teamID, err := h.seriesTeamID(r.Context(), seriesID)
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

	res, err := h.db.ExecContext(r.Context(),
		`DELETE FROM member_series_unavailabilities WHERE id = ? AND training_series_id = ?`,
		uid, seriesID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.broadcastTeam(r.Context(), []int{teamID}, "training-unavailability-changed")
	w.WriteHeader(http.StatusNoContent)
}

// normalizeDatePtr wandelt leere/whitespace-Strings in NULL (nil) um, damit ein
// weggelassenes bzw. leeres Datum als offene Fenstergrenze gespeichert wird.
func normalizeDatePtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}
