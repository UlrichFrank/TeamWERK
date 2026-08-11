package games

// Interner Test-Package (nicht games_test), weil diese Tests runAutoRegen und
// regenSingleDay direkt mit dem neuen skip-Parameter aufrufen — der ist
// unexportiert, siehe duty-bulk-regen design.md §1/§2.

import (
	"context"
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newRegenTestHandler(t *testing.T, db *sql.DB) *Handler {
	t.Helper()
	return NewHandler(db, testutil.TestConfig(), hub.NewHub())
}

func seedAgeClassRuleI(t *testing.T, db *sql.DB, teamID int) {
	t.Helper()
	if _, err := db.Exec(`UPDATE teams SET age_class=? WHERE id=?`, "A-Jugend", teamID); err != nil {
		t.Fatalf("set age_class: %v", err)
	}
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES (?, ?, ?)`,
		"A-Jugend", 30, 15); err != nil {
		t.Fatalf("seed age_class_game_rules: %v", err)
	}
}

// insertDutyTypeI inserts a duty_types row with optional same-day/adjacent-day reduction
// rules. Pass sameDayBehavior/adjacentDayBehavior as "" to leave them at the "normal" default.
func insertDutyTypeI(t *testing.T, db *sql.DB, name string, sameDayBehavior string, sameDayVariantID int, adjacentDayBehavior string, adjacentDayVariantID int) int {
	t.Helper()
	sdb := "normal"
	if sameDayBehavior != "" {
		sdb = sameDayBehavior
	}
	adb := "normal"
	if adjacentDayBehavior != "" {
		adb = adjacentDayBehavior
	}
	var sdv, adv any
	if sameDayVariantID > 0 {
		sdv = sameDayVariantID
	}
	if adjacentDayVariantID > 0 {
		adv = adjacentDayVariantID
	}
	res, err := db.Exec(
		`INSERT INTO duty_types (name, hours_value, same_day_behavior, same_day_variant_id, adjacent_day_behavior, adjacent_day_variant_id)
		 VALUES (?, 2.0, ?, ?, ?, ?)`,
		name, sdb, sdv, adb, adv)
	if err != nil {
		t.Fatalf("insertDutyTypeI: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// insertTemplateI creates a heim template with a single item and returns the template id.
func insertTemplateI(t *testing.T, db *sql.DB, dutyTypeID, offsetMin, slotsCount int) int {
	t.Helper()
	tr, err := db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?, ?, ?)`,
		"Heim", "heim", 75)
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}
	templateID, _ := tr.LastInsertId()
	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order)
		VALUES (?, ?, 'start', ?, ?, 0)`, templateID, dutyTypeID, offsetMin, slotsCount); err != nil {
		t.Fatalf("seed template item: %v", err)
	}
	return int(templateID)
}

// insertGameI inserts a heim game with the given team/template and returns its id.
// templateID=0 leaves template_id NULL.
func insertGameI(t *testing.T, db *sql.DB, seasonID, teamID int, date, timeStr string, templateID int) int {
	t.Helper()
	var tplArg any
	if templateID > 0 {
		tplArg = templateID
	}
	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type, template_id) VALUES (?, ?, ?, ?, 1, 'heim', ?)`,
		seasonID, "Test", date, timeStr, tplArg)
	if err != nil {
		t.Fatalf("insertGameI: %v", err)
	}
	gameID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamID); err != nil {
		t.Fatalf("insertGameI game_teams: %v", err)
	}
	return int(gameID)
}

func regenDate(t *testing.T, h *Handler, db *sql.DB, date string, seasonID int, skip map[int]bool) RegenSummary {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	summary, err := h.runAutoRegen(context.Background(), tx, []string{date}, seasonID, skip)
	if err != nil {
		tx.Rollback()
		t.Fatalf("runAutoRegen: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return summary
}

func autoSlotID(t *testing.T, db *sql.DB, gameID int) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`SELECT id FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).Scan(&id); err != nil {
		t.Fatalf("autoSlotID: %v", err)
	}
	return id
}

func autoSlotDutyType(t *testing.T, db *sql.DB, gameID int) int {
	t.Helper()
	var dt int
	if err := db.QueryRow(`SELECT duty_type_id FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).Scan(&dt); err != nil {
		t.Fatalf("autoSlotDutyType: %v", err)
	}
	return dt
}

