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
	ID                 int    `json:"id"`
	SeasonID           int    `json:"season_id"`
	AgeClass           string `json:"age_class"`
	Gender             string `json:"gender"`
	TeamNumber         int    `json:"team_number"`
	DedicatedBirthYear *int   `json:"dedicated_birth_year"`
}

type memberRow struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	BirthYear int     `json:"birth_year"`
	Gender    string  `json:"gender"`
	Positions *string `json:"positions"`
}

type trainerRow struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type kaderDetail struct {
	kaderRow
	BirthYears   []int        `json:"birth_years"`   // filtered: [dedicated] or [yr1, yr2]
	BracketYears []int        `json:"bracket_years"` // always both bracket years [yr1, yr2]
	Members      []memberRow  `json:"members"`
	MemberCount  int          `json:"member_count"`
	Trainers     []trainerRow `json:"trainers"`
}

func computeBirthYears(k kaderRow, seasonStartYear int) (birthYears []int, bracketYears []int) {
	brackets := ComputeAgeBrackets(seasonStartYear)
	r, ok := brackets[k.AgeClass]
	bracketYears = []int{}
	if ok {
		bracketYears = []int{r[0], r[1]}
	}
	if k.DedicatedBirthYear != nil {
		birthYears = []int{*k.DedicatedBirthYear}
	} else {
		birthYears = bracketYears
	}
	return
}

func scanKaderRow(row interface{ Scan(...any) error }) (kaderRow, int, error) {
	var k kaderRow
	var seasonStartYear int
	var dedicatedBirthYear sql.NullInt64
	err := row.Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender, &k.TeamNumber, &dedicatedBirthYear, &seasonStartYear)
	if err != nil {
		return k, 0, err
	}
	if dedicatedBirthYear.Valid {
		v := int(dedicatedBirthYear.Int64)
		k.DedicatedBirthYear = &v
	}
	return k, seasonStartYear, nil
}

const kaderSelectSQL = `
	SELECT k.id, k.season_id, k.age_class, k.gender, k.team_number, k.dedicated_birth_year,
	       CAST(strftime('%Y', s.start_date) AS INTEGER)
	FROM kader k JOIN seasons s ON s.id = k.season_id`

