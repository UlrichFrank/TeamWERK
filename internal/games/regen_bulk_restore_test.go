package games

// Interner Test-Package — siehe regen_bulk_context_test.go für die Begründung und
// die geteilten Test-Helfer (insertDutyTypeI, insertTemplateI, insertGameI, regenDate, …).

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func assignI(t *testing.T, db *sql.DB, slotID, userID int, status string, cashAmount any, fulfilledAt any) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO duty_assignments (duty_slot_id, user_id, status, cash_amount, fulfilled_at) VALUES (?, ?, ?, ?, ?)`,
		slotID, userID, status, cashAmount, fulfilledAt)
	if err != nil {
		t.Fatalf("assignI: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func assignmentByUser(t *testing.T, db *sql.DB, slotID, userID int) (status string, cashAmount sql.NullFloat64, fulfilledAt sql.NullString, found bool) {
	t.Helper()
	err := db.QueryRow(
		`SELECT status, cash_amount, fulfilled_at FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, userID).Scan(&status, &cashAmount, &fulfilledAt)
	if err == sql.ErrNoRows {
		return "", sql.NullFloat64{}, sql.NullString{}, false
	}
	if err != nil {
		t.Fatalf("assignmentByUser: %v", err)
	}
	return status, cashAmount, fulfilledAt, true
}

// TestRegen_PerGameZweiSpieleGetrenntGezaehlt (duty-bulk-regen tasks.md 2.3): a regen over
// two games on the same day must attribute Created/DeletedAuto separately per game, not
// merge them into a single day-level number.
func TestRegen_PerGameZweiSpieleGetrenntGezaehlt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyA := insertDutyTypeI(t, db, "Getränke", "", 0, "", 0)
	templateA := insertTemplateI(t, db, dutyA, 0, 1)
	dutyB := insertDutyTypeI(t, db, "Kasse", "", 0, "", 0)
	templateB := insertTemplateI(t, db, dutyB, 0, 2)

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", templateA)
	game2 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "16:00", templateB)

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if len(summary.PerGame) != 2 {
		t.Fatalf("expected 2 PerGame entries, got %d: %+v", len(summary.PerGame), summary.PerGame)
	}
	byGame := map[int]GameDelta{}
	for _, pg := range summary.PerGame {
		byGame[pg.GameID] = pg
	}
	pg1, ok1 := byGame[game1]
	pg2, ok2 := byGame[game2]
	if !ok1 || !ok2 {
		t.Fatalf("expected entries for both game1=%d and game2=%d, got %+v", game1, game2, summary.PerGame)
	}
	if pg1.Created != 1 {
		t.Errorf("game1: expected created=1 (slots_count=1), got %d", pg1.Created)
	}
	if pg2.Created != 2 {
		t.Errorf("game2: expected created=2 (slots_count=2), got %d", pg2.Created)
	}
	if pg1.DeletedAuto != 0 || pg2.DeletedAuto != 0 {
		t.Errorf("expected deleted_auto=0 on first-ever regen, got game1=%d game2=%d", pg1.DeletedAuto, pg2.DeletedAuto)
	}
}

// TestRegen_ZuweisungUeberlebtIdentischeRegeneration (tasks.md 3.6): a slot regenerated
// with an unchanged (duty_type_id, event_time, team_id) must restore its assignment,
// keep slots_filled accurate, and trigger no notification.
func TestRegen_ZuweisungUeberlebtIdentischeRegeneration(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	dutyType := insertDutyTypeI(t, db, "Aufbau", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyType, -60, 1)
	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "14:00", templateID)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	oldSlotID := autoSlotID(t, db, gameID)
	assignI(t, db, oldSlotID, userID, "assigned", nil, nil)

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	newSlotID := autoSlotID(t, db, gameID)
	status, _, _, found := assignmentByUser(t, db, newSlotID, userID)
	if !found {
		t.Fatalf("expected assignment to survive identical regen on slot %d", newSlotID)
	}
	if status != "assigned" {
		t.Errorf("expected status 'assigned', got %q", status)
	}
	var filled int
	if err := db.QueryRow(`SELECT slots_filled FROM duty_slots WHERE id=?`, newSlotID).Scan(&filled); err != nil {
		t.Fatalf("read slots_filled: %v", err)
	}
	if filled != 1 {
		t.Errorf("expected slots_filled=1, got %d", filled)
	}
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			t.Errorf("expected no notification for restored assignment, but user %d was notified", userID)
		}
	}
}

// TestRegen_CashSubstituteBleibtErhalten (tasks.md 3.6): status and cash_amount of a
// cash-substitute assignment survive an identical regen.
func TestRegen_CashSubstituteBleibtErhalten(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	dutyType := insertDutyTypeI(t, db, "Aufbau", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyType, -60, 1)
	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "14:00", templateID)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	oldSlotID := autoSlotID(t, db, gameID)
	assignI(t, db, oldSlotID, userID, "cash_substitute", 12.5, "2026-06-01 10:00:00")

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	newSlotID := autoSlotID(t, db, gameID)
	status, cashAmount, fulfilledAt, found := assignmentByUser(t, db, newSlotID, userID)
	if !found {
		t.Fatalf("expected cash_substitute assignment to survive identical regen")
	}
	if status != "cash_substitute" {
		t.Errorf("expected status 'cash_substitute', got %q", status)
	}
	if !cashAmount.Valid || cashAmount.Float64 != 12.5 {
		t.Errorf("expected cash_amount=12.5, got %+v", cashAmount)
	}
	if !fulfilledAt.Valid || fulfilledAt.String != "2026-06-01T10:00:00Z" {
		t.Errorf("expected fulfilled_at preserved, got %+v", fulfilledAt)
	}
}