// TestRegen_AusgenommenesSpielBleibtKontext (duty-bulk-regen tasks.md 1.3): two games on
// the same day, one excluded via skip. The excluded game's slot must survive with the
// exact same id (never touched), while the included game still gets the same_day_behavior
// reduction it would get without the exclusion.
func TestRegen_AusgenommenesSpielBleibtKontext(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	getraenke := insertDutyTypeI(t, db, "Getränke", "", 0, "", 0)
	getraenkeTemplate := insertTemplateI(t, db, getraenke, 0, 1) // slot at game time

	variant := insertDutyTypeI(t, db, "Aufbau-Licht", "", 0, "", 0)
	aufbau := insertDutyTypeI(t, db, "Aufbau", "reduced", variant, "", 0)
	aufbauTemplate := insertTemplateI(t, db, aufbau, -60, 1)

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", getraenkeTemplate)
	game2 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "16:00", aufbauTemplate)

	// Initial regen without exclusion establishes both auto slots.
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	slot1Before := autoSlotID(t, db, game1)
	if got := autoSlotDutyType(t, db, game2); got != variant {
		t.Fatalf("precondition: expected game2 slot to already be reduced to variant %d, got %d", variant, got)
	}

	// Second regen, game1 excluded.
	regenDate(t, h, db, "2026-06-13", seasonID, map[int]bool{game1: true})

	if got := autoSlotID(t, db, game1); got != slot1Before {
		t.Errorf("expected excluded game's slot id unchanged (%d), got %d", slot1Before, got)
	}
	if got := autoSlotDutyType(t, db, game2); got != variant {
		t.Errorf("expected included game to keep same_day_behavior reduction (duty_type %d), got %d", variant, got)
	}
}

// TestRegen_HeimspielKettenKontextUnabhaengigVomZustand (tasks.md 1.4): adjacent_day_behavior
// depends only on whether a neighboring home game EXISTS, never on what state it is in or
// whether it was even part of the current regen run — see design.md §2.
func TestRegen_HeimspielKettenKontextUnabhaengigVomZustand(t *testing.T) {
	t.Run("Nachbar bekommt none im selben Lauf", func(t *testing.T) {
		db := testutil.NewDB(t)
		seasonID := testutil.CreateSeason(t, db, "2025/26")
		teamID := testutil.CreateTeam(t, db, "Team A")
		seedAgeClassRuleI(t, db, teamID)
		h := newRegenTestHandler(t, db)

		variant := insertDutyTypeI(t, db, "Kasse-Kurz", "", 0, "", 0)
		kasse := insertDutyTypeI(t, db, "Kasse", "", 0, "reduced", variant)
		kasseTemplate := insertTemplateI(t, db, kasse, 60, 1) // after game end/time

		day2Template := insertTemplateI(t, db, insertDutyTypeI(t, db, "Sonstiges", "", 0, "", 0), 0, 1)

		day1Game := insertGameI(t, db, seasonID, teamID, "2026-06-13", "18:00", kasseTemplate)
		day2Game := insertGameI(t, db, seasonID, teamID, "2026-06-14", "18:00", day2Template)

		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		if _, err := h.runAutoRegen(context.Background(), tx, []string{"2026-06-13", "2026-06-14"}, seasonID, nil); err != nil {
			t.Fatalf("initial regen: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}

		// Day 2 gets "none": template_id=NULL, run a regen touching both dates again.
		if _, err := db.Exec(`UPDATE games SET template_id=NULL WHERE id=?`, day2Game); err != nil {
			t.Fatalf("set day2 template NULL: %v", err)
		}
		regenDate(t, h, db, "2026-06-14", seasonID, nil) // deletes day2's auto slot, doesn't touch day1
		regenDate(t, h, db, "2026-06-13", seasonID, nil) // re-derive day1's slot with fresh context

		if got := autoSlotDutyType(t, db, day1Game); got != variant {
			t.Errorf("expected day1 slot still reduced via adjacent_day_behavior (duty_type %d) even though day2 has no template, got %d", variant, got)
		}
		if got := countRowsI(t, db, "duty_slots", "game_id=? AND is_custom=0", day2Game); got != 0 {
			t.Errorf("expected day2 to have no auto slot after switching to none, got %d", got)
		}
	})

	t.Run("Nachbar liegt ausserhalb des Regen-Laufs", func(t *testing.T) {
		db := testutil.NewDB(t)
		seasonID := testutil.CreateSeason(t, db, "2025/26")
		teamID := testutil.CreateTeam(t, db, "Team A")
		seedAgeClassRuleI(t, db, teamID)
		h := newRegenTestHandler(t, db)

		variant := insertDutyTypeI(t, db, "Kasse-Kurz", "", 0, "", 0)
		kasse := insertDutyTypeI(t, db, "Kasse", "", 0, "reduced", variant)
		kasseTemplate := insertTemplateI(t, db, kasse, -60, 1)

		// Day 0 (the day before) exists as a plain home game, never passed to runAutoRegen.
		insertGameI(t, db, seasonID, teamID, "2026-06-12", "18:00", 0)
		day1Game := insertGameI(t, db, seasonID, teamID, "2026-06-13", "18:00", kasseTemplate)

		// Only day1 is regenerated — day0 is "outside the range" and never in `dates`.
		regenDate(t, h, db, "2026-06-13", seasonID, nil)

		if got := autoSlotDutyType(t, db, day1Game); got != variant {
			t.Errorf("expected day1 slot reduced via adjacent_day_behavior from a neighbor outside the regen range (duty_type %d), got %d", variant, got)
		}
	})
}

func countRowsI(t *testing.T, db *sql.DB, table, where string, args ...any) int {
	t.Helper()
	var n int
	q := "SELECT COUNT(*) FROM " + table + " WHERE " + where
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("countRowsI %s: %v", q, err)
	}
	return n
}
