package games

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/policy"
)

type Handler struct {
	db  *sql.DB
	cfg *appconfig.Config
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, cfg *appconfig.Config, h *hub.EventHub) *Handler {
	return &Handler{db: db, cfg: cfg, hub: h}
}

// teamMembersAndParents returns user IDs of all active kader members (and their parents)
// for the given team IDs in the current active season.
func (h *Handler) teamMembersAndParents(teamIDs []int) []int {
	if len(teamIDs) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(teamIDs))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(teamIDs))
	for i, id := range teamIDs {
		args[i] = id
	}
	rows, err := h.db.Query(
		`SELECT DISTINCT u.id FROM users u
		 JOIN members m ON m.user_id = u.id
		 JOIN player_memberships pm ON pm.member_id = m.id
		 JOIN seasons s ON s.id = pm.season_id AND s.is_active = 1
		 WHERE pm.team_id IN (`+placeholders+`)
		 UNION
		 SELECT DISTINCT fl.parent_user_id FROM family_links fl
		 JOIN members m ON m.id = fl.member_id
		 JOIN player_memberships pm ON pm.member_id = m.id
		 JOIN seasons s ON s.id = pm.season_id AND s.is_active = 1
		 WHERE pm.team_id IN (`+placeholders+`)`,
		append(args, args...)...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

// gameTeamIDs returns the team IDs for a given game.
func (h *Handler) gameTeamIDs(gameID any) []int {
	rows, err := h.db.Query(`SELECT team_id FROM game_teams WHERE game_id=?`, gameID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

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

// compareTime compares two time strings in HH:MM format (returns -1 if t1 < t2, 0 if equal, 1 if t1 > t2)
func compareTime(t1, t2 string) int {
	if len(t1) < 5 || len(t2) < 5 {
		return 0
	}
	h1, m1 := timeComponents(t1)
	h2, m2 := timeComponents(t2)
	total1 := h1*60 + m1
	total2 := h2*60 + m2
	if total1 < total2 {
		return -1
	} else if total1 > total2 {
		return 1
	}
	return 0
}

func timeComponents(t string) (h, m int) {
	h, _ = strconv.Atoi(t[:2])
	m, _ = strconv.Atoi(t[3:])
	return
}

// classifySlotPosition determines if a slot is before, between, or after games on the same day.
// Classification is based on the game's position in the day (first/last/middle) and whether
// the slot falls before or after the game's kick-off time.
// allGameTimes must be sorted ascending.
func classifySlotPosition(slotTime string, gameTime string, allGameTimes []string) (
	isBeforeAllGames, isAfterAllGames, isBetweenGames bool) {

	if len(allGameTimes) == 0 {
		return false, false, false
	}

	isFirstGame := compareTime(gameTime, allGameTimes[0]) == 0
	isLastGame := compareTime(gameTime, allGameTimes[len(allGameTimes)-1]) == 0

	slotBeforeGame := compareTime(slotTime, gameTime) < 0
	slotAfterGame := compareTime(slotTime, gameTime) > 0

	switch {
	case slotBeforeGame && isFirstGame:
		isBeforeAllGames = true
	case slotBeforeGame && !isFirstGame:
		isBetweenGames = true
	case slotAfterGame && isLastGame:
		isAfterAllGames = true
	case slotAfterGame && !isLastGame:
		isBetweenGames = true
	}

	return isBeforeAllGames, isAfterAllGames, isBetweenGames
}

type templateItemRow struct {
	DutyTypeID           int
	DutyTypeName         string
	Anchor               string
	OffsetMinutes        int
	SlotsCount           int
	SameDayBehavior      string
	SameDayVariantID     sql.NullInt64
	AdjacentDayBehavior  string
	AdjacentDayVariantID sql.NullInt64
	Audiences            sql.NullString
}

func (h *Handler) loadTemplateItems(ctx context.Context, templateID int) ([]templateItemRow, error) {
	rows, err := h.db.QueryContext(ctx,
		`SELECT gti.duty_type_id, dt.name, gti.anchor, gti.offset_minutes, gti.slots_count,
		        dt.same_day_behavior, dt.same_day_variant_id, dt.adjacent_day_behavior, dt.adjacent_day_variant_id,
		        gti.audiences
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
			&it.SlotsCount, &it.SameDayBehavior, &it.SameDayVariantID,
			&it.AdjacentDayBehavior, &it.AdjacentDayVariantID, &it.Audiences)
		result = append(result, it)
	}
	return result, nil
}

// applyBehavior returns the effective dutyTypeID after applying same-day/adjacent-day rules,
// or -1 if the slot should be skipped.
// slotTime: Uhrzeit des Dienstes (berechnet aus game time + offset)
// gameTime: Uhrzeit des aktuellen Spiels
// allGameTimes: Alle Spielzeiten am gleichen Tag (sortiert)
// isBeforeAllGames: Liegt der Service vor allen Spielen des Tages?
// isAfterAllGames: Liegt der Service nach allen Spielen des Tages?
// isBetweenGames: Liegt der Service zwischen zwei Spielen am gleichen Tag?
func applyBehavior(it templateItemRow, gameTime, slotTime string, allGameTimes []string,
	hasPrevDay, hasNextDay, isBeforeAllGames, isAfterAllGames, isBetweenGames bool) int {
	dutyTypeID := it.DutyTypeID
	skip := false

	// Dienste zwischen zwei Spielen am gleichen Tag: same_day_behavior
	if isBetweenGames && it.SameDayBehavior != "normal" {
		if it.SameDayBehavior == "skip" {
			skip = true
		} else if it.SameDayBehavior == "reduced" && it.SameDayVariantID.Valid {
			dutyTypeID = int(it.SameDayVariantID.Int64)
		}
	}

	// Dienste am Anfang (vor allen Spielen) oder am Ende (nach allen Spielen): adjacent_day_behavior
	shouldApplyAdjacent := (isBeforeAllGames && hasPrevDay) || (isAfterAllGames && hasNextDay)
	if shouldApplyAdjacent && it.AdjacentDayBehavior != "normal" {
		if it.AdjacentDayBehavior == "skip" {
			skip = true
		} else if it.AdjacentDayBehavior == "reduced" && it.AdjacentDayVariantID.Valid {
			// Nicht doppelt reduzieren, wenn schon same_day_behavior reduziert wurde
			if !isBetweenGames || it.SameDayBehavior != "reduced" || !it.SameDayVariantID.Valid {
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
	// Load all games (home and away) on the same date
	gtRows, _ := h.db.QueryContext(ctx,
		`SELECT time FROM games WHERE date=? AND season_id=? ORDER BY time`,
		gameDate, seasonID)
	if gtRows != nil {
		defer gtRows.Close()
		for gtRows.Next() {
			var t string
			gtRows.Scan(&t)
			allGameTimes = append(allGameTimes, t)
		}
	}
	// Remove duplicates and sort
	uniqueTimes := make([]string, 0, len(allGameTimes))
	seen := make(map[string]bool)
	for _, t := range allGameTimes {
		if !seen[t] {
			seen[t] = true
			uniqueTimes = append(uniqueTimes, t)
		}
	}
	allGameTimes = uniqueTimes

	// Check if there are home games on previous/next days (for adjacent_day_behavior)
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
	claims := auth.ClaimsFromCtx(r.Context())

	// Event-Sichtbarkeitsregel (Funktionsträger sehen alles, sonst nur Team-
	// Zugehörigkeit). Ersetzt das alte policy.ScopeGamesQuery, das Trainer auf
	// kader_trainers einschränkte und erweiterte Kader-Member ignorierte.
	visClause, visArgs, _, err := auth.GameVisibilityClause(r.Context(), h.db, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	andScope := ""
	scopeArgs := []any{}
	if visClause != "1=1" {
		andScope = " AND " + visClause
		scopeArgs = visArgs
	}

	// confirmed_count berücksichtigt rsvp_opt_out: reguläre Kader-Mitglieder ohne
	// Response-Eintrag werden als "confirmed" gezählt. declined und maybe bleiben
	// rein explizit — Opt-Out kennt keine implizite Absage.
	const base = `
		SELECT g.id, g.date, g.time, g.end_time, g.end_date, g.opponent, g.event_type, g.template_id,
		       COUNT(DISTINCT ds.id), COALESCE(SUM(ds.slots_filled),0), COALESCE(SUM(ds.slots_total),0),
		       CASE WHEN g.rsvp_opt_out = 1
		            THEN COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='confirmed'),0) + (
		                   SELECT COUNT(DISTINCT km.member_id) FROM game_teams gt4
		                   JOIN kader k4 ON k4.team_id = gt4.team_id AND k4.season_id = g.season_id
		                   JOIN kader_members km ON km.kader_id = k4.id
		                   WHERE gt4.game_id = g.id
		                   AND NOT EXISTS (SELECT 1 FROM game_responses gr2 WHERE gr2.game_id = g.id AND gr2.member_id = km.member_id)
		                 )
		            ELSE COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='confirmed'),0)
		       END,
		       COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='declined'),0),
		       COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='maybe'),0),
		       g.rsvp_opt_out, g.rsvp_require_reason,
		       v.id, v.name, v.street, v.city, v.postal_code, v.note
		FROM games g
		LEFT JOIN duty_slots ds ON ds.game_id = g.id
		LEFT JOIN venues v ON v.id = g.venue_id`
	const suffix = ` GROUP BY g.id ORDER BY g.date, g.time`

	var rows *sql.Rows
	if seasonID != "" {
		args := append([]any{seasonID}, scopeArgs...)
		rows, err = h.db.QueryContext(r.Context(), base+` WHERE g.season_id=?`+andScope+suffix, args...)
	} else {
		// Show active-season games plus any future games from other seasons
		// (prevents games from stranding when seasons are switched).
		rows, err = h.db.QueryContext(r.Context(),
			base+` WHERE (g.season_id=(SELECT id FROM seasons WHERE is_active=1 LIMIT 1) OR DATE(g.date) >= DATE('now','-1 day'))`+andScope+suffix,
			scopeArgs...)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type team struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		DisplayShort string `json:"display_short"`
		DisplayLong  string `json:"display_long"`
	}
	type venueRef struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Street     string `json:"street"`
		City       string `json:"city"`
		PostalCode string `json:"postal_code"`
		Note       string `json:"note"`
	}
	type game struct {
		ID                  int                 `json:"id"`
		Date                string              `json:"date"`
		Time                string              `json:"time"`
		EndTime             *string             `json:"end_time,omitempty"`
		EndDate             *string             `json:"end_date"`
		Opponent            string              `json:"opponent"`
		EventType           string              `json:"event_type"`
		TemplateID          *int                `json:"template_id"`
		Teams               []team              `json:"teams"`
		TeamDisplayShortCSV string              `json:"team_display_short_csv"`
		TeamDisplayLongCSV  string              `json:"team_display_long_csv"`
		SlotCount           int                 `json:"slot_count"`
		FilledCount         int                 `json:"filled_count"`
		TotalCount          int                 `json:"total_count"`
		ConfirmedCount      int                 `json:"confirmed_count"`
		DeclinedCount       int                 `json:"declined_count"`
		MaybeCount          int                 `json:"maybe_count"`
		RsvpOptOut          int                 `json:"rsvp_opt_out"`
		RsvpRequireReason   int                 `json:"rsvp_require_reason"`
		Venue               *venueRef           `json:"venue,omitempty"`
		Can                 policy.GameCanFlags `json:"can"`
	}

	var games []*game
	for rows.Next() {
		var g game
		var endTimeNull, endDateNull sql.NullString
		var templateIDNull sql.NullInt64
		var vID sql.NullInt64
		var vName, vStreet, vCity, vPostal, vNote sql.NullString
		if err := rows.Scan(&g.ID, &g.Date, &g.Time, &endTimeNull, &endDateNull, &g.Opponent, &g.EventType, &templateIDNull,
			&g.SlotCount, &g.FilledCount, &g.TotalCount,
			&g.ConfirmedCount, &g.DeclinedCount, &g.MaybeCount,
			&g.RsvpOptOut, &g.RsvpRequireReason,
			&vID, &vName, &vStreet, &vCity, &vPostal, &vNote); err != nil {
			continue
		}
		if templateIDNull.Valid {
			v := int(templateIDNull.Int64)
			g.TemplateID = &v
		}
		if endTimeNull.Valid {
			g.EndTime = &endTimeNull.String
		}
		if endDateNull.Valid {
			g.EndDate = &endDateNull.String
		}
		if vID.Valid {
			g.Venue = &venueRef{
				ID: int(vID.Int64), Name: vName.String, Street: vStreet.String,
				City: vCity.String, PostalCode: vPostal.String, Note: vNote.String,
			}
		}
		g.Teams = []team{}
		games = append(games, &g)
	}

	for _, g := range games {
		teamRows, _ := h.db.QueryContext(r.Context(),
			`SELECT t.id, t.name,
			        COALESCE(`+appdb.TeamDisplayShort("t")+`, t.name) AS display_short,
			        COALESCE(`+appdb.TeamDisplayName("t")+`, t.name) AS display_long
			 FROM teams t
			 JOIN game_teams gt ON gt.team_id = t.id
			 WHERE gt.game_id = ?
			 ORDER BY display_short`, g.ID)
		if teamRows != nil {
			for teamRows.Next() {
				var t team
				teamRows.Scan(&t.ID, &t.Name, &t.DisplayShort, &t.DisplayLong)
				g.Teams = append(g.Teams, t)
			}
			teamRows.Close()
		}
		shorts := make([]string, len(g.Teams))
		longs := make([]string, len(g.Teams))
		for i, t := range g.Teams {
			shorts[i] = t.DisplayShort
			longs[i] = t.DisplayLong
		}
		g.TeamDisplayShortCSV = strings.Join(shorts, ", ")
		g.TeamDisplayLongCSV = strings.Join(longs, ", ")
	}

	gameCan := policy.GameCan(&policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions, IsParent: claims.IsParent})
	result := make([]game, len(games))
	for i, g := range games {
		g.Can = gameCan
		result[i] = *g
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/games/{id}
func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if gid, err := strconv.Atoi(id); err == nil {
		claims := auth.ClaimsFromCtx(r.Context())
		ok, _ := auth.UserCanSeeGame(r.Context(), h.db, claims.UserID, gid)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}

	type venueRef struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Street     string `json:"street"`
		City       string `json:"city"`
		PostalCode string `json:"postal_code"`
		Note       string `json:"note"`
	}
	var g struct {
		ID                int       `json:"id"`
		Date              string    `json:"date"`
		Time              string    `json:"time"`
		EndTime           *string   `json:"end_time,omitempty"`
		EndDate           *string   `json:"end_date"`
		Opponent          string    `json:"opponent"`
		EventType         string    `json:"event_type"`
		IsHome            bool      `json:"is_home"`
		SeasonID          int       `json:"season_id"`
		TemplateID        *int      `json:"template_id"`
		RsvpOptOut        int       `json:"rsvp_opt_out"`
		RsvpRequireReason int       `json:"rsvp_require_reason"`
		ConfirmedCount    int       `json:"confirmed_count"`
		DeclinedCount     int       `json:"declined_count"`
		MaybeCount        int       `json:"maybe_count"`
		Venue             *venueRef `json:"venue,omitempty"`
		Teams             []struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			DisplayShort string `json:"display_short"`
			DisplayLong  string `json:"display_long"`
		} `json:"teams"`
		TeamDisplayShortCSV string              `json:"team_display_short_csv"`
		TeamDisplayLongCSV  string              `json:"team_display_long_csv"`
		Can                 policy.GameCanFlags `json:"can"`
	}
	var templateIDNull sql.NullInt64
	var endTimeNull, endDateNull sql.NullString
	var vID sql.NullInt64
	var vName, vStreet, vCity, vPostal, vNote sql.NullString
	err := h.db.QueryRowContext(r.Context(),
		`SELECT g.id, g.date, g.time, g.end_time, g.end_date, g.opponent, g.event_type, g.is_home, g.season_id, g.template_id,
		        g.rsvp_opt_out, g.rsvp_require_reason,
		        CASE WHEN g.rsvp_opt_out = 1
		             THEN COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='confirmed'),0) + (
		                    SELECT COUNT(DISTINCT km.member_id) FROM game_teams gt4
		                    JOIN kader k4 ON k4.team_id = gt4.team_id AND k4.season_id = g.season_id
		                    JOIN kader_members km ON km.kader_id = k4.id
		                    WHERE gt4.game_id = g.id
		                    AND NOT EXISTS (SELECT 1 FROM game_responses gr2 WHERE gr2.game_id = g.id AND gr2.member_id = km.member_id)
		                  )
		             ELSE COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='confirmed'),0)
		        END,
		        COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='declined'),0),
		        COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='maybe'),0),
		        v.id, v.name, v.street, v.city, v.postal_code, v.note
		 FROM games g LEFT JOIN venues v ON v.id = g.venue_id WHERE g.id=?`, id).
		Scan(&g.ID, &g.Date, &g.Time, &endTimeNull, &endDateNull, &g.Opponent, &g.EventType, &g.IsHome, &g.SeasonID, &templateIDNull,
			&g.RsvpOptOut, &g.RsvpRequireReason,
			&g.ConfirmedCount, &g.DeclinedCount, &g.MaybeCount,
			&vID, &vName, &vStreet, &vCity, &vPostal, &vNote)
	if templateIDNull.Valid {
		v := int(templateIDNull.Int64)
		g.TemplateID = &v
	}
	if endTimeNull.Valid {
		g.EndTime = &endTimeNull.String
	}
	if endDateNull.Valid {
		g.EndDate = &endDateNull.String
	}
	if vID.Valid {
		g.Venue = &venueRef{
			ID: int(vID.Int64), Name: vName.String, Street: vStreet.String,
			City: vCity.String, PostalCode: vPostal.String, Note: vNote.String,
		}
	}
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	teamRows, _ := h.db.QueryContext(r.Context(),
		`SELECT t.id, t.name,
		        COALESCE(`+appdb.TeamDisplayShort("t")+`, t.name) AS display_short,
		        COALESCE(`+appdb.TeamDisplayName("t")+`, t.name) AS display_long
		 FROM teams t
		 JOIN game_teams gt ON gt.team_id = t.id
		 WHERE gt.game_id = ?
		 ORDER BY display_short`, id)
	if teamRows != nil {
		for teamRows.Next() {
			var t struct {
				ID           int    `json:"id"`
				Name         string `json:"name"`
				DisplayShort string `json:"display_short"`
				DisplayLong  string `json:"display_long"`
			}
			teamRows.Scan(&t.ID, &t.Name, &t.DisplayShort, &t.DisplayLong)
			g.Teams = append(g.Teams, t)
		}
		teamRows.Close()
	}
	shorts := make([]string, len(g.Teams))
	longs := make([]string, len(g.Teams))
	for i, t := range g.Teams {
		shorts[i] = t.DisplayShort
		longs[i] = t.DisplayLong
	}
	g.TeamDisplayShortCSV = strings.Join(shorts, ", ")
	g.TeamDisplayLongCSV = strings.Join(longs, ", ")

	claims := auth.ClaimsFromCtx(r.Context())
	gp := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions}
	canEdit := false
	if policy.CanViewAllGames(gp) {
		canEdit = true
	} else if policy.IsTrainerLike(gp) {
		var trains int
		h.db.QueryRowContext(r.Context(), `
			SELECT COUNT(*) FROM trainer_memberships trm
			JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
			JOIN members m ON m.id = trm.member_id AND m.user_id = ?
			JOIN game_teams gt ON gt.team_id = trm.team_id AND gt.game_id = ?`,
			claims.UserID, id).Scan(&trains)
		canEdit = trains > 0
	}
	g.Can = policy.GameCanFlags{Edit: canEdit, Delete: canEdit, ManageLineup: canEdit}

	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT ds.id, dt.name, COALESCE(ds.event_time,''), COALESCE(ds.role_desc,''),
		        ds.slots_total, ds.slots_filled, COALESCE(ds.audiences, dt.audiences)
		 FROM duty_slots ds JOIN duty_types dt ON dt.id = ds.duty_type_id
		 WHERE ds.game_id=? ORDER BY COALESCE(ds.event_time,'99:99'), ds.id`, id)
	defer rows.Close()

	type slot struct {
		ID          int      `json:"id"`
		DutyType    string   `json:"duty_type_name"`
		EventTime   string   `json:"event_time"`
		RoleDesc    string   `json:"role_description"`
		SlotsTotal  int      `json:"slots_total"`
		SlotsFilled int      `json:"slots_filled"`
		Audiences   []string `json:"audiences,omitempty"`
	}
	slots := []slot{}
	for rows.Next() {
		var s slot
		var audiences sql.NullString
		rows.Scan(&s.ID, &s.DutyType, &s.EventTime, &s.RoleDesc, &s.SlotsTotal, &s.SlotsFilled, &audiences)
		s.Audiences = audiencesFromDB(audiences)
		slots = append(slots, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"game": g, "slots": slots})
}

