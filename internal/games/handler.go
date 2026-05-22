package games

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

// addMinutes adds offset to a "HH:MM" string, wrapping around 24 hours.
func addMinutes(t string, offset int) string {
	if len(t) < 5 {
		return t
	}
	h, _ := strconv.Atoi(t[:2])
	m, _ := strconv.Atoi(t[3:])
	total := h*60 + m + offset
	total = ((total % 1440) + 1440) % 1440
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

// findTemplateForGame returns the best-matching template for a game.
// For home games: tries 'heim', falls back to 'generisch'.
// For away games: tries 'auswärts', falls back to 'generisch'.
func (h *Handler) findTemplateForGame(ctx context.Context, isHome bool) (id, durationMins int, err error) {
	targetType := "auswärts"
	if isHome {
		targetType = "heim"
	}
	err = h.db.QueryRowContext(ctx,
		`SELECT id, game_duration_minutes FROM game_templates WHERE template_type=? ORDER BY id LIMIT 1`, targetType).
		Scan(&id, &durationMins)
	if err == sql.ErrNoRows {
		err = h.db.QueryRowContext(ctx,
			`SELECT id, game_duration_minutes FROM game_templates WHERE template_type='generisch' ORDER BY id LIMIT 1`).
			Scan(&id, &durationMins)
	}
	if err == sql.ErrNoRows {
		return 0, 0, fmt.Errorf("kein passendes Dienstplan-Template gefunden (Typ: %s oder generisch)", targetType)
	}
	return id, durationMins, err
}

type templateItemRow struct {
	DutyTypeID           int
	DutyTypeName         string
	Anchor               string
	OffsetMinutes        int
	SlotsCount           int
	RoleDesc             string
	SameDayBehavior      string
	SameDayVariantID     sql.NullInt64
	AdjacentDayBehavior  string
	AdjacentDayVariantID sql.NullInt64
}

func (h *Handler) loadTemplateItems(ctx context.Context, templateID int) ([]templateItemRow, error) {
	rows, err := h.db.QueryContext(ctx,
		`SELECT gti.duty_type_id, dt.name, gti.anchor, gti.offset_minutes, gti.slots_count, COALESCE(gti.role_desc,''),
		        dt.same_day_behavior, dt.same_day_variant_id, dt.adjacent_day_behavior, dt.adjacent_day_variant_id
		 FROM game_template_items gti JOIN duty_types dt ON dt.id = gti.duty_type_id
		 WHERE gti.template_id=? ORDER BY gti.sort_order, gti.id`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []templateItemRow
	for rows.Next() {
		var it templateItemRow
		rows.Scan(&it.DutyTypeID, &it.DutyTypeName, &it.Anchor, &it.OffsetMinutes,
			&it.SlotsCount, &it.RoleDesc, &it.SameDayBehavior, &it.SameDayVariantID,
			&it.AdjacentDayBehavior, &it.AdjacentDayVariantID)
		result = append(result, it)
	}
	return result, nil
}

// applyBehavior returns the effective dutyTypeID after applying same-day/adjacent-day rules,
// or -1 if the slot should be skipped.
func applyBehavior(it templateItemRow, isMultiGame, hasPrevDay, hasNextDay, isFirst, isLast bool) int {
	dutyTypeID := it.DutyTypeID
	skip := false

	if isMultiGame && it.SameDayBehavior != "normal" {
		if it.SameDayBehavior == "skip" {
			skip = true
		} else if it.SameDayBehavior == "reduced" && it.SameDayVariantID.Valid {
			dutyTypeID = int(it.SameDayVariantID.Int64)
		}
	}

	shouldApplyAdjacent := (isFirst && hasPrevDay) || (isLast && hasNextDay)
	if shouldApplyAdjacent && it.AdjacentDayBehavior != "normal" {
		if it.AdjacentDayBehavior == "skip" {
			skip = true
		} else if it.AdjacentDayBehavior == "reduced" && it.AdjacentDayVariantID.Valid {
			if !(it.SameDayBehavior == "reduced" && it.SameDayVariantID.Valid) {
				dutyTypeID = int(it.AdjacentDayVariantID.Int64)
			}
		}
	}

	if skip {
		return -1
	}
	return dutyTypeID
}

func (h *Handler) loadSameDayContext(ctx context.Context, gameDate string, seasonID int) (
	allGameTimes []string, hasPrevDay, hasNextDay bool,
) {
	gtRows, _ := h.db.QueryContext(ctx,
		`SELECT time FROM games WHERE date=? AND is_home=1 AND season_id=? ORDER BY time`,
		gameDate, seasonID)
	if gtRows != nil {
		defer gtRows.Close()
		for gtRows.Next() {
			var t string
			gtRows.Scan(&t)
			allGameTimes = append(allGameTimes, t)
		}
	}
	var prevCount, nextCount int
	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM games WHERE date=date(?, '-1 days') AND is_home=1 AND season_id=?`,
		gameDate, seasonID).Scan(&prevCount)
	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM games WHERE date=date(?, '+1 days') AND is_home=1 AND season_id=?`,
		gameDate, seasonID).Scan(&nextCount)
	return allGameTimes, prevCount > 0, nextCount > 0
}

// ── Games ────────────────────────────────────────────────────────────────────

// GET /api/games
func (h *Handler) ListGames(w http.ResponseWriter, r *http.Request) {
	seasonID := r.URL.Query().Get("season_id")

	const base = `
		SELECT g.id, g.date, g.time, g.opponent, g.event_type,
		       COUNT(DISTINCT ds.id), COALESCE(SUM(ds.slots_filled),0), COALESCE(SUM(ds.slots_total),0)
		FROM games g
		LEFT JOIN duty_slots ds ON ds.game_id = g.id`
	const suffix = ` GROUP BY g.id ORDER BY g.date, g.time`

	var (
		rows *sql.Rows
		err  error
	)
	if seasonID != "" {
		rows, err = h.db.QueryContext(r.Context(), base+` WHERE g.season_id=?`+suffix, seasonID)
	} else {
		rows, err = h.db.QueryContext(r.Context(),
			base+` WHERE g.season_id=(SELECT id FROM seasons WHERE is_active=1 LIMIT 1)`+suffix)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type team struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	type game struct {
		ID          int    `json:"id"`
		Date        string `json:"date"`
		Time        string `json:"time"`
		Opponent    string `json:"opponent"`
		EventType   string `json:"event_type"`
		Teams       []team `json:"teams"`
		SlotCount   int    `json:"slot_count"`
		FilledCount int    `json:"filled_count"`
		TotalCount  int    `json:"total_count"`
	}

	var games []*game
	for rows.Next() {
		var g game
		if err := rows.Scan(&g.ID, &g.Date, &g.Time, &g.Opponent, &g.EventType,
			&g.SlotCount, &g.FilledCount, &g.TotalCount); err != nil {
			continue
		}
		g.Teams = []team{}
		games = append(games, &g)
	}

	for _, g := range games {
		teamRows, _ := h.db.QueryContext(r.Context(),
			`SELECT t.id, t.name FROM teams t
			 JOIN game_teams gt ON gt.team_id = t.id
			 WHERE gt.game_id = ?`, g.ID)
		if teamRows != nil {
			for teamRows.Next() {
				var t team
				teamRows.Scan(&t.ID, &t.Name)
				g.Teams = append(g.Teams, t)
			}
			teamRows.Close()
		}
	}

	result := make([]game, len(games))
	for i, g := range games {
		result[i] = *g
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/games/{id}
func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var g struct {
		ID        int    `json:"id"`
		Date      string `json:"date"`
		Time      string `json:"time"`
		Opponent  string `json:"opponent"`
		EventType string `json:"event_type"`
		SeasonID  int    `json:"season_id"`
		Teams     []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"teams"`
	}
	err := h.db.QueryRowContext(r.Context(),
		`SELECT g.id, g.date, g.time, g.opponent, g.event_type, g.season_id
		 FROM games g WHERE g.id=?`, id).
		Scan(&g.ID, &g.Date, &g.Time, &g.Opponent, &g.EventType, &g.SeasonID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	teamRows, _ := h.db.QueryContext(r.Context(),
		`SELECT t.id, t.name FROM teams t
		 JOIN game_teams gt ON gt.team_id = t.id
		 WHERE gt.game_id = ?`, id)
	if teamRows != nil {
		for teamRows.Next() {
			var t struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}
			teamRows.Scan(&t.ID, &t.Name)
			g.Teams = append(g.Teams, t)
		}
		teamRows.Close()
	}

	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT ds.id, dt.name, COALESCE(ds.event_time,''), COALESCE(ds.role_desc,''),
		        ds.slots_total, ds.slots_filled
		 FROM duty_slots ds JOIN duty_types dt ON dt.id = ds.duty_type_id
		 WHERE ds.game_id=? ORDER BY COALESCE(ds.event_time,'99:99'), ds.id`, id)
	defer rows.Close()

	type slot struct {
		ID          int    `json:"id"`
		DutyType    string `json:"duty_type_name"`
		EventTime   string `json:"event_time"`
		RoleDesc    string `json:"role_description"`
		SlotsTotal  int    `json:"slots_total"`
		SlotsFilled int    `json:"slots_filled"`
	}
	slots := []slot{}
	for rows.Next() {
		var s slot
		rows.Scan(&s.ID, &s.DutyType, &s.EventTime, &s.RoleDesc, &s.SlotsTotal, &s.SlotsFilled)
		slots = append(slots, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"game": g, "slots": slots})
}

// POST /api/admin/games
func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date       string `json:"date"`
		Time       string `json:"time"`
		Opponent   string `json:"opponent"`
		TeamIDs    []int  `json:"team_ids"`
		EventType  string `json:"event_type"`
		SeasonID   int    `json:"season_id"`
		TemplateID *int   `json:"template_id"`
		Slots      []struct {
			DutyTypeID int    `json:"duty_type_id"`
			EventTime  string `json:"event_time"`
			SlotsCount int    `json:"slots_count"`
			RoleDesc   string `json:"role_desc"`
		} `json:"slots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Date == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(req.TeamIDs) == 0 {
		http.Error(w, "team_ids required", http.StatusBadRequest)
		return
	}

	if req.EventType != "heim" && req.EventType != "auswärts" && req.EventType != "generisch" {
		http.Error(w, "invalid event_type", http.StatusBadRequest)
		return
	}

	if req.SeasonID == 0 {
		h.db.QueryRowContext(r.Context(),
			`SELECT id FROM seasons WHERE is_active=1 LIMIT 1`).Scan(&req.SeasonID)
	}

	claims := auth.ClaimsFromCtx(r.Context())
	if claims != nil && claims.Role == "trainer" {
		placeholders := strings.Repeat("?,", len(req.TeamIDs))
		placeholders = placeholders[:len(placeholders)-1]
		args := append([]any{claims.UserID}, toAny(req.TeamIDs)...)
		var count int
		err := h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(DISTINCT k.team_id) FROM kader k
			 JOIN kader_trainers kt ON kt.kader_id = k.id
			 JOIN members m ON m.id = kt.member_id
			 WHERE m.user_id = ? AND k.team_id IN (`+placeholders+`)
			   AND k.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)`,
			args...).Scan(&count)
		if err == nil && count != len(req.TeamIDs) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	isHome := req.EventType == "heim"

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(r.Context(),
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type) VALUES (?,?,?,?,?,?)`,
		req.SeasonID, req.Opponent, req.Date, req.Time, isHome, req.EventType)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	gameID, _ := res.LastInsertId()

	for _, teamID := range req.TeamIDs {
		tx.ExecContext(r.Context(),
			`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, gameID, teamID)
	}

	eventName := ""
	switch req.EventType {
	case "heim":
		eventName = "Heimspiel"
	case "auswärts":
		eventName = "Auswärtsspiel"
	case "generisch":
		eventName = req.Opponent
	}
	if req.EventType != "generisch" && req.Opponent != "" {
		eventName += " vs. " + req.Opponent
	}

	for _, s := range req.Slots {
		n := s.SlotsCount
		if n <= 0 {
			n = 1
		}
		for _, teamID := range req.TeamIDs {
			_, err = tx.ExecContext(r.Context(),
				`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc, slots_total, team_id, season_id, game_id)
				 VALUES (?,?,?,?,?,?,?,?,?)`,
				eventName, req.Date, s.EventTime, s.DutyTypeID, s.RoleDesc, n, teamID, req.SeasonID, gameID)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": gameID})
}

func toAny(teamIDs []int) []any {
	result := make([]any, len(teamIDs))
	for i, id := range teamIDs {
		result[i] = id
	}
	return result
}

// PUT /api/admin/games/{id}
func (h *Handler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Date     string `json:"date"`
		Time     string `json:"time"`
		Opponent string `json:"opponent"`
		TeamIDs  []int  `json:"team_ids"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(r.Context(),
		`UPDATE games SET date=?, time=?, opponent=? WHERE id=?`,
		req.Date, req.Time, req.Opponent, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if len(req.TeamIDs) > 0 {
		tx.ExecContext(r.Context(), `DELETE FROM game_teams WHERE game_id=?`, id)
		for _, teamID := range req.TeamIDs {
			tx.ExecContext(r.Context(),
				`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, id, teamID)
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/admin/games/{id}
func (h *Handler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM games WHERE id=?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/teams — filtered by user role
func (h *Handler) ListTeamsForUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	type team struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		AgeClass string `json:"age_class"`
		Gender   string `json:"gender"`
		IsActive bool   `json:"is_active"`
	}

	const activeSeasonSub = `(SELECT id FROM seasons WHERE is_active=1 LIMIT 1)`

	var rows *sql.Rows
	var err error
	if claims.Role == "trainer" {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT DISTINCT t.id, t.name, t.age_class, t.gender, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 JOIN kader_trainers kt ON kt.kader_id = k.id
			 JOIN members m ON m.id = kt.member_id
			 WHERE k.season_id = `+activeSeasonSub+` AND m.user_id = ?
			 ORDER BY t.name`, claims.UserID)
	} else {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT DISTINCT t.id, t.name, t.age_class, t.gender, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 WHERE k.season_id = `+activeSeasonSub+`
			 ORDER BY t.name`)
	}

	result := []team{}
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t team
			var active int
			rows.Scan(&t.ID, &t.Name, &t.AgeClass, &t.Gender, &active)
			t.IsActive = active == 1
			result = append(result, t)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ── Duty Templates ───────────────────────────────────────────────────────────

