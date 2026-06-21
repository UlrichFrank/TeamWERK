package carpooling

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/push"
)

// POST /api/mitfahrt-paarungen
//
// Akzeptiert drei Body-Formen:
//   - {bieteId, sucheId}            zweiseitig (Altpfad, beide Einträge bestehen)
//   - {bieteId, forUserId?, plaetze?}  einseitig: ich (oder mein Kind) will mitfahren;
//     der Suche-Spiegel wird get-or-create angelegt → initiiert_von='suche'
//   - {sucheId, plaetze?}          einseitig: ich biete einen Platz an;
//     der Biete-Spiegel wird get-or-create für mich angelegt → initiiert_von='biete'
//
// Spiegel-Eintrag + Paarung werden atomar in einer Transaktion erstellt: schlägt
// Berechtigung oder Kapazität fehl, wird nichts persistiert (kein Phantom-Eintrag).
func (h *Handler) RequestPairing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)
	userID := claims.UserID

	var body struct {
		BieteID   int  `json:"bieteId"`
		SucheID   int  `json:"sucheId"`
		ForUserID *int `json:"forUserId,omitempty"`
		Plaetze   *int `json:"plaetze"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || (body.BieteID == 0 && body.SucheID == 0) {
		http.Error(w, "bieteId or sucheId required", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var bieteID, sucheID, gameID, oppositeUserID int
	var initiertVon string

	switch {
	case body.SucheID == 0:
		// Einseitig: Mitfahren bei einem bestehenden Biete-Eintrag.
		var bieteUserID int
		err := tx.QueryRowContext(ctx,
			`SELECT user_id, game_id FROM mitfahrgelegenheiten WHERE id = ? AND typ = 'biete'`,
			body.BieteID).Scan(&bieteUserID, &gameID)
		if err == sql.ErrNoRows {
			http.Error(w, "biete entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// Spiegel-Eintrag entsteht für forUserId (Default: eingeloggter Nutzer).
		// Berechtigung VOR dem Insert prüfen, damit ein fremder forUserId nichts anlegt.
		target := userID
		if body.ForUserID != nil && *body.ForUserID != userID {
			if !h.isChildOf(ctx, userID, *body.ForUserID) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			target = *body.ForUserID
		}
		sid, err := getOrCreateSuche(ctx, tx, gameID, target, body.Plaetze)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		bieteID, sucheID = body.BieteID, sid
		initiertVon = "suche"
		oppositeUserID = bieteUserID

	case body.BieteID == 0:
		// Einseitig: Platz anbieten zu einem bestehenden Suche-Eintrag.
		// Der Biete-Spiegel entsteht immer für den eingeloggten Nutzer (kein forUserId).
		var sucheUserID int
		err := tx.QueryRowContext(ctx,
			`SELECT user_id, game_id FROM mitfahrgelegenheiten WHERE id = ? AND typ = 'suche'`,
			body.SucheID).Scan(&sucheUserID, &gameID)
		if err == sql.ErrNoRows {
			http.Error(w, "suche entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		bid, err := getOrCreateBiete(ctx, tx, gameID, userID, body.Plaetze)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		bieteID, sucheID = bid, body.SucheID
		initiertVon = "biete"
		oppositeUserID = sucheUserID

	default:
		// Zweiseitig (Altpfad): beide IDs gegeben.
		var bieteUserID, sucheUserID, bieteGameID, sucheGameID int
		err := tx.QueryRowContext(ctx,
			`SELECT user_id, game_id FROM mitfahrgelegenheiten WHERE id = ? AND typ = 'biete'`,
			body.BieteID).Scan(&bieteUserID, &bieteGameID)
		if err == sql.ErrNoRows {
			http.Error(w, "biete entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		err = tx.QueryRowContext(ctx,
			`SELECT user_id, game_id FROM mitfahrgelegenheiten WHERE id = ? AND typ = 'suche'`,
			body.SucheID).Scan(&sucheUserID, &sucheGameID)
		if err == sql.ErrNoRows {
			http.Error(w, "suche entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if bieteGameID != sucheGameID {
			http.Error(w, "entries are not for the same game", http.StatusBadRequest)
			return
		}
		switch {
		case userID == bieteUserID || h.isChildOf(ctx, userID, bieteUserID):
			initiertVon = "biete"
			oppositeUserID = sucheUserID
		case userID == sucheUserID || h.isChildOf(ctx, userID, sucheUserID):
			initiertVon = "suche"
			oppositeUserID = bieteUserID
		default:
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		bieteID, sucheID = body.BieteID, body.SucheID
		gameID = bieteGameID
	}

	// Sucher must not already have an active pairing for this gesuch
	var existingCount int
	tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM mitfahrt_paarungen WHERE suche_id = ? AND status IN ('pending','confirmed')`,
		sucheID).Scan(&existingCount)
	if existingCount > 0 {
		http.Error(w, "suche entry already has an active pairing", http.StatusConflict)
		return
	}

	// Capacity check: biete.plaetze - sum(pending+confirmed suche.plaetze) >= suche.plaetze
	// NULL bietePlaetze = unlimited; NULL suchePlaetze = 1 person
	var bietePlaetze, suchePlaetze sql.NullInt64
	tx.QueryRowContext(ctx, `SELECT plaetze FROM mitfahrgelegenheiten WHERE id = ?`, bieteID).Scan(&bietePlaetze)
	tx.QueryRowContext(ctx, `SELECT plaetze FROM mitfahrgelegenheiten WHERE id = ?`, sucheID).Scan(&suchePlaetze)
	suchePlaetzeVal := int64(1)
	if suchePlaetze.Valid && suchePlaetze.Int64 > 0 {
		suchePlaetzeVal = suchePlaetze.Int64
	}
	if bietePlaetze.Valid {
		var usedPlaetze int
		tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(ms.plaetze), 0)
			FROM mitfahrt_paarungen p
			JOIN mitfahrgelegenheiten ms ON ms.id = p.suche_id
			WHERE p.biete_id = ? AND p.status IN ('pending','confirmed')`,
			bieteID).Scan(&usedPlaetze)
		if int(bietePlaetze.Int64)-usedPlaetze < int(suchePlaetzeVal) {
			http.Error(w, "not enough seats available", http.StatusConflict)
			return
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von)
		VALUES (?, ?, ?)
		ON CONFLICT(biete_id, suche_id) DO UPDATE SET
			status = 'pending',
			initiiert_von = excluded.initiiert_von,
			updated_at = CURRENT_TIMESTAMP
		WHERE mitfahrt_paarungen.status = 'rejected'`,
		bieteID, sucheID, initiertVon)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	actorName := h.userName(userID)
	h.writeEvent(gameID, oppositeUserID, "pairing_requested", actorName)

	h.hub.Broadcast("mitfahrgelegenheiten")
	w.WriteHeader(http.StatusNoContent)

	go func() {
		opponent, date := h.gameInfo(gameID)
		var msg string
		if initiertVon == "suche" {
			msg = fmt.Sprintf("%s möchte mitfahren — %s, %s", actorName, opponent, date)
		} else {
			msg = fmt.Sprintf("%s bietet dir einen Platz an — %s, %s", actorName, opponent, date)
		}
		push.SendToUsers(h.db, h.cfg, []int{oppositeUserID}, "Mitfahranfrage", msg, "/mitfahrgelegenheiten")
	}()
}

