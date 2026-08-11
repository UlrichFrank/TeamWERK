package games

// Massen-Regeneration der Dienst-Slots über einen wählbaren Zeitraum. Liegt im
// games-Package (nicht in duties), weil runAutoRegen unexportiert ist — dieselbe
// Begründung wie h4aimport_handler.go. Siehe openspec/changes/duty-bulk-regen/design.md.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// --- Request/Response-Typen (design.md §9) --------------------------------------

type bulkRegenAction struct {
	Action     string `json:"action"` // "template" | "none" | "purge"
	TemplateID *int   `json:"template_id,omitempty"`
}

type bulkRegenOverride struct {
	GameID     int    `json:"game_id"`
	Action     string `json:"action"`
	TemplateID *int   `json:"template_id,omitempty"`
}

type bulkRegenRequest struct {
	From            string                     `json:"from"`
	To              string                     `json:"to"`
	Defaults        map[string]bulkRegenAction `json:"defaults"`
	Overrides       []bulkRegenOverride        `json:"overrides"`
	ExcludedGameIDs []int                      `json:"excluded_game_ids"`
	Notify          *bool                      `json:"notify"`
}

type bulkRegenRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type bulkRegenSlotCount struct {
	Auto   int `json:"auto"`
	Custom int `json:"custom"`
}

type bulkRegenRow struct {
	GameID              int                `json:"game_id"`
	Date                string             `json:"date"`
	Time                string             `json:"time"`
	EventName           string             `json:"event_name"`
	EventType           string             `json:"event_type"`
	CurrentTemplateID   *int               `json:"current_template_id,omitempty"`
	EffectiveAction     string             `json:"effective_action"`
	EffectiveTemplateID *int               `json:"effective_template_id,omitempty"`
	Excluded            bool               `json:"excluded"`
	SlotsBefore         bulkRegenSlotCount `json:"slots_before"`
	SlotsAfter          bulkRegenSlotCount `json:"slots_after"`
	Created             int                `json:"created"`
	DeletedAuto         int                `json:"deleted_auto"`
	DeletedCustom       int                `json:"deleted_custom"`
	AssignmentsKept     int                `json:"assignments_kept"`
	AssignmentsLost     int                `json:"assignments_lost"`
	Conflicts           int                `json:"conflicts"`
}

type bulkRegenTotals struct {
	Games           int `json:"games"`
	Created         int `json:"created"`
	Deleted         int `json:"deleted"`
	CustomKept      int `json:"custom_kept"`
	CustomDeleted   int `json:"custom_deleted"`
	AssignmentsKept int `json:"assignments_kept"`
	AssignmentsLost int `json:"assignments_lost"`
	Conflicts       int `json:"conflicts"`
	NotifiedUsers   int `json:"notified_users"`
}

type bulkRegenResponse struct {
	Range    bulkRegenRange  `json:"range"`
	Rows     []bulkRegenRow  `json:"rows"`
	Totals   bulkRegenTotals `json:"totals"`
	Warnings []string        `json:"warnings"`
	Applied  bool            `json:"applied,omitempty"`
}

// bulkRegenAPIError carries the HTTP status and machine-readable error code for the
// handful of validated failure modes (design.md §6/§9); everything else is a plain 500.
type bulkRegenAPIError struct {
	status int
	code   string
}

func (e *bulkRegenAPIError) Error() string { return e.code }

func bulkRegenErr(status int, code string) *bulkRegenAPIError {
	return &bulkRegenAPIError{status: status, code: code}
}

// --- HTTP entry points ------------------------------------------------------------

