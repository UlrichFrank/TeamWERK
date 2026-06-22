package games

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/notify"
)

// RegenSummary aggregates the effect of an auto-regen window on duty_slots.
// All list fields are capped at summaryCap entries; truncation is signaled implicitly
// by the frontend ("… und N weitere Änderungen").
type RegenSummary struct {
	Created       []CreatedEntry  `json:"created"`
	Reduced       []ReducedEntry  `json:"reduced"`
	Skipped       []SkippedEntry  `json:"skipped"`
	NotifiedUsers []int           `json:"notified_users"`
	Conflicts     []ConflictEntry `json:"conflicts"`

	// Notifications carries per-user dispatch intents. Not serialized — the caller
	// fans these out via notify.Send after tx.Commit.
	Notifications []NotificationIntent `json:"-"`
}

type CreatedEntry struct {
	Date     string `json:"date"`
	DutyType string `json:"duty_type"`
	Count    int    `json:"count"`
}

type ReducedEntry struct {
	Date  string `json:"date"`
	From  string `json:"from"`
	To    string `json:"to"`
	Count int    `json:"count"`
}

type SkippedEntry struct {
	Date     string `json:"date"`
	DutyType string `json:"duty_type"`
}

type ConflictEntry struct {
	Date       string `json:"date"`
	DutyTypeID int    `json:"duty_type_id"`
	EventTime  string `json:"event_time"`
	GameIDs    []int  `json:"game_ids,omitempty"`
}

type NotificationIntent struct {
	UserID    int
	Kind      string // "removed" | "variant_changed"
	EventName string
	EventDate string
	NewType   string // only set for variant_changed
}

const summaryCap = 20

// dateWindow returns the day before, the day itself, and the day after (ISO yyyy-mm-dd).
// Tolerates ISO timestamp inputs like "2026-05-30T00:00:00Z" by slicing to the date part.
// Always returns exactly three entries — on parse failure it returns three copies of the
// normalized input so callers can index safely (window[0], window[2]).
func dateWindow(date string) []string {
	if len(date) > 10 {
		date = date[:10]
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return []string{date, date, date}
	}
	prev := t.AddDate(0, 0, -1).Format("2006-01-02")
	next := t.AddDate(0, 0, 1).Format("2006-01-02")
	return []string{prev, date, next}
}

// runAutoRegen regenerates duty slots for the union of given dates.
// All reads and writes go through tx so the regen sees uncommitted game mutations.
func (h *Handler) runAutoRegen(ctx context.Context, tx *sql.Tx, dates []string, seasonID int) (RegenSummary, error) {
	seen := map[string]bool{}
	unique := make([]string, 0, len(dates))
	for _, d := range dates {
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		unique = append(unique, d)
	}
	sort.Strings(unique)

	var summary RegenSummary
	for _, d := range unique {
		daySummary, err := h.regenSingleDay(ctx, tx, d, seasonID)
		if err != nil {
			return RegenSummary{}, fmt.Errorf("regen %s: %w", d, err)
		}
		mergeSummary(&summary, daySummary)
	}
	capSummary(&summary)
	return summary, nil
}