type templateItem struct {
	ID            int    `json:"id,omitempty"`
	DutyTypeID    int    `json:"duty_type_id"`
	DutyTypeName  string `json:"duty_type_name,omitempty"`
	Anchor        string `json:"anchor"`
	OffsetMinutes int    `json:"offset_minutes"`
	SlotsCount    int    `json:"slots_count"`
	RoleDesc      string `json:"role_desc"`
}

func (h *Handler) scanTemplateItems(ctx context.Context, templateID int) []templateItem {
	rows, _ := h.db.QueryContext(ctx,
		`SELECT gti.id, gti.duty_type_id, dt.name, gti.anchor, gti.offset_minutes, gti.slots_count, COALESCE(gti.role_desc,'')
		 FROM game_template_items gti JOIN duty_types dt ON dt.id = gti.duty_type_id
		 WHERE gti.template_id=? ORDER BY gti.sort_order, gti.id`, templateID)
	items := []templateItem{}
	if rows == nil {
		return items
	}
	defer rows.Close()
	for rows.Next() {
		var it templateItem
		rows.Scan(&it.ID, &it.DutyTypeID, &it.DutyTypeName, &it.Anchor, &it.OffsetMinutes, &it.SlotsCount, &it.RoleDesc)
		items = append(items, it)
	}
	return items
}

