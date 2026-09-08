// Package practicegroups verwaltet Übungsgruppen: benannte Kader ohne
// Altersklasse, Geschlecht, Jahrgang und ohne `teams`-Zwilling.
//
// Eine Übungsgruppe ist eine Zeile in `kader` mit `kind='practice'` — dieselbe
// Tabelle, dieselben Zuordnungstabellen (`kader_members`, `kader_trainers`) wie
// die Mannschaftsvariante. Was sie NICHT hat, ist eine `teams`-Zeile: genau
// diese Abwesenheit hält alle team-gebundenen Flächen (Spiele, Dienste, Kasse,
// Strafen, Aufgaben, Videos, Ordner-Principals, Statistiken, Kalender-Feed) für
// sie unerreichbar, ohne dass eine davon etwas prüfen müsste.
//
// Deshalb ruft die Anlage hier bewusst KEIN ensureTeam.
package practicegroups

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/policy"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

// broadcast meldet eine Änderung an der Übungsgruppen-Liste. Bewusst global:
// die Gruppe ist ein Verwaltungsobjekt des Vorstands, und ihre Zielmenge ändert
// sich mit genau der Mutation, die hier gemeldet wird — eine vorab aufgelöste
// Empfängermenge wäre entweder die alte oder die neue, nie beide.
func (h *Handler) broadcast() {
	if h.hub == nil {
		return
	}
	h.hub.Broadcast("practice-groups")
}

type memberRow struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	UserID *int   `json:"user_id,omitempty"`
	Status string `json:"status"`
}

type group struct {
	ID       int         `json:"id"`
	SeasonID int         `json:"season_id"`
	Name     string      `json:"name"`
	Members  []memberRow `json:"members"`
	Trainers []memberRow `json:"trainers"`
}

// activeSeasonID liefert die aktive Saison. Ohne aktive Saison gibt es keinen
// Ort, an dem eine Übungsgruppe leben könnte (kader.season_id ist NOT NULL).
func (h *Handler) activeSeasonID(ctx context.Context) (int, error) {
	var id int
	err := h.db.QueryRowContext(ctx,
		`SELECT id FROM seasons WHERE is_active=1 LIMIT 1`).Scan(&id)
	return id, err
}