// GET /api/admin/kader?season_id=N
func (h *Handler) ListKader(w http.ResponseWriter, r *http.Request) {
	seasonID := r.URL.Query().Get("season_id")
	var query string
	var args []any

	if seasonID != "" {
		query = kaderSelectSQL + ` WHERE k.season_id=? ORDER BY k.age_class, k.gender, k.team_number`
		args = append(args, seasonID)
	} else {
		query = kaderSelectSQL + ` WHERE k.season_id=(SELECT id FROM seasons WHERE is_active=1 LIMIT 1) ORDER BY k.age_class, k.gender, k.team_number`
	}

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []kaderDetail{}
	for rows.Next() {
		k, seasonStartYear, err := scanKaderRow(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		members, _ := h.loadMembers(r.Context(), k.ID)
		trainers, _ := h.loadTrainers(r.Context(), k.ID)
		bys, bkys := computeBirthYears(k, seasonStartYear)
		result = append(result, kaderDetail{
			kaderRow:     k,
			BirthYears:   bys,
			BracketYears: bkys,
			Members:      members,
			MemberCount:  len(members),
			Trainers:     trainers,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/admin/kader/{id}
func (h *Handler) GetKader(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	k, seasonStartYear, err := scanKaderRow(h.db.QueryRowContext(r.Context(),
		kaderSelectSQL+` WHERE k.id=?`, id))
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	members, _ := h.loadMembers(r.Context(), k.ID)
	trainers, _ := h.loadTrainers(r.Context(), k.ID)
	bys, bkys := computeBirthYears(k, seasonStartYear)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kaderDetail{
		kaderRow:     k,
		BirthYears:   bys,
		BracketYears: bkys,
		Members:      members,
		MemberCount:  len(members),
		Trainers:     trainers,
	})
}

// PUT /api/admin/kader/{id} — add/remove members, optionally update dedicated_birth_year
func (h *Handler) UpdateKader(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		MembersAdd            []int  `json:"members_add"`
		MembersRemove         []int  `json:"members_remove"`
		DedicatedBirthYear    *int   `json:"dedicated_birth_year"`
		SetDedicatedBirthYear bool   `json:"set_dedicated_birth_year"`
		TrainersAdd           []int  `json:"trainers_add"`
		TrainersRemove        []int  `json:"trainers_remove"`
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
	for _, memberID := range req.TrainersAdd {
		tx.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO kader_trainers (kader_id, member_id) VALUES (?,?)`, id, memberID)
	}
	for _, memberID := range req.TrainersRemove {
		tx.ExecContext(r.Context(),
			`DELETE FROM kader_trainers WHERE kader_id=? AND member_id=?`, id, memberID)
	}

	if req.DedicatedBirthYear != nil {
		tx.ExecContext(r.Context(),
			`UPDATE kader SET dedicated_birth_year=?, updated_at=? WHERE id=?`,
			*req.DedicatedBirthYear, time.Now(), id)
	} else if req.SetDedicatedBirthYear {
		// explicit null — reset to mixed mode
		tx.ExecContext(r.Context(),
			`UPDATE kader SET dedicated_birth_year=NULL, updated_at=? WHERE id=?`,
			time.Now(), id)
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(req.MembersAdd) > 0 || len(req.MembersRemove) > 0 {
		h.db.ExecContext(r.Context(), `UPDATE kader SET updated_at=? WHERE id=?`, time.Now(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/admin/kader/{id}/member-suggestions?search=&filter_age_bracket=true
func (h *Handler) MemberSuggestions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	filterByBracket := r.URL.Query().Get("filter_age_bracket") != "false"

	var k kaderRow
	var seasonStartYear int
	var dedicatedBirthYear sql.NullInt64
	err := h.db.QueryRowContext(r.Context(),
		`SELECT k.id, k.season_id, k.age_class, k.gender, k.team_number, k.dedicated_birth_year,
		        CAST(strftime('%Y', s.start_date) AS INTEGER)
		 FROM kader k JOIN seasons s ON s.id=k.season_id
		 WHERE k.id=?`, id).
		Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender, &k.TeamNumber, &dedicatedBirthYear, &seasonStartYear)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if dedicatedBirthYear.Valid {
		v := int(dedicatedBirthYear.Int64)
		k.DedicatedBirthYear = &v
	}

	suggestions, err := suggestMembers(r.Context(), h.db, k.ID, k.AgeClass, k.Gender, seasonStartYear, k.DedicatedBirthYear, search, filterByBracket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"suggestions": suggestions})
}

// POST /api/admin/kader — initialize standard kader OR create a single new kader
func (h *Handler) InitializeKader(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SeasonID           int  `json:"season_id"`
		AgeClass           string `json:"age_class"`
		Gender             string `json:"gender"`
		TeamNumber         int  `json:"team_number"`
		DedicatedBirthYear *int `json:"dedicated_birth_year"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SeasonID == 0 {
		http.Error(w, "season_id required", http.StatusBadRequest)
		return
	}

	// Single kader creation when age_class and gender are provided
	if req.AgeClass != "" && req.Gender != "" {
		if req.TeamNumber == 0 {
			req.TeamNumber = 1
		}
		h.createSingleKader(w, r, req.SeasonID, req.AgeClass, req.Gender, req.TeamNumber, req.DedicatedBirthYear)
		return
	}

	// Bulk initialization: standard set of 7 kader with team_number=1
	type kaderSpec struct{ ageClass, gender string }
	specs := []kaderSpec{
		{"A-Jugend", "m"}, {"A-Jugend", "f"},
		{"B-Jugend", "m"}, {"B-Jugend", "f"},
		{"C-Jugend", "m"}, {"C-Jugend", "f"},
		{"D-Jugend", "mixed"},
	}
	for _, s := range specs {
		h.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO kader (season_id, age_class, gender, team_number) VALUES (?,?,?,1)`,
			req.SeasonID, s.ageClass, s.gender)
	}
	w.WriteHeader(http.StatusCreated)
}

// createSingleKader inserts one kader entry and returns 201 with the new object, or 409 on conflict.
func (h *Handler) createSingleKader(w http.ResponseWriter, r *http.Request, seasonID int, ageClass, gender string, teamNumber int, dedicatedBirthYear *int) {
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO kader (season_id, age_class, gender, team_number, dedicated_birth_year) VALUES (?,?,?,?,?)`,
		seasonID, ageClass, gender, teamNumber, dedicatedBirthYear)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, `{"error":"Kader existiert bereits"}`, http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	newID, _ := res.LastInsertId()

	k, seasonStartYear, err := scanKaderRow(h.db.QueryRowContext(r.Context(),
		kaderSelectSQL+` WHERE k.id=?`, newID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bys, bkys := computeBirthYears(k, seasonStartYear)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(kaderDetail{
		kaderRow:     k,
		BirthYears:   bys,
		BracketYears: bkys,
		Members:      []memberRow{},
		MemberCount:  0,
		Trainers:     []trainerRow{},
	})
}

// DELETE /api/admin/kader/{id}
func (h *Handler) DeleteKader(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var memberCount int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM kader_members WHERE kader_id=?`, id).Scan(&memberCount)

	if memberCount > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error":        "Kader hat noch Mitglieder",
			"member_count": memberCount,
		})
		return
	}

	_, err := h.db.ExecContext(r.Context(), `DELETE FROM kader WHERE id=?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// POST /api/admin/kader/auto-assign — auto-assign members to multiple kader by age/gender bracket
func (h *Handler) AutoAssign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		KaderIDs []int `json:"kader_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.KaderIDs) == 0 {
		http.Error(w, "kader_ids required", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, kaderID := range req.KaderIDs {
		var k kaderRow
		var seasonStartYear int
		var dedicatedBirthYear sql.NullInt64
		err := tx.QueryRowContext(r.Context(),
			`SELECT k.id, k.season_id, k.age_class, k.gender, k.team_number, k.dedicated_birth_year,
			        CAST(strftime('%Y', s.start_date) AS INTEGER)
			 FROM kader k JOIN seasons s ON s.id=k.season_id
			 WHERE k.id=?`, kaderID).
			Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender, &k.TeamNumber, &dedicatedBirthYear, &seasonStartYear)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if dedicatedBirthYear.Valid {
			v := int(dedicatedBirthYear.Int64)
			k.DedicatedBirthYear = &v
		}

		_, err = autoAssignMembers(r.Context(), tx, k.ID, k.AgeClass, k.Gender, seasonStartYear, k.DedicatedBirthYear)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (h *Handler) loadTrainers(ctx context.Context, kaderID int) ([]trainerRow, error) {
	rows, err := h.db.QueryContext(ctx,
		`SELECT m.id, m.first_name || ' ' || m.last_name
		 FROM kader_trainers kt
		 JOIN members m ON m.id = kt.member_id
		 WHERE kt.kader_id = ?
		 ORDER BY m.last_name, m.first_name`, kaderID)
	if err != nil {
		return []trainerRow{}, err
	}
	defer rows.Close()
	result := []trainerRow{}
	for rows.Next() {
		var t trainerRow
		rows.Scan(&t.ID, &t.Name)
		result = append(result, t)
	}
	return result, nil
}

func (h *Handler) loadMembers(ctx context.Context, kaderID int) ([]memberRow, error) {
	rows, err := h.db.QueryContext(ctx,
		`SELECT m.id,
		        m.first_name || ' ' || m.last_name,
		        COALESCE(CAST(strftime('%Y', m.date_of_birth) AS INTEGER), 0),
		        m.gender,
		        m.position
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
		rows.Scan(&m.ID, &m.Name, &m.BirthYear, &m.Gender, &m.Positions)
		result = append(result, m)
	}
	return result, nil
}
