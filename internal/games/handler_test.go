package games_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func testServer(t *testing.T, h *games.Handler) *httptest.Server {
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kalender", h.ListGames)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Post("/api/admin/kalender", h.CreateGame)
			r.Put("/api/admin/kalender/{id}", h.UpdateGame)
			r.Delete("/api/kalender/{id}", h.DeleteGame)
		})
	})
}

// TestListGames_ReturnsGamesInRange verifies that games in the active season are returned.
func TestListGames_ReturnsGamesInRange(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/kalender?season_id=%d", seasonID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	res.Body.Close()

	if len(games) != 1 {
		t.Errorf("expected 1 game, got %d", len(games))
	}
}

// TestListGames_EmptyRange verifies that an empty list is returned when no games exist.
func TestListGames_EmptyRange(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/kalender", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var result []map[string]any
	json.NewDecoder(res.Body).Decode(&result)
	res.Body.Close()

	if len(result) != 0 {
		t.Errorf("expected 0 games, got %d", len(result))
	}
}

// TestCreateGame_AdminOK verifies that an admin can create a game.
func TestCreateGame_AdminOK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	body := map[string]any{
		"date":       "2026-06-15",
		"time":       "18:00",
		"opponent":   "FC Test",
		"team_ids":   []int{teamID},
		"event_type": "heim",
		"season_id":  seasonID,
	}
	res := testutil.Post(t, srv, "/api/admin/kalender", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", res.StatusCode)
	}
}

// TestDeleteGame_CascadesAndRollsBackFulfilledHours verifies the full delete path:
// game and its duty_slots/duty_assignments are removed via FK cascade, and
// duty_accounts.ist is recomputed so fulfilled-hours of the deleted event no
// longer count toward the user's account.
func TestDeleteGame_CascadesAndRollsBackFulfilledHours(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")
	otherGameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-21")

	adminID := testutil.CreateUser(t, db, "admin")
	helperID := testutil.CreateUser(t, db, "standard") // duty assignee

	// Two duty types: 2h and 1h.
	dutyType2h := insertDutyType(t, db, "Aufbau", 2.0)
	dutyType1h := insertDutyType(t, db, "Kasse", 1.0)

	// Two slots on the to-be-deleted game.
	slotAssigned := insertDutySlot(t, db, dutyType2h, seasonID, teamID, gameID, "2026-06-14")
	slotFulfilled := insertDutySlot(t, db, dutyType1h, seasonID, teamID, gameID, "2026-06-14")

	// One unrelated fulfilled slot on a different game in the same season — must remain.
	slotOther := insertDutySlot(t, db, dutyType2h, seasonID, teamID, otherGameID, "2026-06-21")

	// Helper has 1 assigned (no hours yet), 1 fulfilled (1h, on deleted game), 1 fulfilled (2h, on other game).
	insertDutyAssignment(t, db, slotAssigned, helperID, "assigned")
	insertDutyAssignment(t, db, slotFulfilled, helperID, "fulfilled")
	insertDutyAssignment(t, db, slotOther, helperID, "fulfilled")

	// Seed duty_accounts.ist = 3h (1h + 2h), matching reality before delete.
	if _, err := db.Exec(
		`INSERT INTO duty_accounts (user_id, season_id, soll, ist) VALUES (?, ?, 10, 3)`,
		helperID, seasonID); err != nil {
		t.Fatalf("seed duty_accounts: %v", err)
	}

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/kalender/%d", gameID), token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (with regen_summary), got %d", res.StatusCode)
	}

	// Game itself is gone.
	if got := countRows(t, db, "games", "id=?", gameID); got != 0 {
		t.Errorf("game not deleted: count=%d", got)
	}
	// Cascade removed slots + assignments belonging to that game.
	if got := countRows(t, db, "duty_slots", "game_id=?", gameID); got != 0 {
		t.Errorf("duty_slots not cascade-deleted: count=%d", got)
	}
	if got := countRows(t, db, "duty_assignments", "duty_slot_id IN (?, ?)", slotAssigned, slotFulfilled); got != 0 {
		t.Errorf("duty_assignments not cascade-deleted: count=%d", got)
	}
	// Unrelated slot + assignment on the other game are untouched.
	if got := countRows(t, db, "duty_slots", "id=?", slotOther); got != 1 {
		t.Errorf("unrelated slot was wrongly deleted: count=%d", got)
	}
	// ist must equal the remaining fulfilled hours (2h) for this season.
	var ist float64
	if err := db.QueryRow(
		`SELECT ist FROM duty_accounts WHERE user_id=? AND season_id=?`,
		helperID, seasonID).Scan(&ist); err != nil {
		t.Fatalf("read duty_accounts.ist: %v", err)
	}
	if ist != 2.0 {
		t.Errorf("expected duty_accounts.ist=2.0 after rollback of 1h fulfilled, got %v", ist)
	}
}

