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

// Mannschafts-Strafen (Penalties). Alles kader-scoped auf die aktive Saison des
// Teams (konsistent mit den Aufgaben in responsibilities.go). reason/amount_cent
// der vergebenen Strafe (team_penalties) sind Snapshots — ein späterer Edit/Delete
// des Strafen-Catalogs (penalty_types) ändert bereits vergebene Strafen nicht.
//
// Zwei Sicht-/Schreib-Ebenen:
//   - Lesen (Liste, Catalog, Strafenwart-Liste): canReadPenalties — Spieler,
//     Trainer, Erweiterter Kader. Eltern + Außenstehende: 403 (harte Grenze).
//   - Vergeben/Stornieren/Zurücksetzen: isStrafenwartOfTeam — nur der für DIESES
//     Team ernannte Strafenwart (Fremd-Strafenwart strukturell 403).
//   - Catalog-Pflege + Strafenwart-Ernennung: isTrainerOfTeam.
//
// Geld ist überall Integer-Cent.

// Penalty ist eine vergebene Strafe (Snapshot) in der Listen-Antwort.
type Penalty struct {
	ID         int    `json:"id"`
	MemberID   int    `json:"memberId"`
	MemberName string `json:"memberName"`
	AmountCent int    `json:"amountCent"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"createdAt"`
}

// PenaltyTotal ist die Summe der offenen Strafen eines Members (Cent).
type PenaltyTotal struct {
	MemberID   int    `json:"memberId"`
	MemberName string `json:"memberName"`
	TotalCent  int    `json:"totalCent"`
}

// PenaltyListResponse ist die Antwort von ListPenalties. Alle Arrays nie null.
// canLevy = der anfragende Nutzer ist Strafenwart dieses Teams.
type PenaltyListResponse struct {
	Penalties []Penalty      `json:"penalties"`
	Totals    []PenaltyTotal `json:"totals"`
	CanLevy   bool           `json:"canLevy"`
}

// PenaltyType ist ein Catalog-Eintrag (Strafen-Vokabular + Default-Betrag).
type PenaltyType struct {
	ID                int    `json:"id"`
	Reason            string `json:"reason"`
	DefaultAmountCent int    `json:"defaultAmountCent"`
}

// StrafenwartEntry ist ein ernannter Strafenwart des Kaders.
type StrafenwartEntry struct {
	MemberID int    `json:"memberId"`
	Name     string `json:"name"`
}

// memberInActiveKader meldet, ob memberID (regulärer ODER erweiterter) Spieler
// eines Kaders dieses Teams in der aktiven Saison ist — verhindert, dass man
// teamfremden Mitgliedern Strafen andichtet.
func (h *Handler) memberInActiveKader(r *http.Request, teamID, memberID int) (bool, error) {
	var n int
	err := h.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM (
			SELECT km.member_id FROM kader_members km
			JOIN kader k ON k.id = km.kader_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1 AND km.member_id = ?
			UNION
			SELECT kem.member_id FROM kader_extended_members kem
			JOIN kader k ON k.id = kem.kader_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1 AND kem.member_id = ?
		)`, teamID, memberID, teamID, memberID).Scan(&n)
	return n > 0, err
}

// memberInActiveKaderAny meldet, ob memberID Spieler, erweiterter Spieler ODER
// Trainer eines Kaders dieses Teams in der aktiven Saison ist. Gate für die
// Strafenwart-Ernennung (auch ein Trainer darf Strafenwart werden).
func (h *Handler) memberInActiveKaderAny(r *http.Request, teamID, memberID int) (bool, error) {
	var n int
	err := h.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM (
			SELECT km.member_id FROM kader_members km
			JOIN kader k ON k.id = km.kader_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1 AND km.member_id = ?
			UNION
			SELECT kem.member_id FROM kader_extended_members kem
			JOIN kader k ON k.id = kem.kader_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1 AND kem.member_id = ?
			UNION
			SELECT kt.member_id FROM kader_trainers kt
			JOIN kader k ON k.id = kt.kader_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1 AND kt.member_id = ?
		)`, teamID, memberID, teamID, memberID, teamID, memberID).Scan(&n)
	return n > 0, err
}

