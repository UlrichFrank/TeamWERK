package kader

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

type kaderRow struct {
	ID       int    `json:"id"`
	SeasonID int    `json:"season_id"`
	AgeClass string `json:"age_class"`
	Gender   string `json:"gender"`
}

type memberRow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	BirthYear int    `json:"birth_year"`
	Gender    string `json:"gender"`
}

type kaderDetail struct {
	kaderRow
	Members     []memberRow `json:"members"`
	MemberCount int         `json:"member_count"`
}

// GET /api/admin/kader?season_id=N
func (h *Handler) ListKader(w http.ResponseWriter, r *http.Request) {
	seasonID := r.URL.Query().Get("season_id")
	var query string
	var args []any

	if seasonID != "" {
		query = `SELECT id, season_id, age_class, gender FROM kader WHERE season_id=? ORDER BY age_class, gender`
		args = append(args, seasonID)
	} else {
		query = `SELECT id, season_id, age_class, gender FROM kader WHERE season_id=(SELECT id FROM seasons WHERE is_active=1 LIMIT 1) ORDER BY age_class, gender`
	}

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []kaderDetail{}
	for rows.Next() {
		var k kaderDetail
		rows.Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender)
		members, _ := h.loadMembers(r.Context(), k.ID)
		k.Members = members
		k.MemberCount = len(members)
		result = append(result, k)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/admin/kader/:id
func (h *Handler) GetKader(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var k kaderDetail
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, season_id, age_class, gender FROM kader WHERE id=?`, id).
		Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	members, _ := h.loadMembers(r.Context(), k.ID)
	k.Members = members
	k.MemberCount = len(members)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(k)
}

// PUT /api/admin/kader/:id — add/remove members
func (h *Handler) UpdateKader(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		MembersAdd    []int `json:"members_add"`
		MembersRemove []int `json:"members_remove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, memberID := range req.MembersAdd {
		tx.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?,?)`, id, memberID)
	}
	for _, memberID := range req.MembersRemove {
		tx.ExecContext(r.Context(),
			`DELETE FROM kader_members WHERE kader_id=? AND member_id=?`, id, memberID)
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.db.ExecContext(r.Context(), `UPDATE kader SET updated_at=? WHERE id=?`, time.Now(), id)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/admin/kader/:id/member-suggestions?search=&filter_age_bracket=true
func (h *Handler) MemberSuggestions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	filterByBracket := r.URL.Query().Get("filter_age_bracket") != "false"

	var k kaderRow
	var seasonStartYear int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT k.id, k.season_id, k.age_class, k.gender,
		        CAST(strftime('%Y', s.start_date) AS INTEGER)
		 FROM kader k JOIN seasons s ON s.id=k.season_id
		 WHERE k.id=?`, id).
		Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender, &seasonStartYear)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	suggestions, err := suggestMembers(r.Context(), h.db, k.ID, k.AgeClass, k.Gender, seasonStartYear, search, filterByBracket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"suggestions": suggestions})
}

// POST /api/admin/kader — create standard Kader set for a season (first-time setup)
func (h *Handler) InitializeKader(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SeasonID int `json:"season_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SeasonID == 0 {
		http.Error(w, "season_id required", http.StatusBadRequest)
		return
	}

	type kaderSpec struct{ ageClass, gender string }
	specs := []kaderSpec{
		{"A-Jugend", "m"}, {"A-Jugend", "f"},
		{"B-Jugend", "m"}, {"B-Jugend", "f"},
		{"C-Jugend", "m"}, {"C-Jugend", "f"},
		{"D-Jugend", "mixed"},
	}

	for _, s := range specs {
		h.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO kader (season_id, age_class, gender) VALUES (?,?,?)`,
			req.SeasonID, s.ageClass, s.gender)
	}

	w.WriteHeader(http.StatusCreated)
}

// POST /api/admin/kader/copy-from-season
func (h *Handler) CopyFromSeason(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromSeasonID int              `json:"from_season_id"`
		ToSeasonID   int              `json:"to_season_id"`
		Assignments  []CopyAssignment `json:"assignments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.FromSeasonID == 0 || req.ToSeasonID == 0 {
		http.Error(w, "from_season_id and to_season_id required", http.StatusBadRequest)
		return
	}

	var targetStartYear int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT CAST(strftime('%Y', start_date) AS INTEGER) FROM seasons WHERE id=?`, req.ToSeasonID).
		Scan(&targetStartYear)
	if err != nil {
		http.Error(w, "target season not found", http.StatusBadRequest)
		return
	}

	created, err := copyKader(r.Context(), h.db, req.FromSeasonID, req.ToSeasonID, targetStartYear, req.Assignments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"created": created})
}

func (h *Handler) loadMembers(ctx context.Context, kaderID int) ([]memberRow, error) {
	rows, err := h.db.QueryContext(ctx,
		`SELECT m.id,
		        m.first_name || ' ' || m.last_name,
		        COALESCE(CAST(strftime('%Y', m.date_of_birth) AS INTEGER), 0),
		        m.gender
		 FROM kader_members km
		 JOIN members m ON m.id=km.member_id
		 WHERE km.kader_id=?
		 ORDER BY m.last_name, m.first_name`, kaderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []memberRow{}
	for rows.Next() {
		var m memberRow
		rows.Scan(&m.ID, &m.Name, &m.BirthYear, &m.Gender)
		result = append(result, m)
	}
	return result, nil
}