// GET /api/admin/duty-templates
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT gt.id, gt.name, gt.template_type, gt.game_duration_minutes, COUNT(gti.id)
		 FROM game_templates gt
		 LEFT JOIN game_template_items gti ON gti.template_id = gt.id
		 GROUP BY gt.id ORDER BY gt.id`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type listItem struct {
		ID                  int    `json:"id"`
		Name                string `json:"name"`
		TemplateType        string `json:"template_type"`
		GameDurationMinutes int    `json:"game_duration_minutes"`
		ItemCount           int    `json:"item_count"`
	}
	result := []listItem{}
	for rows.Next() {
		var t listItem
		rows.Scan(&t.ID, &t.Name, &t.TemplateType, &t.GameDurationMinutes, &t.ItemCount)
		result = append(result, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/admin/duty-templates/{id}
func (h *Handler) GetTemplateByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var t struct {
		ID                  int    `json:"id"`
		Name                string `json:"name"`
		TemplateType        string `json:"template_type"`
		GameDurationMinutes int    `json:"game_duration_minutes"`
	}
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, template_type, game_duration_minutes FROM game_templates WHERE id=?`, id).
		Scan(&t.ID, &t.Name, &t.TemplateType, &t.GameDurationMinutes)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	items := h.scanTemplateItems(r.Context(), t.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id": t.ID, "name": t.Name, "template_type": t.TemplateType,
		"game_duration_minutes": t.GameDurationMinutes, "items": items,
	})
}