// ListPenalties — GET /api/teams/{id}/penalties.
// Gate: canReadPenalties (Spieler/Trainer/Erweiterter Kader; Eltern/Außenstehende 403).
// Liefert alle Strafen der Kader dieses Teams in der aktiven Saison (nach created_at
// sortiert), pro-Member-Summen und canLevy (=Strafenwart). Alle Arrays nie null.
func (h *Handler) ListPenalties(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.canReadPenalties(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	resp := PenaltyListResponse{
		Penalties: []Penalty{},
		Totals:    []PenaltyTotal{},
	}
	if canLevy, lErr := h.isStrafenwartOfTeam(ctx, claims, teamID); lErr == nil {
		resp.CanLevy = canLevy
	}

	rows, err := h.db.QueryContext(ctx, `
		SELECT tp.id, tp.member_id, m.first_name || ' ' || m.last_name,
		       tp.amount_cent, tp.reason, tp.created_at
		FROM team_penalties tp
		JOIN members m ON m.id = tp.member_id
		JOIN kader k   ON k.id = tp.kader_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1
		ORDER BY tp.created_at, tp.id`, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Summen pro Member in Insertion-Reihenfolge der ersten Strafe aufbauen.
	totalIndex := map[int]int{}
	for rows.Next() {
		var p Penalty
		if err := rows.Scan(&p.ID, &p.MemberID, &p.MemberName, &p.AmountCent, &p.Reason, &p.CreatedAt); err != nil {
			continue
		}
		resp.Penalties = append(resp.Penalties, p)
		if idx, ok := totalIndex[p.MemberID]; ok {
			resp.Totals[idx].TotalCent += p.AmountCent
		} else {
			totalIndex[p.MemberID] = len(resp.Totals)
			resp.Totals = append(resp.Totals, PenaltyTotal{
				MemberID:   p.MemberID,
				MemberName: p.MemberName,
				TotalCent:  p.AmountCent,
			})
		}
	}

	writeRespJSON(w, http.StatusOK, resp)
}

// CreatePenalty — POST /api/teams/{id}/penalties. Body {"memberId":M,"amountCent":N,"reason":"..."}.
// Gate: isStrafenwartOfTeam (Fremd-Strafenwart strukturell 403). 400 bei amountCent<=0
// oder leerem reason bzw. Member nicht im Kader; 404 ohne aktiven Kader.
// created_by_member_id = handelnder Member (nullable, z. B. Admin). Broadcast + 201 {"id":P}.
func (h *Handler) CreatePenalty(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.isStrafenwartOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		MemberID   int    `json:"memberId"`
		AmountCent int    `json:"amountCent"`
		Reason     string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		http.Error(w, "reason required", http.StatusBadRequest)
		return
	}
	if body.AmountCent <= 0 {
		http.Error(w, "amountCent must be > 0", http.StatusBadRequest)
		return
	}
	// Bei Einheit 'striche' sind nur ganze Striche zulässig (amount_cent muss durch
	// 100 teilbar sein, weil ein Strich als 100 Cent gespeichert wird).
	if unit, uErr := h.currentPenaltyUnit(ctx, teamID); uErr != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if unit == "striche" && body.AmountCent%100 != 0 {
		http.Error(w, "amount must be whole striche", http.StatusBadRequest)
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

	if ok, err := h.memberInActiveKader(r, teamID, body.MemberID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "member not in team kader", http.StatusBadRequest)
		return
	}

	// Handelnden Member ermitteln (members.user_id = claims.UserID im aktiven Kader
	// dieses Teams). Nicht auflösbar (z. B. Admin ohne Member) → NULL.
	var createdBy sql.NullInt64
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
		createdBy = sql.NullInt64{Int64: int64(mid), Valid: true}
	}

	res, err := h.db.ExecContext(ctx, `
		INSERT INTO team_penalties (kader_id, member_id, amount_cent, reason, created_by_member_id)
		VALUES (?, ?, ?, ?, ?)`, kaderID, body.MemberID, body.AmountCent, reason, createdBy)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	h.hub.Broadcast("penalties")
	writeRespJSON(w, http.StatusCreated, map[string]int{"id": int(id)})
}

