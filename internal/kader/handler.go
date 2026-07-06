package kader

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/policy"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

// broadcastKader sends the "kader" event only to the team audience of the given
// kader (team members + trainers/sL + parents + vorstand/admin). Replaces the
// former global Broadcast; the Frontend contract (topic string + useLiveUpdates)
// is unchanged, only the recipient set shrinks. A kader without a team assigned
// resolves to only the club-wide staff audience. Resolve the team BEFORE
// deleting the kader (its team_id is gone afterwards).
func (h *Handler) broadcastKader(ctx context.Context, kaderID any) {
	if h.hub == nil {
		return
	}
	a := hub.NewAudience(h.db)
	ids := a.Team(ctx, a.TeamIDsForKader(ctx, kaderID))
	h.hub.BroadcastToUsers(ids, "kader")
}

// broadcastKaderTeams broadcasts "kader" to an already-resolved team-ID set
// (used by DeleteKader, which captures the team before the row is gone).
func (h *Handler) broadcastKaderTeams(ctx context.Context, teamIDs []int) {
	if h.hub == nil {
		return
	}
	ids := hub.NewAudience(h.db).Team(ctx, teamIDs)
	h.hub.BroadcastToUsers(ids, "kader")
}

// broadcastKaderUpdate broadcasts "kader" to the union of (a) the given team IDs
// — which must include BOTH the old and new team when UpdateKader repointed
// team_id via an age-class change — and (b) the audience of members that were
// removed from the roster. Without (a) an age-class/team switch would only reach
// the new team; without (b) a removed member (no longer in the team audience)
// would never learn of their removal and stay on a roster they left. The
// removed-member audience mirrors absences.broadcastMemberEvents / MembersAudience
// (member's own user + parents + their teams).
func (h *Handler) broadcastKaderUpdate(ctx context.Context, teamIDs, removedMemberIDs []int) {
	if h.hub == nil {
		return
	}
	a := hub.NewAudience(h.db)
	set := map[int]struct{}{}
	for _, id := range a.Team(ctx, teamIDs) {
		set[id] = struct{}{}
	}
	if len(removedMemberIDs) > 0 {
		for _, id := range a.MembersAudience(ctx, removedMemberIDs) {
			set[id] = struct{}{}
		}
	}
	ids := make([]int, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	h.hub.BroadcastToUsers(ids, "kader")
}

// unionInts merges several int slices into one with duplicates removed.
func unionInts(slices ...[]int) []int {
	set := map[int]struct{}{}
	out := []int{}
	for _, s := range slices {
		for _, v := range s {
			if _, ok := set[v]; !ok {
				set[v] = struct{}{}
				out = append(out, v)
			}
		}
	}
	return out
}

type dbq interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func teamLabel(ageClass, gender string, teamNumber int) string {
	g := map[string]string{"m": "männlich", "f": "weiblich", "mixed": "gemischt"}
	name := ageClass + " " + g[gender]
	if teamNumber > 1 {
		name += " " + fmt.Sprintf("%d", teamNumber)
	}
	return name
}

// ensureTeam returns the id of the team matching ageClass/gender/teamNumber,
// creating it if it doesn't exist yet.
func ensureTeam(ctx context.Context, db dbq, ageClass, gender string, teamNumber int) (int64, error) {
	name := teamLabel(ageClass, gender, teamNumber)
	var id int64
	err := db.QueryRowContext(ctx,
		`SELECT id FROM teams WHERE name=? AND age_class=? AND gender=? LIMIT 1`,
		name, ageClass, gender).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.ExecContext(ctx,
			`INSERT INTO teams (name, age_class, gender, is_active) VALUES (?,?,?,1)`,
			name, ageClass, gender)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	return id, err
}

type kaderRow struct {
	ID                 int    `json:"id"`
	SeasonID           int    `json:"season_id"`
	AgeClass           string `json:"age_class"`
	Gender             string `json:"gender"`
	TeamNumber         int    `json:"team_number"`
	TeamID             int64  `json:"team_id"`
	DedicatedBirthYear *int   `json:"dedicated_birth_year"`
	GamesPerSeason     int    `json:"games_per_season"`
}

type memberRow struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	BirthYear int     `json:"birth_year"`
	Gender    string  `json:"gender"`
	Positions *string `json:"positions"`
	Status    string  `json:"status"`
}

type trainerRow struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	UserID *int   `json:"user_id,omitempty"`
	Status string `json:"status"`
}

