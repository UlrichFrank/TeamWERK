package teams

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Mannschaftskasse (Cashbook). Eigenes, vom Strafenbuch strukturell getrenntes
// Ledger pro Kader (aktive Saison des Teams). amount_cent ist SIGNED: Einzahlung
// positiv, Ausgabe negativ; der Saldo ist die SQL-Summe (keine denormalisierte
// Spalte). Eine Buchung modifiziert NIE eine Strafe.
//
// Sicht-/Schreib-Ebenen:
//   - Lesen (Ledger, Kassenwart-Liste): canReadCashbook (= canReadPenalties) —
//     Spieler, Trainer, Erweiterter Kader. Eltern + Außenstehende: 403.
//   - Buchen/Löschen: canManageCashbook — Trainer ODER Kassenwart DIESES Kaders
//     (Fremd-Team-Kassenwart strukturell 403).
//   - Kassenwart-Ernennung: isTrainerOfTeam.

// CashbookEntry ist eine Buchung (signierter Cent-Betrag) in der Ledger-Antwort.
// EnteredByUserID (0 = unbekannt/gelöscht) erlaubt dem Frontend die Bold-Me-Markierung.
type CashbookEntry struct {
	ID              int    `json:"id"`
	AmountCent      int    `json:"amountCent"`
	Note            string `json:"note"`
	EnteredBy       string `json:"enteredBy"`
	EnteredByUserID int    `json:"enteredByUserId"`
	EnteredAt       string `json:"enteredAt"`
}

// CashbookResponse ist die Antwort von ListCashbook. entries nie null.
// canManage = der anfragende Nutzer darf buchen (Trainer oder Kassenwart).
type CashbookResponse struct {
	Entries     []CashbookEntry `json:"entries"`
	BalanceCent int             `json:"balanceCent"`
	CanManage   bool            `json:"canManage"`
}

// KassenwartEntry ist ein ernannter Kassenwart des Kaders.
type KassenwartEntry struct {
	MemberID int    `json:"memberId"`
	Name     string `json:"name"`
}

// ListCashbook — GET /api/teams/{id}/cashbook.
// Gate: canReadCashbook (Spieler/Trainer/Erweiterter Kader; Eltern/Außenstehende 403).
// Liefert alle Buchungen der Kader dieses Teams (aktive Saison, nach entered_at),
// den Saldo (Summe) und canManage. entries nie null.
func (h *Handler) ListCashbook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.canReadCashbook(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	resp := CashbookResponse{Entries: []CashbookEntry{}}
	if canManage, mErr := h.canManageCashbook(ctx, claims, teamID); mErr == nil {
		resp.CanManage = canManage
	}

	rows, err := h.db.QueryContext(ctx, `
		SELECT ce.id, ce.amount_cent, ce.note,
		       COALESCE(m.first_name || ' ' || m.last_name, '') AS entered_by,
		       COALESCE(m.user_id, 0) AS entered_by_user_id,
		       ce.entered_at
		FROM team_cashbook_entries ce
		LEFT JOIN members m ON m.id = ce.entered_by_member_id
		JOIN kader k   ON k.id = ce.kader_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1
		ORDER BY ce.entered_at, ce.id`, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var e CashbookEntry
		if err := rows.Scan(&e.ID, &e.AmountCent, &e.Note, &e.EnteredBy, &e.EnteredByUserID, &e.EnteredAt); err != nil {
			continue
		}
		resp.Entries = append(resp.Entries, e)
		resp.BalanceCent += e.AmountCent
	}

	writeRespJSON(w, http.StatusOK, resp)
}

