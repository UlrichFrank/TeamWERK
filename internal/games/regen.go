package games

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/settings"
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

	// Unassigned lists the rotation demand a day could NOT place, because the team
	// queue ran out before it was covered (bewirtung-kuchen-statt-slots design.md
	// Decision 4). One entry per (day, duty type) carrying the number of Kuchen that
	// fell away — there is no catch-all slot anymore, so this list is the only trace.
	Unassigned []UnassignedEntry `json:"unassigned"`

	// PerGame carries one entry per regenerated game, for callers (the bulk-regen
	// preview) that need to attribute deltas to individual games instead of just a
	// day-level total. Deliberately NOT capped at summaryCap — the row list IS the
	// product of the bulk-regen preview (design.md §9); the day-level lists above
	// keep their existing cap.
	PerGame []GameDelta `json:"per_game"`

	// Notifications carries per-user dispatch intents. Not serialized — the caller
	// fans these out via notify.Send after tx.Commit.
	Notifications []NotificationIntent `json:"-"`
}

// GameDelta attributes one game's regeneration outcome. Created/DeletedAuto are total
// slot capacity (slots_total), matching the unit CreatedEntry/ReducedEntry already use —
// not row counts. AssignmentsKept/AssignmentsLost count individual duty_assignments
// (see restoreAssignments); Conflicts counts this game's entries in RegenSummary.Conflicts.
type GameDelta struct {
	GameID          int `json:"game_id"`
	Created         int `json:"created"`
	DeletedAuto     int `json:"deleted_auto"`
	AssignmentsKept int `json:"assignments_kept"`
	AssignmentsLost int `json:"assignments_lost"`
	Conflicts       int `json:"conflicts"`
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

// UnassignedEntry reports the Kuchen of one day and duty type that could not be handed
// to any team without breaking the per-team cap (see RegenSummary.Unassigned). Count is
// a number of Kuchen, not of slots — the shortfall is deliberately NOT turned into a
// team-less slot.
type UnassignedEntry struct {
	Date     string `json:"date"`
	DutyType string `json:"duty_type"`
	Count    int    `json:"count"`
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
// skip names game IDs to exclude from the mutation (their duty_slots stay untouched) —
// they remain part of the same-day/adjacent-day context, see loadSameDayContextTx and
// design.md §2. nil (or empty) skips nothing, matching the pre-bulk-regen callers.
func (h *Handler) runAutoRegen(ctx context.Context, tx *sql.Tx, dates []string, seasonID int, skip map[int]bool) (RegenSummary, error) {
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
		daySummary, err := h.regenSingleDay(ctx, tx, d, seasonID, skip)
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
//
// skip excludes games from this mutation (see runAutoRegen) but NOT from the same-day
// context computed just below — loadSameDayContextTx is deliberately called without it.
func (h *Handler) regenSingleDay(ctx context.Context, tx *sql.Tx, date string, seasonID int, skip map[int]bool) (RegenSummary, error) {
	allGameTimes, hasPrevDay, hasNextDay, err := h.loadSameDayContextTx(ctx, tx, date, seasonID)
	if err != nil {
		return RegenSummary{}, fmt.Errorf("loadSameDayContext: %w", err)
	}

	dayGames, err := h.loadDayGames(ctx, tx, date, seasonID, skip)
	if err != nil {
		return RegenSummary{}, err
	}

	// Tages-Ausrichter: GENAU EINMAL je Tag aufgelöst, nicht je Spiel
	// (heimspieltag-ausrichter design.md Decision 4). Das Ergebnis ist ein einfacher
	// int, der an beide Gate-Stellen weitergereicht wird — dadurch kann keine der
	// beiden nachträglich noch einmal lesen und die Spiele eines Tages können gar
	// nicht gegen unterschiedliche Werte gegatet werden.
	//
	// Die Auflösung ist total: sie liefert immer einen Ausrichter (expliziter
	// Tageswert, sonst der Default). Fehlt die Default-Zeile, ist das ein Datenfehler
	// und der Lauf bricht laut ab — still weiterlaufen hieße, mit einer nicht
	// existierenden ID zu gaten und damit jedes gebundene Item auszufiltern.
	dayAusrichterID, err := settings.ResolveAusrichterForDay(ctx, tx, date, seasonID)
	if err != nil {
		return RegenSummary{}, fmt.Errorf("resolve ausrichter: %w", err)
	}

	// Tagesweite Bewirtungsrotation: muss VOR der Pro-Spiel-Schleife stehen, weil der
	// Kuchenbedarf von der Gesamtzahl der Heimspiele des Tages abhängt (design.md
	// Decision 1). Ohne rotations-aktivierte Items ist der Plan leer und alles unten
	// verhält sich exakt wie vorher.
	rotation, shortfalls, err := h.buildRotationPlan(ctx, tx, dayGames, dayAusrichterID)
	if err != nil {
		return RegenSummary{}, err
	}

	var summary RegenSummary
	// Nicht zugeteilte Kuchen sind eine Eigenschaft des Tages, nicht eines Spiels — sie
	// werden hier einmal übertragen, nicht in der Pro-Spiel-Schleife (design.md
	// Decision 4). buildRotationPlan kennt das Datum nicht, deshalb erst hier.
	for _, sf := range shortfalls {
		summary.Unassigned = append(summary.Unassigned, UnassignedEntry{
			Date: date, DutyType: sf.DutyType, Count: sf.Count,
		})
	}
	for _, g := range dayGames {
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
		slotsByID, err := h.snapshotDeletedSlots(ctx, tx, g.ID)
		if err != nil {
			return RegenSummary{}, err
		}

		// Step 2: load is_custom=1 slots so we can detect conflicts before inserting.
		customSlots, err := h.snapshotCustomSlots(ctx, tx, g.ID)
		if err != nil {
			return RegenSummary{}, err
		}

		// Step 3: delete is_custom=0 slots (assignments cascade).
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM duty_slots WHERE game_id=? AND is_custom=0`, g.ID); err != nil {
			return RegenSummary{}, fmt.Errorf("delete old slots: %w", err)
		}

		// Step 4: per template item, compute behavior and insert.
		gameSummary, outcomeByOriginalType, err := h.regenGameItems(
			ctx, tx, g, items, durationMins, allGameTimes,
			hasPrevDay, hasNextDay, teamIDs, customSlots, eventName, date, seasonID, rotation,
			dayAusrichterID)
		if err != nil {
			return RegenSummary{}, err
		}
		summary.Created = append(summary.Created, gameSummary.Created...)
		summary.Reduced = append(summary.Reduced, gameSummary.Reduced...)
		summary.Skipped = append(summary.Skipped, gameSummary.Skipped...)
		summary.Conflicts = append(summary.Conflicts, gameSummary.Conflicts...)

		// Step 5: restore duty_assignments whose slot reappeared with an identical
		// (duty_type_id, event_time, team_id) — see restoreAssignments doc comment.
		// Must run after the inserts above so the new slot IDs exist to restore into,
		// and before buildNotificationIntents so restored assignments are excluded.
		assignmentsKept, assignmentsLost, err := h.restoreAssignments(ctx, tx, g.ID, slotsByID)
		if err != nil {
			return RegenSummary{}, fmt.Errorf("restore assignments for game %d: %w", g.ID, err)
		}

		// Step 6: turn NOT-restored deleted-slot user assignments into notification intents.
		notifiedUsers, notifications := buildNotificationIntents(slotsByID, outcomeByOriginalType, eventName, date)
		summary.NotifiedUsers = append(summary.NotifiedUsers, notifiedUsers...)
		summary.Notifications = append(summary.Notifications, notifications...)

		created := 0
		for _, c := range gameSummary.Created {
			created += c.Count
		}
		for _, c := range gameSummary.Reduced {
			created += c.Count
		}
		deletedAuto := 0
		for _, ds := range slotsByID {
			deletedAuto += ds.SlotsTotal
		}
		summary.PerGame = append(summary.PerGame, GameDelta{
			GameID:          g.ID,
			Created:         created,
			DeletedAuto:     deletedAuto,
			AssignmentsKept: assignmentsKept,
			AssignmentsLost: assignmentsLost,
			Conflicts:       len(gameSummary.Conflicts),
		})
	}

	return summary, nil
}

// buildNotificationIntents maps the users of the deleted (is_custom=0) slots to notification
// intents, using the per-original-type outcomes: a slot whose type was reduced yields a
// "variant_changed" intent (carrying the new type name), everything else (skipped, or
// recreated identical) yields "removed". Each user is notified at most once per game.
// Assignments already restored by restoreAssignments (Restored=true) are skipped — a
// restored assignment survived the regen and gets no notification.
func buildNotificationIntents(slotsByID map[int]*deletedSlot, outcomeByOriginalType map[int]itemOutcome, eventName, date string) ([]int, []NotificationIntent) {
	var notifiedUsers []int
	var notifications []NotificationIntent
	notifiedSeen := map[int]bool{}
	for _, ds := range slotsByID {
		if len(ds.Assignments) == 0 {
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
		for _, a := range ds.Assignments {
			if a.Restored || notifiedSeen[a.UserID] {
				continue
			}
			notifiedSeen[a.UserID] = true
			notifiedUsers = append(notifiedUsers, a.UserID)
			notifications = append(notifications, NotificationIntent{
				UserID: a.UserID, Kind: kind,
				EventName: eventName, EventDate: date,
				NewType: newType,
			})
		}
	}
	return notifiedUsers, notifications
}

// customKey identifies a slot of ONE game for conflict detection and restore matching.
// Kein team_id: alle vier Aufrufer sind auf ein einzelnes game_id eingeschränkt, und alle
// Slots eines Termins betreffen dieselben Mannschaften — das Feld könnte hier also nie
// zwei Slots unterscheiden. Es wegzulassen macht die Zusage strukturell, statt sie von
// der Disziplin der Aufrufer abhängig zu machen.
type customKey struct {
	DutyTypeID int
	EventTime  string
}

// makeCustomKey baut den Match-/Konflikt-Schlüssel eines Slots: (duty_type_id,
// event_time) für alle Items, mit und ohne Bewirtungsrotation.
//
// team_id war früher Teil des Schlüssels, weil ein Termin einen Slot je beteiligter
// Mannschaft tragen konnte. Seit spielgebundene Slots grundsätzlich kein Team mehr
// tragen (dienst-slot-team-id-ausbauen), gibt es diese Aufteilung nicht mehr — und der
// Wegfall ist zugleich die Übergangszusage der Migration: ein Bestands-Slot mit
// team_id=A und der an seiner Stelle neu entstehende Slot mit team_id=NULL fallen auf
// denselben Schlüssel, die Zusage überlebt den Umstieg.
//
// Einzige Stelle, die diesen Schlüssel baut — snapshotCustomSlots, loadNewAutoSlotsKeyed,
// restoreAssignments und die Konfliktprüfung in regenGameItems müssen deckungsgleich
// bleiben.
func makeCustomKey(dutyTypeID int, eventTime string) customKey {
	return customKey{DutyTypeID: dutyTypeID, EventTime: eventTime}
}

// snapshotCustomSlots reads the is_custom=1 slots of a game into a set keyed by customKey,
// so the regen can skip inserting a template slot that would collide with a manual one.
func (h *Handler) snapshotCustomSlots(ctx context.Context, tx *sql.Tx, gameID int) (map[customKey]bool, error) {
	customRows, err := tx.QueryContext(ctx, `
		SELECT duty_type_id, event_time
		FROM duty_slots WHERE game_id=? AND is_custom=1`, gameID)
	if err != nil {
		return nil, fmt.Errorf("snapshot custom: %w", err)
	}
	customSlots := map[customKey]bool{}
	for customRows.Next() {
		var dutyTypeID int
		var et sql.NullString
		if err := customRows.Scan(&dutyTypeID, &et); err != nil {
			customRows.Close()
			return nil, err
		}
		customSlots[makeCustomKey(dutyTypeID, et.String)] = true
	}
	customRows.Close()
	return customSlots, nil
}

// deletedAssignment snapshots one duty_assignments row of a to-be-deleted slot, so it can
// be restored (see restoreAssignments) or turned into a notification intent.
type deletedAssignment struct {
	ID          int
	UserID      int
	Status      string
	CashAmount  sql.NullFloat64
	FulfilledAt sql.NullString
	// Restored is set by restoreAssignments once this assignment has been
	// reinserted against a matching new slot; buildNotificationIntents skips it.
	Restored bool
}

// deletedSlot captures an is_custom=0 slot (and its assignments) before deletion, so
// assignments can be restored onto an identical reappearing slot or turned into
// notification intents. Assignments is ordered ascending by original id (oldest first) —
// restoreAssignments relies on this order when capacity shrinks.
type deletedSlot struct {
	DutyTypeID  int
	EventTime   string
	SlotsTotal  int
	Assignments []deletedAssignment
}

// snapshotDeletedSlots reads the is_custom=0 slots of a game together with their
// assignments, keyed by slot id. ORDER BY da.id ensures each slot's Assignments arrive
// oldest-first without an extra sort.
func (h *Handler) snapshotDeletedSlots(ctx context.Context, tx *sql.Tx, gameID int) (map[int]*deletedSlot, error) {
	snapRows, err := tx.QueryContext(ctx, `
		SELECT ds.id, ds.duty_type_id, ds.event_time, ds.slots_total,
		       da.id, da.user_id, da.status, da.cash_amount, da.fulfilled_at
		FROM duty_slots ds
		LEFT JOIN duty_assignments da ON da.duty_slot_id = ds.id
		WHERE ds.game_id=? AND ds.is_custom=0
		ORDER BY da.id`, gameID)
	if err != nil {
		return nil, fmt.Errorf("snapshot deleted: %w", err)
	}
	slotsByID := map[int]*deletedSlot{}
	for snapRows.Next() {
		var slotID int
		var s deletedSlot
		var et sql.NullString
		var aid, uid sql.NullInt64
		var status sql.NullString
		var cashAmount sql.NullFloat64
		var fulfilledAt sql.NullString
		if err := snapRows.Scan(&slotID, &s.DutyTypeID, &et, &s.SlotsTotal,
			&aid, &uid, &status, &cashAmount, &fulfilledAt); err != nil {
			snapRows.Close()
			return nil, err
		}
		if et.Valid {
			s.EventTime = et.String
		}
		existing, ok := slotsByID[slotID]
		if !ok {
			existing = &deletedSlot{DutyTypeID: s.DutyTypeID, EventTime: s.EventTime, SlotsTotal: s.SlotsTotal}
			slotsByID[slotID] = existing
		}
		if aid.Valid {
			existing.Assignments = append(existing.Assignments, deletedAssignment{
				ID: int(aid.Int64), UserID: int(uid.Int64), Status: status.String,
				CashAmount: cashAmount, FulfilledAt: fulfilledAt,
			})
		}
	}
	snapRows.Close()
	return slotsByID, nil
}

// newAutoSlot is one freshly (re-)inserted is_custom=0 slot, loaded after regenGameItems
// so restoreAssignments can match deleted slots against it by (duty_type_id, event_time,
// team_id) and knows its capacity for the fill-up-to-slots_total rule.
type newAutoSlot struct {
	ID         int
	SlotsTotal int
}

// restoreAssignments reinserts duty_assignments of deleted (is_custom=0) slots onto the
// freshly regenerated slots of the same game, wherever a new slot with an identical
// (duty_type_id, event_time, team_id) exists — the same key snapshotCustomSlots/insertOne
// use for conflict detection (design.md §4: "Kein Fuzzy-Matching"). Must run AFTER
// regenGameItems has inserted the new slots.
//
// Per matched slot, at most slots_total assignments are restored, oldest original
// duty_assignments.id first (deletedSlot.Assignments is already in that order) — the
// deterministic "wer zuerst da war"-Regel from design.md §4. Any assignment beyond
// capacity, or whose slot key has no match at all, counts as lost and is left for
// buildNotificationIntents (Restored stays false).
//
// duty_slots.slots_filled is denormalized (no trigger, see duties/handler.go) and is
// updated here for every slot that received at least one restored assignment.
func (h *Handler) restoreAssignments(ctx context.Context, tx *sql.Tx, gameID int, slotsByID map[int]*deletedSlot) (kept, lost int, err error) {
	newSlots, err := h.loadNewAutoSlotsKeyed(ctx, tx, gameID)
	if err != nil {
		return 0, 0, err
	}

	// Deterministic iteration over the deleted slots themselves (map order is not).
	// Only matters for the pathological case of two old slots colliding on the same
	// new-slot key; it does not affect the within-slot oldest-first rule.
	oldSlotIDs := make([]int, 0, len(slotsByID))
	for id := range slotsByID {
		oldSlotIDs = append(oldSlotIDs, id)
	}
	sort.Ints(oldSlotIDs)

	for _, oldID := range oldSlotIDs {
		ds := slotsByID[oldID]
		target := newSlots[makeCustomKey(ds.DutyTypeID, ds.EventTime)]
		for i := range ds.Assignments {
			a := &ds.Assignments[i]
			if target == nil || target.filled >= target.SlotsTotal {
				lost++
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO duty_assignments (duty_slot_id, user_id, status, cash_amount, fulfilled_at)
				VALUES (?,?,?,?,?)`,
				target.ID, a.UserID, a.Status, a.CashAmount, a.FulfilledAt); err != nil {
				return 0, 0, fmt.Errorf("restore assignment %d onto slot %d: %w", a.ID, target.ID, err)
			}
			a.Restored = true
			target.filled++
			kept++
		}
	}

	for _, s := range newSlots {
		if s.filled > 0 {
			if _, err := tx.ExecContext(ctx,
				`UPDATE duty_slots SET slots_filled=? WHERE id=?`, s.filled, s.ID); err != nil {
				return 0, 0, fmt.Errorf("update slots_filled for slot %d: %w", s.ID, err)
			}
		}
	}

	return kept, lost, nil
}

// restoreTarget tracks a newly (re-)created slot's remaining restore capacity while
// restoreAssignments fills it up.
type restoreTarget struct {
	newAutoSlot
	filled int
}

// loadNewAutoSlotsKeyed loads the is_custom=0 slots just (re-)created for a game, keyed by
// the same customKey used for is_custom=1 conflict detection — the restore match key is
// deliberately identical (design.md §4: "derselbe Dreier").
func (h *Handler) loadNewAutoSlotsKeyed(ctx context.Context, tx *sql.Tx, gameID int) (map[customKey]*restoreTarget, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, duty_type_id, event_time, slots_total
		FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID)
	if err != nil {
		return nil, fmt.Errorf("load new auto slots: %w", err)
	}
	defer rows.Close()
	byKey := map[customKey]*restoreTarget{}
	for rows.Next() {
		var id, slotsTotal, dutyTypeID int
		var et sql.NullString
		if err := rows.Scan(&id, &dutyTypeID, &et, &slotsTotal); err != nil {
			return nil, err
		}
		k := makeCustomKey(dutyTypeID, et.String)
		byKey[k] = &restoreTarget{newAutoSlot: newAutoSlot{ID: id, SlotsTotal: slotsTotal}}
	}
	return byKey, nil
}

