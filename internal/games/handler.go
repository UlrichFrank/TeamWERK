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
	"time"
	"unicode/utf8"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/httpcache"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/policy"
)

type Handler struct {
	db     *sql.DB
	cfg    *appconfig.Config
	hub    *hub.EventHub
	now    func() time.Time
	newH4A func() h4aFetcher // H4A-Client-Factory, in Tests überschreibbar (kein echter Netzzugriff)
}

func NewHandler(db *sql.DB, cfg *appconfig.Config, h *hub.EventHub) *Handler {
	return &Handler{db: db, cfg: cfg, hub: h, now: time.Now, newH4A: defaultH4AFetcher}
}

// broadcastGame sends event only to the team audience of the given game (team
// members + trainers/sL + parents + vorstand/admin). Replaces the former global
// Broadcast for game-bound topics; the Frontend contract (topic string +
// useLiveUpdates) is unchanged, only the recipient set shrinks. Resolve the
// game's teams BEFORE deleting the game (game_teams cascades on delete).
func (h *Handler) broadcastGame(ctx context.Context, gameID int, event string) {
	if h.hub == nil {
		return
	}
	ids := hub.NewAudience(h.db).GameAudience(ctx, gameID)
	h.hub.BroadcastToUsers(ids, event)
}

// broadcastGameTeams is like broadcastGame but takes already-resolved team IDs
// (used by DeleteGame, which must capture the teams before the game_teams rows
// cascade away). extraUserIDs (e.g. affected duty assignees) are included.
func (h *Handler) broadcastGameTeams(ctx context.Context, teamIDs []int, event string, extraUserIDs ...int) {
	if h.hub == nil {
		return
	}
	ids := hub.NewAudience(h.db).Team(ctx, teamIDs, extraUserIDs...)
	h.hub.BroadcastToUsers(ids, event)
}

// SetNow overrides the clock used for cutoff checks. Intended for tests.
func (h *Handler) SetNow(now func() time.Time) { h.now = now }

// GameRSVPCutoff: bis dahin (vor Spielbeginn) sind RSVP-Änderungen
// für Spieler/Eltern erlaubt. Trainer/Vorstand/Admin können auch danach pflegen.
const GameRSVPCutoff = 18 * time.Hour

var berlinTZ = mustLoadBerlin()

func mustLoadBerlin() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic("games: cannot load Europe/Berlin timezone: " + err.Error())
	}
	return loc
}

// gameLocksAt liefert den UTC-Zeitpunkt, ab dem RSVP-Änderungen
// für reguläre Mitglieder gesperrt sind. dateISO ist `YYYY-MM-DD`,
// timeHHMM ist `HH:MM` (Sekunden werden toleriert) in Europe/Berlin.
func gameLocksAt(dateISO, timeHHMM string) (time.Time, error) {
	// SQLite DATE columns are returned as RFC3339 ("2026-06-15T00:00:00Z");
	// keep only the YYYY-MM-DD prefix. Tolerate "HH:MM:SS" similarly.
	if len(dateISO) > 10 {
		dateISO = dateISO[:10]
	}
	if len(timeHHMM) > 5 {
		timeHHMM = timeHHMM[:5]
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", dateISO+" "+timeHHMM, berlinTZ)
	if err != nil {
		return time.Time{}, err
	}
	return t.Add(-GameRSVPCutoff).UTC(), nil
}

func writeRSVPLocked(w http.ResponseWriter, message string, locksAt time.Time) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":    "rsvp_locked",
		"message":  message,
		"locks_at": locksAt.UTC().Format(time.RFC3339),
	})
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

// diffTeamIDs returns the team IDs in a that are not present in b (a \ b),
// preserving the order of a. Used to isolate the teams removed from a game on a
// team re-assignment so they still receive the live-update / push.
func diffTeamIDs(a, b []int) []int {
	if len(a) == 0 {
		return nil
	}
	inB := make(map[int]struct{}, len(b))
	for _, id := range b {
		inB[id] = struct{}{}
	}
	var out []int
	for _, id := range a {
		if _, ok := inB[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

// isScopedTrainer reports whether the caller's event mutations are limited to
// their own teams: club function `trainer` WITHOUT `sportliche_leitung` or
// `vorstand`, and without the admin system role. Everyone else who reaches the
// mutation routes (admin/vorstand/sportliche_leitung, per the router tier)
// passes the team-scope checks unfiltered — consistent with CanViewAllGames.
func isScopedTrainer(claims *auth.Claims) bool {
	if claims == nil || claims.Role == "admin" {
		return false
	}
	if !claims.HasFunction("trainer") {
		return false
	}
	return !claims.HasFunction("sportliche_leitung") && !claims.HasFunction("vorstand")
}

// uniqueTeamIDs returns teamIDs without duplicates, preserving first-seen
// order. checkTeamScope compares a DISTINCT COUNT against the list length, so a
// request repeating the same team ("1,1,2") must not be able to pass the
// all-own check for heim/auswärts with only one own team.
func uniqueTeamIDs(teamIDs []int) []int {
	seen := make(map[int]struct{}, len(teamIDs))
	out := make([]int, 0, len(teamIDs))
	for _, id := range teamIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// trainerOwnTeamCount returns how many DISTINCT entries of teamIDs the user
// trains in the active season.
func (h *Handler) trainerOwnTeamCount(ctx context.Context, userID int, teamIDs []int) (int, error) {
	if len(teamIDs) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(teamIDs)), ",")
	args := append([]any{userID}, toAny(teamIDs)...)
	var count int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT trm.team_id) FROM trainer_memberships trm
		JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
		JOIN members m ON m.id = trm.member_id AND m.user_id = ?
		WHERE trm.team_id IN (`+placeholders+`)`, args...).Scan(&count)
	return count, err
}

// checkTeamScope validates the team_ids a scoped trainer wants to attach to an
// event of the given type:
//
//	heim/auswärts — every team must be one of their own (unchanged behaviour).
//	generisch     — at least ONE own team must be included; the rest may be any
//	                active team of the club. The "at least one" anchor keeps the
//	                trainer from creating or re-assigning an event that
//	                ScopeGamesQuery would then hide from them, leaving it
//	                neither correctable nor deletable.
func (h *Handler) checkTeamScope(ctx context.Context, claims *auth.Claims, eventType string, teamIDs []int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if !isScopedTrainer(claims) {
		return true, nil
	}
	ids := uniqueTeamIDs(teamIDs)
	own, err := h.trainerOwnTeamCount(ctx, claims.UserID, ids)
	if err != nil {
		return false, err
	}
	if eventType == "generisch" {
		return own >= 1, nil
	}
	return own == len(ids), nil
}

// canMutateGame reports whether the caller may change or delete the given event
// at all. A scoped trainer needs at least one of their own teams among the
// event's CURRENT teams — the router tier alone (vorstand/trainer/sL) would
// otherwise let any trainer edit or delete any club event via the API. The
// team_ids of the RESULTING event are checked separately by checkTeamScope.
func (h *Handler) canMutateGame(ctx context.Context, claims *auth.Claims, gameID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if !isScopedTrainer(claims) {
		return true, nil
	}
	var trains int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trainer_memberships trm
		JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
		JOIN members m ON m.id = trm.member_id AND m.user_id = ?
		JOIN game_teams gt ON gt.team_id = trm.team_id AND gt.game_id = ?`,
		claims.UserID, gameID).Scan(&trains)
	if err != nil {
		return false, err
	}
	return trains > 0, nil
}

