package matchreports

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type createReq struct {
	GameID     int `json:"game_id"`
	DutySlotID int `json:"duty_slot_id"`
}

type createResp struct {
	ID int `json:"id"`
}

// Create legt einen Draft-Bericht an.
//
//	POST /api/match-reports
//	Body: { "game_id": int, "duty_slot_id": int }
//
// Vorbedingungen (siehe specs/match-reports/spec.md Requirement
// "Draft-Erstellung durch Slot-Owner"):
//   - Requester hat role IN ('presseteam','admin')
//   - Requester besitzt den referenzierten Duty-Slot (duty_slots.assigned_user_id)
//     ODER ist Admin (Admin darf für alle publizieren)
//   - Es existiert noch kein match_report für dieses Spiel (UNIQUE game_id)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !isPressTeamOrAdmin(claims) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.GameID <= 0 || req.DutySlotID <= 0 {
		writeErr(w, http.StatusBadRequest, "missing_fields")
		return
	}

	if !h.slotBelongsToUser(req.DutySlotID, claims.UserID, req.GameID, claims.Role == auth.RoleAdmin) {
		writeErr(w, http.StatusForbidden, "slot_not_owned")
		return
	}

	// Default-Titel aus Spieldatum + Gegner. Der Autor kann ihn im Formular
	// überschreiben; der User-Titel läuft dann durch `PUT /match-reports/{id}`.
	var matchDate, opponent string
	if err := h.db.QueryRow(
		`SELECT date, opponent FROM games WHERE id=?`, req.GameID,
	).Scan(&matchDate, &opponent); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeErr(w, http.StatusBadRequest, "game_not_found")
			return
		}
		logErr("matchreports.Create load game", err, "game", req.GameID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	matchDateUnix, err := parseMatchDate(matchDate)
	if err != nil {
		// Ungültiges Datum in games sollte nie vorkommen — Notfall-Fallback.
		logErr("matchreports.Create parse match_date", err, "game", req.GameID)
		matchDateUnix = 0
	}
	defaultTitle := BuildTitle(matchDateUnix, opponent)

	res, err := h.db.Exec(
		`INSERT INTO match_reports (game_id, duty_slot_id, author_user_id, state, title)
		 VALUES (?, ?, ?, ?, ?)`,
		req.GameID, req.DutySlotID, claims.UserID, StateDraft, defaultTitle)
	if err != nil {
		if isUniqueViolation(err) {
			writeErr(w, http.StatusConflict, "report_exists")
			return
		}
		logErr("matchreports.Create insert", err, "user", claims.UserID, "game", req.GameID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	id, _ := res.LastInsertId()

	h.broadcast()
	writeJSON(w, http.StatusCreated, createResp{ID: int(id)})
}

// Delete löscht einen Draft-Bericht (nur im State draft/publish_failed) und
// räumt die zugehörigen Bilder auf. Published-Berichte sind unlöschbar aus
// TeamWERK — dafür gibt es das Typo3-Backend.
//
//	DELETE /api/match-reports/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
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
		logErr("matchreports.Delete select", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	if authorID != claims.UserID && claims.Role != auth.RoleAdmin {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	if state != StateDraft && state != StatePublishFailed {
		writeErr(w, http.StatusConflict, "already_published")
		return
	}

	// Bilder-Dateien vom Draft löschen. DB-Referenzen fallen per ON DELETE
	// CASCADE mit dem Bericht mit.
	h.removeAllImageFiles(id)

	if _, err := h.db.Exec(`DELETE FROM match_reports WHERE id=?`, id); err != nil {
		logErr("matchreports.Delete", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	h.broadcast()
	w.WriteHeader(http.StatusNoContent)
}

// isPressTeamOrAdmin ist die zentrale Rolle-Prüfung. Elternteil-Users mit
// role=presseteam funktionieren gleich wie Members mit role=presseteam.
func isPressTeamOrAdmin(claims *auth.Claims) bool {
	return claims.Role == auth.RolePressTeam || claims.Role == auth.RoleAdmin
}

// isReviewer prüft, ob der Requester Match-Report-Freigaben durchführen darf:
// Vereinsfunktion 'medien' oder 'vorstand' oder System-Rolle 'admin'. Admin
// bypasst alle Vereinsfunktions-Checks (konsistent mit RequireClubFunction).
func isReviewer(claims *auth.Claims) bool {
	if claims == nil {
		return false
	}
	if claims.Role == auth.RoleAdmin {
		return true
	}
	return claims.HasFunction(ClubFunctionMedien) || claims.HasFunction("vorstand")
}

// guardMutation setzt die State-/Rollen-Matrix für alle mutierenden Handler
// (Update, Bild-Upload/Delete) durch. Liefert ("", 0), wenn der Zugriff erlaubt
// ist, sonst (errorCode, httpStatus) für writeErr.
//
// Regeln (spec.md des spielbericht-medien-gate):
//   - state=draft            → nur Autor (oder Admin)
//   - state=pending_review   → nur Freigeber (medien|vorstand|admin), NICHT Autor
//   - state=publish_failed   → nur Freigeber (analog pending_review)
//   - state=publishing       → 409 in_progress (niemand)
//   - state=published        → 409 already_published (niemand)
func guardMutation(claims *auth.Claims, authorID int, state string) (string, int) {
	switch state {
	case StateDraft:
		if authorID == claims.UserID || claims.Role == auth.RoleAdmin {
			return "", 0
		}
		return "forbidden", http.StatusForbidden
	case StatePendingReview, StatePublishFailed:
		if isReviewer(claims) {
			return "", 0
		}
		return "forbidden", http.StatusForbidden
	case StatePublishing:
		return "in_progress", http.StatusConflict
	case StatePublished:
		return "already_published", http.StatusConflict
	default:
		return "invalid_state", http.StatusConflict
	}
}

// matchReportDutyTypeName ist der name-Match für den Spielbericht-Duty-Type
// (Seed in Migration 020) — analog zu duties.matchReportDutyTypeName, per Name
// statt ID, damit Prod- und Test-DBs mit unterschiedlichen IDs beide greifen.
const matchReportDutyTypeName = "Spielbericht"

// slotBelongsToUser prüft, ob der Slot den Bericht für genau dieses Spiel
// autorisiert. Admin darf immer. Für alle anderen müssen drei Bedingungen
// zusammenkommen:
//   - der User ist dem Slot zugewiesen (duty_assignments,
//     status IN ('assigned','fulfilled') — cash_substitute zählt NICHT, der
//     User hat den Dienst finanziell abgelöst und kann nicht Autor sein),
//   - der Slot gehört zum übergebenen game_id (sonst könnte ein Reporter mit
//     seinem Slot für Spiel A den Bericht für ein fremdes Spiel B anlegen —
//     IDOR/Business-Logic-Lücke), und
//   - der Slot ist tatsächlich ein „Spielbericht"-Slot.
func (h *Handler) slotBelongsToUser(slotID, userID, gameID int, isAdmin bool) bool {
	if isAdmin {
		return true
	}
	var exists int
	err := h.db.QueryRow(
		`SELECT 1 FROM duty_assignments da
		 JOIN duty_slots ds ON ds.id = da.duty_slot_id
		 JOIN duty_types dt ON dt.id = ds.duty_type_id
		 WHERE da.duty_slot_id=? AND da.user_id=?
		   AND da.status IN ('assigned','fulfilled')
		   AND ds.game_id=?
		   AND dt.name=?
		 LIMIT 1`,
		slotID, userID, gameID, matchReportDutyTypeName,
	).Scan(&exists)
	return err == nil
}

// isUniqueViolation erkennt SQLite-UNIQUE-Constraint-Verletzungen. Die
// treiber-agnostische Prüfung matcht den Fehlertext, weil modernc/sqlite
// keine typisierten Constraint-Errors exportiert.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg, "UNIQUE constraint failed", "unique constraint failed")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) {
			// case-sensitive Substring-Suche reicht — SQLite-Fehlertexte sind stabil.
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