// POST /api/admin/duty-templates
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                string `json:"name"`
		TemplateType        string `json:"template_type"`
		GameDurationMinutes int    `json:"game_duration_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.TemplateType != "heim" && req.TemplateType != "auswärts" && req.TemplateType != "generisch" {
		http.Error(w, "invalid template_type", http.StatusBadRequest)
		return
	}
	if req.GameDurationMinutes <= 0 {
		req.GameDurationMinutes = 90
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO game_templates (name, template_type, game_duration_minutes) VALUES (?,?,?)`,
		req.Name, req.TemplateType, req.GameDurationMinutes)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newID, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id": newID, "name": req.Name, "template_type": req.TemplateType,
		"game_duration_minutes": req.GameDurationMinutes, "items": []any{},
	})
}

// PUT /api/admin/duty-templates/{id}
func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name                string         `json:"name"`
		TemplateType        string         `json:"template_type"`
		GameDurationMinutes int            `json:"game_duration_minutes"`
		Items               []templateItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.TemplateType != "heim" && req.TemplateType != "auswärts" && req.TemplateType != "generisch" {
		http.Error(w, "invalid template_type", http.StatusBadRequest)
		return
	}
	if req.GameDurationMinutes <= 0 {
		req.GameDurationMinutes = 90
	}
	for _, it := range req.Items {
		var exists int
		if err := h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM duty_types WHERE id=?`, it.DutyTypeID).Scan(&exists); err != nil || exists == 0 {
			http.Error(w, "invalid duty_type_id", http.StatusBadRequest)
			return
		}
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(r.Context(),
		`UPDATE game_templates SET name=?, template_type=?, game_duration_minutes=? WHERE id=?`,
		req.Name, req.TemplateType, req.GameDurationMinutes, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	tx.ExecContext(r.Context(), `DELETE FROM game_template_items WHERE template_id=?`, id)
	for i, it := range req.Items {
		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, role_desc, sort_order)
			 VALUES (?,?,?,?,?,?,?)`,
			id, it.DutyTypeID, it.Anchor, it.OffsetMinutes, it.SlotsCount, it.RoleDesc, i)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/admin/duty-templates/{id}