// POST /api/admin/games
func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date              string  `json:"date"`
		Time              string  `json:"time"`
		EndTime           *string `json:"end_time"`
		Opponent          string  `json:"opponent"`
		TeamIDs           []int   `json:"team_ids"`
		EventType         string  `json:"event_type"`
		SeasonID          int     `json:"season_id"`
		TemplateID        *int    `json:"template_id"`
		VenueID           *int    `json:"venue_id"`
		RsvpOptOut        int     `json:"rsvp_opt_out"`
		RsvpRequireReason *int    `json:"rsvp_require_reason"`
		EndDate           *string `json:"end_date"`
		Slots             []struct {
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
	if claims != nil && claims.HasFunction("trainer") && !claims.HasFunction("sportliche_leitung") {
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

	rsvpRequireReason := 1
	if req.RsvpRequireReason != nil {
		rsvpRequireReason = *req.RsvpRequireReason
	} else if req.EventType == "generisch" {
		rsvpRequireReason = 0
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var templateIDVal interface{}
	if req.TemplateID != nil {
		templateIDVal = *req.TemplateID
	}
	var endTimeVal interface{}
	if req.EndTime != nil && *req.EndTime != "" {
		endTimeVal = *req.EndTime
	}
	var venueIDVal interface{}
	if req.VenueID != nil {
		venueIDVal = *req.VenueID
	}
	var endDateVal interface{}
	if req.EndDate != nil && *req.EndDate != "" {
		if *req.EndDate < req.Date {
			http.Error(w, "end_date must be >= date", http.StatusBadRequest)
			return
		}
		endDateVal = *req.EndDate
	}
	res, err := tx.ExecContext(r.Context(),
		`INSERT INTO games (season_id, opponent, date, time, end_time, end_date, is_home, event_type, template_id, venue_id, rsvp_opt_out, rsvp_require_reason) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		req.SeasonID, req.Opponent, req.Date, req.Time, endTimeVal, endDateVal, isHome, req.EventType, templateIDVal, venueIDVal, req.RsvpOptOut, rsvpRequireReason)
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

	// For generic events: persist user-supplied slots with is_custom=1 (no template).
	// For heim/auswärts: req.Slots is intentionally ignored — runAutoRegen derives
	// all slots from the resolved template + adjacency context below.
	if req.EventType == "generisch" {
		for _, s := range req.Slots {
			n := s.SlotsCount
			if n <= 0 {
				n = 1
			}
			if _, err = tx.ExecContext(r.Context(),
				`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc, slots_total, team_id, season_id, game_id, is_custom)
				 VALUES (?,?,?,?,?,?,NULL,?,?,1)`,
				eventName, req.Date, s.EventTime, s.DutyTypeID, "", n, req.SeasonID, gameID); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
	}

	summary, err := h.runAutoRegen(r.Context(), tx, dateWindow(req.Date), req.SeasonID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	for _, teamID := range req.TeamIDs {
		h.db.ExecContext(r.Context(), `
			INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at, absence_id)
			SELECT ?, km.member_id, m.user_id, 'declined', a.type, datetime('now'), a.id
			FROM member_absences a
			JOIN members m ON m.id = a.member_id
			JOIN kader_members km ON km.member_id = a.member_id
			JOIN kader k ON k.id = km.kader_id AND k.team_id = ? AND k.season_id = ?
			WHERE ? BETWEEN a.start_date AND a.end_date
			ON CONFLICT(game_id, member_id) DO NOTHING`,
			gameID, teamID, req.SeasonID, req.Date)
	}

	h.hub.Broadcast("games")
	notify.Send(h.db, h.cfg, h.teamMembersAndParents(req.TeamIDs),
		"games", "Neues Spiel", eventName+" am "+req.Date, fmt.Sprintf("/termine?focus=game-%d", gameID))
	h.dispatchRegenNotifications(summary)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": gameID, "regen_summary": summary})
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
		Date              string          `json:"date"`
		Time              string          `json:"time"`
		EndTime           *string         `json:"end_time"`
		EndDate           *string         `json:"end_date"`
		Opponent          string          `json:"opponent"`
		TeamIDs           []int           `json:"team_ids"`
		EventType         string          `json:"event_type"`
		VenueID           *int            `json:"venue_id"`
		RsvpOptOut        *int            `json:"rsvp_opt_out,omitempty"`
		RsvpRequireReason *int            `json:"rsvp_require_reason,omitempty"`
		TemplateID        json.RawMessage `json:"template_id,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Tri-State für template_id: Feld fehlt = unverändert, "null" = NULL setzen,
	// Zahl = setzen.
	tplSet := len(req.TemplateID) > 0
	tplToNull := false
	var tplValue int
	if tplSet {
		if string(req.TemplateID) == "null" {
			tplToNull = true
		} else if err := json.Unmarshal(req.TemplateID, &tplValue); err != nil {
			http.Error(w, "bad request: template_id muss null oder Zahl sein", http.StatusBadRequest)
			return
		}
	}

	if req.EndDate != nil && *req.EndDate != "" && req.Date != "" && *req.EndDate < req.Date {
		http.Error(w, "end_date must be >= date", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Capture pre-update state so the regen window can include the old date if it changes.
	var oldDate string
	var oldSeasonID int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT date, season_id FROM games WHERE id=?`, id).Scan(&oldDate, &oldSeasonID); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var endTimeVal interface{}
	if req.EndTime != nil && *req.EndTime != "" {
		endTimeVal = *req.EndTime
	}
	var endDateVal interface{}
	if req.EndDate != nil && *req.EndDate != "" {
		endDateVal = *req.EndDate
	}
	var venueIDVal interface{}
	if req.VenueID != nil {
		venueIDVal = *req.VenueID
	}
	var res sql.Result
	setCols := []string{"date=?", "time=?", "end_time=?", "end_date=?", "opponent=?", "venue_id=?"}
	setArgs := []any{req.Date, req.Time, endTimeVal, endDateVal, req.Opponent, venueIDVal}
	if req.EventType == "heim" || req.EventType == "auswärts" || req.EventType == "generisch" {
		isHome := req.EventType == "heim"
		setCols = append(setCols, "event_type=?", "is_home=?")
		setArgs = append(setArgs, req.EventType, isHome)
	}
	if tplSet {
		if tplToNull {
			setCols = append(setCols, "template_id=NULL")
		} else {
			setCols = append(setCols, "template_id=?")
			setArgs = append(setArgs, tplValue)
		}
	}
	setArgs = append(setArgs, id)
	res, err = tx.ExecContext(r.Context(),
		`UPDATE games SET `+strings.Join(setCols, ", ")+` WHERE id=?`, setArgs...)
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

	// Partial-Update: rsvp_opt_out / rsvp_require_reason nur setzen, wenn im Request enthalten.
	if req.RsvpOptOut != nil || req.RsvpRequireReason != nil {
		setParts := []string{}
		setArgs := []interface{}{}
		if req.RsvpOptOut != nil {
			setParts = append(setParts, "rsvp_opt_out=?")
			setArgs = append(setArgs, *req.RsvpOptOut)
		}
		if req.RsvpRequireReason != nil {
			setParts = append(setParts, "rsvp_require_reason=?")
			setArgs = append(setArgs, *req.RsvpRequireReason)
		}
		setArgs = append(setArgs, id)
		if _, err = tx.ExecContext(r.Context(),
			`UPDATE games SET `+strings.Join(setParts, ", ")+` WHERE id=?`, setArgs...); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	regenDates := append(dateWindow(oldDate), dateWindow(req.Date)...)
	summary, err := h.runAutoRegen(r.Context(), tx, regenDates, oldSeasonID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("games")
	gameIDInt := 0
	fmt.Sscan(id, &gameIDInt)
	notify.Send(h.db, h.cfg,
		h.teamMembersAndParents(h.gameTeamIDs(gameIDInt)),
		"games", "Spielinfo geändert", req.Opponent+" — Details aktualisiert", fmt.Sprintf("/termine?focus=game-%d", gameIDInt))
	h.dispatchRegenNotifications(summary)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"regen_summary": summary})
}

// DELETE /api/admin/games/{id}
// Deletes a game (incl. generic events) together with all duty_slots and
// duty_assignments referencing it (via ON DELETE CASCADE since migration 027).
// For each fulfilled assignment that gets cascade-deleted, the corresponding
// duty_accounts.ist is recomputed in the same transaction so no orphan hours
// remain on user accounts.
func (h *Handler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Fetch team IDs before deleting (game_teams rows are cascade-deleted)
	teamIDs := h.gameTeamIDs(id)

	// Collect event metadata + affected duty assignees before the cascade fires.
	var (
		seasonID  int
		opponent  string
		eventDate string
	)
	err := h.db.QueryRowContext(r.Context(),
		`SELECT season_id, COALESCE(opponent, ''), date FROM games WHERE id=?`, id).
		Scan(&seasonID, &opponent, &eventDate)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	assignedUIDs, fulfilledUIDs, err := h.dutyAssigneesForGame(r.Context(), id)
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

	res, err := tx.ExecContext(r.Context(), `DELETE FROM games WHERE id=?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Re-aggregate duty_accounts.ist for users whose fulfilled assignments just disappeared.
	for _, uid := range fulfilledUIDs {
		if _, err = tx.ExecContext(r.Context(), `
			UPDATE duty_accounts SET ist = (
				SELECT COALESCE(SUM(dt.hours_value), 0)
				FROM duty_assignments da
				JOIN duty_slots ds ON ds.id = da.duty_slot_id
				JOIN duty_types dt ON dt.id = ds.duty_type_id
				WHERE da.user_id = ? AND ds.season_id = ? AND da.status = 'fulfilled'
			)
			WHERE user_id = ? AND season_id = ?`,
			uid, seasonID, uid, seasonID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	// Regen adjacent days — the deleted date itself has no slots anymore.
	window := dateWindow(eventDate)
	neighborDates := []string{window[0], window[2]}
	summary, err := h.runAutoRegen(r.Context(), tx, neighborDates, seasonID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("games")

	// Targeted notification to duty assignees in their "duties" category.
	if len(assignedUIDs) > 0 {
		eventName := opponent
		if eventName == "" {
			eventName = "Termin am " + formatDateDMY(eventDate)
		}
		body := fmt.Sprintf("Dein Dienst zum %s am %s wurde gelöscht.", eventName, formatDateDMY(eventDate))
		notify.Send(h.db, h.cfg, assignedUIDs, "duties", "Dienst entfällt", body, "/dienste")
	}

	// Team-wide event-cancellation notification in "games" category (unchanged audience).
	notify.Send(h.db, h.cfg, h.teamMembersAndParents(teamIDs),
		"games", "Spiel abgesagt", "Ein Spiel wurde abgesagt", "/termine")

	h.dispatchRegenNotifications(summary)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"regen_summary": summary})
}

// dutyAssigneesForGame returns the user IDs of all duty_assignments for slots
// of the given game. The second return is the subset whose status='fulfilled'.
func (h *Handler) dutyAssigneesForGame(ctx context.Context, gameID string) (assigned, fulfilled []int, err error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT da.user_id, da.status
		FROM duty_assignments da
		JOIN duty_slots ds ON ds.id = da.duty_slot_id
		WHERE ds.game_id = ?`, gameID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	seen := map[int]bool{}
	for rows.Next() {
		var uid int
		var status string
		if err = rows.Scan(&uid, &status); err != nil {
			return nil, nil, err
		}
		if !seen[uid] {
			seen[uid] = true
			assigned = append(assigned, uid)
		}
		if status == "fulfilled" {
			fulfilled = append(fulfilled, uid)
		}
	}
	return assigned, fulfilled, rows.Err()
}

// formatDateDMY turns "2026-06-14" (or an ISO timestamp) into "14.06.2026".
func formatDateDMY(s string) string {
	if len(s) < 10 {
		return s
	}
	d := s[:10]
	return d[8:10] + "." + d[5:7] + "." + d[0:4]
}

// GET /api/teams — filtered by user role
func (h *Handler) ListTeamsForUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	type team struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		AgeClass   string `json:"age_class"`
		Gender     string `json:"gender"`
		TeamNumber int    `json:"team_number"`
		GroupCount int    `json:"group_count"`
		IsActive   bool   `json:"is_active"`
	}

	const activeSeasonSub = `(SELECT id FROM seasons WHERE is_active=1 LIMIT 1)`
	const groupCountSub = `(SELECT COUNT(*) FROM kader k2 WHERE k2.season_id=k.season_id AND k2.age_class=k.age_class AND k2.gender=k.gender)`

	var rows *sql.Rows
	var err error
	if claims.Role == "admin" || claims.HasFunction("vorstand") {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT t.id, t.name, t.age_class, t.gender, k.team_number, `+groupCountSub+`, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 WHERE k.season_id = `+activeSeasonSub+`
			 ORDER BY t.age_class, t.gender, k.team_number`)
	} else if claims.IsTrainerLike() && !claims.HasFunction("sportliche_leitung") {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT t.id, t.name, t.age_class, t.gender, k.team_number, `+groupCountSub+`, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 JOIN kader_trainers kt ON kt.kader_id = k.id
			 JOIN members m ON m.id = kt.member_id
			 WHERE k.season_id = `+activeSeasonSub+` AND m.user_id = ?
			 ORDER BY t.age_class, t.gender, k.team_number`, claims.UserID)
	} else if !claims.IsTrainerLike() {
		// spieler / elternteil: only teams the user or their children are in
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT DISTINCT t.id, t.name, t.age_class, t.gender, k.team_number, `+groupCountSub+`, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 JOIN team_memberships tm ON tm.team_id = t.id AND tm.season_id = k.season_id
			 WHERE k.season_id = `+activeSeasonSub+`
			   AND (
			     EXISTS(SELECT 1 FROM members m WHERE m.id = tm.member_id AND m.user_id = ?)
			     OR EXISTS(SELECT 1 FROM family_links fl WHERE fl.member_id = tm.member_id AND fl.parent_user_id = ?)
			   )
			 ORDER BY t.age_class, t.gender, k.team_number`, claims.UserID, claims.UserID)
	} else {
		// sportliche_leitung: all teams
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT t.id, t.name, t.age_class, t.gender, k.team_number, `+groupCountSub+`, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 WHERE k.season_id = `+activeSeasonSub+`
			 ORDER BY t.age_class, t.gender, k.team_number`)
	}

	result := []team{}
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t team
			var active int
			rows.Scan(&t.ID, &t.Name, &t.AgeClass, &t.Gender, &t.TeamNumber, &t.GroupCount, &active)
			t.IsActive = active == 1
			result = append(result, t)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ── Duty Templates ───────────────────────────────────────────────────────────

type templateItem struct {
	ID            int      `json:"id,omitempty"`
	DutyTypeID    int      `json:"duty_type_id"`
	DutyTypeName  string   `json:"duty_type_name,omitempty"`
	Anchor        string   `json:"anchor"`
	OffsetMinutes int      `json:"offset_minutes"`
	SlotsCount    int      `json:"slots_count"`
	Audiences     []string `json:"audiences,omitempty"`
}

func (h *Handler) scanTemplateItems(ctx context.Context, templateID int) []templateItem {
	rows, _ := h.db.QueryContext(ctx,
		`SELECT gti.id, gti.duty_type_id, dt.name, gti.anchor, gti.offset_minutes, gti.slots_count, gti.audiences
		 FROM game_template_items gti JOIN duty_types dt ON dt.id = gti.duty_type_id
		 WHERE gti.template_id=? ORDER BY gti.sort_order, gti.id`, templateID)
	items := []templateItem{}
	if rows == nil {
		return items
	}
	defer rows.Close()
	for rows.Next() {
		var it templateItem
		var audiences sql.NullString
		rows.Scan(&it.ID, &it.DutyTypeID, &it.DutyTypeName, &it.Anchor, &it.OffsetMinutes, &it.SlotsCount, &audiences)
		it.Audiences = audiencesFromDB(audiences)
		items = append(items, it)
	}
	return items
}

// GET /api/admin/duty-templates
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT gt.id, gt.name, gt.template_type, gt.duration_minutes, COUNT(gti.id)
		 FROM game_templates gt
		 LEFT JOIN game_template_items gti ON gti.template_id = gt.id
		 GROUP BY gt.id ORDER BY gt.id`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type listItem struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		TemplateType    string `json:"template_type"`
		DurationMinutes int    `json:"duration_minutes"`
		ItemCount       int    `json:"item_count"`
	}
	result := []listItem{}
	for rows.Next() {
		var t listItem
		rows.Scan(&t.ID, &t.Name, &t.TemplateType, &t.DurationMinutes, &t.ItemCount)
		result = append(result, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/admin/duty-templates/{id}
func (h *Handler) GetTemplateByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var t struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		TemplateType    string `json:"template_type"`
		DurationMinutes int    `json:"duration_minutes"`
	}
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, template_type, duration_minutes FROM game_templates WHERE id=?`, id).
		Scan(&t.ID, &t.Name, &t.TemplateType, &t.DurationMinutes)
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
		"duration_minutes": t.DurationMinutes, "items": items,
	})
}

// POST /api/admin/duty-templates
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string `json:"name"`
		TemplateType    string `json:"template_type"`
		DurationMinutes int    `json:"duration_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.TemplateType != "heim" && req.TemplateType != "auswärts" && req.TemplateType != "generisch" {
		http.Error(w, "invalid template_type", http.StatusBadRequest)
		return
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 90
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?,?,?)`,
		req.Name, req.TemplateType, req.DurationMinutes)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newID, _ := res.LastInsertId()
	h.hub.Broadcast("games")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id": newID, "name": req.Name, "template_type": req.TemplateType,
		"duration_minutes": req.DurationMinutes, "items": []any{},
	})
}

// PUT /api/admin/duty-templates/{id}
func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid template id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name            string         `json:"name"`
		TemplateType    string         `json:"template_type"`
		DurationMinutes int            `json:"duration_minutes"`
		Items           []templateItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.TemplateType != "heim" && req.TemplateType != "auswärts" && req.TemplateType != "generisch" {
		http.Error(w, "invalid template_type", http.StatusBadRequest)
		return
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 90
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
		`UPDATE game_templates SET name=?, template_type=?, duration_minutes=? WHERE id=?`,
		req.Name, req.TemplateType, req.DurationMinutes, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	_, err = tx.ExecContext(r.Context(), `DELETE FROM game_template_items WHERE template_id=?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for i, it := range req.Items {
		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order, audiences)
			 VALUES (?,?,?,?,?,?,?)`,
			id, it.DutyTypeID, it.Anchor, it.OffsetMinutes, it.SlotsCount, i, audiencesToDB(it.Audiences))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("games")
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
	h.hub.Broadcast("games")
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
	gameEndTime := r.URL.Query().Get("end_time")
	gameIDStr := r.URL.Query().Get("game_id")
	dateStr := r.URL.Query().Get("date")

	var templateID, durationMins int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, duration_minutes FROM game_templates WHERE id=?`, templateIDStr).
		Scan(&templateID, &durationMins)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var allGameTimes []string
	var hasPrevDay, hasNextDay bool
	if gameIDStr != "" {
		// Regeneration context: load from existing game
		var gameDate string
		var seasonID int
		var dbEndTime sql.NullString
		if h.db.QueryRowContext(r.Context(),
			`SELECT date, season_id, end_time FROM games WHERE id=?`, gameIDStr).
			Scan(&gameDate, &seasonID, &dbEndTime) == nil {
			if gameEndTime == "" && dbEndTime.Valid {
				gameEndTime = dbEndTime.String
			}
			allGameTimes, hasPrevDay, hasNextDay = h.loadSameDayContext(r.Context(), gameDate, seasonID)
		}
	} else if dateStr != "" {
		// New game context: load by date from active season, then insert new game's time sorted
		var seasonID int
		h.db.QueryRowContext(r.Context(),
			`SELECT id FROM seasons WHERE is_active=1 LIMIT 1`).Scan(&seasonID)
		if seasonID > 0 {
			allGameTimes, hasPrevDay, hasNextDay = h.loadSameDayContext(r.Context(), dateStr, seasonID)
			// Insert the new game's own time into the sorted list
			inserted := false
			for i, t := range allGameTimes {
				if gameTime <= t {
					allGameTimes = append(allGameTimes[:i], append([]string{gameTime}, allGameTimes[i:]...)...)
					inserted = true
					break
				}
			}
			if !inserted {
				allGameTimes = append(allGameTimes, gameTime)
			}
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
		Conflict     bool   `json:"conflict,omitempty"`
	}
	result := []preview{}
	for _, it := range items {
		var eventTime string
		if it.Anchor == "end" && gameEndTime != "" {
			eventTime = addMinutes(gameEndTime, it.OffsetMinutes)
		} else {
			offset := it.OffsetMinutes
			if it.Anchor == "end" {
				offset += durationMins
			}
			eventTime = addMinutes(gameTime, offset)
		}

		dutyTypeID := it.DutyTypeID
		if len(allGameTimes) > 0 {
			isBeforeAllGames, isAfterAllGames, isBetweenGames := classifySlotPosition(eventTime, gameTime, allGameTimes)
			dutyTypeID = applyBehavior(it, gameTime, eventTime, allGameTimes, hasPrevDay, hasNextDay,
				isBeforeAllGames, isAfterAllGames, isBetweenGames)
			if dutyTypeID == -1 {
				continue
			}
		}

		result = append(result, preview{
			DutyTypeID:   dutyTypeID,
			DutyTypeName: it.DutyTypeName,
			EventTime:    eventTime,
			SlotsCount:   it.SlotsCount,
		})
	}

	// Konflikte markieren: gleicher Diensttyp zur gleichen Zeit an diesem Tag für ein anderes Spiel
	if gameIDStr != "" {
		var gameDate string
		h.db.QueryRowContext(r.Context(), `SELECT date FROM games WHERE id=?`, gameIDStr).Scan(&gameDate)
		if gameDate != "" {
			for i, p := range result {
				var count int
				h.db.QueryRowContext(r.Context(),
					`SELECT COUNT(*) FROM duty_slots
					 WHERE duty_type_id=? AND event_time=? AND event_date=? AND game_id != ?`,
					p.DutyTypeID, p.EventTime, gameDate, gameIDStr).Scan(&count)
				if count > 0 {
					result[i].Conflict = true
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/admin/games/regenerate-day
// Thin wrapper around runAutoRegen. Frontend no longer triggers this; kept for
// internal repair workflows (e.g. season-wide rebuild after template change).
func (h *Handler) RegenerateDaySlots(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "date is required", http.StatusBadRequest)
		return
	}

	var seasonID int
	if s := r.URL.Query().Get("season_id"); s != "" {
		seasonID, _ = strconv.Atoi(s)
	}
	if seasonID == 0 {
		h.db.QueryRowContext(r.Context(), `SELECT id FROM seasons WHERE is_active=1 LIMIT 1`).Scan(&seasonID)
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	summary, err := h.runAutoRegen(r.Context(), tx, []string{date}, seasonID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.dispatchRegenNotifications(summary)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// POST /api/admin/games/{id}/regenerate
// Thin wrapper around runAutoRegen scoped to the game's date. Frontend no longer
// triggers this; kept for internal repair workflows.
func (h *Handler) RegenerateSlots(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")
	var seasonID int
	var date string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT season_id, date FROM games WHERE id=?`, gameID).Scan(&seasonID, &date)
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
	summary, err := h.runAutoRegen(r.Context(), tx, []string{date}, seasonID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.dispatchRegenNotifications(summary)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// ── Game RSVP ────────────────────────────────────────────────────────────────

type childRSVP struct {
	MemberID int     `json:"member_id"`
	Name     string  `json:"name"`
	RSVP     *string `json:"rsvp"`
}

type gameVenueRef struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Note       string `json:"note"`
}

type gameListItem struct {
	ID                  int           `json:"id"`
	Date                string        `json:"date"`
	Time                string        `json:"time"`
	Opponent            string        `json:"opponent"`
	EventType           string        `json:"event_type"`
	IsHome              bool          `json:"is_home"`
	SeasonID            int           `json:"season_id"`
	TeamNames           string        `json:"team_names"`
	TeamIDs             []int         `json:"team_ids"`
	TeamDisplayShortCSV string        `json:"team_display_short_csv"`
	TeamDisplayLongCSV  string        `json:"team_display_long_csv"`
	ConfirmedCount      int           `json:"confirmed_count"`
	DeclinedCount       int           `json:"declined_count"`
	MaybeCount          int           `json:"maybe_count"`
	MyRSVP              *string       `json:"my_rsvp"`
	MyRSVPLocked        bool          `json:"my_rsvp_locked"`
	ChildrenRSVP        []childRSVP   `json:"children_rsvp,omitempty"`
	RsvpOptOut          int           `json:"rsvp_opt_out"`
	RsvpRequireReason   int           `json:"rsvp_require_reason"`
	Venue               *gameVenueRef `json:"venue,omitempty"`
}

// memberIDForUser returns the member_id for a user, or 0 if not found.
func (h *Handler) memberIDForUser(ctx context.Context, userID int) (int, error) {
	var memberID int
	err := h.db.QueryRowContext(ctx,
		`SELECT id FROM members WHERE user_id = ?`, userID).Scan(&memberID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return memberID, err
}

// parentHasChild returns true if parentUserID has a family_links entry for memberID.
func (h *Handler) parentHasChild(ctx context.Context, parentUserID, memberID int) (bool, error) {
	var count int
	err := h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id = ? AND member_id = ?`,
		parentUserID, memberID).Scan(&count)
	return count > 0, err
}

// GET /api/games/my
func (h *Handler) ListMyGames(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	if from == "" {
		from = strings.Repeat("0", 10) // no lower bound: "0000-00-00"
	}
	if to == "" {
		to = "9999-12-31"
	}

	memberID, err := h.memberIDForUser(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Build team filter based on role
	var teamSQL string
	var teamArgs []any
	if claims.Role == "admin" || claims.HasFunction("sportliche_leitung") {
		teamSQL = "1=1"
	} else {
		var conds []string
		if claims.HasFunction("trainer") {
			conds = append(conds, `gt.team_id IN (
				SELECT DISTINCT k.team_id FROM kader k
				JOIN kader_trainers kt ON kt.kader_id = k.id
				JOIN members m ON m.id = kt.member_id
				WHERE m.user_id = ?)`)
			teamArgs = append(teamArgs, claims.UserID)
		}
		if claims.IsParent {
			conds = append(conds, `gt.team_id IN (
				SELECT DISTINCT tm.team_id FROM team_memberships tm
				JOIN members m ON m.id = tm.member_id
				JOIN family_links fl ON fl.member_id = m.id
				WHERE fl.parent_user_id = ?
				UNION
				SELECT DISTINCT k.team_id FROM kader_extended_members kem
				JOIN kader k ON k.id = kem.kader_id
				JOIN family_links fl ON fl.member_id = kem.member_id
				WHERE fl.parent_user_id = ?)`)
			teamArgs = append(teamArgs, claims.UserID, claims.UserID)
		}
		conds = append(conds, `gt.team_id IN (
			SELECT DISTINCT tm.team_id FROM team_memberships tm
			JOIN members m ON m.id = tm.member_id
			WHERE m.user_id = ?
			UNION
			SELECT DISTINCT k.team_id FROM kader_extended_members kem
			JOIN kader k ON k.id = kem.kader_id
			JOIN members m2 ON m2.id = kem.member_id
			WHERE m2.user_id = ?)`)
		teamArgs = append(teamArgs, claims.UserID, claims.UserID)
		teamSQL = "(" + strings.Join(conds, " OR ") + ")"
	}

	// Args order: memberID (my_rsvp), memberID (my_rsvp_locked), memberID (in_regular_kader), teamArgs, from, to
	args := append([]any{memberID, memberID, memberID}, teamArgs...)
	args = append(args, from, to)

	query := fmt.Sprintf(`
		SELECT DISTINCT g.id, g.date, g.time, g.opponent, g.event_type, g.is_home, g.season_id,
		       (SELECT GROUP_CONCAT(t.name, ', ') FROM game_teams gt2 JOIN teams t ON t.id = gt2.team_id WHERE gt2.game_id = g.id),
		       (SELECT GROUP_CONCAT(gt3.team_id) FROM game_teams gt3 WHERE gt3.game_id = g.id),
		       (SELECT GROUP_CONCAT(s, ', ') FROM (
		            SELECT COALESCE(`+appdb.TeamDisplayShort("t_s")+`, t_s.name) AS s
		            FROM game_teams gt_s JOIN teams t_s ON t_s.id = gt_s.team_id
		            WHERE gt_s.game_id = g.id ORDER BY s)),
		       (SELECT GROUP_CONCAT(l, ', ') FROM (
		            SELECT COALESCE(`+appdb.TeamDisplayName("t_l")+`, t_l.name) AS l
		            FROM game_teams gt_l JOIN teams t_l ON t_l.id = gt_l.team_id
		            WHERE gt_l.game_id = g.id ORDER BY l)),
		       CASE WHEN g.rsvp_opt_out = 1
		            THEN COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='confirmed'),0) + (
		                   SELECT COUNT(DISTINCT km.member_id) FROM game_teams gt4
		                   JOIN kader k4 ON k4.team_id = gt4.team_id AND k4.season_id = g.season_id
		                   JOIN kader_members km ON km.kader_id = k4.id
		                   WHERE gt4.game_id = g.id
		                   AND NOT EXISTS (SELECT 1 FROM game_responses gr2 WHERE gr2.game_id = g.id AND gr2.member_id = km.member_id)
		                 )
		            ELSE COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='confirmed'),0)
		       END,
		       COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='declined'),0),
		       COALESCE((SELECT COUNT(*) FROM game_responses WHERE game_id=g.id AND status='maybe'),0),
		       (SELECT status FROM game_responses WHERE game_id=g.id AND member_id=?),
		       (SELECT absence_id IS NOT NULL FROM game_responses WHERE game_id=g.id AND member_id=? LIMIT 1),
		       g.rsvp_opt_out, g.rsvp_require_reason,
		       EXISTS(SELECT 1 FROM game_teams gt_r
		              JOIN kader k_r ON k_r.team_id = gt_r.team_id AND k_r.season_id = g.season_id
		              JOIN kader_members km_r ON km_r.kader_id = k_r.id AND km_r.member_id = ?
		              WHERE gt_r.game_id = g.id),
		       v.id, v.name, v.street, v.city, v.postal_code, v.note
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id
		LEFT JOIN venues v ON v.id = g.venue_id
		WHERE %s AND g.date >= ? AND g.date <= ?
		ORDER BY g.date, g.time`, teamSQL)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListMyGames: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []gameListItem{}
	for rows.Next() {
		var g gameListItem
		var isHome, inRegularKader int
		var myRSVP sql.NullString
		var myRSVPLocked sql.NullInt64
		var teamNames, teamIDsCSV, teamShortCSV, teamLongCSV sql.NullString
		var vID sql.NullInt64
		var vName, vStreet, vCity, vPostal, vNote sql.NullString
		if err := rows.Scan(&g.ID, &g.Date, &g.Time, &g.Opponent, &g.EventType, &isHome, &g.SeasonID,
			&teamNames, &teamIDsCSV, &teamShortCSV, &teamLongCSV, &g.ConfirmedCount, &g.DeclinedCount, &g.MaybeCount, &myRSVP, &myRSVPLocked,
			&g.RsvpOptOut, &g.RsvpRequireReason, &inRegularKader,
			&vID, &vName, &vStreet, &vCity, &vPostal, &vNote); err != nil {
			fmt.Fprintf(os.Stderr, "ListMyGames scan: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		g.IsHome = isHome == 1
		g.TeamNames = teamNames.String
		g.TeamDisplayShortCSV = teamShortCSV.String
		g.TeamDisplayLongCSV = teamLongCSV.String
		g.TeamIDs = []int{}
		if teamIDsCSV.Valid {
			for _, s := range strings.Split(teamIDsCSV.String, ",") {
				if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
					g.TeamIDs = append(g.TeamIDs, id)
				}
			}
		}
		if myRSVP.Valid {
			g.MyRSVP = &myRSVP.String
		} else if g.RsvpOptOut == 1 && inRegularKader == 1 {
			confirmed := "confirmed"
			g.MyRSVP = &confirmed
		}
		g.MyRSVPLocked = myRSVPLocked.Valid && myRSVPLocked.Int64 == 1
		if vID.Valid {
			g.Venue = &gameVenueRef{
				ID: int(vID.Int64), Name: vName.String, Street: vStreet.String,
				City: vCity.String, PostalCode: vPostal.String, Note: vNote.String,
			}
		}
		result = append(result, g)
	}

	if claims.IsParent && len(result) > 0 {
		if err := h.attachChildrenRSVPToGames(r.Context(), claims.UserID, result); err != nil {
			fmt.Fprintf(os.Stderr, "ListMyGames children_rsvp: %v\n", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/games/{id}/respond
func (h *Handler) RespondToGame(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if ok, _ := auth.UserCanSeeGame(r.Context(), h.db, claims.UserID, gameID); !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var req struct {
		MemberID int    `json:"member_id"`
		Status   string `json:"status"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Status != "confirmed" && req.Status != "declined" && req.Status != "maybe" {
		http.Error(w, "status must be confirmed, declined, or maybe", http.StatusBadRequest)
		return
	}

	var memberID int
	switch claims.Role {
	case "spieler":
		memberID, err = h.memberIDForUser(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if memberID == 0 {
			http.Error(w, "your account is not linked to a member record", http.StatusUnprocessableEntity)
			return
		}
	case "elternteil":
		if req.MemberID == 0 {
			http.Error(w, "member_id required for elternteil", http.StatusBadRequest)
			return
		}
		ok, err := h.parentHasChild(r.Context(), claims.UserID, req.MemberID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		memberID = req.MemberID
	default:
		if req.MemberID == 0 {
			memberID, err = h.memberIDForUser(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if memberID == 0 {
				http.Error(w, "member_id required", http.StatusBadRequest)
				return
			}
		} else {
			memberID = req.MemberID
		}
	}

	var existingAbsenceID sql.NullInt64
	h.db.QueryRowContext(r.Context(),
		`SELECT absence_id FROM game_responses WHERE game_id = ? AND member_id = ?`,
		gameID, memberID).Scan(&existingAbsenceID)
	if existingAbsenceID.Valid {
		http.Error(w, "response is locked by an absence", http.StatusForbidden)
		return
	}

	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(game_id, member_id) DO UPDATE SET
		  responded_by = excluded.responded_by,
		  status       = excluded.status,
		  reason       = excluded.reason,
		  responded_at = datetime('now')`,
		gameID, memberID, claims.UserID, req.Status, req.Reason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RespondToGame upsert: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("games")
	w.WriteHeader(http.StatusNoContent)
}

type gameResponse struct {
	MemberID   int     `json:"member_id"`
	MemberName string  `json:"member_name"`
	Status     *string `json:"status"`
	Reason     *string `json:"reason"`
}

// GET /api/games/{id}/responses
func (h *Handler) ListGameResponses(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if ok, _ := auth.UserCanSeeGame(r.Context(), h.db, claims.UserID, gameID); !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	isTrainerLike := claims.Role == "admin" || claims.HasFunction("trainer")

	memberID, err := h.memberIDForUser(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	childMemberIDs := map[int]bool{}
	if claims.IsParent {
		childRows, err := h.db.QueryContext(r.Context(),
			`SELECT member_id FROM family_links WHERE parent_user_id = ?`, claims.UserID)
		if err == nil {
			defer childRows.Close()
			for childRows.Next() {
				var cid int
				childRows.Scan(&cid)
				childMemberIDs[cid] = true
			}
		}
	}

	// Return all kader members for the game's teams/season, LEFT JOIN responses
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT DISTINCT m.id, m.first_name || ' ' || m.last_name,
		       gr.status, gr.reason
		FROM members m
		JOIN kader_members km ON km.member_id = m.id
		JOIN kader k ON k.id = km.kader_id AND k.season_id = (SELECT season_id FROM games WHERE id = ?)
		JOIN game_teams gt ON gt.game_id = ? AND gt.team_id = k.team_id
		LEFT JOIN game_responses gr ON gr.game_id = ? AND gr.member_id = m.id
		ORDER BY m.last_name, m.first_name`, gameID, gameID, gameID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListGameResponses: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []gameResponse{}
	for rows.Next() {
		var resp gameResponse
		var status, reason sql.NullString
		rows.Scan(&resp.MemberID, &resp.MemberName, &status, &reason)
		if status.Valid {
			resp.Status = &status.String
		}
		canSeeReason := isTrainerLike ||
			(memberID > 0 && resp.MemberID == memberID) ||
			childMemberIDs[resp.MemberID]
		if canSeeReason && reason.Valid && reason.String != "" {
			resp.Reason = &reason.String
		}
		result = append(result, resp)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type participantItem struct {
	MemberID         int     `json:"member_id"`
	MemberName       string  `json:"member_name"`
	IsExtended       bool    `json:"is_extended"`
	RsvpStatus       *string `json:"rsvp_status"`
	InLineup         bool    `json:"in_lineup"`
	TeamID           int     `json:"team_id"`
	crossTeamVisible bool    `json:"-"`
}

// participantsResponse erlaubt es, neben den sichtbaren Items zusätzlich pro
// Team einen Hinweis zu transportieren, wenn Mitglieder gefiltert wurden. Wir
// behalten Items in einem `items`-Feld, damit das Frontend die `hidden_team_ids`
// für den Footer „Weitere Mitglieder nicht sichtbar" rendern kann.
type participantsResponse struct {
	Items         []participantItem `json:"items"`
	HiddenTeamIDs []int             `json:"hidden_team_ids"`
}

// GET /api/games/{id}/participants
//
// Bei Multi-Team-Events filtert die Antwort für Caller ohne Funktion
// (admin/trainer/sportliche_leitung/vorstand) auf:
//   - Mitglieder aus den Teams, in denen der Caller selbst oder eines seiner
//     Kinder (via family_links) im Kader/erweiterten Kader steht ("meine Teams"),
//   - plus Mitglieder fremder Teams, deren cross_team_visible=1 ist.
//
// Funktionsträger sehen ungefiltert. Single-Team-Events bleiben ungefiltert.
func (h *Handler) GetParticipants(w http.ResponseWriter, r *http.Request) {
	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	if claims != nil {
		if ok, _ := auth.UserCanSeeGame(r.Context(), h.db, claims.UserID, gameID); !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
	bypass := claims != nil && (claims.Role == "admin" ||
		claims.HasFunction("trainer") ||
		claims.HasFunction("sportliche_leitung") ||
		claims.HasFunction("vorstand"))

	// Filter greift nur bei Multi-Team-Events und für Nicht-Funktionsträger.
	var teamCount int
	h.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM game_teams WHERE game_id=?`, gameID).Scan(&teamCount)
	applyFilter := !bypass && teamCount > 1 && claims != nil

	myTeamSet := map[int]bool{}
	if applyFilter {
		myTeamSet, err = h.myTeamsInEvent(r.Context(), gameID, claims.UserID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetParticipants/myTeams: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	// Bei rsvp_opt_out=1 gilt ein regulärer Kader-Spieler ohne Response-Eintrag
	// implizit als "confirmed". Extended-Mitglieder sind davon ausgenommen — sie
	// müssen explizit zusagen.
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT member_id, member_name, is_extended, rsvp_status, in_lineup, team_id, cross_team_visible
		FROM (
			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       0 AS is_extended,
			       COALESCE(gr.status,
			                CASE WHEN (SELECT rsvp_opt_out FROM games WHERE id = ?) = 1
			                     THEN 'confirmed' ELSE NULL END) AS rsvp_status,
			       EXISTS(SELECT 1 FROM game_lineup gl WHERE gl.game_id=? AND gl.member_id=m.id) AS in_lineup,
			       k.team_id AS team_id,
			       m.cross_team_visible AS cross_team_visible
			FROM members m
			JOIN kader_members km ON km.member_id = m.id
			JOIN kader k ON k.id = km.kader_id
			  AND k.season_id = (SELECT season_id FROM games WHERE id = ?)
			JOIN game_teams gt ON gt.game_id = ? AND gt.team_id = k.team_id
			LEFT JOIN game_responses gr ON gr.game_id = ? AND gr.member_id = m.id

			UNION

			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       1 AS is_extended,
			       NULL AS rsvp_status,
			       EXISTS(SELECT 1 FROM game_lineup gl WHERE gl.game_id=? AND gl.member_id=m.id) AS in_lineup,
			       k.team_id AS team_id,
			       m.cross_team_visible AS cross_team_visible
			FROM members m
			JOIN kader_extended_members kem ON kem.member_id = m.id
			JOIN kader k ON k.id = kem.kader_id
			  AND k.season_id = (SELECT season_id FROM games WHERE id = ?)
			JOIN game_teams gt ON gt.game_id = ? AND gt.team_id = k.team_id
		)
		ORDER BY member_name`,
		gameID, gameID, gameID, gameID, gameID,
		gameID, gameID, gameID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetParticipants: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []participantItem{}
	teamsTouched := map[int]bool{}
	teamsHidden := map[int]bool{}
	for rows.Next() {
		var p participantItem
		var status sql.NullString
		var isExtended, inLineup, ctv int
		rows.Scan(&p.MemberID, &p.MemberName, &isExtended, &status, &inLineup, &p.TeamID, &ctv)
		p.IsExtended = isExtended == 1
		p.InLineup = inLineup == 1
		p.crossTeamVisible = ctv == 1
		if status.Valid {
			p.RsvpStatus = &status.String
		}
		teamsTouched[p.TeamID] = true
		if applyFilter && !myTeamSet[p.TeamID] && !p.crossTeamVisible {
			teamsHidden[p.TeamID] = true
			continue
		}
		items = append(items, p)
	}

	hidden := []int{}
	for tid := range teamsHidden {
		hidden = append(hidden, tid)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participantsResponse{Items: items, HiddenTeamIDs: hidden})
}

// myTeamsInEvent liefert die Menge der team_ids im Event gameID, in deren
// (regulärem ODER erweitertem) Kader der userID selbst Mitglied ist ODER eines
// seiner Kinder (via family_links). Maßgeblich ist die Saison des Games.
func (h *Handler) myTeamsInEvent(ctx context.Context, gameID, userID int) (map[int]bool, error) {
	out := map[int]bool{}
	rows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT k.team_id
		FROM kader k
		WHERE k.season_id = (SELECT season_id FROM games WHERE id = ?)
		  AND k.team_id IN (SELECT team_id FROM game_teams WHERE game_id = ?)
		  AND (
			EXISTS (
				SELECT 1 FROM kader_members km
				JOIN members m ON m.id = km.member_id
				WHERE km.kader_id = k.id
				  AND (m.user_id = ?
				       OR m.id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
			)
			OR EXISTS (
				SELECT 1 FROM kader_extended_members kem
				JOIN members m ON m.id = kem.member_id
				WHERE kem.kader_id = k.id
				  AND (m.user_id = ?
				       OR m.id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
			)
		  )`,
		gameID, gameID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tid int
		if err := rows.Scan(&tid); err != nil {
			return nil, err
		}
		out[tid] = true
	}
	return out, nil
}

// POST /api/games/{id}/lineup
func (h *Handler) SaveLineup(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims.Role != "admin" && !claims.HasFunction("trainer") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		MemberIDs []int `json:"member_ids"`
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

	// Delete all existing lineup entries for this game
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM game_lineup WHERE game_id=?`, gameID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert new lineup
	for _, memberID := range req.MemberIDs {
		if _, err := tx.ExecContext(r.Context(),
			`INSERT INTO game_lineup (game_id, member_id, added_by) VALUES (?,?,?)`,
			gameID, memberID, claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("games")
	w.WriteHeader(http.StatusNoContent)
}

func audiencesFromDB(ns sql.NullString) []string {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	var result []string
	json.Unmarshal([]byte(ns.String), &result)
	if len(result) == 0 {
		return nil
	}
	return result
}

func audiencesToDB(audiences []string) *string {
	if len(audiences) == 0 {
		return nil
	}
	b, _ := json.Marshal(audiences)
	s := string(b)
	return &s
}

// attachChildrenRSVPToGames fills ChildrenRSVP on each item for parent users.
// Only includes children who are kader members of one of the game's teams.
func (h *Handler) attachChildrenRSVPToGames(ctx context.Context, parentUserID int, items []gameListItem) error {
	placeholders := make([]string, len(items))
	gameIDs := make([]any, len(items))
	for i, g := range items {
		placeholders[i] = "?"
		gameIDs[i] = g.ID
	}
	// Single query: for each game, return only children who are in the kader of one of the game's teams
	rows, err := h.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT DISTINCT gt.game_id, m.id, m.first_name || ' ' || m.last_name, gr.status, g.rsvp_opt_out
		FROM game_teams gt
		JOIN games g ON g.id = gt.game_id
		JOIN kader k ON k.team_id = gt.team_id
		  AND k.season_id = g.season_id
		JOIN kader_members km ON km.kader_id = k.id
		JOIN members m ON m.id = km.member_id
		JOIN family_links fl ON fl.member_id = m.id AND fl.parent_user_id = ?
		LEFT JOIN game_responses gr ON gr.game_id = gt.game_id AND gr.member_id = m.id
		WHERE gt.game_id IN (%s)
		ORDER BY m.last_name, m.first_name`,
		strings.Join(placeholders, ",")),
		append([]any{parentUserID}, gameIDs...)...)
	if err != nil {
		return err
	}
	defer rows.Close()

	byGame := map[int][]childRSVP{}
	for rows.Next() {
		var gid int
		var c childRSVP
		var rsvp sql.NullString
		var rsvpOptOut int
		rows.Scan(&gid, &c.MemberID, &c.Name, &rsvp, &rsvpOptOut)
		if rsvp.Valid {
			s := rsvp.String
			c.RSVP = &s
		} else if rsvpOptOut == 1 {
			confirmed := "confirmed"
			c.RSVP = &confirmed
		}
		byGame[gid] = append(byGame[gid], c)
	}

	for i := range items {
		if children, ok := byGame[items[i].ID]; ok {
			items[i].ChildrenRSVP = children
		} else {
			items[i].ChildrenRSVP = []childRSVP{}
		}
	}
	return nil
}

// GET /api/teams/names — all active teams for client-side name computation, available to all authenticated users
func (h *Handler) ListTeamNames(w http.ResponseWriter, r *http.Request) {
	const activeSeasonSub = `(SELECT id FROM seasons WHERE is_active=1 LIMIT 1)`
	const groupCountSub = `(SELECT COUNT(*) FROM kader k2 WHERE k2.season_id=k.season_id AND k2.age_class=k.age_class AND k2.gender=k.gender)`

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT DISTINCT t.id, t.age_class, t.gender, k.team_number, `+groupCountSub+`
		 FROM teams t
		 JOIN kader k ON k.team_id = t.id
		 WHERE k.season_id = `+activeSeasonSub+` AND t.is_active = 1
		 ORDER BY t.age_class, t.gender, k.team_number`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type teamName struct {
		ID         int    `json:"id"`
		AgeClass   string `json:"age_class"`
		Gender     string `json:"gender"`
		TeamNumber int    `json:"team_number"`
		GroupCount int    `json:"group_count"`
	}
	result := []teamName{}
	for rows.Next() {
		var t teamName
		rows.Scan(&t.ID, &t.AgeClass, &t.Gender, &t.TeamNumber, &t.GroupCount)
		result = append(result, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