type kaderDetail struct {
	kaderRow
	BirthYears      []int           `json:"birth_years"`   // filtered: [dedicated] or [yr1, yr2]
	BracketYears    []int           `json:"bracket_years"` // always both bracket years [yr1, yr2]
	Members         []memberRow     `json:"members"`
	MemberCount     int             `json:"member_count"`
	Trainers        []trainerRow    `json:"trainers"`
	ExtendedMembers []memberRow     `json:"extended_members"`
	Can             policy.CanFlags `json:"can"`
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
	err := row.Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender, &k.TeamNumber, &k.TeamID, &dedicatedBirthYear, &k.GamesPerSeason, &seasonStartYear)
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
	SELECT k.id, k.season_id, k.age_class, k.gender, k.team_number, k.team_id, k.dedicated_birth_year, k.games_per_season,
	       CAST(strftime('%Y', s.start_date) AS INTEGER)
	FROM kader k JOIN seasons s ON s.id = k.season_id`

// GET /api/admin/kader?season_id=N&limit=&offset=
func (h *Handler) ListKader(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	p := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions}
	kaderCan := policy.CanFlags{Edit: policy.CanEditKader(p), Delete: policy.CanEditKader(p)}

	seasonID := r.URL.Query().Get("season_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if limit < 1 {
		limit = 50
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}
	if offset < 0 {
		offset = 0
	}

	where := ` WHERE k.season_id=(SELECT id FROM seasons WHERE is_active=1 LIMIT 1)`
	var args []any
	if seasonID != "" {
		where = ` WHERE k.season_id=?`
		args = append(args, seasonID)
	}

	// total mit denselben WHERE-Bedingungen wie die Items (Sichtbarkeit invariant).
	var total int
	if err := h.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM kader k`+where, args...).Scan(&total); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := kaderSelectSQL + where + ` ORDER BY k.age_class, k.gender, k.team_number LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

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
		extended, _ := h.loadExtendedMembers(r.Context(), k.ID)
		bys, bkys := computeBirthYears(k, seasonStartYear)
		result = append(result, kaderDetail{
			kaderRow:        k,
			BirthYears:      bys,
			BracketYears:    bkys,
			Members:         members,
			MemberCount:     len(members),
			Trainers:        trainers,
			ExtendedMembers: extended,
			Can:             kaderCan,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": result, "total": total})
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

	gclaims := auth.ClaimsFromCtx(r.Context())
	gp := &policy.Principal{UserID: gclaims.UserID, Role: gclaims.Role, ClubFunctions: gclaims.ClubFunctions}
	members, _ := h.loadMembers(r.Context(), k.ID)
	trainers, _ := h.loadTrainers(r.Context(), k.ID)
	extended, _ := h.loadExtendedMembers(r.Context(), k.ID)
	bys, bkys := computeBirthYears(k, seasonStartYear)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kaderDetail{
		kaderRow:        k,
		BirthYears:      bys,
		BracketYears:    bkys,
		Members:         members,
		MemberCount:     len(members),
		Trainers:        trainers,
		ExtendedMembers: extended,
		Can:             policy.CanFlags{Edit: policy.CanEditKader(gp), Delete: policy.CanEditKader(gp)},
	})
}

// PUT /api/admin/kader/{id} — add/remove members, optionally update dedicated_birth_year
func (h *Handler) UpdateKader(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		MembersAdd            []int   `json:"members_add"`
		MembersRemove         []int   `json:"members_remove"`
		DedicatedBirthYear    *int    `json:"dedicated_birth_year"`
		SetDedicatedBirthYear bool    `json:"set_dedicated_birth_year"`
		TrainersAdd           []int   `json:"trainers_add"`
		TrainersRemove        []int   `json:"trainers_remove"`
		AgeClass              *string `json:"age_class"`
		ExtendedMembersAdd    []int   `json:"extended_members_add"`
		ExtendedMembersRemove []int   `json:"extended_members_remove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Team(s) VOR den Änderungen erfassen: bei einem Age-Class-Wechsel repointet
	// UPDATE kader.team_id auf ein anderes Team, sodass TeamIDsForKader nach dem
	// Commit nur noch das NEUE Team liefert. Das ALTE Team muss aber ebenfalls
	// benachrichtigt werden (sein Roster ändert sich).
	oldTeamIDs := hub.NewAudience(h.db).TeamIDsForKader(r.Context(), id)

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	execTx := func(query string, args ...any) error {
		_, err := tx.ExecContext(r.Context(), query, args...)
		return err
	}

	for _, memberID := range req.MembersAdd {
		if err := execTx(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?,?)`, id, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, memberID := range req.MembersRemove {
		if err := execTx(`DELETE FROM kader_members WHERE kader_id=? AND member_id=?`, id, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, memberID := range req.TrainersAdd {
		if err := execTx(`INSERT OR IGNORE INTO kader_trainers (kader_id, member_id) VALUES (?,?)`, id, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, memberID := range req.TrainersRemove {
		if err := execTx(`DELETE FROM kader_trainers WHERE kader_id=? AND member_id=?`, id, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, memberID := range req.ExtendedMembersAdd {
		if err := execTx(`INSERT OR IGNORE INTO kader_extended_members (kader_id, member_id) VALUES (?,?)`, id, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, memberID := range req.ExtendedMembersRemove {
		if err := execTx(`DELETE FROM kader_extended_members WHERE kader_id=? AND member_id=?`, id, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if req.DedicatedBirthYear != nil {
		if err := execTx(`UPDATE kader SET dedicated_birth_year=?, updated_at=? WHERE id=?`,
			*req.DedicatedBirthYear, time.Now(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if req.SetDedicatedBirthYear {
		if err := execTx(`UPDATE kader SET dedicated_birth_year=NULL, updated_at=? WHERE id=?`,
			time.Now(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if req.AgeClass != nil && *req.AgeClass != "" {
		var gender string
		var teamNumber int
		if err := tx.QueryRowContext(r.Context(), `SELECT gender, team_number FROM kader WHERE id=?`, id).
			Scan(&gender, &teamNumber); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "kader not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		newTeamID, err := ensureTeam(r.Context(), tx, *req.AgeClass, gender, teamNumber)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := execTx(`UPDATE kader SET age_class=?, team_id=?, updated_at=? WHERE id=?`,
			*req.AgeClass, newTeamID, time.Now(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(req.MembersAdd) > 0 || len(req.MembersRemove) > 0 {
		h.db.ExecContext(r.Context(), `UPDATE kader SET updated_at=? WHERE id=?`, time.Now(), id)
	}

	// Publikum = altes ∪ neues Team (deckt Age-Class/Team-Wechsel ab; bei
	// unverändertem Team fallen beide zusammen) plus die entfernten
	// Member/Trainer/Extended-Member, damit auch sie (bzw. Besitzer/Eltern) ihren
	// Rauswurf erfahren, obwohl sie nicht mehr im aktuellen Team-Publikum stehen.
	newTeamIDs := hub.NewAudience(h.db).TeamIDsForKader(r.Context(), id)
	teamIDs := unionInts(oldTeamIDs, newTeamIDs)
	removedMemberIDs := unionInts(req.MembersRemove, req.TrainersRemove, req.ExtendedMembersRemove)
	h.broadcastKaderUpdate(r.Context(), teamIDs, removedMemberIDs)
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
		`SELECT k.id, k.season_id, k.age_class, k.gender, k.team_number, k.team_id, k.dedicated_birth_year,
		        CAST(strftime('%Y', s.start_date) AS INTEGER)
		 FROM kader k JOIN seasons s ON s.id=k.season_id
		 WHERE k.id=?`, id).
		Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender, &k.TeamNumber, &k.TeamID, &dedicatedBirthYear, &seasonStartYear)
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
		SeasonID           int    `json:"season_id"`
		AgeClass           string `json:"age_class"`
		Gender             string `json:"gender"`
		TeamNumber         int    `json:"team_number"`
		DedicatedBirthYear *int   `json:"dedicated_birth_year"`
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
		teamID, _ := ensureTeam(r.Context(), h.db, s.ageClass, s.gender, 1)
		h.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO kader (season_id, age_class, gender, team_number, team_id) VALUES (?,?,?,1,?)`,
			req.SeasonID, s.ageClass, s.gender, teamID)
		h.db.ExecContext(r.Context(),
			`UPDATE kader SET team_id=? WHERE season_id=? AND age_class=? AND gender=? AND team_number=1 AND team_id IS NULL`,
			teamID, req.SeasonID, s.ageClass, s.gender)
	}
	// Initialisiert mehrere Kader über eine ganze Saison → bewusst global.
	h.hub.Broadcast("kader")
	w.WriteHeader(http.StatusCreated)
}