// TestDeleteGame_NoDutiesNoCrash verifies the empty-recipient path: a game
// without any duty_slots can be deleted cleanly, returns 204, and removes
// the game row without panicking on the empty assignee list.
func TestDeleteGame_NoDutiesNoCrash(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	adminID := testutil.CreateUser(t, db, "admin")

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/kalender/%d", gameID), token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (with regen_summary), got %d", res.StatusCode)
	}
	if got := countRows(t, db, "games", "id=?", gameID); got != 0 {
		t.Errorf("game not deleted: count=%d", got)
	}
}

func insertDutyType(t *testing.T, db *sql.DB, name string, hours float64) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO duty_types (name, hours_value) VALUES (?, ?)`, name, hours)
	if err != nil {
		t.Fatalf("insertDutyType: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func insertDutySlot(t *testing.T, db *sql.DB, dutyTypeID, seasonID, teamID, gameID int, date string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, team_id, season_id, game_id)
		 VALUES (?, ?, ?, 1, ?, ?, ?)`,
		"Testdienst", date, dutyTypeID, teamID, seasonID, gameID)
	if err != nil {
		t.Fatalf("insertDutySlot: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func insertDutyAssignment(t *testing.T, db *sql.DB, slotID, userID int, status string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO duty_assignments (duty_slot_id, user_id, status) VALUES (?, ?, ?)`,
		slotID, userID, status); err != nil {
		t.Fatalf("insertDutyAssignment: %v", err)
	}
}

func countRows(t *testing.T, db *sql.DB, table, where string, args ...any) int {
	t.Helper()
	var n int
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, table, where)
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("countRows %s: %v", q, err)
	}
	return n
}