// getOrCreateSuche liefert die ID eines Suche-Eintrags für (gameID, userID) ohne
// aktive Paarung — vorhandenen wiederverwenden, sonst neu anlegen (Default 1 Platz).
func getOrCreateSuche(ctx context.Context, tx *sql.Tx, gameID, userID int, plaetze *int) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `
		SELECT m.id FROM mitfahrgelegenheiten m
		WHERE m.game_id = ? AND m.user_id = ? AND m.typ = 'suche'
		  AND NOT EXISTS (
			SELECT 1 FROM mitfahrt_paarungen p
			WHERE p.suche_id = m.id AND p.status IN ('pending','confirmed'))
		ORDER BY m.id LIMIT 1`, gameID, userID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	p := 1
	if plaetze != nil && *plaetze > 0 {
		p = *plaetze
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', ?)`,
		gameID, userID, p)
	if err != nil {
		return 0, err
	}
	nid, err := res.LastInsertId()
	return int(nid), err
}

// getOrCreateBiete liefert die ID des Biete-Eintrags für (gameID, userID) —
// durch den Unique-Index existiert höchstens einer; sonst neu anlegen.
// plaetze == nil ⇒ NULL (unbegrenzt).
func getOrCreateBiete(ctx context.Context, tx *sql.Tx, gameID, userID int, plaetze *int) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'biete'`,
		gameID, userID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	var p any
	if plaetze != nil && *plaetze > 0 {
		p = *plaetze
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', ?)`,
		gameID, userID, p)
	if err != nil {
		return 0, err
	}
	nid, err := res.LastInsertId()
	return int(nid), err
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

	// Only the opposite party (or their parent) can confirm
	var initiatorUserID int
	switch initiertVon {
	case "suche":
		initiatorUserID = sucheUserID
		if userID != bieteUserID && !h.isChildOf(r.Context(), userID, bieteUserID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	case "biete":
		initiatorUserID = bieteUserID
		if userID != sucheUserID && !h.isChildOf(r.Context(), userID, sucheUserID) {
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

	var gameID int
	h.db.QueryRowContext(r.Context(), `SELECT game_id FROM mitfahrgelegenheiten WHERE id = ?`, bieteID).Scan(&gameID)
	actorName := h.userName(userID)
	h.writeEvent(gameID, initiatorUserID, "pairing_confirmed", actorName)

	h.hub.Broadcast("mitfahrgelegenheiten")
	w.WriteHeader(http.StatusNoContent)

	go func() {
		opponent, date := h.gameInfo(gameID)
		msg := fmt.Sprintf("%s hat die Mitfahrt bestätigt — %s, %s", actorName, opponent, date)
		uids := push.FilterByPushPref(h.db, []int{initiatorUserID}, "carpooling")
		push.SendToUsers(h.db, h.cfg, uids, "Mitfahrt bestätigt", msg, "/mitfahrgelegenheiten")
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

	if userID != bieteUserID && userID != sucheUserID &&
		!h.isChildOf(r.Context(), userID, bieteUserID) &&
		!h.isChildOf(r.Context(), userID, sucheUserID) {
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

	var gameID int
	h.db.QueryRowContext(r.Context(), `SELECT game_id FROM mitfahrgelegenheiten WHERE id = ?`, bieteID).Scan(&gameID)
	actorName := h.userName(userID)
	eventType := "pairing_rejected"
	if status == "confirmed" {
		eventType = "pairing_cancelled"
	}
	h.writeEvent(gameID, oppositeUserID, eventType, actorName)

	h.hub.Broadcast("mitfahrgelegenheiten")
	w.WriteHeader(http.StatusNoContent)

	go func() {
		opponent, date := h.gameInfo(gameID)
		var title, msg string
		if status == "confirmed" {
			title = "Mitfahrt storniert"
			msg = fmt.Sprintf("%s hat die bestätigte Mitfahrt storniert — %s, %s", actorName, opponent, date)
		} else {
			title = "Mitfahranfrage abgelehnt"
			msg = fmt.Sprintf("%s hat die Mitfahranfrage abgelehnt — %s, %s", actorName, opponent, date)
		}
		uids := push.FilterByPushPref(h.db, []int{oppositeUserID}, "carpooling")
		push.SendToUsers(h.db, h.cfg, uids, title, msg, "/mitfahrgelegenheiten")
	}()
}