// POST /api/duty-slots/bulk-regen/preview
// Vorstand-Tier (RequireClubFunction admin-Bypass). Vollständiger Dry-Run derselben
// Transaktion wie Apply, abgeschlossen mit Rollback — siehe design.md §3.
func (h *Handler) PreviewBulkRegen(w http.ResponseWriter, r *http.Request) {
	resp, _, _, apiErr := h.runBulkRegen(r.Context(), r, false)
	if apiErr != nil {
		writeJSON(w, apiErr.status, map[string]string{"error": apiErr.code})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// POST /api/duty-slots/bulk-regen/apply
// Identischer Plan wie Preview, aber committet und broadcastet genau einmal
// ("duties"+"games"), danach dispatchRegenNotifications sofern notify != false.
func (h *Handler) ApplyBulkRegen(w http.ResponseWriter, r *http.Request) {
	resp, summary, notify, apiErr := h.runBulkRegen(r.Context(), r, true)
	if apiErr != nil {
		writeJSON(w, apiErr.status, map[string]string{"error": apiErr.code})
		return
	}
	resp.Applied = true
	if h.hub != nil {
		h.hub.Broadcast("duties")
		h.hub.Broadcast("games")
	}
	if notify {
		h.dispatchRegenNotifications(summary)
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Shared plan/apply core ---------------------------------------------------------

// runBulkRegen builds and executes the bulk-regen plan inside a single transaction,
// committing when apply=true and rolling back otherwise (preview). Returns the API
// response, the RegenSummary (for dispatchRegenNotifications after commit), whether the
// caller requested notifications, and a typed API error for the validated failure modes.
func (h *Handler) runBulkRegen(ctx context.Context, r *http.Request, apply bool) (bulkRegenResponse, RegenSummary, bool, *bulkRegenAPIError) {
	var req bulkRegenRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // fehlende/leere Body → Zero-Value-Request (Default-Range)
	}
	notify := req.Notify == nil || *req.Notify

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusInternalServerError, "internal_error")
	}
	defer tx.Rollback()

	var seasonID int
	if err := tx.QueryRowContext(ctx, `SELECT id FROM seasons WHERE is_active=1 LIMIT 1`).Scan(&seasonID); err != nil {
		return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusBadRequest, "no_active_season")
	}

	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	from := req.From
	if from == "" {
		from = tomorrow
	}
	if _, err := time.Parse("2006-01-02", from); err != nil {
		return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusBadRequest, "invalid_range")
	}
	if from <= today {
		return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusBadRequest, "range_in_past")
	}

	to := req.To
	if to == "" {
		var maxDate sql.NullString
		tx.QueryRowContext(ctx, `SELECT MAX(date) FROM games WHERE season_id=?`, seasonID).Scan(&maxDate)
		if maxDate.Valid && maxDate.String > from {
			to = maxDate.String
		} else {
			to = from
		}
	} else if _, err := time.Parse("2006-01-02", to); err != nil {
		return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusBadRequest, "invalid_range")
	}

	overridesByGame := map[int]bulkRegenOverride{}
	for _, ov := range req.Overrides {
		overridesByGame[ov.GameID] = ov
	}
	excluded := map[int]bool{}
	for _, id := range req.ExcludedGameIDs {
		excluded[id] = true
	}

	games, err := h.loadBulkRangeGames(ctx, tx, seasonID, from, to)
	if err != nil {
		return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusInternalServerError, "internal_error")
	}

	type plannedGame struct {
		g                bulkGame
		action           string
		templateID       *int
		excluded         bool
		before           bulkRegenSlotCount
		deletedCustomNow int
	}
	planned := make([]plannedGame, 0, len(games))

	for _, g := range games {
		action, templateID := resolveBulkAction(g, overridesByGame, req.Defaults)
		if action != "template" && action != "none" && action != "purge" {
			return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusBadRequest, "invalid_action")
		}
		if action == "template" {
			if templateID == nil || !existsID(ctx, tx, "game_templates", *templateID) {
				return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusBadRequest, "invalid_template")
			}
		}
		autoBefore, customBefore, err := countGameSlots(ctx, tx, g.ID)
		if err != nil {
			return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusInternalServerError, "internal_error")
		}
		planned = append(planned, plannedGame{
			g: g, action: action, templateID: templateID, excluded: excluded[g.ID],
			before: bulkRegenSlotCount{Auto: autoBefore, Custom: customBefore},
		})
	}

	// Mutations: template_id per included game, plus a full duty_slots purge for
	// "purge" games (both is_custom values — the regen engine only ever deletes
	// is_custom=0 itself, see design.md's transaction sketch).
	dateSet := map[string]bool{}
	skip := map[int]bool{}
	for i := range planned {
		pg := &planned[i]
		dateSet[pg.g.Date] = true
		if pg.excluded {
			skip[pg.g.ID] = true
			continue
		}
		var newTemplate any
		if pg.action == "template" {
			newTemplate = *pg.templateID
		}
		if _, err := tx.ExecContext(ctx, `UPDATE games SET template_id=? WHERE id=?`, newTemplate, pg.g.ID); err != nil {
			return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusInternalServerError, "internal_error")
		}
		if pg.action == "purge" {
			pg.deletedCustomNow = pg.before.Custom
			if _, err := tx.ExecContext(ctx, `DELETE FROM duty_slots WHERE game_id=?`, pg.g.ID); err != nil {
				return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusInternalServerError, "internal_error")
			}
		}
	}

	dates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	summary, err := h.runAutoRegen(ctx, tx, dates, seasonID, skip)
	if err != nil {
		return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusInternalServerError, "internal_error")
	}

	perGame := map[int]GameDelta{}
	for _, pg := range summary.PerGame {
		perGame[pg.GameID] = pg
	}

	rows := make([]bulkRegenRow, 0, len(planned))
	totals := bulkRegenTotals{}
	notifiedSeen := map[int]bool{}
	for _, pg := range planned {
		row := bulkRegenRow{
			GameID:              pg.g.ID,
			Date:                pg.g.Date,
			Time:                pg.g.Time,
			EventName:           composeEventName(pg.g.EventType, pg.g.IsHome, pg.g.Opponent),
			EventType:           pg.g.EventType,
			EffectiveAction:     pg.action,
			EffectiveTemplateID: pg.templateID,
			Excluded:            pg.excluded,
			SlotsBefore:         pg.before,
			DeletedCustom:       pg.deletedCustomNow,
		}
		if pg.g.TemplateID.Valid {
			id := int(pg.g.TemplateID.Int64)
			row.CurrentTemplateID = &id
		}
		if pg.excluded {
			row.SlotsAfter = pg.before
		} else {
			autoAfter, customAfter, err := countGameSlots(ctx, tx, pg.g.ID)
			if err != nil {
				return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusInternalServerError, "internal_error")
			}
			row.SlotsAfter = bulkRegenSlotCount{Auto: autoAfter, Custom: customAfter}
			if delta, ok := perGame[pg.g.ID]; ok {
				row.Created = delta.Created
				row.DeletedAuto = delta.DeletedAuto
				row.AssignmentsKept = delta.AssignmentsKept
				row.AssignmentsLost = delta.AssignmentsLost
				row.Conflicts = delta.Conflicts
			}
		}
		rows = append(rows, row)

		totals.Created += row.Created
		totals.Deleted += row.DeletedAuto
		totals.CustomDeleted += row.DeletedCustom
		totals.CustomKept += row.SlotsAfter.Custom
		totals.AssignmentsKept += row.AssignmentsKept
		totals.AssignmentsLost += row.AssignmentsLost
		totals.Conflicts += row.Conflicts
	}
	totals.Games = len(rows)
	for _, uid := range summary.NotifiedUsers {
		notifiedSeen[uid] = true
	}
	totals.NotifiedUsers = len(notifiedSeen)

	resp := bulkRegenResponse{
		Range:    bulkRegenRange{From: from, To: to},
		Rows:     rows,
		Totals:   totals,
		Warnings: []string{},
	}

	if apply {
		if err := tx.Commit(); err != nil {
			return bulkRegenResponse{}, RegenSummary{}, notify, bulkRegenErr(http.StatusInternalServerError, "internal_error")
		}
	}
	// Preview: defer tx.Rollback() above runs on return — no write survives.

	return resp, summary, notify, nil
}