func (h *Handler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM game_templates WHERE id=?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GET /api/admin/duty-templates/{id}/preview?time=HH:MM&game_id=N
func (h *Handler) PreviewSlots(w http.ResponseWriter, r *http.Request) {
	templateIDStr := r.PathValue("id")
	gameTime := r.URL.Query().Get("time")
	if gameTime == "" {
		http.Error(w, "time is required", http.StatusBadRequest)
		return
	}
	gameIDStr := r.URL.Query().Get("game_id")

	var templateID, durationMins int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, game_duration_minutes FROM game_templates WHERE id=?`, templateIDStr).
		Scan(&templateID, &durationMins)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var gameDate string
	var seasonID int
	var allGameTimes []string
	var hasPrevDay, hasNextDay bool
	if gameIDStr != "" {
		if h.db.QueryRowContext(r.Context(),
			`SELECT date, season_id FROM games WHERE id=?`, gameIDStr).
			Scan(&gameDate, &seasonID) == nil {
			allGameTimes, hasPrevDay, hasNextDay = h.loadSameDayContext(r.Context(), gameDate, seasonID)
		}
	}

	items, err := h.loadTemplateItems(r.Context(), templateID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type preview struct {
		DutyTypeID   int    `json:"duty_type_id"`
		DutyTypeName string `json:"duty_type_name"`
		EventTime    string `json:"event_time"`
		SlotsCount   int    `json:"slots_count"`
		RoleDesc     string `json:"role_desc"`
	}
	result := []preview{}
	for _, it := range items {
		offset := it.OffsetMinutes
		if it.Anchor == "end" {
			offset += durationMins
		}
		eventTime := addMinutes(gameTime, offset)

		dutyTypeID := it.DutyTypeID
		if gameIDStr != "" && len(allGameTimes) > 0 {
			isFirst, isLast := false, false
			for i, t := range allGameTimes {
				if t == gameTime {
					isFirst = i == 0
					isLast = i == len(allGameTimes)-1
					break
				}
			}
			dutyTypeID = applyBehavior(it, len(allGameTimes) > 1, hasPrevDay, hasNextDay, isFirst, isLast)
			if dutyTypeID == -1 {
				continue
			}
		}

		result = append(result, preview{
			DutyTypeID:   dutyTypeID,
			DutyTypeName: it.DutyTypeName,
			EventTime:    eventTime,
			SlotsCount:   it.SlotsCount,
			RoleDesc:     it.RoleDesc,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/admin/games/{id}/regenerate
func (h *Handler) RegenerateSlots(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")

	var req struct {
		Slots []struct {
			DutyTypeID int    `json:"duty_type_id"`
			EventTime  string `json:"event_time"`
			SlotsCount int    `json:"slots_count"`
			RoleDesc   string `json:"role_desc"`
		} `json:"slots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var game struct {
		SeasonID int
		Date     string
		Opponent string
		IsHome   bool
	}
	err := h.db.QueryRowContext(r.Context(),
		`SELECT season_id, date, opponent, is_home FROM games WHERE id=?`, gameID).
		Scan(&game.SeasonID, &game.Date, &game.Opponent, &game.IsHome)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	teamRows, err := h.db.QueryContext(r.Context(),
		`SELECT team_id FROM game_teams WHERE game_id=?`, gameID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer teamRows.Close()

	var teamIDs []int
	for teamRows.Next() {
		var tid int
		teamRows.Scan(&tid)
		teamIDs = append(teamIDs, tid)
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var keptSlots int
	tx.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM duty_slots WHERE game_id=? AND slots_filled > 0`, gameID).Scan(&keptSlots)
	tx.ExecContext(r.Context(),
		`DELETE FROM duty_slots WHERE game_id=? AND slots_filled = 0`, gameID)

	var gameTime string
	tx.QueryRowContext(r.Context(), `SELECT time FROM games WHERE id=?`, gameID).Scan(&gameTime)

	allGameTimes, hasPrevDay, hasNextDay := h.loadSameDayContext(r.Context(), game.Date, game.SeasonID)

	eventName := "Heimspiel"
	if !game.IsHome {
		eventName = "Auswärtsspiel"
	}
	if game.Opponent != "" {
		eventName += " vs. " + game.Opponent
	}

	for _, s := range req.Slots {
		n := s.SlotsCount
		if n <= 0 {
			n = 1
		}

		isFirst, isLast := false, false
		for i, t := range allGameTimes {
			if t == gameTime {
				isFirst = i == 0
				isLast = i == len(allGameTimes)-1
				break
			}
		}

		var it templateItemRow
		it.DutyTypeID = s.DutyTypeID
		h.db.QueryRowContext(r.Context(),
			`SELECT same_day_behavior, same_day_variant_id, adjacent_day_behavior, adjacent_day_variant_id FROM duty_types WHERE id=?`,
			s.DutyTypeID).Scan(&it.SameDayBehavior, &it.SameDayVariantID, &it.AdjacentDayBehavior, &it.AdjacentDayVariantID)

		dutyTypeID := applyBehavior(it, len(allGameTimes) > 1, hasPrevDay, hasNextDay, isFirst, isLast)
		if dutyTypeID == -1 {
			continue
		}

		for _, teamID := range teamIDs {
			_, err = tx.ExecContext(r.Context(),
				`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc, slots_total, team_id, season_id, game_id)
				 VALUES (?,?,?,?,?,?,?,?,?)`,
				eventName, game.Date, s.EventTime, dutyTypeID, s.RoleDesc, n, teamID, game.SeasonID, gameID)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"kept_slots": keptSlots})
}