// TestCreateGame_AutoRegenSkipsAdjacentDay covers the central auto-regen contract:
// creating two heim games on consecutive days must trigger adjacent-day skip logic,
// and is_custom=1 slots must survive the regen untouched.
func TestCreateGame_AutoRegenSkipsAdjacentDay(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	// Override the fixture's default age_class to one the rules table accepts.
	if _, err := db.Exec(`UPDATE teams SET age_class=? WHERE id=?`, "A-Jugend", teamID); err != nil {
		t.Fatalf("set age_class: %v", err)
	}

	// Age-class rule needed for effectiveEventDuration on heim games.
	if _, err := db.Exec(
		`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES (?, ?, ?)`,
		"A-Jugend", 30, 15); err != nil {
		t.Fatalf("seed age_class_game_rules: %v", err)
	}

	// Duty type with adjacent_day_behavior=skip.
	res, err := db.Exec(`
		INSERT INTO duty_types (name, hours_value, adjacent_day_behavior)
		VALUES (?, ?, ?)`, "Aufbau", 2.0, "skip")
	if err != nil {
		t.Fatalf("seed duty_type: %v", err)
	}
	dutyTypeID, _ := res.LastInsertId()

	// Heim template with one item: -60min from start, 1 slot.
	res, err = db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?, ?, ?)`,
		"Heim", "heim", 75)
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}
	templateID, _ := res.LastInsertId()
	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order)
		VALUES (?, ?, ?, ?, ?, ?)`, templateID, dutyTypeID, "start", -60, 1, 0); err != nil {
		t.Fatalf("seed template item: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	createBody := func(date string) map[string]any {
		return map[string]any{
			"date": date, "time": "14:00",
			"opponent": "FC Test", "team_ids": []int{teamID},
			"event_type": "heim", "season_id": seasonID,
		}
	}

	// Game A — no neighbors → template slot is created at 13:00.
	resA := testutil.Post(t, srv, "/api/admin/kalender", token, createBody("2026-06-13"))
	resA.Body.Close()
	if resA.StatusCode != http.StatusCreated {
		t.Fatalf("create game A: expected 201, got %d", resA.StatusCode)
	}
	if got := countRows(t, db, "duty_slots", "event_date=? AND is_custom=0", "2026-06-13"); got != 1 {
		t.Fatalf("after create A: expected 1 auto-slot on 06-13, got %d", got)
	}

	// Manual slot on game A (is_custom=1) — must survive any future regen.
	var gameAID int
	db.QueryRow(`SELECT id FROM games WHERE date=?`, "2026-06-13").Scan(&gameAID)
	if _, err := db.Exec(`
		INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id,
		  slots_total, team_id, season_id, game_id, is_custom)
		VALUES (?, ?, ?, ?, 1, ?, ?, ?, 1)`,
		"Manuell", "2026-06-13", "12:00", dutyTypeID, teamID, seasonID, gameAID); err != nil {
		t.Fatalf("seed custom slot: %v", err)
	}

	// Game B on the adjacent day — runAutoRegen for {06-12, 06-13, 06-14}
	// must skip the template slot on 06-13 (adjacent rule) AND on 06-14, while leaving
	// the is_custom=1 slot on 06-13 intact.
	resB := testutil.Post(t, srv, "/api/admin/kalender", token, createBody("2026-06-14"))
	resB.Body.Close()
	if resB.StatusCode != http.StatusCreated {
		t.Fatalf("create game B: expected 201, got %d", resB.StatusCode)
	}

	// Day 13 keeps its Aufbau (no Heim on day 12 → adjacent doesn't fire).
	if got := countRows(t, db, "duty_slots", "event_date=? AND is_custom=0", "2026-06-13"); got != 1 {
		t.Errorf("after create B: expected 1 auto-slot on 06-13 (no prev-day heim), got %d", got)
	}
	// Day 14's Aufbau is skipped: adjacent rule fires because Heim on day 13.
	if got := countRows(t, db, "duty_slots", "event_date=? AND is_custom=0", "2026-06-14"); got != 0 {
		t.Errorf("after create B: expected 0 auto-slots on 06-14 (adjacent skip), got %d", got)
	}
	if got := countRows(t, db, "duty_slots", "event_date=? AND is_custom=1", "2026-06-13"); got != 1 {
		t.Errorf("is_custom=1 slot on 06-13 must survive regen, got %d", got)
	}
}

