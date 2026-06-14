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
	appdb "github.com/teamstuttgart/teamwerk/internal/db"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

type Season struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"isActive"`
}

type NextEvent struct {
	ID         int    `json:"id"`
	EventType  string `json:"eventType"` // "training" | "spiel"
	Date       string `json:"date"`
	Time       string `json:"time"`
	Title      string `json:"title"`
	TeamName   string `json:"teamName"`
	DetailURL  string `json:"detailUrl"`
	IsHome     *bool  `json:"isHome"` // nil for training, true/false for games
	IsExtended bool   `json:"isExtended"`
}

type DiensteSlot struct {
	DutyTypeName string `json:"dutyTypeName"`
	EventTime    string `json:"eventTime"`
}

type NextDiensteGame struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Opponent string `json:"opponent"`
}

type MeineDienste struct {
	NextGame       *NextDiensteGame `json:"nextGame"`
	MySlots        []DiensteSlot    `json:"mySlots"`
	OpenSlotsCount int              `json:"openSlotsCount"`
	DutyAccount    *DutyAccount     `json:"dutyAccount"`
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

type CarpoolingPaarung struct {
	PaarungID   int    `json:"paarungId"`
	PartnerName string `json:"partnerName"`
}

type CarpoolingConfirmed struct {
	GameID    int                 `json:"gameId"`
	Date      string              `json:"date"`
	Opponent  string              `json:"opponent"`
	Paarungen []CarpoolingPaarung `json:"paarungen"`
}

type Response struct {
	CurrentSeason       *Season               `json:"currentSeason"`
	MeineTermine        []NextEvent           `json:"meineTermine"`
	MeineDienste        *MeineDienste         `json:"meineDienste"`
	CarpoolingConfirmed []CarpoolingConfirmed `json:"carpoolingConfirmed"`
}

// GET /api/dashboard
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)
	userID := claims.UserID
	role := effectivePersona(claims.ClubFunctions, claims.IsParent)

	resp := Response{
		MeineTermine:        []NextEvent{},
		CarpoolingConfirmed: []CarpoolingConfirmed{},
	}

	var season Season
	err := h.db.QueryRowContext(ctx,
		`SELECT id, name, is_active FROM seasons WHERE is_active = 1 LIMIT 1`,
	).Scan(&season.ID, &season.Name, &season.IsActive)
	if err == sql.ErrNoRows {
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

	resp.MeineTermine = h.queryNextEvents(r, userID, seasonID)
	resp.MeineDienste = h.queryMeineDienste(r, userID, role, seasonID, season.Name)
	resp.CarpoolingConfirmed = h.queryCarpoolingConfirmed(r, userID, seasonID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// teamQueryForUser returns a subquery returning team_ids for the user.
// Uses two positional parameters: userID and seasonID.
func (h *Handler) teamQueryForUser() string {
	return `SELECT uat.team_id FROM user_accessible_teams uat WHERE uat.user_id = ? AND uat.season_id = ?`
}

// queryNextEvents returns all events on the next day that has at least one event.
// Combines training_sessions and games for the user's teams.
// isExtended is true when the user's access to the event team is via kader_extended_members only.
func (h *Handler) queryNextEvents(r *http.Request, userID int, seasonID int) []NextEvent {
	teamSubquery := h.teamQueryForUser()

	rows, err := h.db.QueryContext(r.Context(), fmt.Sprintf(`
		WITH extended_teams AS (
			SELECT k.team_id
			FROM kader_extended_members kem
			JOIN kader k ON k.id = kem.kader_id
			JOIN members m ON m.id = kem.member_id
			JOIN seasons s ON s.id = k.season_id
			WHERE m.user_id = ? AND s.is_active = 1
		),
		primary_teams AS (
			SELECT k.team_id
			FROM kader_members km
			JOIN kader k ON k.id = km.kader_id
			JOIN members m ON m.id = km.member_id
			JOIN seasons s ON s.id = k.season_id
			WHERE m.user_id = ? AND s.is_active = 1
		),
		upcoming AS (
			SELECT ts.id AS event_id,
			       'training' AS event_type,
			       ts.date,
			       ts.start_time AS time,
			       CASE WHEN ts.title != '' THEN ts.title ELSE 'Training' END AS title,
			       COALESCE(`+appdb.TeamDisplayName("t")+`, t.name) AS team_name,
			       '/termine/training/' || ts.id AS detail_url,
			       NULL AS is_home,
			       CASE WHEN ts.team_id IN (SELECT team_id FROM extended_teams) THEN 1 ELSE 0 END AS is_extended
			FROM training_sessions ts
			JOIN teams t ON t.id = ts.team_id
			WHERE ts.team_id IN (%s)
			  AND ts.season_id = ?
			  AND ts.status = 'active'
			  AND DATE(ts.date) >= DATE('now')
			UNION ALL
			SELECT g.id,
			       'spiel',
			       g.date,
			       g.time,
			       CASE WHEN g.event_type = 'generisch' THEN g.opponent ELSE 'vs. ' || g.opponent END,
			       MIN(COALESCE(`+appdb.TeamDisplayName("t")+`, t.name)),
			       '/termine/spiel/' || g.id,
			       g.is_home,
			       CASE WHEN EXISTS (
			           SELECT 1 FROM game_teams gt2
			           WHERE gt2.game_id = g.id
			             AND gt2.team_id IN (SELECT team_id FROM extended_teams)
			       ) AND NOT EXISTS (
			           SELECT 1 FROM game_teams gt3
			           WHERE gt3.game_id = g.id
			             AND gt3.team_id IN (SELECT team_id FROM primary_teams)
			       ) THEN 1 ELSE 0 END
			FROM games g
			JOIN game_teams gt ON g.id = gt.game_id
			JOIN teams t ON t.id = gt.team_id
			WHERE gt.team_id IN (%s)
			  AND g.season_id = ?
			  AND DATE(g.date) >= DATE('now')
			GROUP BY g.id
		),
		min_date AS (SELECT MIN(date) AS d FROM upcoming)
		SELECT event_id, event_type, date, time, title, team_name, detail_url, is_home, is_extended
		FROM upcoming
		WHERE date = (SELECT d FROM min_date)
		ORDER BY time ASC`, teamSubquery, teamSubquery),
		userID, userID, userID, seasonID, seasonID, userID, seasonID, seasonID,
	)
	if err != nil {
		return []NextEvent{}
	}
	defer rows.Close()

	events := []NextEvent{}
	for rows.Next() {
		var e NextEvent
		var isHome sql.NullInt64
		var isExtended int
		rows.Scan(&e.ID, &e.EventType, &e.Date, &e.Time, &e.Title, &e.TeamName, &e.DetailURL, &isHome, &isExtended)
		if isHome.Valid {
			v := isHome.Int64 == 1
			e.IsHome = &v
		}
		e.IsExtended = isExtended == 1
		events = append(events, e)
	}
	return events
}

// queryMeineDienste finds the next game with duty slots and returns user's own
// assignments (or open slot count) plus the season duty account.
func (h *Handler) queryMeineDienste(r *http.Request, userID int, role string, seasonID int, seasonName string) *MeineDienste {
	result := &MeineDienste{
		MySlots:     []DiensteSlot{},
		DutyAccount: h.queryDutyAccount(r, userID, role, seasonID, seasonName),
	}

	teamSubquery := h.teamQueryForUser()
	var game NextDiensteGame
	err := h.db.QueryRowContext(r.Context(), fmt.Sprintf(`
		SELECT g.id, g.date, g.opponent
		FROM games g
		JOIN game_teams gt ON g.id = gt.game_id
		WHERE gt.team_id IN (%s)
		  AND g.season_id = ?
		  AND DATE(g.date) >= DATE('now')
		  AND EXISTS (SELECT 1 FROM duty_slots ds WHERE ds.game_id = g.id AND ds.season_id = ?)
		GROUP BY g.id
		ORDER BY g.date ASC, g.time ASC
		LIMIT 1`, teamSubquery),
		userID, seasonID, seasonID, seasonID,
	).Scan(&game.ID, &game.Date, &game.Opponent)
	if err != nil {
		return result
	}
	result.NextGame = &game

	// User's own assignments for this game
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT dt.name, COALESCE(ds.event_time, '')
		FROM duty_assignments da
		JOIN duty_slots ds ON da.duty_slot_id = ds.id
		JOIN duty_types dt ON ds.duty_type_id = dt.id
		WHERE da.user_id = ?
		  AND ds.game_id = ?
		  AND da.status IN ('assigned','fulfilled','cash_substitute')
		ORDER BY ds.event_time ASC`, userID, game.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var slot DiensteSlot
			rows.Scan(&slot.DutyTypeName, &slot.EventTime)
			result.MySlots = append(result.MySlots, slot)
		}
	}

	if len(result.MySlots) == 0 {
		h.db.QueryRowContext(r.Context(), `
			SELECT COALESCE(SUM(slots_total - slots_filled), 0)
			FROM duty_slots
			WHERE game_id = ? AND season_id = ? AND slots_filled < slots_total`,
			game.ID, seasonID,
		).Scan(&result.OpenSlotsCount)
	}

	return result
}

