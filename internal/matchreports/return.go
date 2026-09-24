package matchreports

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/notify"
)

// MaxReviewCommentLen begrenzt den Rückgabe-Kommentar (Zeichen, nicht Bytes).
const MaxReviewCommentLen = 2000

type returnReq struct {
	Comment string `json:"comment"`
}

// Return gibt einen eingereichten Bericht mit Kommentar an den Autor zurück.
//
//	POST /api/match-reports/{id}/return   {"comment": "…"}
//
// Übergang pending_review → draft: der Autor darf danach wieder bearbeiten
// und erneut einreichen (guardMutation: draft = Autor). Nur Freigeber
// (medien|vorstand|admin); der Kommentar ist Pflicht — eine Rückgabe ohne
// Hinweis, was zu ändern ist, ließe den Autor raten.
//
// Reihenfolge der Prüfungen: Rolle → Objekt/State → Kommentar, damit die
// Objektrechte-Matrix keinen Befund „Validierung vor Autorisierung" sieht.
func (h *Handler) Return(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !isReviewer(claims) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	id, ok := parsePathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_id")
		return
	}

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
		logErr("matchreports.Return select", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if state != StatePendingReview {
		writeErr(w, http.StatusConflict, "not_pending_review")
		return
	}

	var req returnReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request")
		return
	}
	comment := strings.TrimSpace(req.Comment)
	if comment == "" {
		writeErr(w, http.StatusBadRequest, "comment_required")
		return
	}
	if utf8.RuneCountInString(comment) > MaxReviewCommentLen {
		writeErr(w, http.StatusBadRequest, "comment_too_long")
		return
	}

	res, err := h.db.Exec(
		`UPDATE match_reports
		 SET state=?, review_comment=?, returned_at=CURRENT_TIMESTAMP,
		     reviewer_user_id=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=? AND state=?`,
		StateDraft, comment, claims.UserID, id, StatePendingReview,
	)
	if err != nil {
		logErr("matchreports.Return update", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Race: zwischenzeitlich veröffentlicht, gelöscht oder schon zurückgegeben.
		writeErr(w, http.StatusConflict, "not_pending_review")
		return
	}

	h.broadcast()

	// Der Autor muss handeln — Push und (präferenzgesteuert) Mail, anders als
	// die reine Freigeber-Info beim Einreichen.
	body := fmt.Sprintf("Spielbericht %s: %s", opponent, comment)
	notify.SendAsync(h.db, h.cfg, []int{authorID}, "operativ",
		"Spielbericht zurückgegeben", body, fmt.Sprintf("/spielberichte/%d", id))

	writeJSON(w, http.StatusOK, map[string]string{"state": StateDraft})
}