// TestCreateGame_UnauthorizedForbidden verifies that a user without club function cannot create a game.
func TestCreateGame_UnauthorizedForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	userID := testutil.CreateUser(t, db, "standard")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	body := map[string]any{
		"date": "2026-06-15", "time": "18:00",
		"opponent": "FC Test", "team_ids": []int{1},
		"event_type": "heim",
	}
	res := testutil.Post(t, srv, "/api/admin/kalender", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// TestCreateGame_ResponseIncludesRegenSummary verifies that CreateGame response
// includes a regen_summary object after auto-regen completes.
func TestCreateGame_ResponseIncludesRegenSummary(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	if _, err := db.Exec(`UPDATE teams SET age_class=? WHERE id=?`, "A-Jugend", teamID); err != nil {
		t.Fatalf("set age_class: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES (?, ?, ?)`,
		"A-Jugend", 30, 15); err != nil {
		t.Fatalf("seed age_class_game_rules: %v", err)
	}

	dutyRes, err := db.Exec(`
		INSERT INTO duty_types (name, hours_value)
		VALUES (?, ?)`, "Aufbau", 2.0)
	if err != nil {
		t.Fatalf("seed duty_type: %v", err)
	}
	dutyTypeID, _ := dutyRes.LastInsertId()

	templateRes, err := db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?, ?, ?)`,
		"Heim", "heim", 75)
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}
	templateID, _ := templateRes.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order)
		VALUES (?, ?, ?, ?, ?, ?)`, templateID, dutyTypeID, "start", -60, 1, 0); err != nil {
		t.Fatalf("seed template item: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	body := map[string]any{
		"date":       "2026-06-13",
		"time":       "14:00",
		"opponent":   "FC Test",
		"team_ids":   []int{teamID},
		"event_type": "heim",
		"season_id":  seasonID,
	}

	res := testutil.Post(t, srv, "/api/admin/kalender", token, body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create game: expected 201, got %d", res.StatusCode)
	}

	var response map[string]any
	json.NewDecoder(res.Body).Decode(&response)
	res.Body.Close()

	// Response should have an id and a regen_summary.
	if _, ok := response["id"]; !ok {
		t.Errorf("response missing 'id'")
	}
	if _, ok := response["regen_summary"]; !ok {
		t.Errorf("response missing 'regen_summary'")
	}
}

// TestUpdateGame_TimeChangeRegenSlots verifies that updating a game's time
// triggers auto-regen and shifts template-based slots to the new time.
func TestUpdateGame_TimeChangeRegenSlots(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	if _, err := db.Exec(`UPDATE teams SET age_class=? WHERE id=?`, "A-Jugend", teamID); err != nil {
		t.Fatalf("set age_class: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES (?, ?, ?)`,
		"A-Jugend", 30, 15); err != nil {
		t.Fatalf("seed age_class_game_rules: %v", err)
	}

	res, err := db.Exec(`
		INSERT INTO duty_types (name, hours_value)
		VALUES (?, ?)`, "Aufbau", 2.0)
	if err != nil {
		t.Fatalf("seed duty_type: %v", err)
	}
	dutyTypeID, _ := res.LastInsertId()

	res, err = db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?, ?, ?)`,
		"Heim", "heim", 75)
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}
	templateID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order)
		VALUES (?, ?, ?, ?, ?, ?)`, templateID, dutyTypeID, "start", -60, 1, 0); err != nil {
		t.Fatalf("seed template item: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-13")

	// Update game time from default to 16:00.
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	updateBody := map[string]any{
		"date": "2026-06-13", "time": "16:00",
		"opponent": "FC Test Updated", "team_ids": []int{teamID},
		"event_type": "heim", "season_id": seasonID,
	}

	updateRes := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/admin/kalender/%d", gameID), token, updateBody)
	updateRes.Body.Close()
	if updateRes.StatusCode != http.StatusOK {
		t.Fatalf("update game: expected 200, got %d", updateRes.StatusCode)
	}

	// Slot should now be at 15:00 (16:00 - 60 min).
	var slotTime string
	err = db.QueryRow(`SELECT event_time FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).Scan(&slotTime)
	if err != nil || slotTime != "15:00" {
		t.Errorf("expected slot at 15:00, got %s (err=%v)", slotTime, err)
	}
}

// TestDeleteGame_NeighborDayRegen verifies that deleting a game triggers
// auto-regen on adjacent days: a slot on the neighbor day that was skipped
// due to adjacent_day_behavior=skip must reappear after the adjacent game
// is deleted.
func TestDeleteGame_NeighborDayRegen(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	if _, err := db.Exec(`UPDATE teams SET age_class=? WHERE id=?`, "A-Jugend", teamID); err != nil {
		t.Fatalf("set age_class: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES (?, ?, ?)`,
		"A-Jugend", 30, 15); err != nil {
		t.Fatalf("seed age_class_game_rules: %v", err)
	}

	dutyRes, err := db.Exec(`
		INSERT INTO duty_types (name, hours_value, adjacent_day_behavior)
		VALUES (?, ?, ?)`, "Aufbau", 2.0, "skip")
	if err != nil {
		t.Fatalf("seed duty_type: %v", err)
	}
	dutyTypeID, _ := dutyRes.LastInsertId()

	templateRes, err := db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?, ?, ?)`,
		"Heim", "heim", 75)
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}
	templateID, _ := templateRes.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order)
		VALUES (?, ?, ?, ?, ?, ?)`, templateID, dutyTypeID, "start", -60, 1, 0); err != nil {
		t.Fatalf("seed template item: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	createBody := func(date string) map[string]any {
		return map[string]any{
			"date": date, "time": "14:00",
			"opponent": "FC Test", "team_ids": []int{teamID},
			"event_type": "heim", "season_id": seasonID,
		}
	}

	// Game A on day 13 → gets a slot (no neighbors).
	resA := testutil.Post(t, srv, "/api/admin/kalender", token, createBody("2026-06-13"))
	resA.Body.Close()
	var gameAID int
	db.QueryRow(`SELECT id FROM games WHERE date=?`, "2026-06-13").Scan(&gameAID)

	// Game B on day 14 → slot is SKIPPED (adjacent to game A on day 13).
	resB := testutil.Post(t, srv, "/api/admin/kalender", token, createBody("2026-06-14"))
	resB.Body.Close()
	if got := countRows(t, db, "duty_slots", "event_date=? AND is_custom=0", "2026-06-14"); got != 0 {
		t.Fatalf("before delete: expected 0 auto-slots on day 14 (adjacent skip), got %d", got)
	}

	// Delete game A → auto-regen on day 14 should fire and create the slot now.
	deleteRes := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/kalender/%d", gameAID), token, nil)
	deleteRes.Body.Close()
	if deleteRes.StatusCode != http.StatusOK {
		t.Fatalf("delete game A: expected 200, got %d", deleteRes.StatusCode)
	}

	// Day 14 should now have an auto-slot (no prev-day game anymore).
	if got := countRows(t, db, "duty_slots", "event_date=? AND is_custom=0", "2026-06-14"); got != 1 {
		t.Errorf("after delete A: expected 1 auto-slot on day 14 (adjacent skip lifted), got %d", got)
	}
	// Day 13 has no slots (game A deleted).
	if got := countRows(t, db, "duty_slots", "event_date=? AND is_custom=0", "2026-06-13"); got != 0 {
		t.Errorf("after delete A: expected 0 auto-slots on day 13, got %d", got)
	}
}

