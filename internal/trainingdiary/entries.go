package trainingdiary

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// listResponse ist die Hüllstruktur beider Listen-Endpoints.
type listResponse struct {
	Items []entry `json:"items"`
}

// ListOwn — GET /api/training-diary?season=<id>
// Liefert ausschließlich die Einträge des aufrufenden Nutzers.
func (h *Handler) ListOwn(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := h.resolveOwnMember(r.Context(), claims)
	if err == sql.ErrNoRows {
		http.Error(w, "kein Mitglied für diesen Nutzer", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	h.respondEntries(w, r, memberID, r.URL.Query().Get("season"))
}

// GetMemberDiary — GET /api/members/{id}/training-diary?season=<id>
// Fremdzugriff nach den Regeln aus canReadMemberDiary.
func (h *Handler) GetMemberDiary(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	ok, err := h.canReadMemberDiary(r.Context(), claims, memberID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	h.respondEntries(w, r, memberID, r.URL.Query().Get("season"))
}

// respondEntries ist der gemeinsame Lesepfad beider Listen. Ohne season-Filter
// werden alle Einträge geliefert — die eigene Historie soll nicht an der
// Saisongrenze abreißen.
func (h *Handler) respondEntries(w http.ResponseWriter, r *http.Request, memberID int, seasonParam string) {
	query := `SELECT ` + entryColumns + ` FROM training_diary_entries WHERE member_id = ?`
	args := []any{memberID}
	if seasonParam != "" {
		seasonID, err := strconv.Atoi(seasonParam)
		if err != nil {
			http.Error(w, "invalid season", http.StatusBadRequest)
			return
		}
		query += ` AND season_id = ?`
		args = append(args, seasonID)
	}
	query += ` ORDER BY trained_on DESC, id DESC`

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items})
}

// CreateEntry — POST /api/training-diary
// Die member_id stammt immer aus dem Token; ein im Body übergebener Wert wird
// ignoriert (das Feld existiert im Request-Struct gar nicht erst).
func (h *Handler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	// Neuanlage nur für Spieler. Bestandseinträge bleiben für den Eigentümer
	// lesbar und pflegbar (ListOwn/PUT/DELETE) — wer die Spieler-Funktion
	// verliert, soll seine Historie nicht verlieren, aber auch nichts Neues
	// mehr erfassen.
	if !isPlayer(claims) {
		http.Error(w, "Trainingstagebuch nur für Spieler", http.StatusForbidden)
		return
	}
	memberID, err := h.resolveOwnMember(r.Context(), claims)
	if err == sql.ErrNoRows {
		http.Error(w, "kein Mitglied für diesen Nutzer", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	var req entryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	v, err := validateEntry(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Saison-Anker: die bei der Erfassung AKTIVE Saison, nicht die zu
	// trained_on passende. Saisons sind freie Zeiträume ohne Lückenprüfung —
	// eine datumsbasierte Zuordnung verlöre die Sommerpause komplett.
	seasonID, err := h.activeSeasonID(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var seasonArg any
	if seasonID > 0 {
		seasonArg = seasonID
	}

	res, err := h.db.ExecContext(r.Context(), `
		INSERT INTO training_diary_entries
			(member_id, season_id, trained_on, kind, kind_custom, duration_min, rpe, note)
		VALUES (?,?,?,?,?,?,?,?)`,
		memberID, seasonArg, v.trainedOn, v.kind, v.kindCustom, v.durationMin, v.rpe, v.note)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	e, err := h.loadEntry(r.Context(), int(id))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast(diaryEvent)
	writeJSON(w, http.StatusCreated, e)
}

// UpdateEntry — PUT /api/training-diary/{id}
// Nur der Eigentümer. season_id bleibt unangetastet: der Anker gehört zum
// Erfassungszeitpunkt, nicht zur letzten Bearbeitung.
func (h *Handler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	owner, _, _, err := h.entryOwner(r.Context(), id)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !h.isOwner(r.Context(), claims, owner) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req entryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	v, err := validateEntry(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.db.ExecContext(r.Context(), `
		UPDATE training_diary_entries
		   SET trained_on = ?, kind = ?, kind_custom = ?, duration_min = ?, rpe = ?,
		       note = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		v.trainedOn, v.kind, v.kindCustom, v.durationMin, v.rpe, v.note, id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	e, err := h.loadEntry(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast(diaryEvent)
	writeJSON(w, http.StatusOK, e)
}

// DeleteEntry — DELETE /api/training-diary/{id}
// Nur der Eigentümer. Eine anhängende Nachweisdatei wird mit entfernt; eine
// bereits fehlende Datei ist kein Fehlerfall.
func (h *Handler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	owner, diskName, _, err := h.entryOwner(r.Context(), id)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !h.isOwner(r.Context(), claims, owner) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if _, err := h.db.ExecContext(r.Context(),
		`DELETE FROM training_diary_entries WHERE id = ?`, id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	h.removeProofFile(diskName)

	h.hub.Broadcast(diaryEvent)
	w.WriteHeader(http.StatusNoContent)
}

// entryOwner liefert member_id, proof_disk_name und proof_purged_at eines
// Eintrags. sql.ErrNoRows signalisiert „gibt es nicht" → 404, das die Aufrufer
// VOR jeder 403-Antwort auswerten, damit fremde IDs nicht per Statuscode
// enumerierbar sind.
func (h *Handler) entryOwner(ctx context.Context, id int) (memberID int, diskName string, purgedAt string, err error) {
	var disk, purged sql.NullString
	err = h.db.QueryRowContext(ctx,
		`SELECT member_id, proof_disk_name, proof_purged_at FROM training_diary_entries WHERE id = ?`,
		id).Scan(&memberID, &disk, &purged)
	return memberID, disk.String, purged.String, err
}

// isOwner prüft, ob der aufrufende Nutzer das Mitglied hinter dem Eintrag ist.
// Bewusst kein Fallback auf Trainer/Eltern: Schreibrechte hat ausschließlich
// der Eigentümer, Lesen regelt canReadMemberDiary.
func (h *Handler) isOwner(ctx context.Context, claims *auth.Claims, memberID int) bool {
	own, err := h.resolveOwnMember(ctx, claims)
	return err == nil && own == memberID
}

func (h *Handler) loadEntry(ctx context.Context, id int) (entry, error) {
	row := h.db.QueryRowContext(ctx,
		`SELECT `+entryColumns+` FROM training_diary_entries WHERE id = ?`, id)
	return scanEntry(row)
}

// removeProofFile löscht eine Nachweisdatei best-effort. Ein fehlender Name
// oder eine bereits verschwundene Datei sind kein Fehler.
func (h *Handler) removeProofFile(diskName string) {
	if diskName == "" {
		return
	}
	if err := os.Remove(filepath.Join(h.dir, diskName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Kein Abbruch: der Datensatz ist bereits weg, ein verwaister Blob
		// ist das kleinere Übel als eine fehlgeschlagene Nutzeraktion.
		return
	}
}

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}
