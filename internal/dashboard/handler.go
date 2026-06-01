package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

type Season struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"isActive"`
}

type Action struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Text          string `json:"text"`
	Link          string `json:"link"`
	DueDate       string `json:"dueDate,omitempty"`
	EventTime     string `json:"eventTime,omitempty"`
	DutyTypeName  string `json:"dutyTypeName,omitempty"`
	ActionNeeded  bool   `json:"actionNeeded,omitempty"`
}

type Game struct {
	ID          int    `json:"id"`
	Date        string `json:"date"`
	Opponent    string `json:"opponent"`
	IsHome      bool   `json:"isHome"`
	EventType   string `json:"eventType"`
	Team        string `json:"team"`
	SlotsCount  int    `json:"slotsCount"`
	SlotsFilled int    `json:"slotsFilled"`
	Link        string `json:"link"`
}

type TeamStats struct {
	Team          string `json:"team"`
	ActiveMembers int    `json:"activeMembers"`
	TotalMembers  int    `json:"totalMembers"`
	InjuredCount  int    `json:"injuredCount"`
	PausedCount   int    `json:"pausedCount"`
}

type RecentAssignment struct {
	Date     string `json:"date"`
	DutyType string `json:"dutyType"`
	Status   string `json:"status"`
}

type DutyAccount struct {
	Season            string             `json:"season"`
	Ist               int                `json:"ist"`
	Soll              *int               `json:"soll"`
	Children          int                `json:"children"`
	RecentAssignments []RecentAssignment `json:"recentAssignments"`
}

type VehicleInfo struct {
	Seats    int    `json:"seats"`
	Notes    string `json:"notes"`
	UpToDate bool   `json:"upToDate"`
}

type CarpoolingMyEntry struct {
	ID  int    `json:"id"`
	Typ string `json:"typ"`
}

type CarpoolingPaarung struct {
	PaarungID   int    `json:"paarungId"`
	PartnerName string `json:"partnerName"`
}

type CarpoolingOpenEntry struct {
	Typ  string `json:"typ"`
	Name string `json:"name"`
}

type CarpoolingHint struct {
	GameID      int                   `json:"gameId"`
	Date        string                `json:"date"`
	Opponent    string                `json:"opponent"`
	BieteCount  int                   `json:"bieteCount"`
	SucheCount  int                   `json:"sucheCount"`
	MyEntry     *CarpoolingMyEntry    `json:"myEntry"`
	Paarungen   []CarpoolingPaarung   `json:"paarungen"`
	OpenEntries []CarpoolingOpenEntry `json:"openEntries"`
}

type Response struct {
	CurrentSeason  *Season         `json:"currentSeason"`
	NextGameDate   *string         `json:"nextGameDate"`
	Actions        []Action        `json:"actions"`
	NextGames      []Game          `json:"nextGames"`
	TeamStats      *TeamStats      `json:"teamStats"`
	DutyAccount    *DutyAccount    `json:"dutyAccount"`
	VehicleInfo    *VehicleInfo    `json:"vehicleInfo"`
	CarpoolingHint *CarpoolingHint `json:"carpoolingHint"`
}

// GET /api/dashboard
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)
	userID := claims.UserID
	role := effectivePersona(claims.ClubFunctions, claims.IsParent)

	resp := Response{
		Actions:   []Action{},
		NextGames: []Game{},
	}

	// T8: Query active season
	var season Season
	err := h.db.QueryRowContext(ctx,
		`SELECT id, name, is_active FROM seasons WHERE is_active = 1 LIMIT 1`,
	).Scan(&season.ID, &season.Name, &season.IsActive)
	if err == sql.ErrNoRows {
		// No active season: return empty dashboard with warning
		resp.Actions = []Action{{
			ID:   "no-season",
			Type: "team",
			Text: "Keine aktive Saison. Bitte eine Saison aktivieren.",
			Link: "/admin/saisons",
		}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	resp.CurrentSeason = &season
	seasonID := season.ID

	// T2/T3: Action calculation
	actions := h.buildActions(r, userID, role, seasonID)
	resp.Actions = actions

	// T5: Next games
	resp.NextGames = h.queryNextGames(r, userID, seasonID)
	if len(resp.NextGames) > 0 {
		d := resp.NextGames[0].Date
		resp.NextGameDate = &d
	}

	// T6: DutyAccount
	resp.DutyAccount = h.queryDutyAccount(r, userID, role, seasonID, season.Name)

	// T6 (Trainer): TeamStats
	if role == "trainer" {
		resp.TeamStats = h.queryTeamStats(r, userID, seasonID)
	}

	// T7: VehicleInfo
	resp.VehicleInfo = h.queryVehicleInfo(r, userID)

	// CarpoolingHint
	resp.CarpoolingHint = h.queryCarpoolingHint(r, userID, seasonID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) buildActions(r *http.Request, userID int, role string, seasonID int) []Action {
	actions := []Action{}

	switch role {
	case "trainer":
		actions = append(actions, h.trainerDutyActions(r, userID, seasonID)...)
	case "vorstand":
		actions = append(actions, h.vorstandDutyActions(r, seasonID)...)
	case "elternteil", "spieler":
		actions = append(actions, h.memberDutyActions(r, userID, role, seasonID)...)
	}

	// Vehicle action for everyone with team connections
	if va := h.vehicleAction(r, userID, seasonID); va != nil {
		actions = append(actions, *va)
	}

	return actions
}

// T2: Trainer — count open duty slots in their teams this week
func (h *Handler) trainerDutyActions(r *http.Request, userID, seasonID int) []Action {
	var openCount int
	err := h.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*)
		FROM duty_slots ds
		WHERE ds.team_id IN (
			SELECT k.team_id
			FROM kader_trainers kt
			JOIN kader k ON k.id = kt.kader_id
			JOIN members m ON m.id = kt.member_id
			WHERE m.user_id = ?
			  AND k.season_id = ?
			  AND k.team_id IS NOT NULL
		)
		  AND ds.season_id = ?
		  AND DATE(ds.event_date) >= DATE('now')
		  AND DATE(ds.event_date) < DATE('now', '+7 days')
		  AND ds.slots_filled < ds.slots_total`,
		userID, seasonID, seasonID,
	).Scan(&openCount)
	if err != nil || openCount == 0 {
		return nil
	}
	return []Action{{
		ID:   "trainer-duty",
		Type: "duty",
		Text: fmt.Sprintf("%d Dienst(e) diese Woche nicht besetzt — bitte zuweisen", openCount),
		Link: "/dienste",
	}}
}

