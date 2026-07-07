package matchreports

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type updateReq struct {
	HomeGoals    *int   `json:"home_goals"`
	AwayGoals    *int   `json:"away_goals"`
	HomeGoalsHT  *int   `json:"home_goals_ht"`
	AwayGoalsHT  *int   `json:"away_goals_ht"`
	Tournament   *bool  `json:"tournament"`
	Abstract     string `json:"abstract"`
	BodyMarkdown string `json:"body_md"`
}

// Update patcht einen Bericht. Zugriff nach der State-/Rollen-Matrix
// (spielbericht-medien-gate spec.md, Requirement „Draft-Update nur durch
// Autor im State draft"):
//
//   - state=draft: nur Autor (oder Admin)
//
//   - state=pending_review / publish_failed: nur Freigeber (medien|vorstand|admin)
//
//   - state=publishing: 409 in_progress
//
//   - state=published: 409 already_published
//
//     PUT /api/match-reports/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req updateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.Abstract) > 500 {
		writeErr(w, http.StatusBadRequest, "abstract_too_long")
		return
	}

	var authorID int
	var state string
	err := h.db.QueryRow(
		`SELECT author_user_id, state FROM match_reports WHERE id=?`, id,
	).Scan(&authorID, &state)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		logErr("matchreports.Update select", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	// State-/Rollen-Matrix. `guardMutation` liefert bei Verboten den passenden
	// HTTP-Fehler; bei Erlaubnis nil.
	if code, status := guardMutation(claims, authorID, state); code != "" {
		writeErr(w, status, code)
		return
	}

	tournament := 0
	if req.Tournament != nil && *req.Tournament {
		tournament = 1
	}

	_, err = h.db.Exec(
		`UPDATE match_reports
		 SET home_goals=?, away_goals=?, home_goals_ht=?, away_goals_ht=?,
		     tournament=?, abstract=?, body_md=?,
		     updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		nullableInt(req.HomeGoals), nullableInt(req.AwayGoals),
		nullableInt(req.HomeGoalsHT), nullableInt(req.AwayGoalsHT),
		tournament, req.Abstract, req.BodyMarkdown,
		id,
	)
	if err != nil {
		logErr("matchreports.Update", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	h.broadcast()
	w.WriteHeader(http.StatusOK)
}

// nullableInt wandelt einen optionalen int in einen sql-tauglichen Wert.
// Nil ⇒ NULL, sonst der Zahlenwert.
func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
