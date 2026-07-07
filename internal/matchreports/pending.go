package matchreports

import (
	"database/sql"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// PendingItem beschreibt einen Bericht in der Freigabe-Warteschlange.
type PendingItem struct {
	ID          int    `json:"id"`
	GameID      int    `json:"game_id"`
	Opponent    string `json:"opponent"`
	MatchDate   string `json:"match_date"`
	SubmittedAt string `json:"submitted_at"`
	AuthorName  string `json:"author_name"`
	ImageCount  int    `json:"image_count"`
}

// Pending liefert die Liste aller Berichte im State pending_review an
// Freigeber (Vereinsfunktion medien/vorstand/admin), sortiert nach
// submitted_at aufsteigend.
//
//	GET /api/match-reports/pending
//
// Der Router-Guard `RequireClubFunction("medien","vorstand")` filtert bereits;
// hier defensiv trotzdem ein zusätzlicher isReviewer-Check.
func (h *Handler) Pending(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !isReviewer(claims) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}

	rows, err := h.db.Query(
		`SELECT r.id, r.game_id, g.opponent, g.date,
		        r.submitted_at,
		        COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email, '?'),
		        (SELECT COUNT(*) FROM match_report_images i WHERE i.report_id = r.id)
		 FROM match_reports r
		 JOIN games g ON g.id = r.game_id
		 JOIN users u ON u.id = r.author_user_id
		 WHERE r.state = ?
		 ORDER BY r.submitted_at ASC`,
		StatePendingReview,
	)
	if err != nil {
		logErr("matchreports.Pending query", err)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	defer rows.Close()

	out := make([]PendingItem, 0)
	for rows.Next() {
		var it PendingItem
		var submitted sql.NullString
		var authorName sql.NullString
		if err := rows.Scan(&it.ID, &it.GameID, &it.Opponent, &it.MatchDate,
			&submitted, &authorName, &it.ImageCount); err != nil {
			logErr("matchreports.Pending scan", err)
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
		it.SubmittedAt = submitted.String
		it.AuthorName = authorName.String
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		logErr("matchreports.Pending rows", err)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
