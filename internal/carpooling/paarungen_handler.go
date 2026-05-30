package carpooling

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
)

// POST /api/mitfahrt-paarungen
func (h *Handler) RequestPairing(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	var body struct {
		BieteID int `json:"bieteId"`
		SucheID int `json:"sucheId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.BieteID == 0 || body.SucheID == 0 {
		http.Error(w, "bieteId and sucheId required", http.StatusBadRequest)
		return
	}

	// Load biete entry
	var bieteUserID int
	var bietePlaetze sql.NullInt64
	var bieteGameID int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT user_id, plaetze, game_id FROM mitfahrgelegenheiten WHERE id = ? AND typ = 'biete'`,
		body.BieteID).Scan(&bieteUserID, &bietePlaetze, &bieteGameID)
	if err == sql.ErrNoRows {
		http.Error(w, "biete entry not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Load suche entry
	var sucheUserID int
	var suchePlaetze sql.NullInt64
	var sucheGameID int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT user_id, plaetze, game_id FROM mitfahrgelegenheiten WHERE id = ? AND typ = 'suche'`,
		body.SucheID).Scan(&sucheUserID, &suchePlaetze, &sucheGameID)
	if err == sql.ErrNoRows {
		http.Error(w, "suche entry not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Entries must be for the same game
	if bieteGameID != sucheGameID {
		http.Error(w, "entries are not for the same game", http.StatusBadRequest)
		return
	}

	// Current user must be either bieter or sucher
	var initiertVon string
	var oppositeUserID int
	switch userID {
	case bieteUserID:
		initiertVon = "biete"
		oppositeUserID = sucheUserID
	case sucheUserID:
		initiertVon = "suche"
		oppositeUserID = bieteUserID
	default:
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Sucher must not already have an active pairing for this gesuch
	var existingCount int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM mitfahrt_paarungen WHERE suche_id = ? AND status IN ('pending','confirmed')`,
		body.SucheID).Scan(&existingCount)
	if existingCount > 0 {
		http.Error(w, "suche entry already has an active pairing", http.StatusConflict)
		return
	}

	// Capacity check: biete.plaetze - sum(pending+confirmed suche.plaetze) >= suche.plaetze
	// NULL bietePlaetze = unlimited; NULL suchePlaetze = 1 person
	suchePlaetzeVal := int64(1)
	if suchePlaetze.Valid && suchePlaetze.Int64 > 0 {
		suchePlaetzeVal = suchePlaetze.Int64
	}
	if bietePlaetze.Valid {
		var usedPlaetze int
		h.db.QueryRowContext(r.Context(), `
			SELECT COALESCE(SUM(ms.plaetze), 0)
			FROM mitfahrt_paarungen p
			JOIN mitfahrgelegenheiten ms ON ms.id = p.suche_id
			WHERE p.biete_id = ? AND p.status IN ('pending','confirmed')`,
			body.BieteID).Scan(&usedPlaetze)
		if int(bietePlaetze.Int64)-usedPlaetze < int(suchePlaetzeVal) {
			http.Error(w, "not enough seats available", http.StatusConflict)
			return
		}
	}

	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von)
		VALUES (?, ?, ?)
		ON CONFLICT(biete_id, suche_id) DO UPDATE SET
			status = 'pending',
			initiiert_von = excluded.initiiert_von,
			updated_at = CURRENT_TIMESTAMP
		WHERE mitfahrt_paarungen.status = 'rejected'`,
		body.BieteID, body.SucheID, initiertVon)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("mitfahrgelegenheiten")
	w.WriteHeader(http.StatusNoContent)

	go func() {
		name := h.userName(userID)
		opponent, date := h.gameInfo(bieteGameID)
		var msg string
		if initiertVon == "suche" {
			msg = fmt.Sprintf("%s möchte mitfahren — %s, %s", name, opponent, date)
		} else {
			msg = fmt.Sprintf("%s bietet dir einen Platz an — %s, %s", name, opponent, date)
		}
		notifications.SendToUsers(h.db, h.cfg, []int{oppositeUserID}, "Mitfahranfrage", msg, "/mitfahrgelegenheiten")
	}()
}

// POST /api/mitfahrt-paarungen/{id}/confirm
func (h *Handler) ConfirmPairing(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	var bieteID, sucheID int
	var initiertVon, status string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT biete_id, suche_id, initiiert_von, status FROM mitfahrt_paarungen WHERE id = ?`, id).
		Scan(&bieteID, &sucheID, &initiertVon, &status)
	if err == sql.ErrNoRows {
		http.Error(w, "pairing not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if status != "pending" {
		http.Error(w, "pairing is not pending", http.StatusConflict)
		return
	}

	var bieteUserID int
	var bietePlaetze sql.NullInt64
	h.db.QueryRowContext(r.Context(),
		`SELECT user_id, plaetze FROM mitfahrgelegenheiten WHERE id = ?`, bieteID).
		Scan(&bieteUserID, &bietePlaetze)

	var sucheUserID int
	var suchePlaetze sql.NullInt64
	h.db.QueryRowContext(r.Context(),
		`SELECT user_id, plaetze FROM mitfahrgelegenheiten WHERE id = ?`, sucheID).
		Scan(&sucheUserID, &suchePlaetze)

	// Only the opposite party can confirm
	var initiatorUserID int
	switch initiertVon {
	case "suche":
		initiatorUserID = sucheUserID
		if userID != bieteUserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	case "biete":
		initiatorUserID = bieteUserID
		if userID != sucheUserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	// Race-condition guard: re-check capacity (excluding this pairing itself)
	if bietePlaetze.Valid && suchePlaetze.Valid {
		var usedPlaetze int
		h.db.QueryRowContext(r.Context(), `
			SELECT COALESCE(SUM(ms.plaetze), 0)
			FROM mitfahrt_paarungen p
			JOIN mitfahrgelegenheiten ms ON ms.id = p.suche_id
			WHERE p.biete_id = ? AND p.status IN ('pending','confirmed') AND p.id != ?`,
			bieteID, id).Scan(&usedPlaetze)
		if int(bietePlaetze.Int64)-usedPlaetze < int(suchePlaetze.Int64) {
			http.Error(w, "not enough seats available", http.StatusConflict)
			return
		}
	}

	_, err = h.db.ExecContext(r.Context(),
		`UPDATE mitfahrt_paarungen SET status = 'confirmed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("mitfahrgelegenheiten")
	w.WriteHeader(http.StatusNoContent)

	go func() {
		name := h.userName(userID)
		var gameID int
		h.db.QueryRow(`SELECT game_id FROM mitfahrgelegenheiten WHERE id = ?`, bieteID).Scan(&gameID)
		opponent, date := h.gameInfo(gameID)
		msg := fmt.Sprintf("%s hat die Mitfahrt bestätigt — %s, %s", name, opponent, date)
		notifications.SendToUsers(h.db, h.cfg, []int{initiatorUserID}, "Mitfahrt bestätigt", msg, "/mitfahrgelegenheiten")
	}()
}

// POST /api/mitfahrt-paarungen/{id}/reject
func (h *Handler) RejectPairing(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	var bieteID, sucheID int
	var status string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT biete_id, suche_id, status FROM mitfahrt_paarungen WHERE id = ?`, id).
		Scan(&bieteID, &sucheID, &status)
	if err == sql.ErrNoRows {
		http.Error(w, "pairing not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if status == "rejected" {
		http.Error(w, "pairing already rejected", http.StatusConflict)
		return
	}

	var bieteUserID int
	h.db.QueryRowContext(r.Context(),
		`SELECT user_id FROM mitfahrgelegenheiten WHERE id = ?`, bieteID).Scan(&bieteUserID)
	var sucheUserID int
	h.db.QueryRowContext(r.Context(),
		`SELECT user_id FROM mitfahrgelegenheiten WHERE id = ?`, sucheID).Scan(&sucheUserID)

	if userID != bieteUserID && userID != sucheUserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var oppositeUserID int
	if userID == bieteUserID {
		oppositeUserID = sucheUserID
	} else {
		oppositeUserID = bieteUserID
	}

	_, err = h.db.ExecContext(r.Context(),
		`UPDATE mitfahrt_paarungen SET status = 'rejected', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("mitfahrgelegenheiten")
	w.WriteHeader(http.StatusNoContent)

	go func() {
		name := h.userName(userID)
		var gameID int
		h.db.QueryRow(`SELECT game_id FROM mitfahrgelegenheiten WHERE id = ?`, bieteID).Scan(&gameID)
		opponent, date := h.gameInfo(gameID)
		var title, msg string
		if status == "confirmed" {
			title = "Mitfahrt storniert"
			msg = fmt.Sprintf("%s hat die bestätigte Mitfahrt storniert — %s, %s", name, opponent, date)
		} else {
			title = "Mitfahranfrage abgelehnt"
			msg = fmt.Sprintf("%s hat die Mitfahranfrage abgelehnt — %s, %s", name, opponent, date)
		}
		notifications.SendToUsers(h.db, h.cfg, []int{oppositeUserID}, title, msg, "/mitfahrgelegenheiten")
	}()
}
