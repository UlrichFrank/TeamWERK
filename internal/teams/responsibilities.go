package teams

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Mannschafts-Aufgaben (Responsibilities). Alles kader-scoped auf die aktive
// Saison des Teams. Der Aufgaben-Catalog (responsibility_types) ist reines
// Vorschlags-Vokabular; die Zuweisung (member_responsibilities) speichert das
// Label als Snapshot — ein späterer Catalog-Edit/-Delete ändert vergebene
// Zuweisungen nicht rückwirkend.

// ResponsibilityType ist ein Catalog-Eintrag (Aufgaben-Vokabular pro Kader).
type ResponsibilityType struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

// responsibilitiesFor liefert die Aufgaben-Labels eines Members, gescoped auf die
// Kader dieses Teams in der angegebenen (aktiven) Saison, alphabetisch sortiert.
// Liefert nie nil — für die Roster-JSON-Invariante (responsibilities != null).
func (h *Handler) responsibilitiesFor(ctx context.Context, teamID, seasonID, memberID int) []string {
	labels := []string{}
	rows, err := h.db.QueryContext(ctx, `
		SELECT mr.label
		FROM member_responsibilities mr
		JOIN kader k ON k.id = mr.kader_id
		WHERE mr.member_id = ? AND k.team_id = ? AND k.season_id = ?
		ORDER BY mr.label`, memberID, teamID, seasonID)
	if err != nil {
		return labels
	}
	defer rows.Close()
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			continue
		}
		labels = append(labels, l)
	}
	return labels
}

// ListResponsibilityTypes — GET /api/teams/{id}/responsibility-types.
// Gate: Trainer/Admin des Teams. Liefert den Catalog des Kaders der aktiven
// Saison als JSON-Array (nie null). Ohne aktiven Kader → leeres Array.
func (h *Handler) ListResponsibilityTypes(w http.ResponseWriter, r *http.Request) {
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

	types := []ResponsibilityType{}
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
		`SELECT id, label FROM responsibility_types WHERE kader_id = ? ORDER BY label`, kaderID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rt ResponsibilityType
		if err := rows.Scan(&rt.ID, &rt.Label); err != nil {
			continue
		}
		types = append(types, rt)
	}
	writeRespJSON(w, http.StatusOK, types)
}

// CreateResponsibilityType — POST /api/teams/{id}/responsibility-types.
// Body {"label":"Harz"}. Gate: Trainer/Admin. 400 bei leerem Label, 404 ohne
// aktiven Kader, 409 bei Duplikat. Broadcast + 201 {"id":N,"label":"..."}.
func (h *Handler) CreateResponsibilityType(w http.ResponseWriter, r *http.Request) {
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
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	label := strings.TrimSpace(body.Label)
	if label == "" {
		http.Error(w, "label required", http.StatusBadRequest)
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
		`INSERT INTO responsibility_types (kader_id, label) VALUES (?, ?)`, kaderID, label)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "responsibility type already exists", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	h.hub.Broadcast("responsibilities")
	writeRespJSON(w, http.StatusCreated, ResponsibilityType{ID: int(id), Label: label})
}

// DeleteResponsibilityType — DELETE /api/teams/{id}/responsibility-types/{typeId}.
// Gate: Trainer/Admin. Löscht den Catalog-Eintrag (auf die aktive Saison des Teams
// gescoped). Vergebene member_responsibilities bleiben unberührt (Snapshot).
// Broadcast + 204.
func (h *Handler) DeleteResponsibilityType(w http.ResponseWriter, r *http.Request) {
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
		DELETE FROM responsibility_types
		WHERE id = ? AND kader_id IN (
			SELECT k.id FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)`, typeID, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("responsibilities")
	w.WriteHeader(http.StatusNoContent)
}

// CreateResponsibility — POST /api/teams/{id}/responsibilities.
// Body {"memberId":M,"label":"Harz"}. Gate: Trainer/Admin. 400 bei leerem Label,
// 404 ohne aktiven Kader, 400 wenn der Member nicht (erweiterter) Kader-Spieler ist,
// 409 bei Duplikat. Broadcast + 201 {"id":N}.
func (h *Handler) CreateResponsibility(w http.ResponseWriter, r *http.Request) {
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
		MemberID int    `json:"memberId"`
		Label    string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	label := strings.TrimSpace(body.Label)
	if label == "" {
		http.Error(w, "label required", http.StatusBadRequest)
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

	// Der Member muss (regulärer ODER erweiterter) Spieler eines Kaders dieses
	// Teams in der aktiven Saison sein — sonst kann man teamfremden Mitgliedern
	// Aufgaben andichten.
	var inKader int
	err = h.db.QueryRowContext(ctx, `
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
		)`, teamID, body.MemberID, teamID, body.MemberID).Scan(&inKader)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if inKader == 0 {
		http.Error(w, "member not in team kader", http.StatusBadRequest)
		return
	}

	res, err := h.db.ExecContext(ctx,
		`INSERT INTO member_responsibilities (kader_id, member_id, label) VALUES (?, ?, ?)`,
		kaderID, body.MemberID, label)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "responsibility already assigned", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	h.hub.Broadcast("responsibilities")
	writeRespJSON(w, http.StatusCreated, map[string]int{"id": int(id)})
}

// DeleteResponsibility — DELETE /api/teams/{id}/responsibilities/{respId}.
// Gate: Trainer/Admin. Löscht die Zuweisung (auf die aktive Saison des Teams
// gescoped). Broadcast + 204.
func (h *Handler) DeleteResponsibility(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	respID, err := strconv.Atoi(chi.URLParam(r, "respId"))
	if err != nil {
		http.Error(w, "invalid responsibility id", http.StatusBadRequest)
		return
	}
	if ok, err := h.isTrainerOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_, err = h.db.ExecContext(ctx, `
		DELETE FROM member_responsibilities
		WHERE id = ? AND kader_id IN (
			SELECT k.id FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1
		)`, respID, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("responsibilities")
	w.WriteHeader(http.StatusNoContent)
}

// writeRespJSON schreibt eine JSON-Antwort mit dem gegebenen Statuscode.
// Bewusst responsibility-lokaler Name, um Kollisionen mit anderen teams-Dateien
// (Strafen) zu vermeiden.
func writeRespJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