// DeletePenalty (Storno) — DELETE /api/teams/{id}/penalties/{penaltyId}.
// Gate: isStrafenwartOfTeam. Hard-Delete der einzelnen Row, auf die aktive Saison
// des Teams gescoped. Broadcast + 204.
func (h *Handler) DeletePenalty(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	penaltyID, err := strconv.Atoi(chi.URLParam(r, "penaltyId"))
	if err != nil {
		http.Error(w, "invalid penalty id", http.StatusBadRequest)
		return
	}
	if ok, err := h.isStrafenwartOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_, err = h.db.ExecContext(ctx, `
		DELETE FROM team_penalties
		WHERE id = ? AND kader_id IN (
			SELECT k.id FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)`, penaltyID, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("penalties")
	w.WriteHeader(http.StatusNoContent)
}

// ResetMemberPenalties — DELETE /api/teams/{id}/penalties?member={memberId}.
// Gate: isStrafenwartOfTeam. Löscht ALLE Strafen dieses Members in den Kadern des
// Teams (aktive Saison). 400 bei fehlendem/ungültigem member. Broadcast + 204.
func (h *Handler) ResetMemberPenalties(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	// Gate VOR der Input-Validierung: ein Nicht-Strafenwart bekommt 403, nicht 400
	// (konsistente Autorisierungs-Semantik, unabhängig vom fehlenden member-Param).
	if ok, err := h.isStrafenwartOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	memberID, err := strconv.Atoi(r.URL.Query().Get("member"))
	if err != nil || memberID <= 0 {
		http.Error(w, "member query param required", http.StatusBadRequest)
		return
	}

	_, err = h.db.ExecContext(ctx, `
		DELETE FROM team_penalties
		WHERE member_id = ? AND kader_id IN (
			SELECT k.id FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)`, memberID, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("penalties")
	w.WriteHeader(http.StatusNoContent)
}

// ListPenaltyTypes — GET /api/teams/{id}/penalty-types.
// Gate: canReadPenalties. Liefert den Strafen-Catalog des Kaders der aktiven Saison
// als JSON-Array (nie null). Ohne aktiven Kader → leeres Array.
func (h *Handler) ListPenaltyTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.canReadPenalties(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	types := []PenaltyType{}
	kaderID, err := h.resolveKaderID(ctx, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		writeRespJSON(w, http.StatusOK, types)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	rows, err := h.db.QueryContext(ctx,
		`SELECT id, reason, default_amount_cent FROM penalty_types WHERE kader_id = ? ORDER BY reason`, kaderID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var pt PenaltyType
		if err := rows.Scan(&pt.ID, &pt.Reason, &pt.DefaultAmountCent); err != nil {
			continue
		}
		types = append(types, pt)
	}
	writeRespJSON(w, http.StatusOK, types)
}

// CreatePenaltyType — POST /api/teams/{id}/penalty-types. Body {"reason":"...","defaultAmountCent":N}.
// Gate: isTrainerOfTeam. 400 bei leerem reason oder defaultAmountCent<0, 404 ohne
// aktiven Kader, 409 bei Duplikat. Broadcast + 201 {"id":N}.
func (h *Handler) CreatePenaltyType(w http.ResponseWriter, r *http.Request) {
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
		Reason            string `json:"reason"`
		DefaultAmountCent int    `json:"defaultAmountCent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		http.Error(w, "reason required", http.StatusBadRequest)
		return
	}
	if body.DefaultAmountCent < 0 {
		http.Error(w, "defaultAmountCent must be >= 0", http.StatusBadRequest)
		return
	}
	// Bei Einheit 'striche' nur ganze Striche im Katalog (durch 100 teilbar).
	if unit, uErr := h.currentPenaltyUnit(ctx, teamID); uErr != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if unit == "striche" && body.DefaultAmountCent%100 != 0 {
		http.Error(w, "amount must be whole striche", http.StatusBadRequest)
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

	res, err := h.db.ExecContext(ctx,
		`INSERT INTO penalty_types (kader_id, reason, default_amount_cent) VALUES (?, ?, ?)`,
		kaderID, reason, body.DefaultAmountCent)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "penalty type already exists", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	h.hub.Broadcast("penalties")
	writeRespJSON(w, http.StatusCreated, map[string]int{"id": int(id)})
}

// DeletePenaltyType — DELETE /api/teams/{id}/penalty-types/{typeId}.
// Gate: isTrainerOfTeam. Löscht den Catalog-Eintrag (auf die aktive Saison des Teams
// gescoped). Vergebene team_penalties bleiben unberührt (Snapshot). Broadcast + 204.
func (h *Handler) DeletePenaltyType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	typeID, err := strconv.Atoi(chi.URLParam(r, "typeId"))
	if err != nil {
		http.Error(w, "invalid type id", http.StatusBadRequest)
		return
	}
	if ok, err := h.isTrainerOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_, err = h.db.ExecContext(ctx, `
		DELETE FROM penalty_types
		WHERE id = ? AND kader_id IN (
			SELECT k.id FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)`, typeID, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("penalties")
	w.WriteHeader(http.StatusNoContent)
}

// ListStrafenwarte — GET /api/teams/{id}/penalty-wardens.
// Gate: canReadPenalties. Liefert die ernannten Strafenwarte des Kaders der aktiven
// Saison als JSON-Array (nie null).
func (h *Handler) ListStrafenwarte(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.canReadPenalties(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	warte := []StrafenwartEntry{}
	rows, err := h.db.QueryContext(ctx, `
		SELECT ks.member_id, m.first_name || ' ' || m.last_name
		FROM kader_strafenwarte ks
		JOIN members m ON m.id = ks.member_id
		JOIN kader k   ON k.id = ks.kader_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1
		ORDER BY m.first_name, m.last_name`, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var e StrafenwartEntry
		if err := rows.Scan(&e.MemberID, &e.Name); err != nil {
			continue
		}
		warte = append(warte, e)
	}
	writeRespJSON(w, http.StatusOK, warte)
}

// AppointStrafenwart — POST /api/teams/{id}/penalty-wardens. Body {"memberId":M}.
// Gate: isTrainerOfTeam. 404 ohne aktiven Kader, 400 wenn der Member nicht im
// (regulären/erweiterten/Trainer-)Kader des Teams ist. INSERT OR IGNORE (idempotent).
// Broadcast + 201 {"memberId":M}.
func (h *Handler) AppointStrafenwart(w http.ResponseWriter, r *http.Request) {
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
		`INSERT OR IGNORE INTO kader_strafenwarte (kader_id, member_id) VALUES (?, ?)`,
		kaderID, body.MemberID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("penalties")
	writeRespJSON(w, http.StatusCreated, map[string]int{"memberId": body.MemberID})
}

// RemoveStrafenwart — DELETE /api/teams/{id}/penalty-wardens/{memberId}.
// Gate: isTrainerOfTeam. Löscht das Appointment (auf die aktive Saison des Teams
// gescoped). Broadcast + 204.
func (h *Handler) RemoveStrafenwart(w http.ResponseWriter, r *http.Request) {
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
		DELETE FROM kader_strafenwarte
		WHERE member_id = ? AND kader_id IN (
			SELECT k.id FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)`, memberID, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("penalties")
	w.WriteHeader(http.StatusNoContent)
}