// T2: Vorstand — count all open duty slots this week
func (h *Handler) vorstandDutyActions(r *http.Request, seasonID int) []Action {
	var openCount int
	err := h.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*)
		FROM duty_slots ds
		WHERE ds.season_id = ?
		  AND DATE(ds.event_date) >= DATE('now')
		  AND DATE(ds.event_date) < DATE('now', '+7 days')
		  AND ds.slots_filled < ds.slots_total`,
		seasonID,
	).Scan(&openCount)
	if err != nil || openCount == 0 {
		return nil
	}
	return []Action{{
		ID:   "vorstand-duty",
		Type: "duty",
		Text: fmt.Sprintf("%d Dienst(e) diese Woche nicht besetzt (alle Mannschaften)", openCount),
		Link: "/dienste",
	}}
}

// T3: Elternteil/Spieler — find applicable open duty slots this week for their teams
func (h *Handler) memberDutyActions(r *http.Request, userID int, role string, seasonID int) []Action {
	teamSubquery := h.teamQueryForUser()
	rows, err := h.db.QueryContext(r.Context(), fmt.Sprintf(`
		SELECT ds.id, dt.name, ds.event_date, COALESCE(ds.event_time, '')
		FROM duty_slots ds
		JOIN duty_types dt ON ds.duty_type_id = dt.id
		LEFT JOIN duty_assignments da ON ds.id = da.duty_slot_id AND da.user_id = ?
		WHERE ds.slots_filled < ds.slots_total
		  AND da.id IS NULL
		  AND dt.target_role = ?
		  AND ds.season_id = ?
		  AND ds.team_id IN (%s)
		  AND DATE(ds.event_date) >= DATE('now')
		  AND DATE(ds.event_date) < DATE('now', '+7 days')
		ORDER BY ds.event_date ASC, ds.event_time ASC`, teamSubquery),
		userID, role, seasonID, userID, seasonID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var actions []Action
	for rows.Next() {
		var slotID int
		var typeName, date, eventTime string
		rows.Scan(&slotID, &typeName, &date, &eventTime)
		actions = append(actions, Action{
			ID:           fmt.Sprintf("duty-%d", slotID),
			Type:         "duty",
			Text:         typeName,
			Link:         "/dienste",
			DueDate:      date,
			EventTime:    eventTime,
			DutyTypeName: typeName,
		})
	}
	return actions
}

// T4: Vehicle action — check for upcoming away games and vehicle_info status
func (h *Handler) vehicleAction(r *http.Request, userID int, seasonID int) *Action {
	teamQuery := h.teamQueryForUser()

	var gameID int
	var gameDate, gameTime, opponent string
	err := h.db.QueryRowContext(r.Context(), fmt.Sprintf(`
		SELECT g.id, g.date, g.time, g.opponent
		FROM games g
		JOIN game_teams gt ON g.id = gt.game_id
		WHERE gt.team_id IN (%s)
		  AND g.is_home = 0
		  AND g.season_id = ?
		  AND DATE(g.date) >= DATE('now')
		  AND DATE(g.date) < DATE('now', '+14 days')
		ORDER BY g.date, g.time ASC
		LIMIT 1`, teamQuery),
		userID, seasonID, seasonID,
	).Scan(&gameID, &gameDate, &gameTime, &opponent)
	if err != nil {
		return nil
	}

	// Check vehicle_info
	var seats int
	hasVehicle := true
	if err2 := h.db.QueryRowContext(r.Context(),
		`SELECT seats FROM vehicle_info WHERE user_id = ?`, userID,
	).Scan(&seats); err2 != nil {
		hasVehicle = false
	}

	var text string
	if !hasVehicle {
		text = fmt.Sprintf("Auswärtsspiel %s vs. %s — Fahrzeuginfo fehlt, bitte eintragen", formatDate(gameDate), opponent)
	} else {
		text = fmt.Sprintf("Auswärtsspiel %s vs. %s — %d Plätze gemeldet", formatDate(gameDate), opponent, seats)
	}

	return &Action{
		ID:           fmt.Sprintf("vehicle-%d", gameID),
		Type:         "vehicle",
		Text:         text,
		Link:         "/profil",
		DueDate:      gameDate,
		ActionNeeded: !hasVehicle,
	}
}

// teamQueryForUser returns a subquery returning team_ids for the user based on actual
// team relationships (kader member, kader trainer, or parent via family_links).
// The subquery uses two positional parameters: userID and seasonID.
func (h *Handler) teamQueryForUser() string {
	return `SELECT uat.team_id FROM user_accessible_teams uat WHERE uat.user_id = ? AND uat.season_id = ?`
}

// T5: Next games for user's teams
func (h *Handler) queryNextGames(r *http.Request, userID int, seasonID int) []Game {
	teamQuery := h.teamQueryForUser()

	rows, err := h.db.QueryContext(r.Context(), fmt.Sprintf(`
		SELECT g.id, g.date, g.time, g.opponent, g.is_home, g.event_type, t.name,
		       COALESCE((SELECT SUM(slots_total) FROM duty_slots WHERE game_id = g.id AND season_id = ?), 0),
		       COALESCE((SELECT SUM(slots_filled) FROM duty_slots WHERE game_id = g.id AND season_id = ?), 0)
		FROM games g
		JOIN game_teams gt ON g.id = gt.game_id
		JOIN teams t ON t.id = gt.team_id
		WHERE gt.team_id IN (%s)
		  AND g.season_id = ?
		  AND DATE(g.date) >= DATE('now')
		GROUP BY g.id, t.id
		ORDER BY g.date, g.time ASC
		LIMIT 3`, teamQuery),
		seasonID, seasonID, userID, seasonID, seasonID,
	)
	if err != nil {
		return []Game{}
	}
	defer rows.Close()

	games := []Game{}
	for rows.Next() {
		var g Game
		var isHome int
		var gameTime string
		rows.Scan(&g.ID, &g.Date, &gameTime, &g.Opponent, &isHome, &g.EventType, &g.Team, &g.SlotsCount, &g.SlotsFilled)
		g.IsHome = isHome == 1
		g.Link = fmt.Sprintf("/spielplan/%d", g.ID)
		games = append(games, g)
	}
	return games
}

// T6: DutyAccount for all roles
func (h *Handler) queryDutyAccount(r *http.Request, userID int, role string, seasonID int, seasonName string) *DutyAccount {
	acc := &DutyAccount{
		Season:            seasonName,
		RecentAssignments: []RecentAssignment{},
	}

	// Count ist
	h.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*)
		FROM duty_assignments da
		JOIN duty_slots ds ON da.duty_slot_id = ds.id
		JOIN duty_types dt ON ds.duty_type_id = dt.id
		WHERE da.user_id = ?
		  AND ds.season_id = ?
		  AND dt.target_role = ?
		  AND da.status IN ('assigned', 'fulfilled', 'cash_substitute')`,
		userID, seasonID, role,
	).Scan(&acc.Ist)

	// Recent assignments (last 5)
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT ds.event_date, dt.name, da.status
		FROM duty_assignments da
		JOIN duty_slots ds ON da.duty_slot_id = ds.id
		JOIN duty_types dt ON ds.duty_type_id = dt.id
		WHERE da.user_id = ?
		  AND ds.season_id = ?
		  AND dt.target_role = ?
		ORDER BY ds.event_date DESC
		LIMIT 5`,
		userID, seasonID, role,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ra RecentAssignment
			rows.Scan(&ra.Date, &ra.DutyType, &ra.Status)
			acc.RecentAssignments = append(acc.RecentAssignments, ra)
		}
	}

	// Soll calculation
	switch role {
	case "elternteil":
		var childCount int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM family_links WHERE parent_user_id = ?`, userID,
		).Scan(&childCount)
		acc.Children = childCount
		avgPerGame, _ := computeAvgSlotsPerGame(r.Context(), h.db)
		soll, _ := computeSollForElternteil(r.Context(), h.db, userID, seasonID, avgPerGame)
		acc.Soll = &soll
	case "spieler":
		soll := 5
		acc.Soll = &soll
	}

	return acc
}