// TestRegen_SchrumpfungBehaeltAelteste (tasks.md 3.6): when a slot shrinks from 3 to 2
// seats, the two oldest assignments (by original duty_assignments.id) survive and exactly
// the third is notified as removed.
func TestRegen_SchrumpfungBehaeltAelteste(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)
	u1 := testutil.CreateUser(t, db, "standard")
	u2 := testutil.CreateUser(t, db, "standard")
	u3 := testutil.CreateUser(t, db, "standard")

	dutyType := insertDutyTypeI(t, db, "Aufbau", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyType, -60, 3)
	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "14:00", templateID)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	oldSlotID := autoSlotID(t, db, gameID)
	assignI(t, db, oldSlotID, u1, "assigned", nil, nil) // ascending IDs: u1 oldest
	assignI(t, db, oldSlotID, u2, "assigned", nil, nil)
	assignI(t, db, oldSlotID, u3, "assigned", nil, nil) // youngest

	if _, err := db.Exec(`UPDATE game_template_items SET slots_count=2 WHERE template_id=?`, templateID); err != nil {
		t.Fatalf("shrink template: %v", err)
	}
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	newSlotID := autoSlotID(t, db, gameID)
	if _, _, _, found := assignmentByUser(t, db, newSlotID, u1); !found {
		t.Errorf("expected oldest assignee u1=%d to survive the shrink", u1)
	}
	if _, _, _, found := assignmentByUser(t, db, newSlotID, u2); !found {
		t.Errorf("expected second-oldest assignee u2=%d to survive the shrink", u2)
	}
	if _, _, _, found := assignmentByUser(t, db, newSlotID, u3); found {
		t.Errorf("expected youngest assignee u3=%d to NOT survive the shrink", u3)
	}
	var filled int
	if err := db.QueryRow(`SELECT slots_filled FROM duty_slots WHERE id=?`, newSlotID).Scan(&filled); err != nil {
		t.Fatalf("read slots_filled: %v", err)
	}
	if filled != 2 {
		t.Errorf("expected slots_filled=2, got %d", filled)
	}
	notified := map[int]bool{}
	for _, uid := range summary.NotifiedUsers {
		notified[uid] = true
	}
	if notified[u1] || notified[u2] {
		t.Errorf("expected u1/u2 (restored) to NOT be notified, got NotifiedUsers=%v", summary.NotifiedUsers)
	}
	if !notified[u3] {
		t.Errorf("expected u3 (not restored) to be notified, got NotifiedUsers=%v", summary.NotifiedUsers)
	}
}

// TestRegen_VerschobeneUhrzeitLoestZuweisung (tasks.md 3.6): a slot whose event_time
// shifts (template offset change) does not match its predecessor — the assignment is
// lost, not restored, and the user is notified.
func TestRegen_VerschobeneUhrzeitLoestZuweisung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	dutyType := insertDutyTypeI(t, db, "Aufbau", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyType, -60, 1)
	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "14:00", templateID)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	oldSlotID := autoSlotID(t, db, gameID)
	assignI(t, db, oldSlotID, userID, "assigned", nil, nil)

	if _, err := db.Exec(`UPDATE game_template_items SET offset_minutes=-30 WHERE template_id=?`, templateID); err != nil {
		t.Fatalf("shift offset: %v", err)
	}
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	newSlotID := autoSlotID(t, db, gameID)
	if newSlotID == oldSlotID {
		t.Fatalf("expected a new slot id after regen")
	}
	if _, _, _, found := assignmentByUser(t, db, newSlotID, userID); found {
		t.Errorf("expected assignment NOT to be restored after a time shift")
	}
	found := false
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			found = true
		}
	}
	if !found {
		t.Errorf("expected user %d to be notified after the time-shifted slot lost its assignment", userID)
	}
}

// TestRegen_ReduzierteVarianteKeinTreffer (tasks.md 3.6): same_day_behavior turning a slot
// into its reduced variant changes duty_type_id — the old assignment does not carry over
// (different match key) and the existing "variant_changed" notification still fires.
func TestRegen_ReduzierteVarianteKeinTreffer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	variant := insertDutyTypeI(t, db, "Aufbau-Licht", "", 0, "", 0)
	aufbau := insertDutyTypeI(t, db, "Aufbau", "reduced", variant, "", 0)
	templateID := insertTemplateI(t, db, aufbau, -60, 1)

	game2 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "16:00", templateID)

	// First regen: game2 alone → no same-day context → slot stays base type "Aufbau".
	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	if got := autoSlotDutyType(t, db, game2); got != aufbau {
		t.Fatalf("precondition: expected base duty type %d before same-day context, got %d", aufbau, got)
	}
	oldSlotID := autoSlotID(t, db, game2)
	assignI(t, db, oldSlotID, userID, "assigned", nil, nil)

	// A second, earlier game on the same day puts the slot "between" games → reduces.
	insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", 0)
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got := autoSlotDutyType(t, db, game2); got != variant {
		t.Fatalf("expected slot reduced to variant %d, got %d", variant, got)
	}
	newSlotID := autoSlotID(t, db, game2)
	if _, _, _, found := assignmentByUser(t, db, newSlotID, userID); found {
		t.Errorf("expected assignment NOT to carry over onto the reduced-variant slot")
	}
	var gotKind, gotNewType string
	for _, n := range summary.Notifications {
		if n.UserID == userID {
			gotKind = n.Kind
			gotNewType = n.NewType
		}
	}
	if gotKind != "variant_changed" {
		t.Errorf("expected notification kind 'variant_changed', got %q", gotKind)
	}
	if gotNewType != "Aufbau-Licht" {
		t.Errorf("expected notification new_type 'Aufbau-Licht', got %q", gotNewType)
	}
}