// dayGame is one row of games for the regen target date+season.
type dayGame struct {
	ID         int
	Time       string
	EndTime    sql.NullString
	Opponent   string
	IsHome     bool
	EventType  string
	TemplateID sql.NullInt64
}

// loadDayGames loads all games for the given date+season, ordered by time then id,
// excluding any game ID present in skip — those stay out of the mutation set entirely
// (design.md §2: they remain part of the same-day context via loadSameDayContextTx,
// which is queried separately and never filtered).
func (h *Handler) loadDayGames(ctx context.Context, tx *sql.Tx, date string, seasonID int, skip map[int]bool) ([]dayGame, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, time, end_time, opponent, is_home, event_type, template_id
		 FROM games WHERE date=? AND season_id=? ORDER BY time, id`,
		date, seasonID)
	if err != nil {
		return nil, fmt.Errorf("load games: %w", err)
	}
	var dayGames []dayGame
	for rows.Next() {
		var g dayGame
		var isHome int
		if err := rows.Scan(&g.ID, &g.Time, &g.EndTime, &g.Opponent, &isHome, &g.EventType, &g.TemplateID); err != nil {
			rows.Close()
			return nil, err
		}
		if skip[g.ID] {
			continue
		}
		g.IsHome = isHome == 1
		dayGames = append(dayGames, g)
	}
	rows.Close()
	return dayGames, nil
}

// rotationAssignment ist die Kuchen-Zuteilung an genau eine Mannschaft: sie bäckt Cakes
// Kuchen, und der zugehörige Slot hängt an ihrem Anker-Spiel (dem Spiel, unter dem diese
// Zuteilung im rotationPlan steht). Cakes ist immer >= 1 — eine Mannschaft ohne Kuchen
// bekommt gar keinen Eintrag.
type rotationAssignment struct {
	TeamID int
	Cakes  int
}

// rotationPlan bildet duty_type_id → Anker-Spiel → Zuteilungen ab. Anker ist das
// chronologisch erste Heimspiel der jeweiligen Mannschaft, das dieses Item trägt
// (bewirtung-kuchen-statt-slots design.md Decision 1/2) — dort und nur dort entsteht ihr
// Slot. Ein Spiel ohne Eintrag bekommt für dieses Item keinen Rotations-Slot; ein Spiel
// mit mehreren Kader-Teams kann mehrere Zuteilungen tragen, daher ein Slice.
type rotationPlan map[int]map[int][]rotationAssignment

// rotationShortfall meldet den Teil des Tagesbedarfs, den die Warteschlange nicht mehr
// aufnehmen konnte. Das Datum fehlt bewusst: buildRotationPlan kennt es nicht,
// regenSingleDay setzt es beim Übertragen in RegenSummary.Unassigned.
type rotationShortfall struct {
	DutyType string
	Count    int
}

// rotationGroup sammelt pro rotations-aktiviertem duty_type_id die Anzahl der Heimspiele
// des Tages, die dieses Item tragen (Basis der Bedarfsrechnung), und die Team-
// Warteschlange: jedes Team genau einmal, an der Position seines ersten solchen
// Heimspiels. anchorByTeam hält zu jedem Team genau dieses Spiel fest — per Konstruktion
// dasselbe, das seine Warteschlangen-Position bestimmt hat, damit Reihenfolge und
// Slot-Ort nicht auseinanderlaufen können (design.md Decision 2).
// Der Cap pro Mannschaft gehört bewusst NICHT hierher: er ist seit
// bewirtung-cap-global vereinsweit und damit für alle Gruppen eines Laufs derselbe.
type rotationGroup struct {
	dutyTypeName string
	gameCount    int
	queue        []int
	inQueue      map[int]bool
	anchorByTeam map[int]int
}

// itemPassesAusrichterGate entscheidet, ob ein Vorlagen-Item am Spieltag mit dem
// aufgelösten Ausrichter dayAusrichterID überhaupt Slots erzeugen darf
// (heimspieltag-ausrichter design.md Decision 4).
//
//	ausrichter_id IS NULL  → die Zeile gilt an jedem Spieltag (Bestandsverhalten,
//	                          rein additiver Change: solange niemand bindet, ist das
//	                          Gate für alle Zeilen offen)
//	ausrichter_id gesetzt  → nur an Spieltagen mit passendem Tages-Ausrichter
//
// Das Gate wirkt AUSSCHLIESSLICH bei event_type='heim'. Auswärts- und generische
// Termine ignorieren die Bindung vollständig: Vorlagen mit template_type != 'heim'
// dürfen das Feld laut Route-Validierung gar nicht erst tragen (Decision 5), dieser
// Zweig ist also eine Sicherung gegen Altbestand/Direkteingriffe, keine Fachlogik.
//
// Einzige Stelle, die diese Bedingung formuliert — beide Gates (buildRotationPlan
// und regenGameItems) müssen deckungsgleich bleiben, sonst rechnet der Bedarf über
// Slots, die anschließend verworfen werden.
func itemPassesAusrichterGate(it templateItemRow, eventType string, dayAusrichterID int) bool {
	if eventType != "heim" || !it.AusrichterID.Valid {
		return true
	}
	return int(it.AusrichterID.Int64) == dayAusrichterID
}

// buildRotationPlan berechnet die tagesweite Bewirtungsrotation VOR der Pro-Spiel-
// Schleife (kuchendienst-rotation design.md Decision 1–3). Nötig, weil der Bedarf
// eines Tages von der Gesamtzahl seiner Heimspiele abhängt — die Pro-Spiel-Schleife
// allein kann das strukturell nicht wissen.
//
// Zugeteilt werden KUCHEN, nicht Slots (bewirtung-kuchen-statt-slots): der Tagesbedarf
// ist eine Zahl Kuchen, die auf möglichst wenige Mannschaften gebündelt wird. Jede
// herangezogene Mannschaft bekommt genau EINEN Slot, dessen slots_total ihre Kuchenzahl
// ist — an ihrem eigenen Termin, nicht am i-ten Spiel des Tages.
//
// Ablauf je Gruppe (gruppiert nach duty_type_id der Items mit rotation_enabled=1):
//  1. Heimspiele des Tages (event_type='heim') in chronologischer Reihenfolge, aber
//     nur die, deren Vorlage dieses Item überhaupt trägt.
//  2. Team-Warteschlange: jedes Team des Spiels tritt bei seinem ersten Auftreten ein
//     (mehrere Teams an einem Spiel treten unabhängig ein, Reihenfolge = DB-Rückgabe
//     von loadGameTeamIDsTx, siehe design.md Risks). Dabei wird sein Anker-Spiel
//     festgehalten — dasselbe Spiel, das die Position bestimmt.
//  3. Bedarf = aufgerundet(Anzahl Heimspiele × Verhältnis). KEINE Deckelung auf die
//     Spieleanzahl mehr: eine Mannschaft trägt jetzt mehrere Kuchen in einem Slot, ein
//     Verhältnis > 1 ist damit ausdrückbar (design.md Decision 6).
//  4. Greedy-Zuteilung entlang der Warteschlange: jede Mannschaft nimmt
//     min(maxPerTeam, Rest) Kuchen, bis der Bedarf gedeckt ist. `maxPerTeam` ist
//     vereinsweit (system_settings) und gilt für alle Gruppen dieses Laufs gleichermaßen.
//  5. Bleibt danach Bedarf übrig, VERFÄLLT er (design.md Decision 4): kein Auffang-Slot
//     ohne Team, kein Überschreiten des Caps — nur ein rotationShortfall für die
//     Zusammenfassung.
//
// Es gibt bewusst KEINEN Zustand über Tagesgrenzen hinweg: jeder Spieltag startet die
// Warteschlange bei Position 1 (Non-Goal aus design.md).
//
// dayAusrichterID ist das GATE #1 des heimspieltag-ausrichter-Changes: es filtert die
// rotations-aktiven Items schon beim Sammeln, also vor Warteschlange und Bedarf (siehe
// die Schleife unten).
func (h *Handler) buildRotationPlan(ctx context.Context, tx *sql.Tx, dayGames []dayGame, dayAusrichterID int) (rotationPlan, []rotationShortfall, error) {
	groups := map[int]*rotationGroup{}

	for _, g := range dayGames {
		if g.EventType != "heim" || !g.TemplateID.Valid {
			continue
		}
		items, err := h.loadTemplateItemsTx(ctx, tx, int(g.TemplateID.Int64))
		if err != nil {
			return nil, nil, fmt.Errorf("rotation: load template %d: %w", g.TemplateID.Int64, err)
		}
		// GATE #1 (heimspieltag-ausrichter design.md Decision 4): das Ausrichter-Gate
		// muss GENAU HIER greifen — im selben Durchlauf, der die rotations-aktiven
		// Items sammelt — und damit vor beidem, was daraus folgt:
		//   - dem Füllen der Team-Warteschlange (unten), und
		//   - der Bedarfsrechnung (demand, weiter unten).
		// Filterte man erst in regenGameItems (Gate #2), rechnete buildRotationPlan
		// den Kuchenbedarf über Heimspiele, deren Slots danach verworfen werden: die
		// Warteschlange verbrauchte Positionen für nie entstehende Slots, und der im
		// RegenSummary ausgewiesene Bedarf wäre schlicht falsch. Gate #2 allein
		// genügt also nicht, Gate #1 allein aber auch nicht — nicht-rotierende Items
		// laufen an dieser Funktion vorbei.
		var rotationItems []templateItemRow
		for _, it := range items {
			if it.RotationEnabled && itemPassesAusrichterGate(it, g.EventType, dayAusrichterID) {
				rotationItems = append(rotationItems, it)
			}
		}
		if len(rotationItems) == 0 {
			continue
		}

		teamIDs, err := h.loadGameTeamIDsTx(ctx, tx, g.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("rotation: load teams for game %d: %w", g.ID, err)
		}

		for _, it := range rotationItems {
			grp := groups[it.DutyTypeID]
			if grp == nil {
				grp = &rotationGroup{
					dutyTypeName: it.DutyTypeName,
					inQueue:      map[int]bool{},
					anchorByTeam: map[int]int{},
				}
				groups[it.DutyTypeID] = grp
			}
			grp.gameCount++
			for _, tid := range teamIDs {
				if !grp.inQueue[tid] {
					grp.inQueue[tid] = true
					grp.queue = append(grp.queue, tid)
					// Anker = das Spiel, mit dem dieses Team in die Warteschlange
					// eintritt. Weil loadDayGames nach (time, id) sortiert, ist das
					// automatisch sein chronologisch erstes passendes Heimspiel.
					grp.anchorByTeam[tid] = g.ID
				}
			}
		}
	}
	if len(groups) == 0 {
		return rotationPlan{}, nil, nil
	}

	// Beide Vereinswerte einmal pro Lauf, in derselben tx wie der Rest des Regens.
	verhaeltnis, err := settings.GetBewirtungVerhaeltnis(ctx, tx)
	if err != nil {
		return nil, nil, fmt.Errorf("rotation: read bewirtung_verhaeltnis: %w", err)
	}
	maxPerTeam, err := settings.GetBewirtungMaxPerTeam(ctx, tx)
	if err != nil {
		return nil, nil, fmt.Errorf("rotation: read bewirtung_max_per_team: %w", err)
	}

	plan := rotationPlan{}
	var shortfalls []rotationShortfall
	// Deterministische Reihenfolge der Gruppen — sonst wechselt die Reihenfolge der
	// shortfalls (und damit der Zusammenfassung) von Lauf zu Lauf.
	dutyTypeIDs := make([]int, 0, len(groups))
	for id := range groups {
		dutyTypeIDs = append(dutyTypeIDs, id)
	}
	sort.Ints(dutyTypeIDs)

	for _, dutyTypeID := range dutyTypeIDs {
		grp := groups[dutyTypeID]
		demand := int(math.Ceil(float64(grp.gameCount) * verhaeltnis))
		if demand < 0 {
			demand = 0
		}

		byGame := map[int][]rotationAssignment{}
		rest := demand
		for _, tid := range grp.queue {
			if rest <= 0 || maxPerTeam <= 0 {
				break
			}
			cakes := maxPerTeam
			if cakes > rest {
				cakes = rest
			}
			anchor := grp.anchorByTeam[tid]
			byGame[anchor] = append(byGame[anchor], rotationAssignment{TeamID: tid, Cakes: cakes})
			rest -= cakes
		}
		if rest > 0 {
			// Warteschlange erschöpft (oder Cap unbrauchbar): der Rest verfällt, statt
			// den Cap zu verletzen oder einen team-losen Slot zu erzeugen.
			shortfalls = append(shortfalls, rotationShortfall{DutyType: grp.dutyTypeName, Count: rest})
		}
		plan[dutyTypeID] = byGame
	}
	return plan, shortfalls, nil
}

// itemAppliesToAnyTeam entscheidet, ob ein Vorlagen-Item für einen Termin gilt: greift
// die Team-Allowlist bei mindestens einer der beteiligten Mannschaften? Leere Allowlist =
// gilt immer. Bei leerer Teamliste (Aufrufer kennt die Teams nicht, oder generisches
// Event) bleibt das Item gültig — nicht filtern ist ehrlicher als raten.
//
// Einzige Quelle dieser Bedingung: Erzeugung (regenGameItems) und Vorschau (PreviewSlots)
// benutzen sie gemeinsam, sonst zeigt die Vorschau Einträge, die real nie entstehen. Die
// frühere Pro-Team-Variante itemAppliesToTeam ist entfallen, seit ein Item je Termin
// höchstens einen Slot erzeugt (dienst-slot-team-id-ausbauen).
func itemAppliesToAnyTeam(allowlist []int, teamIDs []int) bool {
	if len(allowlist) == 0 || len(teamIDs) == 0 {
		return true
	}
	for _, tid := range teamIDs {
		if slices.Contains(allowlist, tid) {
			return true
		}
	}
	return false
}

// itemOutcome records what happened to one template item's duty type, keyed by the
// item's original DutyTypeID, so notification intents can distinguish removed vs. variant-changed.
type itemOutcome struct {
	kind    string // "created" | "reduced" | "skipped"
	newType string // duty_type name after reduction (for "reduced")
}

// regenGameItems runs the per-template-item insertion loop for a single game (after its
// is_custom=0 slots were deleted). It returns a gameSummary carrying only the
// Created/Reduced/Skipped/Conflicts deltas for this game plus the per-original-type outcomes
// used downstream to build notification intents. It performs all duty_slots INSERTs on tx.
func (h *Handler) regenGameItems(
	ctx context.Context, tx *sql.Tx, g dayGame, items []templateItemRow, durationMins int,
	allGameTimes []string, hasPrevDay, hasNextDay bool, teamIDs []int,
	customSlots map[customKey]bool, eventName, date string, seasonID int,
	rotation rotationPlan, dayAusrichterID int,
) (RegenSummary, map[int]itemOutcome, error) {
	var summary RegenSummary
	outcomeByOriginalType := map[int]itemOutcome{}

	for _, it := range items {
		// GATE #2 (heimspieltag-ausrichter design.md Decision 4): an einen anderen
		// Ausrichter gebundene Zeilen erzeugen an diesem Tag nichts. Bewusst ein
		// nacktes `continue` OHNE Eintrag in outcomeByOriginalType — exakt wie der
		// Rotations-Miss und der Allowlist-Miss weiter unten: buildNotificationIntents
		// behandelt einen fehlenden Eintrag als "removed", was für eine Zusage auf
		// einem eben gelöschten Bestandsslot genau richtig ist. Ein eigener
		// Skipped-Eintrag wäre falsch (das meint die Varianten-Logik, nicht "gilt hier
		// nicht"), ein eigener Sonderfall unnötig.
		if !itemPassesAusrichterGate(it, g.EventType, dayAusrichterID) {
			continue
		}

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

		// slotsTotal ist für gewöhnliche Items die Anzahl aus der Vorlage; für
		// rotations-aktive Items überschreibt der Aufrufer sie mit der zugeteilten
		// Kuchenzahl (bewirtung-kuchen-statt-slots: slots_count bleibt dort wirkungslos).
		// team_id wird bewusst nicht gesetzt: der Slot hängt an g.ID, seine
		// Mannschaften stehen in game_teams. Eine Kopie hier wäre eine zweite,
		// einfrierende Quelle für dieselbe Tatsache.
		insertOne := func(slotsTotal int) (bool, error) {
			k := makeCustomKey(resultDutyTypeID, eventTime)
			if customSlots[k] {
				summary.Conflicts = append(summary.Conflicts, ConflictEntry{
					Date: date, DutyTypeID: resultDutyTypeID,
					EventTime: eventTime, GameIDs: []int{g.ID},
				})
				return false, nil
			}
			_, err := tx.ExecContext(ctx, `
				INSERT INTO duty_slots
				  (event_name, event_date, event_time, duty_type_id, role_desc,
				   slots_total, team_id, season_id, game_id, audiences, is_custom)
				VALUES (?,?,?,?,?,?,NULL,?,?,?,0)`,
				eventName, date, eventTime, resultDutyTypeID, "",
				slotsTotal, seasonID, g.ID, slotAudiences)
			if err != nil {
				return false, err
			}
			return true, nil
		}

		// createdCount = Kapazität (slots_total), die dieses Item an diesem Spiel
		// erzeugt hat — je Item höchstens ein Slot. Bei Rotation ist es die Summe der
		// Kuchen, die buildRotationPlan diesem Spiel zugeteilt hat.
		// Die Zählung in der Zusammenfassung darf nicht mehr melden, als entstanden ist.
		createdCount := 0
		switch {
		case it.RotationEnabled:
			// Rotations-Item: welche Mannschaft hier zum Zug kommt und über wie viele
			// Kuchen, hat buildRotationPlan tagesweit entschieden — Einträge stehen nur
			// unter dem Anker-Spiel der jeweiligen Mannschaft. Kein Eintrag (Bedarf
			// gedeckt, oder dieses Spiel ist für niemanden Anker) → wie ein
			// Allowlist-Miss: kein Slot, kein Skipped-Eintrag.
			//
			// Trägt ein Heimspiel ausnahmsweise mehrere Mannschaften, treten sie an
			// derselben Ankerposition ein; ihre Zuteilungen werden hier zu EINEM Slot
			// mit summierten Kuchen verschmolzen, statt zu je einem pro Mannschaft
			// (design.md Decision 4). Ohne team_id wären zwei Slots am selben Termin
			// und derselben Uhrzeit im Restore ohnehin nicht mehr unterscheidbar.
			for _, a := range rotation[it.DutyTypeID][g.ID] {
				createdCount += a.Cakes
			}
			if createdCount > 0 {
				if _, err := insertOne(createdCount); err != nil {
					return RegenSummary{}, nil, err
				}
			}
		default:
			// Ein Slot je Item, nicht je Mannschaft: die Team-Allowlist entscheidet, OB
			// das Item für diesen Termin gilt, nicht für wie viele Mannschaften
			// (dienst-slot-team-id-ausbauen). itemAppliesToAnyTeam ist dieselbe
			// Bedingung, die die Vorschau schon benutzt — Vorschau und Erzeugung fallen
			// damit auf eine Regel zusammen. Generische Events erreichen diesen Zweig
			// defensiv mit leerer Teamliste, für die die Bedingung ebenfalls hält.
			if itemAppliesToAnyTeam(it.TeamIDs, teamIDs) {
				createdCount = n
				if _, err := insertOne(n); err != nil {
					return RegenSummary{}, nil, err
				}
			}
		}
		if createdCount == 0 {
			// Kein Team des Spiels steht in der Allowlist, bzw. die Rotation hat diesem
			// Spiel nichts zugeteilt → dieses Item hat nichts erzeugt und taucht deshalb
			// auch nicht in der Zusammenfassung auf. Der outcome bleibt ungesetzt;
			// buildNotificationIntents behandelt einen fehlenden Eintrag als "removed",
			// was für einen gelöschten Bestandsslot genau richtig ist.
			continue
		}

		if isReduce {
			outcomeByOriginalType[it.DutyTypeID] = itemOutcome{kind: "reduced", newType: resultTypeName}
			summary.Reduced = append(summary.Reduced, ReducedEntry{
				Date: date, From: it.DutyTypeName, To: resultTypeName,
				Count: createdCount,
			})
		} else {
			outcomeByOriginalType[it.DutyTypeID] = itemOutcome{kind: "created"}
			summary.Created = append(summary.Created, CreatedEntry{
				Date: date, DutyType: it.DutyTypeName,
				Count: createdCount,
			})
		}
	}

	return summary, outcomeByOriginalType, nil
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
	dst.Unassigned = append(dst.Unassigned, src.Unassigned...)
	dst.NotifiedUsers = append(dst.NotifiedUsers, src.NotifiedUsers...)
	dst.PerGame = append(dst.PerGame, src.PerGame...)
	dst.Notifications = append(dst.Notifications, src.Notifications...)
}

// capSummary truncates the day-level display lists to summaryCap entries.
// PerGame is deliberately excluded — see the RegenSummary.PerGame doc comment.
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
	if len(s.Unassigned) > summaryCap {
		s.Unassigned = s.Unassigned[:summaryCap]
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

// effectiveEventDurationTx berechnet die Spieldauer in Minuten. Für
// heim/auswärts kommt die Dauer aus der Altersklassen-Regel des Teams; für
// generische Events aus game_templates.duration_minutes.
func (h *Handler) effectiveEventDurationTx(ctx context.Context, tx *sql.Tx, eventType string, templateID, teamID int) (int, error) {
	if eventType == "generisch" {
		var dur int
		err := tx.QueryRowContext(ctx,
			`SELECT duration_minutes FROM game_templates WHERE id=?`, templateID).Scan(&dur)
		if err != nil {
			return 0, fmt.Errorf("vorlage nicht gefunden")
		}
		if dur <= 0 {
			return 0, fmt.Errorf("vorlage hat keine Spieldauer konfiguriert")
		}
		return dur, nil
	}
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
		        gti.audiences, gti.team_ids, gti.rotation_enabled, gti.ausrichter_id
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
		if err := rows.Scan(&it.DutyTypeID, &it.DutyTypeName, &it.Anchor, &it.OffsetMinutes,
			&it.SlotsCount, &it.SameDayBehavior, &it.SameDayVariantID,
			&it.AdjacentDayBehavior, &it.AdjacentDayVariantID, &it.Audiences, &teamIDs,
			&it.RotationEnabled, &it.AusrichterID); err != nil {
			return nil, err
		}
		it.TeamIDs = teamIDsFromDB(teamIDs)
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