// canEditGameNote reports whether the caller may set a game's note: admin,
// vorstand, sportliche_leitung, or a trainer of a participating team. Mirrors
// the canEdit logic of GetGame.
func (h *Handler) canEditGameNote(ctx context.Context, claims *auth.Claims, gameID int) bool {
	gp := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions}
	if policy.CanViewAllGames(gp) {
		return true
	}
	if !policy.IsTrainerLike(gp) {
		return false
	}
	var trains int
	h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trainer_memberships trm
		JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
		JOIN members m ON m.id = trm.member_id AND m.user_id = ?
		JOIN game_teams gt ON gt.team_id = trm.team_id AND gt.game_id = ?`,
		claims.UserID, gameID).Scan(&trains)
	return trains > 0
}

// PUT /api/games/{id}/note — setzt das Hinweisfeld eines Spiels/Events.
// Berechtigung: Vorstand / Trainer eines beteiligten Teams / sportliche_leitung / Admin.
func (h *Handler) UpdateGameNote(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(req.Note) > 200 {
		http.Error(w, "note too long", http.StatusBadRequest)
		return
	}

	var exists int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT 1 FROM games WHERE id=?`, gameID).Scan(&exists); err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !h.canEditGameNote(r.Context(), claims, gameID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	req.Note = strings.TrimSpace(req.Note)
	if _, err = tx.ExecContext(r.Context(),
		`UPDATE games SET note = ? WHERE id = ?`, req.Note, gameID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(req.Note) == "" {
		if _, err = tx.ExecContext(r.Context(),
			`DELETE FROM pending_event_notes_push WHERE ref_type='game' AND ref_id=?`,
			gameID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		if _, err = tx.ExecContext(r.Context(), `
			INSERT INTO pending_event_notes_push (ref_type, ref_id, note_text, notify_after, updated_by)
			VALUES ('game', ?, ?, datetime('now', '+5 minutes'), ?)
			ON CONFLICT(ref_type, ref_id) DO UPDATE SET
				note_text    = excluded.note_text,
				notify_after = excluded.notify_after,
				updated_by   = excluded.updated_by`,
			gameID, req.Note, claims.UserID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.broadcastGame(r.Context(), gameID, "event-note")
	w.WriteHeader(http.StatusOK)
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
	TeamIDs              []int
	// RotationMaxPerTeam aktiviert für dieses Item den Bewirtungsrotations-
	// Modus (kuchendienst-rotation): NULL = bestehendes Verhalten (ein Slot
	// pro Team des jeweiligen Spiels), gesetzt = Cap pro Team in der
	// tagesweiten Team-Warteschlange (siehe regen.go, buildRotationPlan).
	RotationMaxPerTeam sql.NullInt64
}

func (h *Handler) loadTemplateItems(ctx context.Context, templateID int) ([]templateItemRow, error) {
	rows, err := h.db.QueryContext(ctx,
		`SELECT gti.duty_type_id, dt.name, gti.anchor, gti.offset_minutes, gti.slots_count,
		        dt.same_day_behavior, dt.same_day_variant_id, dt.adjacent_day_behavior, dt.adjacent_day_variant_id,
		        gti.audiences, gti.team_ids, gti.rotation_max_per_team
		 FROM game_template_items gti JOIN duty_types dt ON dt.id = gti.duty_type_id
		 WHERE gti.template_id=? ORDER BY gti.sort_order, gti.id`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []templateItemRow
	for rows.Next() {
		var it templateItemRow
		var teamIDs sql.NullString
		rows.Scan(&it.DutyTypeID, &it.DutyTypeName, &it.Anchor, &it.OffsetMinutes,
			&it.SlotsCount, &it.SameDayBehavior, &it.SameDayVariantID,
			&it.AdjacentDayBehavior, &it.AdjacentDayVariantID, &it.Audiences, &teamIDs,
			&it.RotationMaxPerTeam)
		it.TeamIDs = teamIDsFromDB(teamIDs)
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

// validRsvpDefault reports whether v is one of the accepted enum values.
func validRsvpDefault(v string) bool {
	return v == "confirmed" || v == "declined" || v == "none"
}

// gameRegularNoResp counts regular kader members (across all of the game's teams)
// without a game_responses row, excluding trainers. Correlates on the outer alias g.
const gameRegularNoResp = `(SELECT COUNT(DISTINCT km.member_id) FROM game_teams gt4
	JOIN kader k4 ON k4.team_id = gt4.team_id AND k4.season_id = g.season_id
	JOIN kader_members km ON km.kader_id = k4.id
	WHERE gt4.game_id = g.id
	  AND NOT EXISTS (SELECT 1 FROM game_responses gr2 WHERE gr2.game_id = g.id AND gr2.member_id = km.member_id)
	  AND km.member_id NOT IN (SELECT kt.member_id FROM kader_trainers kt JOIN kader k ON k.id=kt.kader_id AND k.season_id=g.season_id WHERE k.team_id IN (SELECT team_id FROM game_teams WHERE game_id=g.id)))`

// gameExtendedNoResp counts extended-only kader members (not also in the regular
// kader, not trainers) without a game_responses row. Correlates on outer alias g.
const gameExtendedNoResp = `(SELECT COUNT(DISTINCT kem.member_id) FROM game_teams gt5
	JOIN kader k5 ON k5.team_id = gt5.team_id AND k5.season_id = g.season_id
	JOIN kader_extended_members kem ON kem.kader_id = k5.id
	WHERE gt5.game_id = g.id
	  AND NOT EXISTS (SELECT 1 FROM game_responses gr3 WHERE gr3.game_id = g.id AND gr3.member_id = kem.member_id)
	  AND kem.member_id NOT IN (SELECT km2.member_id FROM game_teams gt6 JOIN kader k6 ON k6.team_id=gt6.team_id AND k6.season_id=g.season_id JOIN kader_members km2 ON km2.kader_id=k6.id WHERE gt6.game_id=g.id)
	  AND kem.member_id NOT IN (SELECT kt.member_id FROM kader_trainers kt JOIN kader k ON k.id=kt.kader_id AND k.season_id=g.season_id WHERE k.team_id IN (SELECT team_id FROM game_teams WHERE game_id=g.id)))`

// gameRsvpCountCols yields the three header-counter columns (confirmed, declined,
// maybe) as a comma-separated SQL expression list. Explicit responses (excluding
// trainers) plus role-specific defaults ('none' counts nowhere); maybe has no
// default. Correlates on the outer alias g.
const gameRsvpCountCols = `
	COALESCE((SELECT COUNT(*) FROM game_responses gr_c WHERE gr_c.game_id=g.id AND gr_c.status='confirmed'
	           AND gr_c.member_id NOT IN (SELECT kt.member_id FROM kader_trainers kt JOIN kader k ON k.id=kt.kader_id AND k.season_id=g.season_id WHERE k.team_id IN (SELECT team_id FROM game_teams WHERE game_id=g.id))),0)
	  + CASE WHEN g.rsvp_default_players='confirmed' THEN ` + gameRegularNoResp + ` ELSE 0 END
	  + CASE WHEN g.rsvp_default_extended='confirmed' THEN ` + gameExtendedNoResp + ` ELSE 0 END,
	COALESCE((SELECT COUNT(*) FROM game_responses gr_d WHERE gr_d.game_id=g.id AND gr_d.status='declined'
	           AND gr_d.member_id NOT IN (SELECT kt.member_id FROM kader_trainers kt JOIN kader k ON k.id=kt.kader_id AND k.season_id=g.season_id WHERE k.team_id IN (SELECT team_id FROM game_teams WHERE game_id=g.id))),0)
	  + CASE WHEN g.rsvp_default_players='declined' THEN ` + gameRegularNoResp + ` ELSE 0 END
	  + CASE WHEN g.rsvp_default_extended='declined' THEN ` + gameExtendedNoResp + ` ELSE 0 END,
	COALESCE((SELECT COUNT(*) FROM game_responses gr_m WHERE gr_m.game_id=g.id AND gr_m.status='maybe'
	           AND gr_m.member_id NOT IN (SELECT kt.member_id FROM kader_trainers kt JOIN kader k ON k.id=kt.kader_id AND k.season_id=g.season_id WHERE k.team_id IN (SELECT team_id FROM game_teams WHERE game_id=g.id))),0)`

// GET /api/games?season_id=&limit=&offset=
func (h *Handler) ListGames(w http.ResponseWriter, r *http.Request) {
	seasonID := r.URL.Query().Get("season_id")
	claims := auth.ClaimsFromCtx(r.Context())

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

	// Header-Zähler beziehen die Rollen-Voreinstellungen ein: reguläre bzw.
	// erweiterte Kader-Mitglieder ohne Response werden gemäß rsvp_default_players/
	// rsvp_default_extended als confirmed/declined gezählt; 'none' zählt nirgends,
	// Trainer sind stets ausgeschlossen (siehe gameRsvpCountCols).
	const base = `
		SELECT g.id, g.date, g.time, g.end_time, g.end_date, g.opponent, g.event_type, g.template_id,
		       COUNT(DISTINCT ds.id), COALESCE(SUM(ds.slots_filled),0), COALESCE(SUM(ds.slots_total),0),
		       ` + gameRsvpCountCols + `,
		       g.rsvp_default_players, g.rsvp_default_extended, g.rsvp_require_reason, g.note,
		       v.id, v.name, v.street, v.city, v.postal_code, v.note
		FROM games g
		LEFT JOIN duty_slots ds ON ds.game_id = g.id
		LEFT JOIN venues v ON v.id = g.venue_id`
	const suffix = ` GROUP BY g.id ORDER BY g.date, g.time, g.id LIMIT ? OFFSET ?`

	// where/whereArgs: identisch für COUNT(*) und Items (Sichtbarkeit invariant).
	var where string
	var whereArgs []any
	if seasonID != "" {
		where = ` WHERE g.season_id=?` + andScope
		whereArgs = append([]any{seasonID}, scopeArgs...)
	} else {
		// Show active-season games plus any future games from other seasons
		// (prevents games from stranding when seasons are switched).
		where = ` WHERE (g.season_id=(SELECT id FROM seasons WHERE is_active=1 LIMIT 1) OR DATE(g.date) >= DATE('now','-1 day'))` + andScope
		whereArgs = scopeArgs
	}

	var total int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM games g`+where, whereArgs...).Scan(&total); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		base+where+suffix, append(append([]any{}, whereArgs...), limit, offset)...)
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
		RsvpDefaultPlayers  string              `json:"rsvp_default_players"`
		RsvpDefaultExtended string              `json:"rsvp_default_extended"`
		RsvpRequireReason   int                 `json:"rsvp_require_reason"`
		RsvpLocksAt         string              `json:"rsvp_locks_at,omitempty"`
		Note                string              `json:"note"`
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
			&g.RsvpDefaultPlayers, &g.RsvpDefaultExtended, &g.RsvpRequireReason, &g.Note,
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
		if locksAt, err := gameLocksAt(g.Date, g.Time); err == nil {
			g.RsvpLocksAt = locksAt.Format(time.RFC3339)
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
	json.NewEncoder(w).Encode(map[string]any{"items": result, "total": total})
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
		ID                  int       `json:"id"`
		Date                string    `json:"date"`
		Time                string    `json:"time"`
		EndTime             *string   `json:"end_time,omitempty"`
		EndDate             *string   `json:"end_date"`
		Opponent            string    `json:"opponent"`
		EventType           string    `json:"event_type"`
		IsHome              bool      `json:"is_home"`
		SeasonID            int       `json:"season_id"`
		TemplateID          *int      `json:"template_id"`
		RsvpDefaultPlayers  string    `json:"rsvp_default_players"`
		RsvpDefaultExtended string    `json:"rsvp_default_extended"`
		RsvpRequireReason   int       `json:"rsvp_require_reason"`
		RsvpLocksAt         string    `json:"rsvp_locks_at,omitempty"`
		Note                string    `json:"note"`
		ConfirmedCount      int       `json:"confirmed_count"`
		DeclinedCount       int       `json:"declined_count"`
		MaybeCount          int       `json:"maybe_count"`
		Venue               *venueRef `json:"venue,omitempty"`
		Teams               []struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			DisplayShort string `json:"display_short"`
			DisplayLong  string `json:"display_long"`
		} `json:"teams"`
		TeamDisplayShortCSV string              `json:"team_display_short_csv"`
		TeamDisplayLongCSV  string              `json:"team_display_long_csv"`
		AmIParticipant      bool                `json:"am_i_participant"`
		AttendanceTracked   bool                `json:"attendance_tracked"`
		Can                 policy.GameCanFlags `json:"can"`
	}
	var templateIDNull sql.NullInt64
	var endTimeNull, endDateNull sql.NullString
	var vID sql.NullInt64
	var vName, vStreet, vCity, vPostal, vNote sql.NullString
	var attendanceTracked int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT g.id, g.date, g.time, g.end_time, g.end_date, g.opponent, g.event_type, g.is_home, g.season_id, g.template_id,
		        g.rsvp_default_players, g.rsvp_default_extended, g.rsvp_require_reason, g.note,
		        `+gameRsvpCountCols+`,
		        g.attendance_tracked,
		        v.id, v.name, v.street, v.city, v.postal_code, v.note
		 FROM games g LEFT JOIN venues v ON v.id = g.venue_id WHERE g.id=?`, id).
		Scan(&g.ID, &g.Date, &g.Time, &endTimeNull, &endDateNull, &g.Opponent, &g.EventType, &g.IsHome, &g.SeasonID, &templateIDNull,
			&g.RsvpDefaultPlayers, &g.RsvpDefaultExtended, &g.RsvpRequireReason, &g.Note,
			&g.ConfirmedCount, &g.DeclinedCount, &g.MaybeCount,
			&attendanceTracked,
			&vID, &vName, &vStreet, &vCity, &vPostal, &vNote)
	g.AttendanceTracked = attendanceTracked == 1
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
	if locksAt, lerr := gameLocksAt(g.Date, g.Time); lerr == nil {
		g.RsvpLocksAt = locksAt.Format(time.RFC3339)
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

	// am_i_participant: aufrufender User selbst in Stamm-, Erweitert- oder Trainer-Kader
	// eines beteiligten Teams für die Spielsaison.
	var participant int
	h.db.QueryRowContext(r.Context(), `
		SELECT CASE WHEN EXISTS (
		  SELECT 1 FROM game_teams gt
		  JOIN kader k ON k.team_id = gt.team_id AND k.season_id = ?
		  JOIN members m ON m.user_id = ?
		  WHERE gt.game_id = ?
		    AND (
		      EXISTS (SELECT 1 FROM kader_members       km  WHERE km.kader_id  = k.id AND km.member_id  = m.id)
		      OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = m.id)
		      OR EXISTS (SELECT 1 FROM kader_trainers  kt  WHERE kt.kader_id  = k.id AND kt.member_id  = m.id)
		    )
		) THEN 1 ELSE 0 END`, g.SeasonID, claims.UserID, id).Scan(&participant)
	g.AmIParticipant = participant == 1

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
		Date                string  `json:"date"`
		Time                string  `json:"time"`
		EndTime             *string `json:"end_time"`
		Opponent            string  `json:"opponent"`
		TeamIDs             []int   `json:"team_ids"`
		EventType           string  `json:"event_type"`
		SeasonID            int     `json:"season_id"`
		TemplateID          *int    `json:"template_id"`
		VenueID             *int    `json:"venue_id"`
		RsvpDefaultPlayers  string  `json:"rsvp_default_players"`
		RsvpDefaultExtended string  `json:"rsvp_default_extended"`
		RsvpRequireReason   *int    `json:"rsvp_require_reason"`
		EndDate             *string `json:"end_date"`
		Slots               []struct {
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

	// Team-Scope für reine Trainer: bei heim/auswärts müssen alle Mannschaften
	// eigene sein, bei generisch reicht eine eigene (mannschaftsübergreifende
	// Events sind der Regelfall für diesen Typ). Siehe checkTeamScope.
	claims := auth.ClaimsFromCtx(r.Context())
	allowed, err := h.checkTeamScope(r.Context(), claims, req.EventType, req.TeamIDs)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	isHome := req.EventType == "heim"

	if req.RsvpDefaultPlayers == "" {
		req.RsvpDefaultPlayers = "none"
	}
	if req.RsvpDefaultExtended == "" {
		req.RsvpDefaultExtended = "none"
	}
	if !validRsvpDefault(req.RsvpDefaultPlayers) || !validRsvpDefault(req.RsvpDefaultExtended) {
		http.Error(w, "invalid rsvp_default_*", http.StatusBadRequest)
		return
	}

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
		`INSERT INTO games (season_id, opponent, date, time, end_time, end_date, is_home, event_type, template_id, venue_id, rsvp_default_players, rsvp_default_extended, rsvp_require_reason) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		req.SeasonID, req.Opponent, req.Date, req.Time, endTimeVal, endDateVal, isHome, req.EventType, templateIDVal, venueIDVal, req.RsvpDefaultPlayers, req.RsvpDefaultExtended, rsvpRequireReason)
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

	summary, err := h.runAutoRegen(r.Context(), tx, dateWindow(req.Date), req.SeasonID, nil)
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

	h.broadcastGame(r.Context(), int(gameID), "games")
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
		Date                string          `json:"date"`
		Time                string          `json:"time"`
		EndTime             *string         `json:"end_time"`
		EndDate             *string         `json:"end_date"`
		Opponent            string          `json:"opponent"`
		TeamIDs             []int           `json:"team_ids"`
		EventType           string          `json:"event_type"`
		VenueID             *int            `json:"venue_id"`
		RsvpDefaultPlayers  *string         `json:"rsvp_default_players,omitempty"`
		RsvpDefaultExtended *string         `json:"rsvp_default_extended,omitempty"`
		RsvpRequireReason   *int            `json:"rsvp_require_reason,omitempty"`
		TemplateID          json.RawMessage `json:"template_id,omitempty"`
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

	// Autorisierung vor jedem Schreibzugriff. Die 404-Prüfung steht bewusst VOR
	// der 403-Antwort, damit fremde Event-IDs nicht über den Statuscode
	// unterscheidbar werden.
	gameIDInt, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	var storedEventType string
	switch err := h.db.QueryRowContext(r.Context(),
		`SELECT event_type FROM games WHERE id=?`, gameIDInt).Scan(&storedEventType); {
	case err == sql.ErrNoRows:
		http.Error(w, "not found", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	mayMutate, err := h.canMutateGame(r.Context(), claims, gameIDInt)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !mayMutate {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	// Hängt der Request die Mannschaften um, gilt die Typ-Regel für den ZIEL-Typ:
	// req.EventType, falls gesetzt, sonst der gespeicherte Typ.
	if len(req.TeamIDs) > 0 {
		targetType := storedEventType
		if req.EventType == "heim" || req.EventType == "auswärts" || req.EventType == "generisch" {
			targetType = req.EventType
		}
		allowed, err := h.checkTeamScope(r.Context(), claims, targetType, req.TeamIDs)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
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
		`SELECT date, season_id FROM games WHERE id=?`, id).
		Scan(&oldDate, &oldSeasonID); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Enum-Validierung der bereitgestellten RSVP-Felder (keine Konflikt-Prüfung mehr).
	if req.RsvpDefaultPlayers != nil && !validRsvpDefault(*req.RsvpDefaultPlayers) {
		http.Error(w, "invalid rsvp_default_players", http.StatusBadRequest)
		return
	}
	if req.RsvpDefaultExtended != nil && !validRsvpDefault(*req.RsvpDefaultExtended) {
		http.Error(w, "invalid rsvp_default_extended", http.StatusBadRequest)
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

	// Erfasse die ALTEN Team-IDs VOR dem game_teams-Rewrite. broadcastGame und der
	// notify.Send unten lösen das Publikum aus den game_teams auf; würde man das erst
	// nach dem Rewrite tun, erführen die entfernten (alten) Teams nichts von der
	// Umhängung und behielten ein Spiel in ihrer Liste, das ihnen nicht mehr gehört.
	// Nur relevant, wenn TeamIDs im Request stehen (sonst bleibt game_teams unverändert).
	var oldTeamIDs []int
	if len(req.TeamIDs) > 0 {
		oldTeamIDs = hub.NewAudience(h.db).TeamIDsForGame(r.Context(), gameIDInt)
		tx.ExecContext(r.Context(), `DELETE FROM game_teams WHERE game_id=?`, id)
		for _, teamID := range req.TeamIDs {
			tx.ExecContext(r.Context(),
				`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, id, teamID)
		}
	}

	// Partial-Update: RSVP-Felder nur setzen, wenn im Request enthalten.
	if req.RsvpDefaultPlayers != nil || req.RsvpDefaultExtended != nil || req.RsvpRequireReason != nil {
		setParts := []string{}
		setArgs := []interface{}{}
		if req.RsvpDefaultPlayers != nil {
			setParts = append(setParts, "rsvp_default_players=?")
			setArgs = append(setArgs, *req.RsvpDefaultPlayers)
		}
		if req.RsvpDefaultExtended != nil {
			setParts = append(setParts, "rsvp_default_extended=?")
			setArgs = append(setArgs, *req.RsvpDefaultExtended)
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
	summary, err := h.runAutoRegen(r.Context(), tx, regenDates, oldSeasonID, nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// broadcastGame/gameTeamIDs adressieren die AKTUELLEN (neuen) game_teams. Bei einer
	// Team-Umhängung müssen aber auch die ENTFERNTEN Teams erfahren, dass ihnen das Spiel
	// nicht mehr gehört. Deshalb ein zusätzlicher, gezielter Broadcast/Push an die alten
	// Teams, die nicht mehr im neuen Set sind (removed = old \ new) — kleinster Eingriff,
	// keine Doppelung mit dem aktuellen Publikum.
	removedTeamIDs := diffTeamIDs(oldTeamIDs, req.TeamIDs)
	h.broadcastGame(r.Context(), gameIDInt, "games")
	if len(removedTeamIDs) > 0 {
		h.broadcastGameTeams(r.Context(), removedTeamIDs, "games")
	}
	notify.Send(h.db, h.cfg,
		h.teamMembersAndParents(h.gameTeamIDs(gameIDInt)),
		"games", "Spielinfo geändert", req.Opponent+" — Details aktualisiert", fmt.Sprintf("/termine?focus=game-%d", gameIDInt))
	if len(removedTeamIDs) > 0 {
		notify.Send(h.db, h.cfg,
			h.teamMembersAndParents(removedTeamIDs),
			"games", "Spielinfo geändert", req.Opponent+" — Details aktualisiert", fmt.Sprintf("/termine?focus=game-%d", gameIDInt))
	}
	h.dispatchRegenNotifications(summary)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"regen_summary": summary})
}

// DELETE /api/games/{id}
// Deletes a game (incl. generic events) together with all duty_slots and
// duty_assignments referencing it (via ON DELETE CASCADE since migration 027).
// For each fulfilled assignment that gets cascade-deleted, the corresponding
// duty_accounts.ist is recomputed in the same transaction so no orphan hours
// remain on user accounts.
func (h *Handler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	// Der optionale {reason, silent}-Body wird als Allererstes gelesen — danach
	// ist er verbraucht. Fehlt er (alte PWA aus dem Service-Worker-Cache) oder
	// ist er kaputt, heißt das „kein Grund, nicht stumm", nie HTTP 400.
	reason, silent := notify.DecodeCancellation(r)
	claims := auth.ClaimsFromCtx(r.Context())
	// Stummschalten ist ein eigenes, engeres Recht als das Löschrecht. Fehlt es,
	// wird das Flag ignoriert statt die Löschung mit 403 abzubrechen —
	// benachrichtigen ist der sichere Default.
	if silent {
		silent = claims != nil && policy.CanSuppressEventNotification(&policy.Principal{
			UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions,
		})
	}

	id := r.PathValue("id")
	// Fetch team IDs before deleting (game_teams rows are cascade-deleted)
	teamIDs := h.gameTeamIDs(id)

	// Collect event metadata + affected duty assignees before the cascade fires.
	var (
		seasonID  int
		opponent  string
		eventDate string
		eventType string
	)
	err := h.db.QueryRowContext(r.Context(),
		`SELECT season_id, COALESCE(opponent, ''), date, event_type FROM games WHERE id=?`, id).
		Scan(&seasonID, &opponent, &eventDate, &eventType)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Autorisierung nach der Existenzprüfung (404 vor 403) und vor dem Cascade:
	// ein reiner Trainer löscht nur Events, an denen eine eigene Mannschaft
	// beteiligt ist.
	gameIDInt, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	mayMutate, err := h.canMutateGame(r.Context(), claims, gameIDInt)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !mayMutate {
		http.Error(w, "forbidden", http.StatusForbidden)
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

	// event-notes: etwaige pending Push-Row mitlöschen, sonst Karteileiche.
	if _, err = tx.ExecContext(r.Context(),
		`DELETE FROM pending_event_notes_push WHERE ref_type='game' AND ref_id=?`, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
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
	summary, err := h.runAutoRegen(r.Context(), tx, neighborDates, seasonID, nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Game is gone (game_teams cascaded) — use the teamIDs captured before the
	// delete, plus the duty assignees whose slots vanished, as the audience.
	h.broadcastGameTeams(r.Context(), teamIDs, "games", append(append([]int{}, assignedUIDs...), fulfilledUIDs...)...)

	// Absage-Meldungen. `silent` betrifft ausschließlich notify.Send — Broadcast
	// (oben) und Regen-Meldungen (unten) laufen immer, sonst zeigten offene
	// Sessions den gelöschten Termin weiter an.
	if !silent {
		actor := notify.ActorName(h.db, claims.UserID)
		eventName := opponent
		if eventName == "" {
			eventName = "Termin" // generische Events ohne Gegnerfeld
		}
		eventDay := formatDateDMY(eventDate)

		// Targeted notification to duty assignees in their "duties" category.
		// Der erste Satz ist als Wortlaut in specs/push-duties festgeschrieben
		// und bleibt deshalb unverändert; angehängt werden nur Aktor und Grund.
		// Der Link bleibt /dienste — die Dienstbörse existiert nach der Löschung
		// weiter, der Empfänger kann sich dort neu eintragen.
		if len(assignedUIDs) > 0 {
			body := fmt.Sprintf("Dein Dienst zum %s am %s wurde gelöscht. %s",
				eventName, eventDay, notify.ActorClause(actor, reason))
			notify.Send(h.db, h.cfg, assignedUIDs, "duties", "Dienst entfällt", body, "/dienste")
		}

		// Team-wide event-cancellation notification in "games" category (unchanged
		// audience). Das Linkziel ist bewusst leer: /termine zeigt den Termin nach
		// der Löschung nicht mehr, ein Sprung dorthin wirkt wie ein Fehler.
		notify.Send(h.db, h.cfg, h.teamMembersAndParents(teamIDs),
			"games", cancellationTitle(eventType),
			notify.CancellationBody(eventName, "am "+eventDay, actor, reason), "")
	}

	h.dispatchRegenNotifications(summary)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"regen_summary": summary})
}

// cancellationTitle wählt die Überschrift der Absage-Meldung passend zum
// games.event_type. Die Route trägt drei fachlich verschiedene Termine: heim und
// auswärts sind Spiele, `generisch` ist alles andere (Turnier, Vereinsfest,
// Zusatztraining) — dort war "Spiel abgesagt" schlicht falsch. Unbekannte Werte
// fallen bewusst auf den Spiel-Titel zurück: der CHECK-Constraint der Tabelle
// lässt nur diese drei zu, ein vierter wäre ein Migrations-Fehler und soll die
// Meldung nicht verschlucken.
func cancellationTitle(eventType string) string {
	if eventType == "generisch" {
		return "Termin abgesagt"
	}
	return "Spiel abgesagt"
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
// Dünner Alias auf notify.FormatDateDMY, damit das Datumsformat der
// Absage-Texte nur eine Implementierung hat.
func formatDateDMY(s string) string { return notify.FormatDateDMY(s) }

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
			 ORDER BY `+appdb.AgeClassSortKey("t.age_class")+`, t.gender, k.team_number`)
	} else if claims.IsTrainerLike() && !claims.HasFunction("sportliche_leitung") {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT t.id, t.name, t.age_class, t.gender, k.team_number, `+groupCountSub+`, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 JOIN kader_trainers kt ON kt.kader_id = k.id
			 JOIN members m ON m.id = kt.member_id
			 WHERE k.season_id = `+activeSeasonSub+` AND m.user_id = ?
			 ORDER BY `+appdb.AgeClassSortKey("t.age_class")+`, t.gender, k.team_number`, claims.UserID)
	} else if !claims.IsTrainerLike() {
		// spieler / elternteil: only teams the user or their children belong to.
		// user_accessible_teams covers regular AND extended squad (kader_extended_members)
		// for both the player themselves and their parents (via family_links).
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT DISTINCT t.id, t.name, t.age_class, t.gender, k.team_number, `+groupCountSub+`, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 WHERE k.season_id = `+activeSeasonSub+`
			   AND t.id IN (
			     SELECT team_id FROM user_accessible_teams
			     WHERE user_id = ? AND season_id = `+activeSeasonSub+`
			   )
			 ORDER BY `+appdb.AgeClassSortKey("t.age_class")+`, t.gender, k.team_number`, claims.UserID)
	} else {
		// sportliche_leitung: all teams
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT t.id, t.name, t.age_class, t.gender, k.team_number, `+groupCountSub+`, t.is_active
			 FROM teams t
			 JOIN kader k ON k.team_id = t.id
			 WHERE k.season_id = `+activeSeasonSub+`
			 ORDER BY `+appdb.AgeClassSortKey("t.age_class")+`, t.gender, k.team_number`)
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
	// Nutzergefilterte Referenzdaten: ETag aus dem Body (pro Nutzer korrekt),
	// Revalidierung per 304 — bewusst KEIN geteiltes public/max-age, damit
	// Zwischencaches nie die Antwort eines anderen Nutzers ausliefern.
	httpcache.ServeJSON(w, r, "private, no-cache", result)
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
	// TeamIDs schränkt ein, für welche Teams eines Spiels aus diesem Item ein
	// Slot entsteht. Leer/NULL = alle Teams (umgekehrte Leer-Semantik zu Audiences).
	TeamIDs []int `json:"team_ids,omitempty"`
	// RotationMaxPerTeam (kuchendienst-rotation): nil = Rotation deaktiviert
	// (bestehendes Verhalten), gesetzt = Cap pro Team in der tagesweiten
	// Bewirtungs-Warteschlange. Setzt same_day_behavior/adjacent_day_behavior
	// des Duty-Types = 'normal' voraus (UpdateTemplate validiert das).
	RotationMaxPerTeam *int `json:"rotation_max_per_team,omitempty"`
}

func (h *Handler) scanTemplateItems(ctx context.Context, templateID int) []templateItem {
	rows, _ := h.db.QueryContext(ctx,
		`SELECT gti.id, gti.duty_type_id, dt.name, gti.anchor, gti.offset_minutes, gti.slots_count, gti.audiences, gti.team_ids, gti.rotation_max_per_team
		 FROM game_template_items gti JOIN duty_types dt ON dt.id = gti.duty_type_id
		 WHERE gti.template_id=? ORDER BY gti.sort_order, gti.id`, templateID)
	items := []templateItem{}
	if rows == nil {
		return items
	}
	defer rows.Close()
	for rows.Next() {
		var it templateItem
		var audiences, teamIDs sql.NullString
		var rotationMax sql.NullInt64
		rows.Scan(&it.ID, &it.DutyTypeID, &it.DutyTypeName, &it.Anchor, &it.OffsetMinutes, &it.SlotsCount, &audiences, &teamIDs, &rotationMax)
		it.Audiences = audiencesFromDB(audiences)
		it.TeamIDs = teamIDsFromDB(teamIDs)
		if rotationMax.Valid {
			v := int(rotationMax.Int64)
			it.RotationMaxPerTeam = &v
		}
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
	// Templates sind nicht team-gebunden (sie steuern die Dienst-Generierung
	// vereinsweit) → bewusst global, kein Team-Scoping.
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
		// same_day_behavior/adjacent_day_behavior werden zusammen mit der
		// Existenzprüfung geladen — für die Rotations-Validierung unten
		// gebraucht (design.md kuchendienst-rotation, Decision 4).
		var sameDayBehavior, adjacentDayBehavior string
		err := h.db.QueryRowContext(r.Context(),
			`SELECT same_day_behavior, adjacent_day_behavior FROM duty_types WHERE id=?`, it.DutyTypeID,
		).Scan(&sameDayBehavior, &adjacentDayBehavior)
		if err == sql.ErrNoRows {
			http.Error(w, "invalid duty_type_id", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// Rotations-Cap setzt same_day_behavior='normal' UND
		// adjacent_day_behavior='normal' voraus — sonst müsste die Rotation
		// zusätzlich mit variantenwechselndem Duty-Type umgehen (siehe
		// design.md). Nichts wird persistiert, wenn diese Prüfung fehlschlägt.
		if it.RotationMaxPerTeam != nil && (sameDayBehavior != "normal" || adjacentDayBehavior != "normal") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rotation_requires_normal_behavior"})
			return
		}
		if it.RotationMaxPerTeam != nil && *it.RotationMaxPerTeam <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_rotation_max_per_team"})
			return
		}
		// team_ids wird nur gegen die Existenz in teams geprüft, bewusst NICHT
		// gegen die aktive Saison: eine Vorlage überlebt Saisonwechsel, und
		// zwischen Saisonende und Kader-Kopie gäbe es sonst ein Speicher-Loch.
		for _, tid := range it.TeamIDs {
			var teamExists int
			if err := h.db.QueryRowContext(r.Context(),
				`SELECT COUNT(*) FROM teams WHERE id=?`, tid).Scan(&teamExists); err != nil || teamExists == 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_team"})
				return
			}
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
		var rotationMaxVal any
		if it.RotationMaxPerTeam != nil {
			rotationMaxVal = *it.RotationMaxPerTeam
		}
		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order, audiences, team_ids, rotation_max_per_team)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			id, it.DutyTypeID, it.Anchor, it.OffsetMinutes, it.SlotsCount, i,
			audiencesToDB(it.Audiences), teamIDsToDB(it.TeamIDs), rotationMaxVal)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Template-Änderung ist vereinsweit (nicht team-gebunden) → bewusst global.
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
	// Template-Löschung ist vereinsweit (nicht team-gebunden) → bewusst global.
	h.hub.Broadcast("games")
	w.WriteHeader(http.StatusOK)
}

// GET /api/admin/duty-templates/{id}/preview?time=HH:MM&game_id=N&team_ids=1,2
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
	var templateType string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, duration_minutes, template_type FROM game_templates WHERE id=?`, templateIDStr).
		Scan(&templateID, &durationMins, &templateType)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Teams des geplanten Events, damit die Vorschau dieselbe Team-Einschränkung
	// anwendet wie der Regen. Explizite team_ids gewinnen (Anlege-Wizard, das Event
	// existiert noch nicht); sonst aus dem bestehenden Spiel ableiten. Ist keins von
	// beidem da, bleibt die Vorschau ungefiltert — nicht filtern ist ehrlicher als
	// raten und hält Bestandsaufrufer unverändert.
	// generisch ist ausgenommen: dort ignoriert auch der Regen team_ids, ein Filter
	// würde hier einen Slot verschweigen, der real entsteht.
	var previewTeamIDs []int
	if templateType != "generisch" {
		if raw := r.URL.Query().Get("team_ids"); raw != "" {
			for _, part := range strings.Split(raw, ",") {
				if id, convErr := strconv.Atoi(strings.TrimSpace(part)); convErr == nil && id > 0 {
					previewTeamIDs = append(previewTeamIDs, id)
				}
			}
		} else if gameIDStr != "" {
			rows, qErr := h.db.QueryContext(r.Context(),
				`SELECT team_id FROM game_teams WHERE game_id=?`, gameIDStr)
			if qErr == nil {
				defer rows.Close()
				for rows.Next() {
					var id int
					if rows.Scan(&id) == nil {
						previewTeamIDs = append(previewTeamIDs, id)
					}
				}
			}
		}
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
		if !itemAppliesToAnyTeam(it.TeamIDs, previewTeamIDs) {
			continue // Item erzeugt für keines der gewählten Teams einen Slot
		}
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
	summary, err := h.runAutoRegen(r.Context(), tx, []string{date}, seasonID, nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.dispatchRegenNotifications(summary)
	// Regen kann Slots mehrerer Teams betreffen (alle Events des Datums) → globaler
	// duties-Broadcast, damit die Dienstbörse (useLiveUpdates('duties')) neu lädt.
	if h.hub != nil {
		h.hub.Broadcast("duties")
	}
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
	summary, err := h.runAutoRegen(r.Context(), tx, []string{date}, seasonID, nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.dispatchRegenNotifications(summary)
	// Regen betrifft die Dienst-Slots des Spieltags → globaler duties-Broadcast,
	// damit die Dienstbörse (useLiveUpdates('duties')) neu lädt.
	if h.hub != nil {
		h.hub.Broadcast("duties")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// ── Game RSVP ────────────────────────────────────────────────────────────────

type childRSVP struct {
	MemberID int     `json:"member_id"`
	Name     string  `json:"name"`
	RSVP     *string `json:"rsvp"`
	Reason   *string `json:"reason,omitempty"`
	Locked   bool    `json:"rsvp_locked"`
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
	MyRSVPIsDefault     bool          `json:"my_rsvp_is_default,omitempty"`
	MyRSVPLocked        bool          `json:"my_rsvp_locked"`
	MyReason            *string       `json:"my_reason,omitempty"`
	AmIParticipant      bool          `json:"am_i_participant"`
	ChildrenRSVP        []childRSVP   `json:"children_rsvp,omitempty"`
	RsvpDefaultPlayers  string        `json:"rsvp_default_players"`
	RsvpDefaultExtended string        `json:"rsvp_default_extended"`
	RsvpRequireReason   int           `json:"rsvp_require_reason"`
	RsvpLocksAt         string        `json:"rsvp_locks_at,omitempty"`
	Note                string        `json:"note"`
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
		// Sichtbarkeit ist saison-gebunden: die Kader-Zugehörigkeit muss aus
		// derselben Saison stammen wie das Spiel (g.season_id). Alt-Saison-
		// Zugehörigkeiten sonst leaken Spiele der neuen Saison ins gleiche
		// team_id (Bug: Elternteil sah gD-Spiel, weil das Kind eine Saison
		// vorher im gD-Kader stand). Konsistent zu event_visibility.go.
		var conds []string
		if claims.HasFunction("trainer") {
			conds = append(conds, `EXISTS (
				SELECT 1 FROM kader k
				JOIN kader_trainers kt ON kt.kader_id = k.id
				JOIN members m ON m.id = kt.member_id
				WHERE m.user_id = ?
				  AND k.team_id = gt.team_id
				  AND k.season_id = g.season_id)`)
			teamArgs = append(teamArgs, claims.UserID)
		}
		if claims.IsParent {
			conds = append(conds, `(EXISTS (
				SELECT 1 FROM team_memberships tm
				JOIN members m ON m.id = tm.member_id
				JOIN family_links fl ON fl.member_id = m.id
				WHERE fl.parent_user_id = ?
				  AND tm.team_id = gt.team_id
				  AND tm.season_id = g.season_id)
				OR EXISTS (
				SELECT 1 FROM kader_extended_members kem
				JOIN kader k ON k.id = kem.kader_id
				JOIN family_links fl ON fl.member_id = kem.member_id
				WHERE fl.parent_user_id = ?
				  AND k.team_id = gt.team_id
				  AND k.season_id = g.season_id))`)
			teamArgs = append(teamArgs, claims.UserID, claims.UserID)
		}
		conds = append(conds, `(EXISTS (
			SELECT 1 FROM team_memberships tm
			JOIN members m ON m.id = tm.member_id
			WHERE m.user_id = ?
			  AND tm.team_id = gt.team_id
			  AND tm.season_id = g.season_id)
			OR EXISTS (
			SELECT 1 FROM kader_extended_members kem
			JOIN kader k ON k.id = kem.kader_id
			JOIN members m2 ON m2.id = kem.member_id
			WHERE m2.user_id = ?
			  AND k.team_id = gt.team_id
			  AND k.season_id = g.season_id))`)
		teamArgs = append(teamArgs, claims.UserID, claims.UserID)
		teamSQL = "(" + strings.Join(conds, " OR ") + ")"
	}

	// Args order: memberID (my_rsvp), memberID (my_rsvp_locked), memberID (my_reason),
	// memberID (in_regular_kader), memberID (in_extended_kader), memberID (in_trainer_kader),
	// teamArgs, from, to
	args := append([]any{memberID, memberID, memberID, memberID, memberID, memberID}, teamArgs...)
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
		       `+gameRsvpCountCols+`,
		       (SELECT status FROM game_responses WHERE game_id=g.id AND member_id=?),
		       (SELECT absence_id IS NOT NULL FROM game_responses WHERE game_id=g.id AND member_id=? LIMIT 1),
		       (SELECT reason FROM game_responses WHERE game_id=g.id AND member_id=?),
		       g.rsvp_default_players, g.rsvp_default_extended, g.rsvp_require_reason, g.note,
		       EXISTS(SELECT 1 FROM game_teams gt_r
		              JOIN kader k_r ON k_r.team_id = gt_r.team_id AND k_r.season_id = g.season_id
		              JOIN kader_members km_r ON km_r.kader_id = k_r.id AND km_r.member_id = ?
		              WHERE gt_r.game_id = g.id),
		       EXISTS(SELECT 1 FROM game_teams gt_e
		              JOIN kader k_e ON k_e.team_id = gt_e.team_id AND k_e.season_id = g.season_id
		              JOIN kader_extended_members kem_e ON kem_e.kader_id = k_e.id AND kem_e.member_id = ?
		              WHERE gt_e.game_id = g.id),
		       EXISTS(SELECT 1 FROM game_teams gt_t
		              JOIN kader k_t ON k_t.team_id = gt_t.team_id AND k_t.season_id = g.season_id
		              JOIN kader_trainers kt_t ON kt_t.kader_id = k_t.id AND kt_t.member_id = ?
		              WHERE gt_t.game_id = g.id),
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
		var isHome, inRegularKader, inExtendedKader, inTrainerKader int
		var myRSVP, myReason sql.NullString
		var myRSVPLocked sql.NullInt64
		var teamNames, teamIDsCSV, teamShortCSV, teamLongCSV sql.NullString
		var vID sql.NullInt64
		var vName, vStreet, vCity, vPostal, vNote sql.NullString
		if err := rows.Scan(&g.ID, &g.Date, &g.Time, &g.Opponent, &g.EventType, &isHome, &g.SeasonID,
			&teamNames, &teamIDsCSV, &teamShortCSV, &teamLongCSV, &g.ConfirmedCount, &g.DeclinedCount, &g.MaybeCount, &myRSVP, &myRSVPLocked, &myReason,
			&g.RsvpDefaultPlayers, &g.RsvpDefaultExtended, &g.RsvpRequireReason, &g.Note, &inRegularKader, &inExtendedKader, &inTrainerKader,
			&vID, &vName, &vStreet, &vCity, &vPostal, &vNote); err != nil {
			fmt.Fprintf(os.Stderr, "ListMyGames scan: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		g.IsHome = isHome == 1
		g.AmIParticipant = inRegularKader == 1 || inExtendedKader == 1 || inTrainerKader == 1
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
		// Priorität: explizite Response > rsvp_default_players (Stammkader) >
		// rsvp_default_extended (nur Erweiterter Kader) > Trainer-confirmed > null.
		// 'none' liefert nichts.
		if myRSVP.Valid {
			g.MyRSVP = &myRSVP.String
			if myReason.Valid && myReason.String != "" {
				g.MyReason = &myReason.String
			}
		} else if inRegularKader == 1 && (g.RsvpDefaultPlayers == "confirmed" || g.RsvpDefaultPlayers == "declined") {
			v := g.RsvpDefaultPlayers
			g.MyRSVP = &v
			g.MyRSVPIsDefault = true
		} else if inExtendedKader == 1 && (g.RsvpDefaultExtended == "confirmed" || g.RsvpDefaultExtended == "declined") {
			v := g.RsvpDefaultExtended
			g.MyRSVP = &v
			g.MyRSVPIsDefault = true
		} else if inTrainerKader == 1 {
			confirmed := "confirmed"
			g.MyRSVP = &confirmed
			g.MyRSVPIsDefault = true
		}
		g.MyRSVPLocked = myRSVPLocked.Valid && myRSVPLocked.Int64 == 1
		if vID.Valid {
			g.Venue = &gameVenueRef{
				ID: int(vID.Int64), Name: vName.String, Street: vStreet.String,
				City: vCity.String, PostalCode: vPostal.String, Note: vNote.String,
			}
		}
		if locksAt, err := gameLocksAt(g.Date, g.Time); err == nil {
			g.RsvpLocksAt = locksAt.Format(time.RFC3339)
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

	// Autorisierung keyed auf echte Claim-Signale (own member / parent-of-child / staff) —
	// NICHT den System-Rollen-String. `claims.Role` ist nur admin/standard/presseteam;
	// "spieler"/"elternteil" sind Vereinsfunktionen, nie Rollen. Der frühere Rollen-Switch
	// ließ jeden Request in den ungeprüften default-Zweig fallen → jeder eingeloggte Nutzer
	// (mit Spiel-Sichtbarkeit) konnte fremde Spiel-RSVP setzen (Broken Access Control).
	ownMemberID, err := h.memberIDForUser(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var memberID int
	if req.MemberID == 0 || req.MemberID == ownMemberID {
		if ownMemberID == 0 {
			http.Error(w, "your account is not linked to a member record", http.StatusUnprocessableEntity)
			return
		}
		memberID = ownMemberID
	} else {
		memberID = req.MemberID
		staff := claims.Role == auth.RoleAdmin || claims.HasFunction("vorstand") || claims.IsTrainerLike()
		if !staff {
			if !claims.IsParent {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			okParent, perr := h.parentHasChild(r.Context(), claims.UserID, req.MemberID)
			if perr != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if !okParent {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
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

	if !claims.CanOverrideRSVPCutoff() {
		var gameDate, gameTime string
		if err := h.db.QueryRowContext(r.Context(),
			`SELECT date(date), substr(time,1,5) FROM games WHERE id = ?`,
			gameID).Scan(&gameDate, &gameTime); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		locksAt, err := gameLocksAt(gameDate, gameTime)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if h.now().After(locksAt) {
			writeRSVPLocked(w, "Spiel kann nur bis 18 Stunden vor Beginn umgesagt werden.", locksAt)
			return
		}
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
	h.broadcastGame(r.Context(), gameID, "games")
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
	IsTrainer        bool    `json:"is_trainer"`
	RsvpStatus       *string `json:"rsvp_status"`
	RsvpIsDefault    bool    `json:"rsvp_is_default,omitempty"`
	Reason           *string `json:"reason,omitempty"`
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
	Total         int               `json:"total"`
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

	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if limit < 1 {
		limit = 200
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}
	if offset < 0 {
		offset = 0
	}
	bypass := claims != nil && (claims.Role == "admin" ||
		claims.HasFunction("trainer") ||
		claims.HasFunction("sportliche_leitung") ||
		claims.HasFunction("vorstand"))

	// Reason-Sichtbarkeit: Trainer/Admin/Vorstand/sL sehen alle, Mitglied nur eigene,
	// Elternteil zusätzlich Zeilen ihrer Kinder.
	var callerMemberID int
	childMemberIDs := map[int]bool{}
	if claims != nil {
		callerMemberID, err = h.memberIDForUser(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
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
	}

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

	// Rollen-Voreinstellungen des Spiels; werden für Zeilen ohne Response virtuell
	// angewandt (Stammkader → rsvp_default_players, Erweiterter Kader →
	// rsvp_default_extended, Trainer immer 'confirmed').
	var defPlayers, defExtended string
	h.db.QueryRowContext(r.Context(),
		`SELECT rsvp_default_players, rsvp_default_extended FROM games WHERE id = ?`, gameID).
		Scan(&defPlayers, &defExtended)

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT member_id, member_name, is_extended, is_trainer, rsvp_status, reason, in_lineup, team_id, cross_team_visible
		FROM (
			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       0 AS is_extended,
			       1 AS is_trainer,
			       gr.status AS rsvp_status,
			       gr.reason AS reason,
			       0 AS in_lineup,
			       k.team_id AS team_id,
			       m.cross_team_visible AS cross_team_visible
			FROM members m
			JOIN kader_trainers kt ON kt.member_id = m.id
			JOIN kader k ON k.id = kt.kader_id
			  AND k.season_id = (SELECT season_id FROM games WHERE id = ?)
			JOIN game_teams gt ON gt.game_id = ? AND gt.team_id = k.team_id
			LEFT JOIN game_responses gr ON gr.game_id = ? AND gr.member_id = m.id

			UNION

			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       0 AS is_extended,
			       0 AS is_trainer,
			       gr.status AS rsvp_status,
			       gr.reason AS reason,
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
			       0 AS is_trainer,
			       gr.status AS rsvp_status,
			       gr.reason AS reason,
			       EXISTS(SELECT 1 FROM game_lineup gl WHERE gl.game_id=? AND gl.member_id=m.id) AS in_lineup,
			       k.team_id AS team_id,
			       m.cross_team_visible AS cross_team_visible
			FROM members m
			JOIN kader_extended_members kem ON kem.member_id = m.id
			JOIN kader k ON k.id = kem.kader_id
			  AND k.season_id = (SELECT season_id FROM games WHERE id = ?)
			JOIN game_teams gt ON gt.game_id = ? AND gt.team_id = k.team_id
			LEFT JOIN game_responses gr ON gr.game_id = ? AND gr.member_id = m.id
		)
		ORDER BY member_name`,
		gameID, gameID, gameID,
		gameID, gameID, gameID, gameID,
		gameID, gameID, gameID, gameID)
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
		var status, reason sql.NullString
		var isExtended, isTrainer, inLineup, ctv int
		rows.Scan(&p.MemberID, &p.MemberName, &isExtended, &isTrainer, &status, &reason, &inLineup, &p.TeamID, &ctv)
		p.IsExtended = isExtended == 1
		p.IsTrainer = isTrainer == 1
		p.InLineup = inLineup == 1
		p.crossTeamVisible = ctv == 1
		canSeeReason := bypass ||
			(callerMemberID > 0 && p.MemberID == callerMemberID) ||
			childMemberIDs[p.MemberID]
		if canSeeReason && reason.Valid && reason.String != "" {
			p.Reason = &reason.String
		}
		if status.Valid {
			p.RsvpStatus = &status.String
		} else if p.IsTrainer {
			confirmed := "confirmed"
			p.RsvpStatus = &confirmed
		} else {
			def := defPlayers
			if p.IsExtended {
				def = defExtended
			}
			if def == "confirmed" || def == "declined" {
				d := def
				p.RsvpStatus = &d
				p.RsvpIsDefault = true
			}
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

	// total = Gesamtzahl der sichtbaren Teilnehmer (nach Sichtbarkeitsfilter,
	// vor limit/offset). Paginierung ist ein reiner Umfangs-Schnitt auf der
	// bereits sichtbaren Menge — dieselben WHERE-/Sichtbarkeitsregeln wie items.
	total := len(items)
	if offset > len(items) {
		offset = len(items)
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	items = items[offset:end]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participantsResponse{Items: items, Total: total, HiddenTeamIDs: hidden})
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

	h.broadcastGame(r.Context(), gameID, "games")
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

// teamIDsFromDB/teamIDsToDB sind die Geschwister von audiencesFromDB/audiencesToDB
// für die Team-Allowlist eines Vorlagen-Items. NULL und [] werden beide als nil
// gelesen — „leer" ist ein einziger Zustand (kein Tri-State) und bedeutet
// „gilt für alle Teams des Spiels".
func teamIDsFromDB(ns sql.NullString) []int {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	var result []int
	json.Unmarshal([]byte(ns.String), &result)
	if len(result) == 0 {
		return nil
	}
	return result
}

func teamIDsToDB(teamIDs []int) *string {
	if len(teamIDs) == 0 {
		return nil
	}
	b, _ := json.Marshal(teamIDs)
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
	ph := strings.Join(placeholders, ",")
	// Two branches: regular squad and extended squad across all of the game's
	// teams. The extended branch excludes members already counted as regular for
	// one of the game's teams so a child in both squads appears exactly once
	// (regular wins). Without an explicit response the role-specific default
	// applies (rsvp_default_players for regular, rsvp_default_extended for
	// extended); 'none' leaves the RSVP empty.
	rows, err := h.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT DISTINCT gt.game_id, m.id, m.first_name || ' ' || m.last_name, gr.status, g.rsvp_default_players, gr.reason, gr.absence_id IS NOT NULL AS locked
		FROM game_teams gt
		JOIN games g ON g.id = gt.game_id
		JOIN kader k ON k.team_id = gt.team_id
		  AND k.season_id = g.season_id
		JOIN kader_members km ON km.kader_id = k.id
		JOIN members m ON m.id = km.member_id
		JOIN family_links fl ON fl.member_id = m.id AND fl.parent_user_id = ?
		LEFT JOIN game_responses gr ON gr.game_id = gt.game_id AND gr.member_id = m.id
		WHERE gt.game_id IN (%s)

		UNION

		SELECT DISTINCT gt.game_id, m.id, m.first_name || ' ' || m.last_name, gr.status, g.rsvp_default_extended, gr.reason, gr.absence_id IS NOT NULL AS locked
		FROM game_teams gt
		JOIN games g ON g.id = gt.game_id
		JOIN kader k ON k.team_id = gt.team_id
		  AND k.season_id = g.season_id
		JOIN kader_extended_members kem ON kem.kader_id = k.id
		JOIN members m ON m.id = kem.member_id
		JOIN family_links fl ON fl.member_id = m.id AND fl.parent_user_id = ?
		LEFT JOIN game_responses gr ON gr.game_id = gt.game_id AND gr.member_id = m.id
		WHERE gt.game_id IN (%s)
		  AND NOT EXISTS (
			SELECT 1 FROM game_teams gt2
			JOIN kader k2 ON k2.team_id = gt2.team_id AND k2.season_id = g.season_id
			JOIN kader_members km2 ON km2.kader_id = k2.id AND km2.member_id = m.id
			WHERE gt2.game_id = gt.game_id
		  )

		ORDER BY 3`, ph, ph),
		append(append(append([]any{parentUserID}, gameIDs...), parentUserID), gameIDs...)...)
	if err != nil {
		return err
	}
	defer rows.Close()

	byGame := map[int][]childRSVP{}
	for rows.Next() {
		var gid int
		var c childRSVP
		var rsvp, reason sql.NullString
		var roleDefault string
		var locked sql.NullInt64
		rows.Scan(&gid, &c.MemberID, &c.Name, &rsvp, &roleDefault, &reason, &locked)
		c.Locked = locked.Valid && locked.Int64 == 1
		if rsvp.Valid {
			s := rsvp.String
			c.RSVP = &s
			if reason.Valid && reason.String != "" {
				r := reason.String
				c.Reason = &r
			}
		} else if roleDefault == "confirmed" || roleDefault == "declined" {
			d := roleDefault
			c.RSVP = &d
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
		 ORDER BY `+appdb.AgeClassSortKey("t.age_class")+`, t.gender, k.team_number`)
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

// gameAttendanceItem ist die Repräsentation eines Kader-Mitglieds in der
// Spiel-Anwesenheitsliste (GET /api/games/{id}/attendances).
type gameAttendanceItem struct {
	MemberID      int     `json:"member_id"`
	MemberName    string  `json:"member_name"`
	IsExtended    bool    `json:"is_extended"`
	IsTrainer     bool    `json:"is_trainer"`
	RSVPStatus    *string `json:"rsvp_status"`
	RSVPIsDefault bool    `json:"rsvp_is_default,omitempty"`
	Reason        *string `json:"reason"`
	Present       *bool   `json:"present"`
}

// canRecordGameAttendance prüft die Authz für Spiel-Anwesenheits-Routen
// gemäß Design D7: admin / sportliche_leitung / Trainer eines beteiligten
// Teams. Vorstand darf nicht (anders als bei game-note).
func (h *Handler) canRecordGameAttendance(ctx context.Context, claims *auth.Claims, gameID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasFunction("sportliche_leitung") {
		return true, nil
	}
	if !claims.HasFunction("trainer") {
		return false, nil
	}
	var trains int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trainer_memberships trm
		JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
		JOIN members m ON m.id = trm.member_id AND m.user_id = ?
		JOIN game_teams gt ON gt.team_id = trm.team_id AND gt.game_id = ?`,
		claims.UserID, gameID).Scan(&trains)
	if err != nil {
		return false, err
	}
	return trains > 0, nil
}

// POST /api/games/{id}/attendances — Bulk-Upsert der Spiel-Anwesenheit.
// Erlaubt: admin, sportliche_leitung, Trainer eines beteiligten Teams.
// Nur für Spiele, deren Datum <= heute liegt.
// declinedMembersForGame liefert alle Mitglieder, die für dieses Spiel abgesagt
// haben — unabhängig davon, ob die Absage automatisch aus einer erfassten
// Abwesenheit stammt (absence_id gesetzt) oder manuell erfolgte (absence_id
// NULL). Beide Formen zählen fachlich als entschuldigtes Fehlen, deshalb darf
// ein `present=false` aus dem Bulk-Save sie nicht auf "fehlt" herabstufen.
func declinedMembersForGame(ctx context.Context, q interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}, gameID int) (map[int]bool, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT member_id FROM game_responses WHERE game_id = ? AND status = 'declined'`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]bool{}
	for rows.Next() {
		var mid int
		if err := rows.Scan(&mid); err != nil {
			return nil, err
		}
		out[mid] = true
	}
	return out, rows.Err()
}

func (h *Handler) SaveAttendances(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var isPastOrToday bool
	err = h.db.QueryRowContext(r.Context(),
		`SELECT date(date) <= date('now') FROM games WHERE id = ?`, gameID).Scan(&isPastOrToday)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ok, err := h.canRecordGameAttendance(r.Context(), claims, gameID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if !isPastOrToday {
		http.Error(w, "attendance can only be recorded for past or current games", http.StatusUnprocessableEntity)
		return
	}

	var entries []struct {
		MemberID int  `json:"member_id"`
		Present  bool `json:"present"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Abgesagte Mitglieder werden gegen ein `present=false` aus dem Bulk-Save
	// geschützt: Das Frontend schickt bei jedem Checkbox-Klick das komplette
	// Roster, unberührte Mitglieder defaulten dabei auf false. Für eine Absage
	// (egal ob automatisch aus einer Abwesenheit oder manuell) wäre das eine
	// stille Herabstufung von "entschuldigt" auf "fehlt". `present=true` bleibt
	// erlaubt — der Trainer kann bewusst erfassen, dass jemand trotz Absage da war.
	declined, err := declinedMembersForGame(r.Context(), tx, gameID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SaveGameAttendances declined lookup: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	wroteAny := false
	for _, e := range entries {
		if !e.Present && declined[e.MemberID] {
			continue
		}
		// Trainer haben keine Anwesenheitserfassung — Ziel-Members, die als Trainer eines
		// beteiligten Teams eingetragen sind und nicht auch als Spieler geführt werden,
		// werden still übersprungen. Die Teilnehmer-Antwort enthält Trainer-Zeilen (eigene
		// Sektion), die der Bulk-Save des Frontends mitschicken kann; ein Trainer-Eintrag
		// darf das Speichern der Spieler nicht blockieren (leeres/trainer-only-Paket → 204).
		var isTrainerOnly int
		if err := tx.QueryRowContext(r.Context(), `
			SELECT CASE
			  WHEN EXISTS (
			    SELECT 1 FROM kader_trainers kt
			    JOIN kader k ON k.id=kt.kader_id AND k.season_id=(SELECT season_id FROM games WHERE id=?)
			    WHERE kt.member_id=? AND k.team_id IN (SELECT team_id FROM game_teams WHERE game_id=?)
			  ) AND NOT EXISTS (
			    SELECT 1 FROM kader_members km
			    JOIN kader k2 ON k2.id=km.kader_id AND k2.season_id=(SELECT season_id FROM games WHERE id=?)
			    WHERE km.member_id=? AND k2.team_id IN (SELECT team_id FROM game_teams WHERE game_id=?)
			  )
			  THEN 1 ELSE 0 END`,
			gameID, e.MemberID, gameID, gameID, e.MemberID, gameID).Scan(&isTrainerOnly); err != nil {
			fmt.Fprintf(os.Stderr, "SaveGameAttendances trainer check: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if isTrainerOnly == 1 {
			continue
		}
		present := 0
		if e.Present {
			present = 1
		}
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO game_attendances (game_id, member_id, present, noted_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(game_id, member_id) DO UPDATE SET present=excluded.present, noted_at=CURRENT_TIMESTAMP`,
			gameID, e.MemberID, present); err != nil {
			fmt.Fprintf(os.Stderr, "SaveGameAttendances upsert: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		wroteAny = true
	}
	// Nur wenn tatsächlich ein Spieler-Eintrag persistiert wurde, kippt das
	// Spiel auf "bewertet". Ein rein-Trainer-Paket ist ein No-op und lässt
	// das Flag ruhen.
	if wroteAny {
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE games SET attendance_tracked=1 WHERE id=?`, gameID); err != nil {
			fmt.Fprintf(os.Stderr, "SaveGameAttendances set tracked: %v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.broadcastGame(r.Context(), gameID, "attendance-changed")
	w.WriteHeader(http.StatusNoContent)
}

// ResetAttendanceTracking — DELETE /api/games/{id}/attendance-tracking
// Setzt games.attendance_tracked=0. Vorhandene game_attendances-Rows bleiben
// unverändert; die Statistik ignoriert sie, solange das Flag 0 ist.
func (h *Handler) ResetAttendanceTracking(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var exists int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT 1 FROM games WHERE id = ?`, gameID).Scan(&exists)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ok, err := h.canRecordGameAttendance(r.Context(), claims, gameID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE games SET attendance_tracked=0 WHERE id=?`, gameID); err != nil {
		fmt.Fprintf(os.Stderr, "ResetGameAttendanceTracking: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.broadcastGame(r.Context(), gameID, "attendance-changed")
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/games/{id}/attendances — Anwesenheitsliste eines Spiels.
// Liefert pro Kader-Mitglied (Stamm + erweitert dedupliziert) RSVP-Status,
// reason und present (nullable). Authz wie bei SaveAttendances.
func (h *Handler) GetAttendances(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var seasonID int
	var defPlayers, defExtended string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT season_id, rsvp_default_players, rsvp_default_extended FROM games WHERE id = ?`, gameID).
		Scan(&seasonID, &defPlayers, &defExtended)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ok, err := h.canRecordGameAttendance(r.Context(), claims, gameID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Reason-Sichtbarkeit: Trainer sehen alle, Mitglied nur eigene Zeile,
	// Elternteil zusätzlich Zeilen ihrer Kinder (family_links).
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

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT member_id, member_name, is_extended, is_trainer, rsvp_status, reason, present
		FROM (
			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       0 AS is_extended,
			       1 AS is_trainer,
			       gr.status AS rsvp_status,
			       gr.reason AS reason,
			       NULL AS present
			FROM members m
			JOIN kader_trainers kt ON kt.member_id = m.id
			JOIN kader k ON k.id = kt.kader_id
			  AND k.season_id = ?
			  AND k.team_id IN (SELECT team_id FROM game_teams WHERE game_id = ?)
			LEFT JOIN game_responses gr ON gr.game_id = ? AND gr.member_id = m.id

			UNION

			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       0 AS is_extended,
			       0 AS is_trainer,
			       gr.status AS rsvp_status,
			       gr.reason AS reason,
			       ga.present AS present
			FROM members m
			JOIN kader_members km ON km.member_id = m.id
			JOIN kader k ON k.id = km.kader_id
			  AND k.season_id = ?
			  AND k.team_id IN (SELECT team_id FROM game_teams WHERE game_id = ?)
			LEFT JOIN game_responses gr ON gr.game_id = ? AND gr.member_id = m.id
			LEFT JOIN game_attendances ga ON ga.game_id = ? AND ga.member_id = m.id
			WHERE NOT EXISTS (
				SELECT 1 FROM kader_trainers kt2
				JOIN kader k2 ON k2.id = kt2.kader_id
				  AND k2.season_id = ?
				  AND k2.team_id IN (SELECT team_id FROM game_teams WHERE game_id = ?)
				WHERE kt2.member_id = m.id
			)

			UNION

			SELECT DISTINCT m.id AS member_id,
			       m.first_name || ' ' || m.last_name AS member_name,
			       1 AS is_extended,
			       0 AS is_trainer,
			       gr.status AS rsvp_status,
			       gr.reason AS reason,
			       ga.present AS present
			FROM members m
			JOIN kader_extended_members kem ON kem.member_id = m.id
			JOIN kader k ON k.id = kem.kader_id
			  AND k.season_id = ?
			  AND k.team_id IN (SELECT team_id FROM game_teams WHERE game_id = ?)
			LEFT JOIN game_responses gr ON gr.game_id = ? AND gr.member_id = m.id
			LEFT JOIN game_attendances ga ON ga.game_id = ? AND ga.member_id = m.id
			WHERE NOT EXISTS (
				SELECT 1 FROM kader_members km2
				JOIN kader k2 ON k2.id = km2.kader_id
				  AND k2.season_id = ?
				  AND k2.team_id IN (SELECT team_id FROM game_teams WHERE game_id = ?)
				WHERE km2.member_id = m.id
			)
			AND NOT EXISTS (
				SELECT 1 FROM kader_trainers kt3
				JOIN kader k3 ON k3.id = kt3.kader_id
				  AND k3.season_id = ?
				  AND k3.team_id IN (SELECT team_id FROM game_teams WHERE game_id = ?)
				WHERE kt3.member_id = m.id
			)
		)
		ORDER BY member_name`,
		seasonID, gameID, gameID,
		seasonID, gameID, gameID, gameID, seasonID, gameID,
		seasonID, gameID, gameID, gameID, seasonID, gameID, seasonID, gameID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetGameAttendances: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Dedupe per member_id: Trainer schlägt Stammkader schlägt erweiterten Kader.
	// (Fachlich gibt es keine Spielertrainer, aber Query kann duplizieren; Priorität
	// robust festhalten.)
	byID := map[int]gameAttendanceItem{}
	order := []int{}
	for rows.Next() {
		var item gameAttendanceItem
		var isExtended, isTrainer int
		var rsvp, reason sql.NullString
		var present sql.NullInt64
		rows.Scan(&item.MemberID, &item.MemberName, &isExtended, &isTrainer, &rsvp, &reason, &present)
		item.IsExtended = isExtended == 1
		item.IsTrainer = isTrainer == 1
		if rsvp.Valid {
			item.RSVPStatus = &rsvp.String
		} else if item.IsTrainer {
			// Trainer sind immer confirmed, unabhängig von der Voreinstellung.
			confirmed := "confirmed"
			item.RSVPStatus = &confirmed
		} else {
			// Rollen-Voreinstellung greift virtuell; 'none' bleibt ohne Status.
			def := defPlayers
			if item.IsExtended {
				def = defExtended
			}
			if def == "confirmed" || def == "declined" {
				d := def
				item.RSVPStatus = &d
				item.RSVPIsDefault = true
			}
		}
		canSeeReason := isTrainerLike ||
			(memberID > 0 && item.MemberID == memberID) ||
			childMemberIDs[item.MemberID]
		if canSeeReason && reason.Valid && reason.String != "" {
			item.Reason = &reason.String
		}
		// Trainer haben keine Anwesenheitserfassung.
		if present.Valid && !item.IsTrainer {
			b := present.Int64 == 1
			item.Present = &b
		}
		if existing, dup := byID[item.MemberID]; dup {
			// Priorität: Trainer > Stammkader > Erweiterter Kader.
			if item.IsTrainer && !existing.IsTrainer {
				byID[item.MemberID] = item
			} else if existing.IsExtended && !item.IsExtended && !existing.IsTrainer {
				byID[item.MemberID] = item
			}
			continue
		}
		byID[item.MemberID] = item
		order = append(order, item.MemberID)
	}

	result := make([]gameAttendanceItem, 0, len(order))
	for _, id := range order {
		result = append(result, byID[id])
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
