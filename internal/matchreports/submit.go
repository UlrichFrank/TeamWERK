package matchreports

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type submitResp struct {
	State       string `json:"state"`
	SubmittedAt string `json:"submitted_at"`
}

// SubmitForReview transitioniert einen Draft in den Review-Zustand und
// benachrichtigt die Freigeber (Vereinsfunktion 'medien' + 'vorstand').
//
//	POST /api/match-reports/{id}/submit-for-review
//
// Vorbedingungen (siehe specs/match-reports/spec.md Requirement
// „POST /submit-for-review durch Autor" des spielbericht-medien-gate-Change):
//   - Requester = author_user_id (oder Admin)
//   - State = draft
//
// Nach dem Submit verliert der Autor die Edit-Rechte — die Freigeber
// übernehmen (siehe design.md D-3: kein Rückweg).
func (h *Handler) SubmitForReview(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parsePathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_id")
		return
	}

	// Autor + State laden für Guard-Check.
	var (
		authorID int
		state    string
		opponent string
	)
	err := h.db.QueryRow(
		`SELECT r.author_user_id, r.state, g.opponent
		 FROM match_reports r
		 JOIN games g ON g.id = r.game_id
		 WHERE r.id=?`, id,
	).Scan(&authorID, &state, &opponent)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		logErr("matchreports.SubmitForReview select", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if authorID != claims.UserID && claims.Role != auth.RoleAdmin {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	switch state {
	case StatePendingReview:
		writeErr(w, http.StatusConflict, "already_submitted")
		return
	case StatePublishing:
		writeErr(w, http.StatusConflict, "in_progress")
		return
	case StatePublished:
		writeErr(w, http.StatusConflict, "already_published")
		return
	case StatePublishFailed:
		writeErr(w, http.StatusConflict, "already_submitted")
		return
	}

	// Atomarer Übergang draft → pending_review. `RETURNING submitted_at` wäre
	// eleganter, aber wir wollen 2. Query-Ergebnis auch bei Race-Verlierer nicht
	// verschmutzen — daher UPDATE + separates SELECT.
	res, err := h.db.Exec(
		`UPDATE match_reports
		 SET state=?, submitted_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP
		 WHERE id=? AND state=?`,
		StatePendingReview, id, StateDraft,
	)
	if err != nil {
		logErr("matchreports.SubmitForReview update", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Race verloren — zwischenzeitlich hat jemand anders (oder ein
		// paralleler Klick des Autors) den State bereits weitergedreht.
		writeErr(w, http.StatusConflict, "already_submitted")
		return
	}

	var submittedAt sql.NullString
	if err := h.db.QueryRow(
		`SELECT submitted_at FROM match_reports WHERE id=?`, id,
	).Scan(&submittedAt); err != nil {
		logErr("matchreports.SubmitForReview readback", err, "id", id)
	}

	h.broadcast()

	// Push an alle Freigeber (fire-and-forget).
	body := fmt.Sprintf("Spielbericht %s wartet auf Freigabe", opponent)
	go notifyReviewers(h.db, h.cfg, "Neuer Spielbericht zur Prüfung", body,
		fmt.Sprintf("/spielberichte/%d", id))

	writeJSON(w, http.StatusOK, submitResp{
		State:       StatePendingReview,
		SubmittedAt: submittedAt.String,
	})
}