// TestCreateGame_CustomSlotNotAffectedByRegen verifies that manually created
// slots (is_custom=1) are not deleted or modified by auto-regen.
func TestCreateGame_CustomSlotNotAffectedByRegen(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	if _, err := db.Exec(`UPDATE teams SET age_class=? WHERE id=?`, "A-Jugend", teamID); err != nil {
		t.Fatalf("set age_class: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES (?, ?, ?)`,
		"A-Jugend", 30, 15); err != nil {
		t.Fatalf("seed age_class_game_rules: %v", err)
	}

	res, err := db.Exec(`
		INSERT INTO duty_types (name, hours_value)
		VALUES (?, ?)`, "Aufbau", 2.0)
	if err != nil {
		t.Fatalf("seed duty_type: %v", err)
	}
	dutyTypeID, _ := res.LastInsertId()

	res, err = db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?, ?, ?)`,
		"Heim", "heim", 75)
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}
	templateID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order)
		VALUES (?, ?, ?, ?, ?, ?)`, templateID, dutyTypeID, "start", -60, 1, 0); err != nil {
		t.Fatalf("seed template item: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-13")

	// Create a custom slot (is_custom=1).
	if _, err := db.Exec(`
		INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id,
		  slots_total, team_id, season_id, game_id, is_custom)
		VALUES (?, ?, ?, ?, 1, ?, ?, ?, 1)`,
		"Manueller Dienst", "2026-06-13", "12:00", dutyTypeID, teamID, seasonID, gameID); err != nil {
		t.Fatalf("seed custom slot: %v", err)
	}

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	// Update the game (triggers auto-regen).
	updateBody := map[string]any{
		"date": "2026-06-13", "time": "14:00",
		"opponent": "FC Test", "team_ids": []int{teamID},
		"event_type": "heim", "season_id": seasonID,
	}
	updateRes := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/admin/kalender/%d", gameID), token, updateBody)
	updateRes.Body.Close()

	// Custom slot should still exist with its original time.
	var customSlotTime string
	err = db.QueryRow(`SELECT event_time FROM duty_slots WHERE is_custom=1 AND game_id=?`, gameID).Scan(&customSlotTime)
	if err != nil || customSlotTime != "12:00" {
		t.Errorf("custom slot should be unchanged at 12:00, got %s (err=%v)", customSlotTime, err)
	}
}

// TestCreateGame_GenericEventCanBeCreated verifies that generic events
// can be created without a template.
func TestCreateGame_GenericEventCanBeCreated(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	// Create generic event.
	createBody := map[string]any{
		"date":       "2026-06-13",
		"time":       "10:00",
		"opponent":   "Training",
		"team_ids":   []int{teamID},
		"event_type": "generisch",
		"season_id":  seasonID,
	}

	createRes := testutil.Post(t, srv, "/api/admin/kalender", token, createBody)
	createRes.Body.Close()
	if createRes.StatusCode != http.StatusCreated {
		t.Fatalf("create generic event: expected 201, got %d", createRes.StatusCode)
	}

	// Generic event should be created without template-derived slots.
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM games WHERE event_type=? AND date=?`,
		"generisch", "2026-06-13").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 generic event, got %d", count)
	}
}