// resolveBulkAction implements the override → defaults → keep-current precedence
// (design.md §9/§7). Excluded games still get a resolved action for row display, but
// the mutation loop in runBulkRegen skips applying it.
func resolveBulkAction(g bulkGame, overrides map[int]bulkRegenOverride, defaults map[string]bulkRegenAction) (string, *int) {
	if ov, ok := overrides[g.ID]; ok {
		return ov.Action, ov.TemplateID
	}
	if def, ok := defaults[g.EventType]; ok {
		return def.Action, def.TemplateID
	}
	if g.TemplateID.Valid {
		id := int(g.TemplateID.Int64)
		return "template", &id
	}
	return "none", nil
}

// bulkGame is one row of games loaded for the bulk-regen range.
type bulkGame struct {
	ID         int
	Date       string
	Time       string
	Opponent   string
	EventType  string
	IsHome     bool
	TemplateID sql.NullInt64
}

func (h *Handler) loadBulkRangeGames(ctx context.Context, tx *sql.Tx, seasonID int, from, to string) ([]bulkGame, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, date, time, COALESCE(opponent,''), event_type, is_home, template_id
		FROM games WHERE season_id=? AND date BETWEEN ? AND ?
		ORDER BY date, time, id`, seasonID, from, to)
	if err != nil {
		return nil, fmt.Errorf("load range games: %w", err)
	}
	defer rows.Close()
	var games []bulkGame
	for rows.Next() {
		var g bulkGame
		var isHome int
		if err := rows.Scan(&g.ID, &g.Date, &g.Time, &g.Opponent, &g.EventType, &isHome, &g.TemplateID); err != nil {
			return nil, err
		}
		g.IsHome = isHome == 1
		// SQLite DATE-Gotcha (docs/agent/06-gotchas.md): der Scan liefert einen
		// ISO-Timestamp ("2026-08-21T00:00:00Z") statt der reinen Datumszeichenkette,
		// unabhängig vom insertierten Format. Muss vor jeder Weiterverwendung als
		// Datums-Schlüssel (dateSet → runAutoRegen, WHERE date=?) normalisiert werden
		// — sonst matcht regenSingleDay das Datum nie.
		if len(g.Date) > 10 {
			g.Date = g.Date[:10]
		}
		games = append(games, g)
	}
	return games, nil
}

func countGameSlots(ctx context.Context, tx *sql.Tx, gameID int) (auto, custom int, err error) {
	if err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).Scan(&auto); err != nil {
		return 0, 0, err
	}
	if err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM duty_slots WHERE game_id=? AND is_custom=1`, gameID).Scan(&custom); err != nil {
		return 0, 0, err
	}
	return auto, custom, nil
}