// T6 (Trainer): TeamStats
func (h *Handler) queryTeamStats(r *http.Request, userID, seasonID int) *TeamStats {
	var stats TeamStats
	err := h.db.QueryRowContext(r.Context(), `
		SELECT t.name,
		       COUNT(CASE WHEN m.status = 'aktiv' THEN 1 END),
		       COUNT(*),
		       COUNT(CASE WHEN m.status = 'verletzt' THEN 1 END),
		       COUNT(CASE WHEN m.status = 'pausiert' THEN 1 END)
		FROM kader_trainers kt
		JOIN kader k ON k.id = kt.kader_id
		JOIN members mem_t ON mem_t.id = kt.member_id
		JOIN kader_members km ON km.kader_id = k.id
		JOIN members m ON m.id = km.member_id
		JOIN teams t ON t.id = k.team_id
		WHERE mem_t.user_id = ?
		  AND k.season_id = ?
		  AND k.team_id IS NOT NULL
		GROUP BY t.id
		LIMIT 1`,
		userID, seasonID,
	).Scan(&stats.Team, &stats.ActiveMembers, &stats.TotalMembers, &stats.InjuredCount, &stats.PausedCount)
	if err != nil {
		return nil
	}
	return &stats
}

// T7: VehicleInfo
func (h *Handler) queryVehicleInfo(r *http.Request, userID int) *VehicleInfo {
	var vi VehicleInfo
	var notes sql.NullString
	err := h.db.QueryRowContext(r.Context(),
		`SELECT seats, COALESCE(notes, '') FROM vehicle_info WHERE user_id = ?`, userID,
	).Scan(&vi.Seats, &notes)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return nil
	}
	vi.Notes = notes.String
	vi.UpToDate = true
	return &vi
}