// createSingleKader inserts one kader entry and returns 201 with the new object, or 409 on conflict.
func (h *Handler) createSingleKader(w http.ResponseWriter, r *http.Request, seasonID int, ageClass, gender string, teamNumber int, dedicatedBirthYear *int) {
	teamID, err := ensureTeam(r.Context(), h.db, ageClass, gender, teamNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO kader (season_id, age_class, gender, team_number, dedicated_birth_year, team_id) VALUES (?,?,?,?,?,?)`,
		seasonID, ageClass, gender, teamNumber, dedicatedBirthYear, teamID)
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
	h.broadcastKaderTeams(r.Context(), []int{int(teamID)})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(kaderDetail{
		kaderRow:        k,
		BirthYears:      bys,
		BracketYears:    bkys,
		Members:         []memberRow{},
		MemberCount:     0,
		Trainers:        []trainerRow{},
		ExtendedMembers: []memberRow{},
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

	// Team vor dem Löschen auflösen (team_id ist danach weg).
	teamIDs := hub.NewAudience(h.db).TeamIDsForKader(r.Context(), id)

	_, err := h.db.ExecContext(r.Context(), `DELETE FROM kader WHERE id=?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.broadcastKaderTeams(r.Context(), teamIDs)
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

	// Kopiert mehrere Kader saisonuebergreifend → bewusst global.
	h.hub.Broadcast("kader")
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
			`SELECT k.id, k.season_id, k.age_class, k.gender, k.team_number, k.team_id, k.dedicated_birth_year,
			        CAST(strftime('%Y', s.start_date) AS INTEGER)
			 FROM kader k JOIN seasons s ON s.id=k.season_id
			 WHERE k.id=?`, kaderID).
			Scan(&k.ID, &k.SeasonID, &k.AgeClass, &k.Gender, &k.TeamNumber, &k.TeamID, &dedicatedBirthYear, &seasonStartYear)
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

	// Bulk-Zuordnung über mehrere Kader (req.KaderIDs) → bewusst global.
	h.hub.Broadcast("kader")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// PATCH /api/admin/kader/{id}/games-per-season
func (h *Handler) PatchGamesPerSeason(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		GamesPerSeason int `json:"games_per_season"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.GamesPerSeason < 0 {
		http.Error(w, "games_per_season must be >= 0", http.StatusBadRequest)
		return
	}
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE kader SET games_per_season=?, updated_at=? WHERE id=?`,
		req.GamesPerSeason, time.Now(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.broadcastKader(r.Context(), id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) loadTrainers(ctx context.Context, kaderID int) ([]trainerRow, error) {
	rows, err := h.db.QueryContext(ctx,
		`SELECT m.id, m.first_name || ' ' || m.last_name, m.user_id, m.status
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
		var userID sql.NullInt64
		rows.Scan(&t.ID, &t.Name, &userID, &t.Status)
		if userID.Valid {
			n := int(userID.Int64)
			t.UserID = &n
		}
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
		        m.position,
		        m.status
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
		rows.Scan(&m.ID, &m.Name, &m.BirthYear, &m.Gender, &m.Positions, &m.Status)
		result = append(result, m)
	}
	return result, nil
}

func (h *Handler) loadExtendedMembers(ctx context.Context, kaderID int) ([]memberRow, error) {
	rows, err := h.db.QueryContext(ctx,
		`SELECT m.id,
		        m.first_name || ' ' || m.last_name,
		        COALESCE(CAST(strftime('%Y', m.date_of_birth) AS INTEGER), 0),
		        m.gender,
		        m.position,
		        m.status
		 FROM kader_extended_members kem
		 JOIN members m ON m.id=kem.member_id
		 WHERE kem.kader_id=?
		 ORDER BY m.last_name, m.first_name`, kaderID)
	if err != nil {
		return []memberRow{}, err
	}
	defer rows.Close()

	result := []memberRow{}
	for rows.Next() {
		var m memberRow
		rows.Scan(&m.ID, &m.Name, &m.BirthYear, &m.Gender, &m.Positions, &m.Status)
		result = append(result, m)
	}
	return result, nil
}

// GET /api/admin/kader/{id}/extended-member-suggestions?search=
func (h *Handler) ExtendedMemberSuggestions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	var kaderID int
	if err := h.db.QueryRowContext(r.Context(), `SELECT id FROM kader WHERE id=?`, id).Scan(&kaderID); err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT m.id,
		        m.first_name || ' ' || m.last_name,
		        COALESCE(CAST(strftime('%Y', m.date_of_birth) AS INTEGER), 0),
		        m.gender,
		        EXISTS(SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id=? AND kem.member_id=m.id) AS in_extended
		 FROM members m
		 WHERE m.status != 'ausgetreten'
		   AND (? = '' OR (m.first_name || ' ' || m.last_name) LIKE ?)
		 ORDER BY m.last_name, m.first_name
		 LIMIT 20`,
		kaderID, search, "%"+search+"%")
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
	result := []suggestion{}
	for rows.Next() {
		var s suggestion
		var inExtended int
		rows.Scan(&s.ID, &s.Name, &s.BirthYear, &s.Gender, &inExtended)
		s.AlreadyInKader = inExtended == 1
		result = append(result, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"suggestions": result})
}