// queryDutyAccount returns the duty account for a user in the active season.
func (h *Handler) queryDutyAccount(r *http.Request, userID int, role string, seasonID int, seasonName string) *DutyAccount {
	acc := &DutyAccount{
		Season:            seasonName,
		RecentAssignments: []RecentAssignment{},
	}

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

// queryCarpoolingConfirmed returns confirmed pairings for the next max. 3 away games.
func (h *Handler) queryCarpoolingConfirmed(r *http.Request, userID int, seasonID int) []CarpoolingConfirmed {
	teamQuery := h.teamQueryForUser()

	rows, err := h.db.QueryContext(r.Context(), fmt.Sprintf(`
		SELECT g.id, g.date, g.opponent
		FROM games g
		JOIN game_teams gt ON g.id = gt.game_id
		WHERE gt.team_id IN (%s)
		  AND g.is_home = 0
		  AND g.season_id = ?
		  AND DATE(g.date) >= DATE('now')
		GROUP BY g.id
		ORDER BY g.date ASC
		LIMIT 3`, teamQuery),
		userID, seasonID, seasonID,
	)
	if err != nil {
		return []CarpoolingConfirmed{}
	}
	defer rows.Close()

	var games []CarpoolingConfirmed
	for rows.Next() {
		var g CarpoolingConfirmed
		g.Paarungen = []CarpoolingPaarung{}
		rows.Scan(&g.GameID, &g.Date, &g.Opponent)
		games = append(games, g)
	}

	for i, g := range games {
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
			userID, userID, userID, g.GameID)
		if err != nil {
			continue
		}
		for pRows.Next() {
			var p CarpoolingPaarung
			pRows.Scan(&p.PaarungID, &p.PartnerName)
			games[i].Paarungen = append(games[i].Paarungen, p)
		}
		pRows.Close()
	}

	if games == nil {
		return []CarpoolingConfirmed{}
	}
	return games
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