func (h *Handler) queryCarpoolingHint(r *http.Request, userID int, seasonID int) *CarpoolingHint {
	var hint CarpoolingHint
	teamQuery := h.teamQueryForUser()
	err := h.db.QueryRowContext(r.Context(), fmt.Sprintf(`
		SELECT g.id, g.date, g.opponent
		FROM games g
		JOIN game_teams gt ON g.id = gt.game_id
		WHERE gt.team_id IN (%s)
		  AND g.is_home = 0
		  AND g.season_id = ?
		  AND DATE(g.date) >= DATE('now')
		ORDER BY g.date ASC
		LIMIT 1`, teamQuery),
		userID, seasonID, seasonID,
	).Scan(&hint.GameID, &hint.Date, &hint.Opponent)
	if err != nil {
		return nil
	}

	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(CASE WHEN typ='biete' THEN 1 END), COUNT(CASE WHEN typ='suche' THEN 1 END)
		 FROM mitfahrgelegenheiten WHERE game_id = ?`, hint.GameID,
	).Scan(&hint.BieteCount, &hint.SucheCount)

	// MyEntry
	var myEntry CarpoolingMyEntry
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT id, typ FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? LIMIT 1`,
		hint.GameID, userID).Scan(&myEntry.ID, &myEntry.Typ); err == nil {
		hint.MyEntry = &myEntry
	}

	// Confirmed paarungen involving this user
	hint.Paarungen = make([]CarpoolingPaarung, 0)
	pRows, err := h.db.QueryContext(r.Context(), `
		SELECT p.id,
		       CASE WHEN mb.user_id = ? THEN us.first_name || ' ' || us.last_name
		                                ELSE ub.first_name || ' ' || ub.last_name END
		FROM mitfahrt_paarungen p
		JOIN mitfahrgelegenheiten mb ON mb.id = p.biete_id
		JOIN mitfahrgelegenheiten ms ON ms.id = p.suche_id
		JOIN users ub ON ub.id = mb.user_id
		JOIN users us ON us.id = ms.user_id
		WHERE p.status = 'confirmed'
		  AND (mb.user_id = ? OR ms.user_id = ?)
		  AND mb.game_id = ?
		ORDER BY p.updated_at DESC`,
		userID, userID, userID, hint.GameID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var p CarpoolingPaarung
			pRows.Scan(&p.PaarungID, &p.PartnerName)
			hint.Paarungen = append(hint.Paarungen, p)
		}
	}

	// Open entries from other users (no confirmed pairing)
	hint.OpenEntries = make([]CarpoolingOpenEntry, 0)
	oRows, err := h.db.QueryContext(r.Context(), `
		SELECT m.typ, u.first_name || ' ' || u.last_name
		FROM mitfahrgelegenheiten m
		JOIN users u ON u.id = m.user_id
		WHERE m.game_id = ?
		  AND m.user_id != ?
		  AND NOT EXISTS (
		      SELECT 1 FROM mitfahrt_paarungen p
		      WHERE p.status = 'confirmed'
		        AND (p.biete_id = m.id OR p.suche_id = m.id)
		  )
		ORDER BY m.typ, u.first_name
		LIMIT 5`,
		hint.GameID, userID)
	if err == nil {
		defer oRows.Close()
		for oRows.Next() {
			var e CarpoolingOpenEntry
			oRows.Scan(&e.Typ, &e.Name)
			hint.OpenEntries = append(hint.OpenEntries, e)
		}
	}

	return &hint
}