// CreateCashbookEntry — POST /api/teams/{id}/cashbook. Body {"amountCent":N,"note":"..."}.
// Gate: canManageCashbook (Trainer ODER Kassenwart; Fremd-Team strukturell 403).
// 400 bei amountCent==0 oder leerem note; 404 ohne aktiven Kader. Broadcast + 201.
func (h *Handler) CreateCashbookEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.canManageCashbook(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		AmountCent int    `json:"amountCent"`
		Note       string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	note := strings.TrimSpace(body.Note)
	if note == "" {
		http.Error(w, "note required", http.StatusBadRequest)
		return
	}
	if body.AmountCent == 0 {
		http.Error(w, "amountCent must be non-zero", http.StatusBadRequest)
		return
	}

	kaderID, err := h.resolveKaderID(ctx, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "no active kader for team", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Buchenden Member ermitteln (members.user_id im aktiven Kader dieses Teams);
	// nicht auflösbar (z. B. Admin ohne Member) → NULL.
	var enteredBy sql.NullInt64
	var mid int
	mErr := h.db.QueryRowContext(ctx, `
		SELECT m.id FROM members m
		WHERE m.user_id = ? AND m.id IN (
			SELECT km.member_id FROM kader_members km
			JOIN kader k ON k.id = km.kader_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
			UNION
			SELECT kt.member_id FROM kader_trainers kt
			JOIN kader k ON k.id = kt.kader_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)
		LIMIT 1`, claims.UserID, teamID, teamID).Scan(&mid)
	if mErr == nil {
		enteredBy = sql.NullInt64{Int64: int64(mid), Valid: true}
	}

	res, err := h.db.ExecContext(ctx, `
		INSERT INTO team_cashbook_entries (kader_id, amount_cent, note, entered_by_member_id)
		VALUES (?, ?, ?, ?)`, kaderID, body.AmountCent, note, enteredBy)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	h.hub.Broadcast("cashbook")
	writeRespJSON(w, http.StatusCreated, map[string]int{"id": int(id)})
}

// DeleteCashbookEntry — DELETE /api/teams/{id}/cashbook/{entryId}.
// Gate: canManageCashbook. Hard-Delete der einzelnen Row, auf die aktive Saison
// des Teams gescoped. Broadcast + 204.
func (h *Handler) DeleteCashbookEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	entryID, err := strconv.Atoi(chi.URLParam(r, "entryId"))
	if err != nil {
		http.Error(w, "invalid entry id", http.StatusBadRequest)
		return
	}
	if ok, err := h.canManageCashbook(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_, err = h.db.ExecContext(ctx, `
		DELETE FROM team_cashbook_entries
		WHERE id = ? AND kader_id IN (
			SELECT k.id FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)`, entryID, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("cashbook")
	w.WriteHeader(http.StatusNoContent)
}

// ListKassenwarte — GET /api/teams/{id}/treasurers.
// Gate: canReadCashbook. Liefert die ernannten Kassenwarte des Kaders der aktiven
// Saison als JSON-Array (nie null).
func (h *Handler) ListKassenwarte(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.canReadCashbook(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	warte := []KassenwartEntry{}
	rows, err := h.db.QueryContext(ctx, `
		SELECT kk.member_id, m.first_name || ' ' || m.last_name
		FROM kader_kassenwarte kk
		JOIN members m ON m.id = kk.member_id
		JOIN kader k   ON k.id = kk.kader_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1
		ORDER BY m.first_name, m.last_name`, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var e KassenwartEntry
		if err := rows.Scan(&e.MemberID, &e.Name); err != nil {
			continue
		}
		warte = append(warte, e)
	}
	writeRespJSON(w, http.StatusOK, warte)
}

// AppointKassenwart — POST /api/teams/{id}/treasurers. Body {"memberId":M}.
// Gate: isTrainerOfTeam. 404 ohne aktiven Kader, 400 wenn der Member nicht im
// (regulären/erweiterten/Trainer-)Kader des Teams ist. INSERT OR IGNORE (idempotent).
// Broadcast + 201 {"memberId":M}.
func (h *Handler) AppointKassenwart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.isTrainerOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		MemberID int `json:"memberId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.MemberID <= 0 {
		http.Error(w, "memberId required", http.StatusBadRequest)
		return
	}

	kaderID, err := h.resolveKaderID(ctx, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "no active kader for team", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if ok, err := h.memberInActiveKaderAny(r, teamID, body.MemberID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "member not in team kader", http.StatusBadRequest)
		return
	}

	_, err = h.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO kader_kassenwarte (kader_id, member_id) VALUES (?, ?)`,
		kaderID, body.MemberID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("treasurers")
	writeRespJSON(w, http.StatusCreated, map[string]int{"memberId": body.MemberID})
}

// RemoveKassenwart — DELETE /api/teams/{id}/treasurers/{memberId}.
// Gate: isTrainerOfTeam. Löscht das Appointment (auf die aktive Saison des Teams
// gescoped). Broadcast + 204.
func (h *Handler) RemoveKassenwart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	memberID, err := strconv.Atoi(chi.URLParam(r, "memberId"))
	if err != nil {
		http.Error(w, "invalid member id", http.StatusBadRequest)
		return
	}
	if ok, err := h.isTrainerOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_, err = h.db.ExecContext(ctx, `
		DELETE FROM kader_kassenwarte
		WHERE member_id = ? AND kader_id IN (
			SELECT k.id FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)`, memberID, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("treasurers")
	w.WriteHeader(http.StatusNoContent)
}