// loadPeople lädt Mitglieder (kader_members) oder Trainer (kader_trainers)
// einer Gruppe. table ist eine Konstante aus diesem Paket, nie Nutzereingabe.
func (h *Handler) loadPeople(ctx context.Context, table string, kaderID int) ([]memberRow, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT m.id, m.first_name || ' ' || m.last_name, m.user_id, m.status
		FROM `+table+` x
		JOIN members m ON m.id = x.member_id
		WHERE x.kader_id = ?
		ORDER BY m.last_name, m.first_name`, kaderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []memberRow{}
	for rows.Next() {
		var mr memberRow
		var uid sql.NullInt64
		if err := rows.Scan(&mr.ID, &mr.Name, &uid, &mr.Status); err != nil {
			return nil, err
		}
		if uid.Valid {
			v := int(uid.Int64)
			mr.UserID = &v
		}
		out = append(out, mr)
	}
	return out, rows.Err()
}

func (h *Handler) loadGroup(ctx context.Context, id int) (group, error) {
	var g group
	err := h.db.QueryRowContext(ctx,
		`SELECT id, season_id, COALESCE(name, '') FROM kader WHERE id=? AND kind='practice'`, id).
		Scan(&g.ID, &g.SeasonID, &g.Name)
	if err != nil {
		return g, err
	}
	if g.Members, err = h.loadPeople(ctx, "kader_members", id); err != nil {
		return g, err
	}
	g.Trainers, err = h.loadPeople(ctx, "kader_trainers", id)
	return g, err
}

// GET /api/practice-groups — nur die Gruppen der aktiven Saison.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id FROM kader
		WHERE kind='practice' AND season_id=(SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
		ORDER BY name`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ids = append(ids, id)
	}
	items := []group{}
	for _, id := range ids {
		g, err := h.loadGroup(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, g)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

// GET /api/practice-groups/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	g, err := h.loadGroup(r.Context(), id)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// POST /api/practice-groups
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	seasonID, err := h.activeSeasonID(r.Context())
	if err == sql.ErrNoRows {
		http.Error(w, "keine aktive Saison", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Kein ensureTeam: die Abwesenheit der teams-Zeile IST das Gate.
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO kader (season_id, kind, name) VALUES (?, 'practice', ?)`, seasonID, name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": "Eine Übungsgruppe mit diesem Namen existiert in dieser Saison bereits",
			})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	newID, _ := res.LastInsertId()
	g, err := h.loadGroup(r.Context(), int(newID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusCreated, g)
}

// PUT /api/practice-groups/{id} — umbenennen, Mitglieder und Trainer pflegen.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Name           *string `json:"name"`
		MembersAdd     []int   `json:"members_add"`
		MembersRemove  []int   `json:"members_remove"`
		TrainersAdd    []int   `json:"trainers_add"`
		TrainersRemove []int   `json:"trainers_remove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var exists int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM kader WHERE id=? AND kind='practice'`, id).Scan(&exists); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exists == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE kader SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, name, id); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				writeJSON(w, http.StatusConflict, map[string]any{
					"error": "Eine Übungsgruppe mit diesem Namen existiert in dieser Saison bereits",
				})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	type change struct {
		query string
		ids   []int
	}
	for _, c := range []change{
		{`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?,?)`, req.MembersAdd},
		{`DELETE FROM kader_members WHERE kader_id=? AND member_id=?`, req.MembersRemove},
		{`INSERT OR IGNORE INTO kader_trainers (kader_id, member_id) VALUES (?,?)`, req.TrainersAdd},
		{`DELETE FROM kader_trainers WHERE kader_id=? AND member_id=?`, req.TrainersRemove},
	} {
		for _, memberID := range c.ids {
			if _, err := tx.ExecContext(r.Context(), c.query, id, memberID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	g, err := h.loadGroup(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, g)
}

// DELETE /api/practice-groups/{id}
//
// 409 mit training_count, solange Termine oder Serien an der Gruppe hängen —
// dieselbe Form wie die Mitglieder-Guard in kader.DeleteKader. Ohne sie verlöre
// ein Aufräumen am Saisonende die Anwesenheits- und RSVP-Historie
// (training_sessions.kader_id trägt ON DELETE RESTRICT).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var exists int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM kader WHERE id=? AND kind='practice'`, id).Scan(&exists); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exists == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if n, err := policy.KaderTrainingCount(r.Context(), h.db, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if n > 0 {
		policy.WriteKaderTrainingConflict(w, n)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM kader WHERE id=?`, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.broadcast()
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/practice-groups/{id}/member-suggestions?search=
//
// Anders als bei der Mannschaftsvariante gibt es keinen Jahrgangs- oder
// Geschlechtsfilter: eine Übungsgruppe hat weder Altersklasse noch Geschlecht.
func (h *Handler) MemberSuggestions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	query := `
		SELECT m.id, m.first_name || ' ' || m.last_name,
		       COALESCE(CAST(strftime('%Y', m.date_of_birth) AS INTEGER), 0),
		       m.gender,
		       EXISTS(SELECT 1 FROM kader_members km WHERE km.kader_id=? AND km.member_id=m.id)
		FROM members m
		WHERE m.status != 'ausgetreten'`
	args := []any{id}
	if search != "" {
		query += ` AND (m.first_name || ' ' || m.last_name) LIKE ?`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY m.last_name, m.first_name LIMIT 20`

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type suggestion struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		BirthYear      int    `json:"birth_year"`
		Gender         string `json:"gender"`
		AlreadyInKader bool   `json:"already_in_kader"`
	}
	out := []suggestion{}
	for rows.Next() {
		var s suggestion
		var in int
		if err := rows.Scan(&s.ID, &s.Name, &s.BirthYear, &s.Gender, &in); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.AlreadyInKader = in == 1
		out = append(out, s)
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": out})
}

func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Fprintf(w, "\n")
	}
}