func computeAvgSlotsPerGame(ctx context.Context, db *sql.DB) (float64, error) {
	var heimSlots, auswärtsSlots int
	db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(gti.slots_count), 0)
		FROM game_template_items gti
		JOIN game_templates gt ON gt.id = gti.template_id
		WHERE gt.template_type = 'heim' AND gt.is_active = 1`,
	).Scan(&heimSlots)
	db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(gti.slots_count), 0)
		FROM game_template_items gti
		JOIN game_templates gt ON gt.id = gti.template_id
		WHERE gt.template_type = 'auswärts' AND gt.is_active = 1`,
	).Scan(&auswärtsSlots)
	return float64(heimSlots+auswärtsSlots) / 2.0, nil
}

func computeSollForElternteil(ctx context.Context, db *sql.DB, userID int, seasonID int, avgPerGame float64) (int, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT member_id FROM family_links WHERE parent_user_id = ?`, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total float64
	for rows.Next() {
		var memberID int
		rows.Scan(&memberID)

		var kaderID, gamesPerSeason int
		err := db.QueryRowContext(ctx, `
			SELECT k.id, k.games_per_season
			FROM kader_members km
			JOIN kader k ON k.id = km.kader_id
			WHERE km.member_id = ? AND k.season_id = ?
			LIMIT 1`, memberID, seasonID,
		).Scan(&kaderID, &gamesPerSeason)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return 0, err
		}
		if gamesPerSeason == 0 {
			continue
		}

		var playerCount int
		db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM kader_members WHERE kader_id = ?`, kaderID,
		).Scan(&playerCount)
		if playerCount == 0 {
			continue
		}

		var parentCount int
		db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM family_links WHERE member_id = ?`, memberID,
		).Scan(&parentCount)
		if parentCount == 0 {
			parentCount = 1
		}

		childSoll := float64(gamesPerSeason) * avgPerGame / float64(playerCount) / float64(parentCount)
		total += childSoll
	}
	return int(math.Round(total)), nil
}

func formatDate(date string) string {
	if len(date) >= 10 {
		// "2026-05-30" → "30.05.2026"
		return date[8:10] + "." + date[5:7] + "." + date[0:4]
	}
	return date
}

// effectivePersona returns the single duty/dashboard persona for a user based on
// their club functions and parent status. Priority: trainer > vorstand > spieler > elternteil.
func effectivePersona(clubFunctions []string, isParent bool) string {
	for _, p := range []string{"trainer", "vorstand", "vorstand_beisitzer", "spieler"} {
		if slices.Contains(clubFunctions, p) {
			return p
		}
	}
	if isParent {
		return "elternteil"
	}
	return ""
}
