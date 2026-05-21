package games

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

// GET /api/games
func (h *Handler) ListGames(w http.ResponseWriter, r *http.Request) {
	seasonID := r.URL.Query().Get("season_id")

	const base = `
		SELECT g.id, g.date, g.time, g.opponent, g.team_id, t.name,
		       COUNT(ds.id), COALESCE(SUM(ds.slots_filled),0), COALESCE(SUM(ds.slots_total),0)
		FROM games g
		JOIN teams t ON t.id = g.team_id
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

	type game struct {
		ID          int    `json:"id"`
		Date        string `json:"date"`
		Time        string `json:"time"`
		Opponent    string `json:"opponent"`
		TeamID      int    `json:"team_id"`
		TeamName    string `json:"team_name"`
		SlotCount   int    `json:"slot_count"`
		FilledCount int    `json:"filled_count"`
		TotalCount  int    `json:"total_count"`
	}
	result := []game{}
	for rows.Next() {
		var g game
		rows.Scan(&g.ID, &g.Date, &g.Time, &g.Opponent, &g.TeamID, &g.TeamName,
			&g.SlotCount, &g.FilledCount, &g.TotalCount)
		result = append(result, g)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/games/{id}
func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var g struct {
		ID       int    `json:"id"`
		Date     string `json:"date"`
		Time     string `json:"time"`
		Opponent string `json:"opponent"`
		TeamID   int    `json:"team_id"`
		TeamName string `json:"team_name"`
		SeasonID int    `json:"season_id"`
	}
	err := h.db.QueryRowContext(r.Context(),
		`SELECT g.id, g.date, g.time, g.opponent, g.team_id, t.name, g.season_id
		 FROM games g JOIN teams t ON t.id = g.team_id WHERE g.id=?`, id).
		Scan(&g.ID, &g.Date, &g.Time, &g.Opponent, &g.TeamID, &g.TeamName, &g.SeasonID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
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
		Date     string `json:"date"`
		Time     string `json:"time"`
		Opponent string `json:"opponent"`
		TeamID   int    `json:"team_id"`
		SeasonID int    `json:"season_id"`
		Slots    []struct {
			DutyTypeID int    `json:"duty_type_id"`
			EventTime  string `json:"event_time"`
			SlotsCount int    `json:"slots_count"`
			RoleDesc   string `json:"role_desc"`
		} `json:"slots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Date == "" || req.TeamID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.SeasonID == 0 {
		h.db.QueryRowContext(r.Context(),
			`SELECT id FROM seasons WHERE is_active=1 LIMIT 1`).Scan(&req.SeasonID)
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(r.Context(),
		`INSERT INTO games (team_id, season_id, opponent, date, time) VALUES (?,?,?,?,?)`,
		req.TeamID, req.SeasonID, req.Opponent, req.Date, req.Time)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	gameID, _ := res.LastInsertId()

	eventName := "Heimspiel"
	if req.Opponent != "" {
		eventName = "Heimspiel vs. " + req.Opponent
	}
	for _, s := range req.Slots {
		n := s.SlotsCount
		if n <= 0 {
			n = 1
		}
		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc, slots_total, team_id, season_id, game_id)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			eventName, req.Date, s.EventTime, s.DutyTypeID, s.RoleDesc, n, req.TeamID, req.SeasonID, gameID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
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

// PUT /api/admin/games/{id}
func (h *Handler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Date     string `json:"date"`
		Time     string `json:"time"`
		Opponent string `json:"opponent"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	res, err := h.db.ExecContext(r.Context(),
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

// GET /api/admin/game-template
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	var templateID, durationMins int
	var name string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, game_duration_minutes FROM game_templates WHERE is_active=1 LIMIT 1`).
		Scan(&templateID, &name, &durationMins)

	type item struct {
		ID            int    `json:"id"`
		DutyTypeID    int    `json:"duty_type_id"`
		DutyTypeName  string `json:"duty_type_name"`
		Anchor        string `json:"anchor"`
		OffsetMinutes int    `json:"offset_minutes"`
		SlotsCount    int    `json:"slots_count"`
		RoleDesc      string `json:"role_desc"`
	}
	type resp struct {
		ID                  *int   `json:"id"`
		Name                string `json:"name"`
		GameDurationMinutes int    `json:"game_duration_minutes"`
		Items               []item `json:"items"`
	}

	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp{Name: "Heimspiel Standard", GameDurationMinutes: 90, Items: []item{}})
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT gti.id, gti.duty_type_id, dt.name, gti.anchor, gti.offset_minutes, gti.slots_count, COALESCE(gti.role_desc,'')
		 FROM game_template_items gti JOIN duty_types dt ON dt.id = gti.duty_type_id
		 WHERE gti.template_id=? ORDER BY gti.sort_order, gti.id`, templateID)
	defer rows.Close()

	items := []item{}
	for rows.Next() {
		var it item
		rows.Scan(&it.ID, &it.DutyTypeID, &it.DutyTypeName, &it.Anchor,
			&it.OffsetMinutes, &it.SlotsCount, &it.RoleDesc)
		items = append(items, it)
	}
	idPtr := &templateID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp{ID: idPtr, Name: name, GameDurationMinutes: durationMins, Items: items})
}

// PUT /api/admin/game-template
func (h *Handler) SetTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                string `json:"name"`
		GameDurationMinutes int    `json:"game_duration_minutes"`
		Items               []struct {
			DutyTypeID    int    `json:"duty_type_id"`
			Anchor        string `json:"anchor"`
			OffsetMinutes int    `json:"offset_minutes"`
			SlotsCount    int    `json:"slots_count"`
			RoleDesc      string `json:"role_desc"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "Heimspiel Standard"
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

	var templateID int
	err = tx.QueryRowContext(r.Context(),
		`SELECT id FROM game_templates WHERE is_active=1 LIMIT 1`).Scan(&templateID)
	if err == sql.ErrNoRows {
		res, err2 := tx.ExecContext(r.Context(),
			`INSERT INTO game_templates (name, game_duration_minutes) VALUES (?,?)`,
			req.Name, req.GameDurationMinutes)
		if err2 != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		id64, _ := res.LastInsertId()
		templateID = int(id64)
	} else if err == nil {
		tx.ExecContext(r.Context(),
			`UPDATE game_templates SET name=?, game_duration_minutes=? WHERE id=?`,
			req.Name, req.GameDurationMinutes, templateID)
	} else {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tx.ExecContext(r.Context(), `DELETE FROM game_template_items WHERE template_id=?`, templateID)

	for i, it := range req.Items {
		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, role_desc, sort_order)
			 VALUES (?,?,?,?,?,?,?)`,
			templateID, it.DutyTypeID, it.Anchor, it.OffsetMinutes, it.SlotsCount, it.RoleDesc, i)
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

// GET /api/admin/game-template/preview
func (h *Handler) PreviewSlots(w http.ResponseWriter, r *http.Request) {
	gameTime := r.URL.Query().Get("time")
	if gameTime == "" {
		http.Error(w, "time is required", http.StatusBadRequest)
		return
	}
	gameIDStr := r.URL.Query().Get("game_id")

	var templateID, durationMins int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, game_duration_minutes FROM game_templates WHERE is_active=1 LIMIT 1`).
		Scan(&templateID, &durationMins)
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// If game_id provided, load game context (date, season) for applies_when calculation
	var gameDate string
	var seasonID int
	var allGameTimes []string
	var hasPrevDay, hasNextDay bool
	if gameIDStr != "" {
		err := h.db.QueryRowContext(r.Context(),
			`SELECT date, season_id FROM games WHERE id=?`, gameIDStr).
			Scan(&gameDate, &seasonID)
		if err == nil {
			// Get all games on the same day
			gameRows, _ := h.db.QueryContext(r.Context(),
				`SELECT time FROM games WHERE date=? AND is_home=1 AND season_id=? ORDER BY time`,
				gameDate, seasonID)
			if gameRows != nil {
				defer gameRows.Close()
				for gameRows.Next() {
					var t string
					gameRows.Scan(&t)
					allGameTimes = append(allGameTimes, t)
				}
			}
			// Check adjacent days
			var prevCount, nextCount int
			h.db.QueryRowContext(r.Context(),
				`SELECT COUNT(*) FROM games WHERE date=date(?, '-1 days') AND is_home=1 AND season_id=?`,
				gameDate, seasonID).Scan(&prevCount)
			h.db.QueryRowContext(r.Context(),
				`SELECT COUNT(*) FROM games WHERE date=date(?, '+1 days') AND is_home=1 AND season_id=?`,
				gameDate, seasonID).Scan(&nextCount)
			hasPrevDay = prevCount > 0
			hasNextDay = nextCount > 0
		}
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT gti.duty_type_id, dt.name, gti.anchor, gti.offset_minutes, gti.slots_count, COALESCE(gti.role_desc,''), dt.consecutive_behavior, dt.consecutive_variant_id
		 FROM game_template_items gti JOIN duty_types dt ON dt.id = gti.duty_type_id
		 WHERE gti.template_id=? ORDER BY gti.sort_order, gti.id`, templateID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type preview struct {
		DutyTypeID   int    `json:"duty_type_id"`
		DutyTypeName string `json:"duty_type_name"`
		EventTime    string `json:"event_time"`
		SlotsCount   int    `json:"slots_count"`
		RoleDesc     string `json:"role_desc"`
	}
	result := []preview{}
	for rows.Next() {
		var p preview
		var anchor string
		var offset int
		var consB string
		var consV sql.NullInt64
		rows.Scan(&p.DutyTypeID, &p.DutyTypeName, &anchor, &offset, &p.SlotsCount, &p.RoleDesc, &consB, &consV)

		if anchor == "end" {
			offset += durationMins
		}
		p.EventTime = addMinutes(gameTime, offset)

		// If game_id context provided, check applies_when
		if gameIDStr != "" && len(allGameTimes) > 0 {
			// Determine if this is first/last game
			isFirst := gameTime == allGameTimes[0]
			isLast := gameTime == allGameTimes[len(allGameTimes)-1]

			// Calculate applies_when
			appliesWhen := "always"
			if isFirst && !hasPrevDay {
				appliesWhen = "day_open"
			} else if isLast && !hasNextDay {
				appliesWhen = "day_close"
			}

			// Apply consecutive_behavior only for day_open/day_close
			if appliesWhen != "always" && consB != "normal" {
				shouldApplyConsecutive := false
				if appliesWhen == "day_open" && hasPrevDay {
					shouldApplyConsecutive = true
				} else if appliesWhen == "day_close" && hasNextDay {
					shouldApplyConsecutive = true
				}

				if shouldApplyConsecutive {
					if consB == "skip" {
						continue
					} else if consB == "reduced" && consV.Valid {
						p.DutyTypeID = int(consV.Int64)
					}
				}
			}
		}

		result = append(result, p)
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
		TeamID   int
		SeasonID int
		Date     string
		Opponent string
	}
	err := h.db.QueryRowContext(r.Context(),
		`SELECT team_id, season_id, date, opponent FROM games WHERE id=?`, gameID).
		Scan(&game.TeamID, &game.SeasonID, &game.Date, &game.Opponent)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
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

	// Load game's time and all games on same day for applies_when calculation
	var gameTime string
	tx.QueryRowContext(r.Context(),
		`SELECT time FROM games WHERE id=?`, gameID).Scan(&gameTime)

	allGameTimes := []string{}
	gtRows, _ := tx.QueryContext(r.Context(),
		`SELECT time FROM games WHERE date=? AND is_home=1 AND season_id=? ORDER BY time`,
		game.Date, game.SeasonID)
	if gtRows != nil {
		defer gtRows.Close()
		for gtRows.Next() {
			var t string
			gtRows.Scan(&t)
			allGameTimes = append(allGameTimes, t)
		}
	}

	// Check adjacent days
	var prevCount, nextCount int
	tx.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM games WHERE date=date(?, '-1 days') AND is_home=1 AND season_id=?`,
		game.Date, game.SeasonID).Scan(&prevCount)
	tx.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM games WHERE date=date(?, '+1 days') AND is_home=1 AND season_id=?`,
		game.Date, game.SeasonID).Scan(&nextCount)
	hasPrevDay := prevCount > 0
	hasNextDay := nextCount > 0

	eventName := "Heimspiel"
	if game.Opponent != "" {
		eventName = "Heimspiel vs. " + game.Opponent
	}
	for _, s := range req.Slots {
		n := s.SlotsCount
		if n <= 0 {
			n = 1
		}

		// Determine if this is first/last game and apply applies_when logic
		isFirst := false
		isLast := false
		for i, t := range allGameTimes {
			if t == gameTime {
				isFirst = (i == 0)
				isLast = (i == len(allGameTimes)-1)
				break
			}
		}

		// Load duty type info for consecutive_behavior
		var consB string
		var consV sql.NullInt64
		tx.QueryRowContext(r.Context(),
			`SELECT consecutive_behavior, consecutive_variant_id FROM duty_types WHERE id=?`,
			s.DutyTypeID).Scan(&consB, &consV)

		// Calculate applies_when
		appliesWhen := "always"
		if isFirst && !hasPrevDay {
			appliesWhen = "day_open"
		} else if isLast && !hasNextDay {
			appliesWhen = "day_close"
		}

		// Apply consecutive_behavior logic
		dutyTypeID := s.DutyTypeID
		skip := false
		if appliesWhen != "always" && consB != "normal" {
			shouldApplyConsecutive := false
			if appliesWhen == "day_open" && hasPrevDay {
				shouldApplyConsecutive = true
			} else if appliesWhen == "day_close" && hasNextDay {
				shouldApplyConsecutive = true
			}

			if shouldApplyConsecutive {
				if consB == "skip" {
					skip = true
				} else if consB == "reduced" && consV.Valid {
					dutyTypeID = int(consV.Int64)
				}
			}
		}

		if skip {
			continue
		}

		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc, slots_total, team_id, season_id, game_id)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			eventName, game.Date, s.EventTime, dutyTypeID, s.RoleDesc, n, game.TeamID, game.SeasonID, gameID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"kept_slots": keptSlots})
}
