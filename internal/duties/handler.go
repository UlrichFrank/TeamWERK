package duties

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// eligibleDutyUsers returns user IDs that could be relevant recipients for a duty slot notification:
// members with club function spieler or trainer, plus parents of members with the spieler function.
// Optionally filtered to a specific team (player/trainer/parent must be connected to that team's
// active-season kader).
func (h *Handler) eligibleDutyUsers(teamID *int) []int {
	var (
		rows *sql.Rows
		err  error
	)
	if teamID != nil {
		rows, err = h.db.Query(
			`SELECT DISTINCT u.id FROM users u
			 LEFT JOIN members m ON m.user_id = u.id
			 LEFT JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function IN ('spieler','trainer')
			 LEFT JOIN player_memberships pm ON pm.member_id = m.id
			 LEFT JOIN family_links fl ON fl.parent_user_id = u.id
			 LEFT JOIN members cm ON cm.id = fl.member_id
			 LEFT JOIN member_club_functions cmcf ON cmcf.member_id = cm.id AND cmcf.function = 'spieler'
			 LEFT JOIN player_memberships cpm ON cpm.member_id = cm.id
			 WHERE (mcf.member_id IS NOT NULL OR cmcf.member_id IS NOT NULL)
			   AND (pm.team_id = ? OR cpm.team_id = ?)`, *teamID, *teamID)
	} else {
		rows, err = h.db.Query(
			`SELECT DISTINCT u.id FROM users u
			 LEFT JOIN members m ON m.user_id = u.id
			 LEFT JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function IN ('spieler','trainer')
			 LEFT JOIN family_links fl ON fl.parent_user_id = u.id
			 WHERE mcf.member_id IS NOT NULL OR fl.parent_user_id IS NOT NULL`)
	}
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

// assignedUsers returns user IDs assigned to a duty slot.
func (h *Handler) assignedUsers(slotID string) []int {
	rows, err := h.db.Query(
		`SELECT user_id FROM duty_assignments WHERE duty_slot_id = ?`, slotID)
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

// GET /api/admin/duty-types
func (h *Handler) ListTypes(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, name, hours_value, cash_substitute, default_anchor, default_offset_minutes,
		        same_day_behavior, same_day_variant_id, adjacent_day_behavior, adjacent_day_variant_id, audiences
		 FROM duty_types ORDER BY name`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListTypes query error: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type dt struct {
		ID                   int      `json:"id"`
		Name                 string   `json:"name"`
		HoursValue           float64  `json:"hours_value"`
		CashSubstitute       *float64 `json:"cash_substitute,omitempty"`
		DefaultAnchor        string   `json:"default_anchor"`
		DefaultOffsetMinutes int      `json:"default_offset_minutes"`
		SameDayBehavior      string   `json:"same_day_behavior"`
		SameDayVariantID     *int     `json:"same_day_variant_id,omitempty"`
		AdjacentDayBehavior  string   `json:"adjacent_day_behavior"`
		AdjacentDayVariantID *int     `json:"adjacent_day_variant_id,omitempty"`
		Audiences            []string `json:"audiences,omitempty"`
	}
	result := []dt{}
	for rows.Next() {
		var d dt
		var cs sql.NullFloat64
		var sdvi sql.NullInt64
		var advi sql.NullInt64
		var audiences sql.NullString
		rows.Scan(&d.ID, &d.Name, &d.HoursValue, &cs, &d.DefaultAnchor, &d.DefaultOffsetMinutes,
			&d.SameDayBehavior, &sdvi, &d.AdjacentDayBehavior, &advi, &audiences)
		if cs.Valid {
			d.CashSubstitute = &cs.Float64
		}
		if sdvi.Valid {
			id := int(sdvi.Int64)
			d.SameDayVariantID = &id
		}
		if advi.Valid {
			id := int(advi.Int64)
			d.AdjacentDayVariantID = &id
		}
		d.Audiences = audiencesFromDB(audiences)
		result = append(result, d)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/admin/duty-types
func (h *Handler) CreateType(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                 string   `json:"name"`
		HoursValue           float64  `json:"hours_value"`
		CashSubstitute       *float64 `json:"cash_substitute"`
		DefaultAnchor        string   `json:"default_anchor"`
		DefaultOffsetMinutes int      `json:"default_offset_minutes"`
		SameDayBehavior      string   `json:"same_day_behavior"`
		SameDayVariantID     *int     `json:"same_day_variant_id"`
		AdjacentDayBehavior  string   `json:"adjacent_day_behavior"`
		AdjacentDayVariantID *int     `json:"adjacent_day_variant_id"`
		Audiences            []string `json:"audiences"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.DefaultAnchor == "" {
		req.DefaultAnchor = "start"
	}
	if req.SameDayBehavior == "" {
		req.SameDayBehavior = "normal"
	}
	if req.AdjacentDayBehavior == "" {
		req.AdjacentDayBehavior = "normal"
	}
	if req.SameDayBehavior == "reduced" && req.SameDayVariantID == nil {
		http.Error(w, "same_day_behavior 'reduced' requires same_day_variant_id", http.StatusBadRequest)
		return
	}
	if req.AdjacentDayBehavior == "reduced" && req.AdjacentDayVariantID == nil {
		http.Error(w, "adjacent_day_behavior 'reduced' requires adjacent_day_variant_id", http.StatusBadRequest)
		return
	}
	h.db.ExecContext(r.Context(),
		`INSERT INTO duty_types (name, hours_value, cash_substitute, default_anchor, default_offset_minutes,
		                          same_day_behavior, same_day_variant_id, adjacent_day_behavior, adjacent_day_variant_id, audiences)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		req.Name, req.HoursValue, req.CashSubstitute, req.DefaultAnchor, req.DefaultOffsetMinutes,
		req.SameDayBehavior, req.SameDayVariantID, req.AdjacentDayBehavior, req.AdjacentDayVariantID, audiencesToDB(req.Audiences))
	w.WriteHeader(http.StatusCreated)
}

// PUT /api/admin/duty-types/:id
func (h *Handler) UpdateType(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name                 string   `json:"name"`
		HoursValue           float64  `json:"hours_value"`
		CashSubstitute       *float64 `json:"cash_substitute"`
		DefaultAnchor        string   `json:"default_anchor"`
		DefaultOffsetMinutes int      `json:"default_offset_minutes"`
		SameDayBehavior      string   `json:"same_day_behavior"`
		SameDayVariantID     *int     `json:"same_day_variant_id"`
		AdjacentDayBehavior  string   `json:"adjacent_day_behavior"`
		AdjacentDayVariantID *int     `json:"adjacent_day_variant_id"`
		Audiences            []string `json:"audiences"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.DefaultAnchor == "" {
		req.DefaultAnchor = "start"
	}
	if req.SameDayBehavior == "" {
		req.SameDayBehavior = "normal"
	}
	if req.AdjacentDayBehavior == "" {
		req.AdjacentDayBehavior = "normal"
	}
	if req.SameDayBehavior == "reduced" && req.SameDayVariantID == nil {
		http.Error(w, "same_day_behavior 'reduced' requires same_day_variant_id", http.StatusBadRequest)
		return
	}
	if req.AdjacentDayBehavior == "reduced" && req.AdjacentDayVariantID == nil {
		http.Error(w, "adjacent_day_behavior 'reduced' requires adjacent_day_variant_id", http.StatusBadRequest)
		return
	}
	h.db.ExecContext(r.Context(),
		`UPDATE duty_types SET name=?, hours_value=?, cash_substitute=?, default_anchor=?, default_offset_minutes=?,
		                       same_day_behavior=?, same_day_variant_id=?, adjacent_day_behavior=?, adjacent_day_variant_id=?,
		                       audiences=?
		 WHERE id=?`,
		req.Name, req.HoursValue, req.CashSubstitute, req.DefaultAnchor, req.DefaultOffsetMinutes,
		req.SameDayBehavior, req.SameDayVariantID, req.AdjacentDayBehavior, req.AdjacentDayVariantID,
		audiencesToDB(req.Audiences), id)
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/admin/duty-types/:id
func (h *Handler) DeleteType(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.db.ExecContext(r.Context(), `DELETE FROM duty_types WHERE id=?`, id)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/duty-slots
func (h *Handler) ListSlots(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	p := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions}
	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT ds.id, ds.event_name, ds.event_date, ds.slots_total, ds.slots_filled,
		        dt.name, COALESCE(ds.role_desc,'')
		 FROM duty_slots ds JOIN duty_types dt ON dt.id = ds.duty_type_id
		 ORDER BY ds.event_date DESC`)
	defer rows.Close()
	type slot struct {
		ID          int                 `json:"id"`
		EventName   string              `json:"event_name"`
		EventDate   string              `json:"event_date"`
		SlotsTotal  int                 `json:"slots_total"`
		SlotsFilled int                 `json:"slots_filled"`
		DutyType    string              `json:"duty_type"`
		RoleDesc    string              `json:"role_desc,omitempty"`
		Can         policy.DutyCanFlags `json:"can"`
	}
	dutyCan := policy.DutyCan(p)
	result := []slot{}
	for rows.Next() {
		var s slot
		rows.Scan(&s.ID, &s.EventName, &s.EventDate, &s.SlotsTotal, &s.SlotsFilled, &s.DutyType, &s.RoleDesc)
		s.Can = dutyCan
		result = append(result, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/duty-slots
func (h *Handler) CreateSlot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventName  string   `json:"event_name"`
		EventDate  string   `json:"event_date"`
		EventTime  string   `json:"event_time"`
		DutyTypeID int      `json:"duty_type_id"`
		RoleDesc   string   `json:"role_desc"`
		SlotsTotal int      `json:"slots_total"`
		TeamID     *int     `json:"team_id"`
		SeasonID   int      `json:"season_id"`
		GameID     *int     `json:"game_id"`
		Audiences  []string `json:"audiences"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var eventTime any = nil
	if req.EventTime != "" {
		eventTime = req.EventTime
	}
	h.db.ExecContext(r.Context(),
		`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc, slots_total, team_id, season_id, game_id, audiences, is_custom)
		 VALUES (?,?,?,?,?,?,?,?,?,?,1)`,
		req.EventName, req.EventDate, eventTime, req.DutyTypeID, req.RoleDesc, req.SlotsTotal, req.TeamID, req.SeasonID, req.GameID, audiencesToDB(req.Audiences))
	h.hub.Broadcast("duties")
	notify.Send(h.db, h.cfg, h.eligibleDutyUsers(req.TeamID),
		"duties", "Neuer Dienst verfügbar", req.EventName+" — jetzt eintragen", "/dienste")
	w.WriteHeader(http.StatusCreated)
}

// PUT /api/duty-slots/:id
func (h *Handler) UpdateSlot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		EventName  string   `json:"event_name"`
		EventDate  string   `json:"event_date"`
		EventTime  string   `json:"event_time"`
		RoleDesc   string   `json:"role_desc"`
		SlotsTotal int      `json:"slots_total"`
		Audiences  []string `json:"audiences"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var eventTime any = nil
	if req.EventTime != "" {
		eventTime = req.EventTime
	}
	h.db.ExecContext(r.Context(),
		`UPDATE duty_slots SET event_name=?, event_date=?, event_time=?, role_desc=?, slots_total=?, audiences=?, is_custom=1 WHERE id=?`,
		req.EventName, req.EventDate, eventTime, req.RoleDesc, req.SlotsTotal, audiencesToDB(req.Audiences), id)
	h.hub.Broadcast("duties")
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/duty-slots/:id
func (h *Handler) DeleteSlot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	assigned := h.assignedUsers(id)
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM duty_slots WHERE id=?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.hub.Broadcast("duties")
	if len(assigned) > 0 {
		notify.Send(h.db, h.cfg, assigned, "duties",
			"Dienst abgesagt", "Ein Dienst, für den du eingetragen warst, wurde abgesagt", "/dienste")
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/duty-board — grouped by game, filtered to user's teams
func (h *Handler) Board(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	userID := claims.UserID

	// Audience filter rules:
	//   - System role admin: always bypass (sees all audiences).
	//   - Privileged functions (vorstand, vorstand_beisitzer, trainer, sportliche_leitung):
	//     audience filter is active by default, but ?audience=all disables it.
	//   - Everyone else: audience filter always active; ?audience=all is ignored.
	isPrivileged := claims.HasAnyFunction("vorstand", "vorstand_beisitzer", "trainer", "sportliche_leitung")
	audienceBypass := claims.Role == "admin" || (isPrivileged && r.URL.Query().Get("audience") == "all")

	args := []any{userID} // first ? is for the da LEFT JOIN
	var whereParts string

	if claims.Role == "admin" || claims.HasFunction("vorstand") {
		whereParts = `WHERE ds.season_id = (SELECT id FROM seasons WHERE is_active = 1)`
	} else {
		// Team source = teams the user plays in (or a family member plays in)
		// OR teams the user trains in (via trainer_memberships).
		whereParts = `WHERE (
		     ds.team_id IN (
		         SELECT DISTINCT tm.team_id
		         FROM player_memberships tm
		         JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
		         WHERE tm.member_id IN (
		             SELECT id FROM members WHERE user_id = ?
		             UNION
		             SELECT fl.member_id FROM family_links fl WHERE fl.parent_user_id = ?
		         )
		         UNION
		         SELECT DISTINCT trm.team_id
		         FROM trainer_memberships trm
		         JOIN seasons strn ON strn.id = trm.season_id AND strn.is_active = 1
		         WHERE trm.member_id IN (SELECT id FROM members WHERE user_id = ?)
		     )
		     OR (ds.team_id IS NULL AND ds.game_id IN (
		         SELECT gt.game_id FROM game_teams gt
		         WHERE gt.team_id IN (
		             SELECT DISTINCT tm2.team_id
		             FROM player_memberships tm2
		             JOIN seasons s2 ON s2.id = tm2.season_id AND s2.is_active = 1
		             WHERE tm2.member_id IN (
		                 SELECT id FROM members WHERE user_id = ?
		                 UNION
		                 SELECT fl2.member_id FROM family_links fl2 WHERE fl2.parent_user_id = ?
		             )
		             UNION
		             SELECT DISTINCT trm2.team_id
		             FROM trainer_memberships trm2
		             JOIN seasons strn2 ON strn2.id = trm2.season_id AND strn2.is_active = 1
		             WHERE trm2.member_id IN (SELECT id FROM members WHERE user_id = ?)
		         )
		     ))
		 )
		 AND ds.season_id = (SELECT id FROM seasons WHERE is_active = 1)`
		args = append(args, userID, userID, userID, userID, userID, userID)
	}

	if !audienceBypass {
		// The 'eltern' audience match is team-scoped: a parent only matches
		// when their linked child plays (player_memberships) in the slot's
		// team — or, for game-less slots, in any team participating in the
		// slot's game.
		whereParts += ` AND (
		     COALESCE(ds.audiences, dt.audiences) IS NULL
		     OR (
		         json_valid(COALESCE(ds.audiences, dt.audiences)) AND (
		             (EXISTS (
		                 SELECT 1 FROM json_each(COALESCE(ds.audiences, dt.audiences)) je
		                 WHERE je.value = 'eltern'
		             ) AND EXISTS (
		                 SELECT 1 FROM family_links fl_a
		                 JOIN player_memberships pm_a ON pm_a.member_id = fl_a.member_id
		                 JOIN seasons sa ON sa.id = pm_a.season_id AND sa.is_active = 1
		                 WHERE fl_a.parent_user_id = ?
		                 AND (
		                     pm_a.team_id = ds.team_id
		                     OR (ds.team_id IS NULL AND ds.game_id IS NOT NULL AND pm_a.team_id IN (
		                         SELECT gt_a.team_id FROM game_teams gt_a WHERE gt_a.game_id = ds.game_id
		                     ))
		                 )
		             ))
		             OR EXISTS (
		                 SELECT 1 FROM json_each(COALESCE(ds.audiences, dt.audiences)) je
		                 JOIN member_club_functions mcf_a ON mcf_a.function = je.value
		                 JOIN members m_a ON m_a.id = mcf_a.member_id
		                 WHERE m_a.user_id = ?
		             )
		         )
		     )
		 )`
		args = append(args, userID, userID)
	}

	if r.URL.Query().Get("view") == "mine" {
		whereParts += ` AND EXISTS (SELECT 1 FROM duty_assignments da2 WHERE da2.duty_slot_id = ds.id AND da2.user_id = ?)`
		args = append(args, userID)
	}

	if gameIDStr := r.URL.Query().Get("game_id"); gameIDStr != "" {
		whereParts += ` AND ds.game_id = ?`
		args = append(args, gameIDStr)
	}

	rows, err := h.db.QueryContext(r.Context(), `SELECT
		    ds.id,
		    COALESCE(ds.event_date, '') AS event_date,
		    COALESCE(ds.event_time, '') AS event_time,
		    ds.slots_total,
		    ds.slots_filled,
		    dt.name,
		    COALESCE(ds.role_desc, ''),
		    CASE WHEN da.id IS NOT NULL THEN 1 ELSE 0 END,
		    ds.game_id,
		    COALESCE(g.opponent, ''),
		    COALESCE(g.event_type, ''),
		    COALESCE(g.time, ''),
		    COALESCE(ds.team_id, 0),
		    COALESCE(`+appdb.TeamDisplayShort("t")+`, t.name, ''),
		    CASE WHEN ds.event_date < date('now') THEN 1 ELSE 0 END,
		    COALESCE(ds.audiences, dt.audiences),
		    COALESCE(ds.event_name, '')
		 FROM duty_slots ds
		 JOIN duty_types dt ON dt.id = ds.duty_type_id
		 LEFT JOIN duty_assignments da ON da.duty_slot_id = ds.id AND da.user_id = ?
		 LEFT JOIN games g ON g.id = ds.game_id
		 LEFT JOIN teams t ON t.id = ds.team_id
		 `+whereParts+`
		 ORDER BY ds.event_date, COALESCE(ds.event_time, ''), ds.id`, args...)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type publicAssignee struct {
		UserID   int     `json:"user_id"`
		Name     string  `json:"name"`
		PhotoURL *string `json:"photo_url,omitempty"`
	}
	bp := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions}
	boardDutyCan := policy.DutyCan(bp)

	type boardSlot struct {
		ID          int                 `json:"id"`
		DutyType    string              `json:"duty_type"`
		EventTime   string              `json:"event_time,omitempty"`
		SlotsTotal  int                 `json:"slots_total"`
		Vacancies   int                 `json:"vacancies"`
		ClaimedByMe bool                `json:"claimed_by_me"`
		RoleDesc    string              `json:"role_desc,omitempty"`
		Audiences   []string            `json:"audiences,omitempty"`
		Assignees   []publicAssignee    `json:"assignees"`
		Can         policy.DutyCanFlags `json:"can"`
	}
	type boardGroup struct {
		GameID    *int        `json:"game_id"`
		TeamID    *int        `json:"team_id,omitempty"`
		Date      string      `json:"date,omitempty"`
		EventTime string      `json:"event_time,omitempty"`
		Opponent  string      `json:"opponent,omitempty"`
		EventType string      `json:"event_type,omitempty"`
		TeamName  string      `json:"team_name"`
		Label     string      `json:"label,omitempty"`
		Past      bool        `json:"past"`
		Slots     []boardSlot `json:"slots"`
	}

	groupOrder := []string{}
	groupMap := map[string]*boardGroup{}

	for rows.Next() {
		var slotID, slotsTotal, slotsFilled, claimedInt, teamID, isPastInt int
		var eventDate, eventTime, dutyType, roleDesc, opponent, eventType, gameTime, teamName, eventName string
		var gameID sql.NullInt64
		var audiences sql.NullString
		rows.Scan(&slotID, &eventDate, &eventTime, &slotsTotal, &slotsFilled,
			&dutyType, &roleDesc, &claimedInt, &gameID, &opponent, &eventType, &gameTime,
			&teamID, &teamName, &isPastInt, &audiences, &eventName)

		var key string
		if gameID.Valid {
			key = fmt.Sprintf("game-%d", gameID.Int64)
		} else {
			key = fmt.Sprintf("other-%d-%s", teamID, eventDate)
		}

		if _, ok := groupMap[key]; !ok {
			g := &boardGroup{TeamName: teamName, Slots: []boardSlot{}, Past: isPastInt == 1}
			if teamID > 0 {
				tid := teamID
				g.TeamID = &tid
			}
			if gameID.Valid {
				id := int(gameID.Int64)
				g.GameID = &id
				g.Date = eventDate
				g.EventTime = gameTime
				g.Opponent = opponent
				g.EventType = eventType
			} else {
				g.Date = eventDate
				g.EventType = "generisch"
				if eventName != "" {
					g.Label = eventName
				} else {
					g.Label = "Sonstige Dienste"
				}
			}
			groupMap[key] = g
			groupOrder = append(groupOrder, key)
		}
		grp := groupMap[key]
		if !gameID.Valid && isPastInt == 0 {
			grp.Past = false
		}
		grp.Slots = append(grp.Slots, boardSlot{
			ID:          slotID,
			DutyType:    dutyType,
			EventTime:   eventTime,
			SlotsTotal:  slotsTotal,
			Vacancies:   slotsTotal - slotsFilled,
			ClaimedByMe: claimedInt == 1,
			RoleDesc:    roleDesc,
			Audiences:   audiencesFromDB(audiences),
			Assignees:   []publicAssignee{},
			Can:         boardDutyCan,
		})
	}

	// Fetch assignees for all slots with privacy filtering
	var slotIDs []int
	for _, grp := range groupMap {
		for _, s := range grp.Slots {
			slotIDs = append(slotIDs, s.ID)
		}
	}
	if len(slotIDs) > 0 {
		ph := make([]string, len(slotIDs))
		aArgs := make([]any, len(slotIDs))
		for i, id := range slotIDs {
			ph[i] = "?"
			aArgs[i] = id
		}
		aRows, aErr := h.db.QueryContext(r.Context(), `
			SELECT da.duty_slot_id,
			       u.id,
			       u.first_name || ' ' || u.last_name,
			       CASE WHEN COALESCE(uv.photo_visible,0)=1 AND COALESCE(u.photo_path,'') != '' THEN '/api/uploads/' || u.photo_path END
			FROM duty_assignments da
			JOIN users u ON u.id = da.user_id
			LEFT JOIN user_visibility uv ON uv.user_id = u.id
			WHERE da.duty_slot_id IN (`+strings.Join(ph, ",")+`)
			ORDER BY da.created_at`, aArgs...)
		if aErr == nil {
			defer aRows.Close()
			assigneeMap := map[int][]publicAssignee{}
			for aRows.Next() {
				var slotID, userID int
				var name string
				var photoURL sql.NullString
				aRows.Scan(&slotID, &userID, &name, &photoURL)
				a := publicAssignee{UserID: userID, Name: name}
				if photoURL.Valid && photoURL.String != "" {
					a.PhotoURL = &photoURL.String
				}
				assigneeMap[slotID] = append(assigneeMap[slotID], a)
			}
			for _, grp := range groupMap {
				for i := range grp.Slots {
					if assignees, ok := assigneeMap[grp.Slots[i].ID]; ok {
						grp.Slots[i].Assignees = assignees
					}
				}
			}
		}
	}

	result := make([]*boardGroup, 0, len(groupOrder))
	for _, k := range groupOrder {
		result = append(result, groupMap[k])
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DELETE /api/duty-board/:slotId/claim
func (h *Handler) Unclaim(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slotId")
	claims := auth.ClaimsFromCtx(r.Context())
	var status string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT status FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, claims.UserID).Scan(&status)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if status == "fulfilled" {
		http.Error(w, "already fulfilled", http.StatusConflict)
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(),
		`DELETE FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, claims.UserID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE duty_slots SET slots_filled = slots_filled - 1 WHERE id=?`, slotID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/duty-board/:slotId/claim
func (h *Handler) Claim(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slotId")
	claims := auth.ClaimsFromCtx(r.Context())

	var req struct {
		UserID *int `json:"user_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	targetUserID := claims.UserID
	if req.UserID != nil {
		targetUserID = *req.UserID
	}

	if targetUserID != claims.UserID {
		// Verify the target is a proxy child linked to the logged-in user
		var allowed bool
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*)>0
			 FROM family_links fl
			 JOIN members m ON m.id = fl.member_id
			 JOIN users u ON u.id = m.user_id
			 WHERE fl.parent_user_id = ? AND u.id = ? AND u.can_login = 0`,
			claims.UserID, targetUserID,
		).Scan(&allowed)
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	// Atomically increment slots_filled only if capacity remains. This prevents
	// concurrent over-claim without a transaction (SQLite serializes writes).
	res, err := h.db.ExecContext(r.Context(),
		`UPDATE duty_slots SET slots_filled = slots_filled + 1
		 WHERE id = ? AND slots_filled < slots_total`, slotID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "slot full or not found", http.StatusConflict)
		return
	}
	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO duty_assignments (duty_slot_id, user_id) VALUES (?,?)`, slotID, targetUserID)
	if err != nil {
		// Rollback the counter increment if assignment insert fails (e.g. duplicate).
		h.db.ExecContext(r.Context(),
			`UPDATE duty_slots SET slots_filled = slots_filled - 1 WHERE id=?`, slotID)
		http.Error(w, "already claimed", http.StatusConflict)
		return
	}
	// Ensure a duty_accounts row exists for the target user in the active season
	h.db.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO duty_accounts (user_id, season_id, soll, ist)
		 SELECT ?, id, 0, 0 FROM seasons WHERE is_active = 1`,
		targetUserID)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/duty-slots/:id/assignments
func (h *Handler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("id")
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT da.id, u.first_name || ' ' || u.last_name, da.status, COALESCE(da.cash_amount,0)
		 FROM duty_assignments da JOIN users u ON u.id = da.user_id
		 WHERE da.duty_slot_id=? ORDER BY da.created_at`, slotID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type assignment struct {
		ID         int     `json:"id"`
		UserName   string  `json:"user_name"`
		Status     string  `json:"status"`
		CashAmount float64 `json:"cash_amount,omitempty"`
	}
	result := []assignment{}
	for rows.Next() {
		var a assignment
		rows.Scan(&a.ID, &a.UserName, &a.Status, &a.CashAmount)
		result = append(result, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/duty-assignments/:id/fulfill
func (h *Handler) Fulfill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.db.ExecContext(r.Context(),
		`UPDATE duty_assignments SET status='fulfilled', fulfilled_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	h.hub.Broadcast("duties")
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/duty-assignments/:id/cash-substitute
func (h *Handler) CashSubstitute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Amount float64 `json:"amount"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(),
		`UPDATE duty_assignments SET status='cash_substitute', cash_amount=?, fulfilled_at=CURRENT_TIMESTAMP WHERE id=?`,
		req.Amount, id)
	h.hub.Broadcast("duties")
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/duty-accounts
func (h *Handler) Accounts(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var rows *sql.Rows
	if claims.Role == "admin" {
		rows, _ = h.db.QueryContext(r.Context(),
			`SELECT da.user_id, u.first_name || ' ' || u.last_name, da.season_id, da.soll, da.ist
			 FROM duty_accounts da JOIN users u ON u.id = da.user_id
			 ORDER BY u.last_name, u.first_name`)
	} else {
		rows, _ = h.db.QueryContext(r.Context(),
			`SELECT da.user_id, u.first_name || ' ' || u.last_name, da.season_id, da.soll, da.ist
			 FROM duty_accounts da JOIN users u ON u.id = da.user_id
			 WHERE da.user_id=?`, claims.UserID)
	}
	defer rows.Close()
	type account struct {
		UserID   int     `json:"user_id"`
		Name     string  `json:"name"`
		SeasonID int     `json:"season_id"`
		Soll     float64 `json:"soll"`
		Ist      float64 `json:"ist"`
		Balance  float64 `json:"balance"`
	}
	result := []account{}
	for rows.Next() {
		var a account
		rows.Scan(&a.UserID, &a.Name, &a.SeasonID, &a.Soll, &a.Ist)
		a.Balance = a.Soll - a.Ist
		result = append(result, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/admin/duty-accounts/export
func (h *Handler) ExportAccounts(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT u.first_name || ' ' || u.last_name, da.soll, da.ist, da.soll - da.ist as balance,
		        COALESCE(SUM(CASE WHEN dassign.status='cash_substitute' THEN dassign.cash_amount ELSE 0 END), 0)
		 FROM duty_accounts da
		 JOIN users u ON u.id = da.user_id
		 LEFT JOIN duty_assignments dassign ON dassign.user_id = da.user_id
		 GROUP BY da.user_id, da.season_id
		 ORDER BY u.last_name, u.first_name`)
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="dienstkonten.csv"`)
	cw := csv.NewWriter(w)
	cw.Write([]string{"Name", "Soll", "Ist", "Saldo", "Geldersatz"})
	for rows.Next() {
		var name string
		var soll, ist, balance, cash float64
		rows.Scan(&name, &soll, &ist, &balance, &cash)
		cw.Write([]string{name,
			fmtFloat(soll), fmtFloat(ist), fmtFloat(balance), fmtFloat(cash)})
	}
	cw.Flush()
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

// PUT /api/admin/seasons/:id/duty-targets
func (h *Handler) SetSeasonTargets(w http.ResponseWriter, r *http.Request) {
	seasonID := r.PathValue("id")
	var req []struct {
		DutyTypeID  int     `json:"duty_type_id"`
		TargetHours float64 `json:"target_hours"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	for _, t := range req {
		h.db.ExecContext(r.Context(),
			`INSERT INTO duty_season_targets (season_id, duty_type_id, target_hours) VALUES (?,?,?)
			 ON CONFLICT(season_id, duty_type_id) DO UPDATE SET target_hours=excluded.target_hours`,
			seasonID, t.DutyTypeID, t.TargetHours)
	}
	w.WriteHeader(http.StatusNoContent)
}

func fmtFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