// regenSingleDay regenerates all template-derived duty_slots for a given date+season.
// Per-game flow:
//  1. Snapshot the to-be-deleted is_custom=0 slots (user_id, duty_type_id, event_time, name).
//  2. Delete those slots — duty_assignments cascade away.
//  3. For each template item: compute event_time, applyBehavior, then either skip,
//     insert (with potential conflict against is_custom=1 slots), or insert variant.
//  4. Match deleted-slot users to "removed" or "variant_changed" notification intents.
func (h *Handler) regenSingleDay(ctx context.Context, tx *sql.Tx, date string, seasonID int) (RegenSummary, error) {
	allGameTimes, hasPrevDay, hasNextDay, err := h.loadSameDayContextTx(ctx, tx, date, seasonID)
	if err != nil {
		return RegenSummary{}, fmt.Errorf("loadSameDayContext: %w", err)
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT id, time, end_time, opponent, is_home, event_type, template_id
		 FROM games WHERE date=? AND season_id=? ORDER BY time, id`,
		date, seasonID)
	if err != nil {
		return RegenSummary{}, fmt.Errorf("load games: %w", err)
	}
	type dayGame struct {
		ID         int
		Time       string
		EndTime    sql.NullString
		Opponent   string
		IsHome     bool
		EventType  string
		TemplateID sql.NullInt64
	}
	var dayGames []dayGame
	for rows.Next() {
		var g dayGame
		var isHome int
		if err := rows.Scan(&g.ID, &g.Time, &g.EndTime, &g.Opponent, &isHome, &g.EventType, &g.TemplateID); err != nil {
			rows.Close()
			return RegenSummary{}, err
		}
		g.IsHome = isHome == 1
		dayGames = append(dayGames, g)
	}
	rows.Close()

	var summary RegenSummary
	for _, g := range dayGames {
		// Generic events have no template — their is_custom=1 slots stay, nothing to derive.
		if g.EventType == "generisch" {
			continue
		}

		// templateID == 0 bedeutet: keine Auto-Slots für dieses Event. Es gibt
		// keinen Fallback mehr (frühere `findTemplateForGameTx`-Auflösung auf
		// die kleinste passende Template-ID entfällt). Existierende
		// is_custom=0-Slots werden trotzdem gelöscht, damit ein Wechsel auf
		// "keine Vorlage" sichtbar wird; is_custom=1-Slots bleiben unberührt.
		templateID := 0
		if g.TemplateID.Valid {
			templateID = int(g.TemplateID.Int64)
		}

		teamIDs, err := h.loadGameTeamIDsTx(ctx, tx, g.ID)
		if err != nil {
			return RegenSummary{}, fmt.Errorf("load teams for game %d: %w", g.ID, err)
		}
		firstTeamID := 0
		if len(teamIDs) > 0 {
			firstTeamID = teamIDs[0]
		}

		var items []templateItemRow
		var durationMins int
		if templateID > 0 {
			durationMins, err = h.effectiveEventDurationTx(ctx, tx, g.EventType, templateID, firstTeamID)
			if err != nil {
				// Duration unknown → can't position slots safely. Skip this game.
				continue
			}
			items, err = h.loadTemplateItemsTx(ctx, tx, templateID)
			if err != nil {
				return RegenSummary{}, fmt.Errorf("load template %d: %w", templateID, err)
			}
		}

		eventName := composeEventName(g.EventType, g.IsHome, g.Opponent)

		// Step 1: snapshot to-be-deleted slots with their assignments.
		type deletedSlot struct {
			DutyTypeID int
			EventTime  string
			TeamID     sql.NullInt64
			UserIDs    []int
		}
		snapRows, err := tx.QueryContext(ctx, `
			SELECT ds.id, ds.duty_type_id, ds.event_time, ds.team_id, da.user_id
			FROM duty_slots ds
			LEFT JOIN duty_assignments da ON da.duty_slot_id = ds.id
			WHERE ds.game_id=? AND ds.is_custom=0`, g.ID)
		if err != nil {
			return RegenSummary{}, fmt.Errorf("snapshot deleted: %w", err)
		}
		slotsByID := map[int]*deletedSlot{}
		for snapRows.Next() {
			var slotID int
			var s deletedSlot
			var et sql.NullString
			var uid sql.NullInt64
			if err := snapRows.Scan(&slotID, &s.DutyTypeID, &et, &s.TeamID, &uid); err != nil {
				snapRows.Close()
				return RegenSummary{}, err
			}
			if et.Valid {
				s.EventTime = et.String
			}
			existing, ok := slotsByID[slotID]
			if !ok {
				existing = &deletedSlot{DutyTypeID: s.DutyTypeID, EventTime: s.EventTime, TeamID: s.TeamID}
				slotsByID[slotID] = existing
			}
			if uid.Valid {
				existing.UserIDs = append(existing.UserIDs, int(uid.Int64))
			}
		}
		snapRows.Close()

		// Step 2: load is_custom=1 slots so we can detect conflicts before inserting.
		customRows, err := tx.QueryContext(ctx, `
			SELECT duty_type_id, event_time, team_id
			FROM duty_slots WHERE game_id=? AND is_custom=1`, g.ID)
		if err != nil {
			return RegenSummary{}, fmt.Errorf("snapshot custom: %w", err)
		}
		type customKey struct {
			DutyTypeID int
			EventTime  string
			TeamID     int64
			HasTeam    bool
		}
		customSlots := map[customKey]bool{}
		for customRows.Next() {
			var k customKey
			var et sql.NullString
			var tid sql.NullInt64
			if err := customRows.Scan(&k.DutyTypeID, &et, &tid); err != nil {
				customRows.Close()
				return RegenSummary{}, err
			}
			if et.Valid {
				k.EventTime = et.String
			}
			if tid.Valid {
				k.TeamID = tid.Int64
				k.HasTeam = true
			}
			customSlots[k] = true
		}
		customRows.Close()

		// Step 3: delete is_custom=0 slots (assignments cascade).
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM duty_slots WHERE game_id=? AND is_custom=0`, g.ID); err != nil {
			return RegenSummary{}, fmt.Errorf("delete old slots: %w", err)
		}

		// Step 4: per template item, compute behavior and insert.
		type itemOutcome struct {
			kind    string // "created" | "reduced" | "skipped"
			newType string // duty_type name after reduction (for "reduced")
		}
		outcomeByOriginalType := map[int]itemOutcome{}

		for _, it := range items {
			var eventTime string
			if it.Anchor == "end" && g.EndTime.Valid {
				eventTime = addMinutes(g.EndTime.String, it.OffsetMinutes)
			} else {
				offset := it.OffsetMinutes
				if it.Anchor == "end" {
					offset += durationMins
				}
				eventTime = addMinutes(g.Time, offset)
			}

			isBefore, isAfter, isBetween := classifySlotPosition(eventTime, g.Time, allGameTimes)
			resultDutyTypeID := applyBehavior(it, g.Time, eventTime, allGameTimes,
				hasPrevDay, hasNextDay, isBefore, isAfter, isBetween)

			if resultDutyTypeID == -1 {
				outcomeByOriginalType[it.DutyTypeID] = itemOutcome{kind: "skipped"}
				summary.Skipped = append(summary.Skipped, SkippedEntry{
					Date: date, DutyType: it.DutyTypeName,
				})
				continue
			}

			resultTypeName := it.DutyTypeName
			isReduce := resultDutyTypeID != it.DutyTypeID
			if isReduce {
				name, lerr := h.lookupDutyTypeNameTx(ctx, tx, resultDutyTypeID)
				if lerr == nil {
					resultTypeName = name
				}
			}

			n := it.SlotsCount
			if n <= 0 {
				n = 1
			}
			slotAudiences := audiencesToDB(audiencesFromDB(it.Audiences))

			insertOne := func(teamID sql.NullInt64) error {
				k := customKey{DutyTypeID: resultDutyTypeID, EventTime: eventTime}
				if teamID.Valid {
					k.TeamID = teamID.Int64
					k.HasTeam = true
				}
				if customSlots[k] {
					summary.Conflicts = append(summary.Conflicts, ConflictEntry{
						Date: date, DutyTypeID: resultDutyTypeID,
						EventTime: eventTime, GameIDs: []int{g.ID},
					})
					return nil
				}
				var teamVal any
				if teamID.Valid {
					teamVal = teamID.Int64
				}
				_, err := tx.ExecContext(ctx, `
					INSERT INTO duty_slots
					  (event_name, event_date, event_time, duty_type_id, role_desc,
					   slots_total, team_id, season_id, game_id, audiences, is_custom)
					VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
					eventName, date, eventTime, resultDutyTypeID, "",
					n, teamVal, seasonID, g.ID, slotAudiences)
				return err
			}

			if g.EventType == "generisch" {
				// generisch never reaches here (skipped above), but kept defensive.
				if err := insertOne(sql.NullInt64{}); err != nil {
					return RegenSummary{}, err
				}
			} else {
				for _, tid := range teamIDs {
					if err := insertOne(sql.NullInt64{Int64: int64(tid), Valid: true}); err != nil {
						return RegenSummary{}, err
					}
				}
			}

			if isReduce {
				outcomeByOriginalType[it.DutyTypeID] = itemOutcome{kind: "reduced", newType: resultTypeName}
				summary.Reduced = append(summary.Reduced, ReducedEntry{
					Date: date, From: it.DutyTypeName, To: resultTypeName,
					Count: max(1, len(teamIDs)) * n,
				})
			} else {
				outcomeByOriginalType[it.DutyTypeID] = itemOutcome{kind: "created"}
				summary.Created = append(summary.Created, CreatedEntry{
					Date: date, DutyType: it.DutyTypeName,
					Count: max(1, len(teamIDs)) * n,
				})
			}
		}

		// Step 5: turn deleted-slot user assignments into notification intents.
		notifiedSeen := map[int]bool{}
		for _, ds := range slotsByID {
			if len(ds.UserIDs) == 0 {
				continue
			}
			outcome, ok := outcomeByOriginalType[ds.DutyTypeID]
			kind := "removed"
			newType := ""
			if ok {
				switch outcome.kind {
				case "skipped":
					kind = "removed"
				case "reduced":
					kind = "variant_changed"
					newType = outcome.newType
				case "created":
					// Slot recreated identical-type — user assignment still gone (we deleted
					// the slot), so treat as removed. Could be no-op-noisy in rare edge case
					// but better than silent loss.
					kind = "removed"
				}
			}
			for _, uid := range ds.UserIDs {
				if notifiedSeen[uid] {
					continue
				}
				notifiedSeen[uid] = true
				summary.NotifiedUsers = append(summary.NotifiedUsers, uid)
				summary.Notifications = append(summary.Notifications, NotificationIntent{
					UserID: uid, Kind: kind,
					EventName: eventName, EventDate: date,
					NewType: newType,
				})
			}
		}
	}

	return summary, nil
}

func composeEventName(eventType string, isHome bool, opponent string) string {
	var name string
	switch eventType {
	case "heim":
		name = "Heimspiel"
	case "auswärts":
		name = "Auswärtsspiel"
	case "generisch":
		name = opponent
	default:
		if isHome {
			name = "Heimspiel"
		} else {
			name = "Auswärtsspiel"
		}
	}
	if eventType != "generisch" && opponent != "" {
		name += " vs. " + opponent
	}
	return name
}

func mergeSummary(dst *RegenSummary, src RegenSummary) {
	dst.Created = append(dst.Created, src.Created...)
	dst.Reduced = append(dst.Reduced, src.Reduced...)
	dst.Skipped = append(dst.Skipped, src.Skipped...)
	dst.Conflicts = append(dst.Conflicts, src.Conflicts...)
	dst.NotifiedUsers = append(dst.NotifiedUsers, src.NotifiedUsers...)
	dst.Notifications = append(dst.Notifications, src.Notifications...)
}

func capSummary(s *RegenSummary) {
	if len(s.Created) > summaryCap {
		s.Created = s.Created[:summaryCap]
	}
	if len(s.Reduced) > summaryCap {
		s.Reduced = s.Reduced[:summaryCap]
	}
	if len(s.Skipped) > summaryCap {
		s.Skipped = s.Skipped[:summaryCap]
	}
	if len(s.Conflicts) > summaryCap {
		s.Conflicts = s.Conflicts[:summaryCap]
	}
	if len(s.NotifiedUsers) > summaryCap {
		s.NotifiedUsers = s.NotifiedUsers[:summaryCap]
	}
}

// ── tx-aware variants of existing helpers ────────────────────────────────────

func (h *Handler) loadSameDayContextTx(ctx context.Context, tx *sql.Tx, gameDate string, seasonID int) (
	allGameTimes []string, hasPrevDay, hasNextDay bool, err error,
) {
	rows, err := tx.QueryContext(ctx,
		`SELECT time FROM games WHERE date=? AND season_id=? ORDER BY time`,
		gameDate, seasonID)
	if err != nil {
		return nil, false, false, err
	}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return nil, false, false, err
		}
		allGameTimes = append(allGameTimes, t)
	}
	rows.Close()

	seen := map[string]bool{}
	unique := make([]string, 0, len(allGameTimes))
	for _, t := range allGameTimes {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}
	allGameTimes = unique

	var prev, next int
	tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM games WHERE date=date(?, '-1 days') AND is_home=1 AND season_id=?`,
		gameDate, seasonID).Scan(&prev)
	tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM games WHERE date=date(?, '+1 days') AND is_home=1 AND season_id=?`,
		gameDate, seasonID).Scan(&next)
	return allGameTimes, prev > 0, next > 0, nil
}

func (h *Handler) loadGameTeamIDsTx(ctx context.Context, tx *sql.Tx, gameID int) ([]int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT team_id FROM game_teams WHERE game_id=?`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// effectiveEventDurationTx berechnet die Spieldauer in Minuten. Wird nur für
// heim/auswärts-Events mit gesetztem template_id aufgerufen — der frühere
// generisch-Zweig (Dauer aus game_templates.duration_minutes) ist obsolet,
// seit generische Events im Auto-Regen früher übersprungen werden.
func (h *Handler) effectiveEventDurationTx(ctx context.Context, tx *sql.Tx, eventType string, templateID, teamID int) (int, error) {
	_ = templateID
	if eventType != "heim" && eventType != "auswärts" {
		return 0, fmt.Errorf("unerwarteter event_type %q im Auto-Regen", eventType)
	}
	var ageClass sql.NullString
	tx.QueryRowContext(ctx, `SELECT age_class FROM teams WHERE id=?`, teamID).Scan(&ageClass)
	if !ageClass.Valid || ageClass.String == "" {
		return 0, fmt.Errorf("team hat keine Altersklasse")
	}
	var half, brk int
	err := tx.QueryRowContext(ctx,
		`SELECT half_duration_minutes, break_minutes FROM age_class_game_rules WHERE age_class=?`,
		ageClass.String).Scan(&half, &brk)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("keine Altersklassen-Regel für %s", ageClass.String)
	}
	if err != nil {
		return 0, err
	}
	return 2*half + brk, nil
}

func (h *Handler) loadTemplateItemsTx(ctx context.Context, tx *sql.Tx, templateID int) ([]templateItemRow, error) {
	rows, err := tx.QueryContext(ctx,
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
		if err := rows.Scan(&it.DutyTypeID, &it.DutyTypeName, &it.Anchor, &it.OffsetMinutes,
			&it.SlotsCount, &it.SameDayBehavior, &it.SameDayVariantID,
			&it.AdjacentDayBehavior, &it.AdjacentDayVariantID, &it.Audiences); err != nil {
			return nil, err
		}
		result = append(result, it)
	}
	return result, nil
}

func (h *Handler) lookupDutyTypeNameTx(ctx context.Context, tx *sql.Tx, id int) (string, error) {
	var name string
	err := tx.QueryRowContext(ctx, `SELECT name FROM duty_types WHERE id=?`, id).Scan(&name)
	return name, err
}

// dispatchRegenNotifications fans out one notify.Send per intent. Must be called
// AFTER tx.Commit so users never see notifications about a rolled-back change.
func (h *Handler) dispatchRegenNotifications(summary RegenSummary) {
	for _, n := range summary.Notifications {
		var body string
		switch n.Kind {
		case "variant_changed":
			body = fmt.Sprintf("Dein Dienst zum %s am %s wurde zur Variante %s geändert. Bitte überprüfe deinen Dienstplan.",
				n.EventName, formatDateDMY(n.EventDate), n.NewType)
		default:
			body = fmt.Sprintf("Dein Dienst zum %s am %s wurde aufgrund einer Spielplanänderung entfernt.",
				n.EventName, formatDateDMY(n.EventDate))
		}
		go notify.Send(h.db, h.cfg, []int{n.UserID}, "duties", "Dienst angepasst", body, "/dienste")
	}
}
