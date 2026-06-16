package games_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// testServer mounts the production router. Pass the test DB; the handler
// argument is kept for backwards compatibility with existing call sites and
// can be nil — only the DB is used.
func testServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	return prodserver.New(t, db)
}

// TestListGames_ReturnsGamesInRange verifies that games in the active season are returned.
func TestListGames_ReturnsGamesInRange(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d", seasonID), token)
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
	srv := testServer(t, db)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/games", token)
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

// TestGetGame_HappyPath verifies GET /api/games/{id} returns the game.
func TestGetGame_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", gameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

// TestGetGame_NotFound verifies GET /api/games/{id} on missing game returns 404.
func TestGetGame_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, "/api/games/9999", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

// TestUpdateGame_Forbidden verifies non-vorstand/trainer cannot update games.
func TestUpdateGame_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	spielerID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, spielerID, "standard", []string{"spieler"})

	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/games/%d", gameID), token,
		map[string]any{"opponent": "Hacker"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// TestCreateGame_AdminOK verifies that an admin can create a game.
func TestCreateGame_AdminOK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)

	token := testutil.Token(t, adminUserID, "admin", nil)
	body := map[string]any{
		"date":       "2026-06-15",
		"time":       "18:00",
		"opponent":   "FC Test",
		"team_ids":   []int{teamID},
		"event_type": "heim",
		"season_id":  seasonID,
	}
	res := testutil.Post(t, srv, "/api/games", token, body)
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

	srv := testServer(t, db)
	token := testutil.Token(t, adminID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/games/%d", gameID), token, nil)
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

	srv := testServer(t, db)
	token := testutil.Token(t, adminID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/games/%d", gameID), token, nil)
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
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	createBody := func(date string) map[string]any {
		return map[string]any{
			"date": date, "time": "14:00",
			"opponent": "FC Test", "team_ids": []int{teamID},
			"event_type": "heim", "season_id": seasonID,
		}
	}

	// Game A — no neighbors → template slot is created at 13:00.
	resA := testutil.Post(t, srv, "/api/games", token, createBody("2026-06-13"))
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
	resB := testutil.Post(t, srv, "/api/games", token, createBody("2026-06-14"))
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
	srv := testServer(t, db)

	token := testutil.Token(t, userID, "standard", nil)
	body := map[string]any{
		"date": "2026-06-15", "time": "18:00",
		"opponent": "FC Test", "team_ids": []int{1},
		"event_type": "heim",
	}
	res := testutil.Post(t, srv, "/api/games", token, body)
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
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	body := map[string]any{
		"date":       "2026-06-13",
		"time":       "14:00",
		"opponent":   "FC Test",
		"team_ids":   []int{teamID},
		"event_type": "heim",
		"season_id":  seasonID,
	}

	res := testutil.Post(t, srv, "/api/games", token, body)
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
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	updateBody := map[string]any{
		"date": "2026-06-13", "time": "16:00",
		"opponent": "FC Test Updated", "team_ids": []int{teamID},
		"event_type": "heim", "season_id": seasonID,
	}

	updateRes := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/games/%d", gameID), token, updateBody)
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
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	createBody := func(date string) map[string]any {
		return map[string]any{
			"date": date, "time": "14:00",
			"opponent": "FC Test", "team_ids": []int{teamID},
			"event_type": "heim", "season_id": seasonID,
		}
	}

	// Game A on day 13 → gets a slot (no neighbors).
	resA := testutil.Post(t, srv, "/api/games", token, createBody("2026-06-13"))
	resA.Body.Close()
	var gameAID int
	db.QueryRow(`SELECT id FROM games WHERE date=?`, "2026-06-13").Scan(&gameAID)

	// Game B on day 14 → slot is SKIPPED (adjacent to game A on day 13).
	resB := testutil.Post(t, srv, "/api/games", token, createBody("2026-06-14"))
	resB.Body.Close()
	if got := countRows(t, db, "duty_slots", "event_date=? AND is_custom=0", "2026-06-14"); got != 0 {
		t.Fatalf("before delete: expected 0 auto-slots on day 14 (adjacent skip), got %d", got)
	}

	// Delete game A → auto-regen on day 14 should fire and create the slot now.
	deleteRes := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/games/%d", gameAID), token, nil)
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

	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	// Update the game (triggers auto-regen).
	updateBody := map[string]any{
		"date": "2026-06-13", "time": "14:00",
		"opponent": "FC Test", "team_ids": []int{teamID},
		"event_type": "heim", "season_id": seasonID,
	}
	updateRes := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/games/%d", gameID), token, updateBody)
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
	srv := testServer(t, db)
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

	createRes := testutil.Post(t, srv, "/api/games", token, createBody)
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

// ── TC-G-EXT: ListTeamsForUser ────────────────────────────────────────────────

func teamsServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	return prodserver.New(t, db)
}

// TC-G-EXT01: Trainer sees only teams they manage via kader_trainers.
func TestListTeamsForUser_TrainerSeesOwnTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	testutil.CreateTeam(t, db, "Team B") // not linked to trainer

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID)

	srv := teamsServer(t, db)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/teams", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var teams []map[string]any
	json.NewDecoder(res.Body).Decode(&teams)
	res.Body.Close()

	if len(teams) != 1 {
		t.Errorf("trainer: expected 1 team, got %d", len(teams))
	}
	if len(teams) > 0 {
		if int(teams[0]["id"].(float64)) != teamA {
			t.Errorf("expected team A (id=%d), got id=%.0f", teamA, teams[0]["id"])
		}
	}
}

// TC-G-EXT02: Admin sees all teams with an active kader.
func TestListTeamsForUser_AdminSeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	testutil.CreateKader(t, db, teamA, seasonID)
	testutil.CreateKader(t, db, teamB, seasonID)

	adminID := testutil.CreateUser(t, db, "admin")
	srv := teamsServer(t, db)

	res := testutil.Get(t, srv, "/api/teams", testutil.Token(t, adminID, "admin", nil))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var teams []map[string]any
	json.NewDecoder(res.Body).Decode(&teams)
	res.Body.Close()

	if len(teams) < 2 {
		t.Errorf("admin: expected ≥2 teams, got %d", len(teams))
	}
}

// TC-G-EXT03: Spieler (non-trainer) sees only teams they are a member of.
func TestListTeamsForUser_SpielerSeesOwnTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.CreateKader(t, db, teamB, seasonID)

	spielerUserID := testutil.CreateUser(t, db, "standard")
	spielerMemberID := testutil.CreateMember(t, db, spielerUserID)
	// Add spieler to team A via kader_members (player_memberships is a view).
	db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, spielerMemberID)

	srv := teamsServer(t, db)

	token := testutil.Token(t, spielerUserID, "standard", nil)
	res := testutil.Get(t, srv, "/api/teams", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var teams []map[string]any
	json.NewDecoder(res.Body).Decode(&teams)
	res.Body.Close()

	if len(teams) != 1 {
		t.Errorf("spieler: expected 1 team, got %d", len(teams))
	}
	if len(teams) > 0 {
		if int(teams[0]["id"].(float64)) != teamA {
			t.Errorf("expected team A (id=%d), got id=%.0f", teamA, teams[0]["id"])
		}
	}
}

// TestListDutyTemplates_TrainerCanRead verifies that a user with club_function=trainer
// can list duty templates (GET /api/duty-templates returns 200).
func TestListDutyTemplates_TrainerCanRead(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)

	token := testutil.Token(t, userID, "spieler", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/duty-templates", token)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for trainer, got %d", res.StatusCode)
	}
}

// TestCreateDutyTemplate_TrainerForbidden verifies that a trainer cannot create
// duty templates (POST /api/duty-templates returns 403).
func TestCreateDutyTemplate_TrainerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)

	token := testutil.Token(t, userID, "spieler", []string{"trainer"})
	res := testutil.Post(t, srv, "/api/duty-templates", token, map[string]any{
		"name":             "Test-Vorlage",
		"template_type":    "heim",
		"duration_minutes": 90,
	})
	res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for trainer creating template, got %d", res.StatusCode)
	}
}

// TestListMyGames_ExtendedKaderSiehtSpiel verifies that a player only in the extended kader
// of a team sees that team's games in /api/games/my.
func TestListMyGames_ExtendedKaderSiehtSpiel(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")

	extUserID := testutil.CreateUser(t, db, "standard")
	extMemberID := testutil.CreateMember(t, db, extUserID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, extMemberID)

	srv := testServer(t, db)

	token := testutil.Token(t, extUserID, "standard", nil)
	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var gameList []map[string]any
	json.NewDecoder(res.Body).Decode(&gameList)
	res.Body.Close()

	if len(gameList) != 1 {
		t.Errorf("extended kader member: expected 1 game, got %d", len(gameList))
	}
}

// TestListMyGames_ExtendedKaderKeinAutoConfirm verifies that opt-out auto-confirm does NOT
// apply to members who are only in the extended kader of a team.
func TestListMyGames_ExtendedKaderKeinAutoConfirm(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")
	db.Exec(`UPDATE games SET rsvp_opt_out=1 WHERE id=?`, gameID)

	extUserID := testutil.CreateUser(t, db, "standard")
	extMemberID := testutil.CreateMember(t, db, extUserID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, extMemberID)

	srv := testServer(t, db)

	token := testutil.Token(t, extUserID, "standard", nil)
	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var gameList []map[string]any
	json.NewDecoder(res.Body).Decode(&gameList)
	res.Body.Close()

	if len(gameList) != 1 {
		t.Fatalf("expected 1 game, got %d", len(gameList))
	}
	if gameList[0]["my_rsvp"] != nil {
		t.Errorf("extended kader: expected my_rsvp=null for opt-out game, got %v", gameList[0]["my_rsvp"])
	}
}

// TestListMyGames_RegularKaderAutoConfirmBleibt verifies that opt-out auto-confirm still
// applies to members in the regular kader.
func TestListMyGames_RegularKaderAutoConfirmBleibt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")
	db.Exec(`UPDATE games SET rsvp_opt_out=1 WHERE id=?`, gameID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	srv := testServer(t, db)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var gameList []map[string]any
	json.NewDecoder(res.Body).Decode(&gameList)
	res.Body.Close()

	if len(gameList) != 1 {
		t.Fatalf("expected 1 game, got %d", len(gameList))
	}
	if gameList[0]["my_rsvp"] != "confirmed" {
		t.Errorf("regular kader: expected my_rsvp=confirmed for opt-out game, got %v", gameList[0]["my_rsvp"])
	}
}

// TestUpdateGame_RsvpFlagsPersisted verifies that PUT /api/games/{id} with
// rsvp_opt_out and rsvp_require_reason in the payload writes both values to DB.
func TestUpdateGame_RsvpFlagsPersisted(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	body := map[string]any{
		"date":                "2026-06-14",
		"time":                "18:00",
		"opponent":            "FC Test",
		"team_ids":            []int{teamID},
		"event_type":          "heim",
		"rsvp_opt_out":        1,
		"rsvp_require_reason": 0,
	}
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/games/%d", gameID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var optOut, reqReason int
	if err := db.QueryRow(`SELECT rsvp_opt_out, rsvp_require_reason FROM games WHERE id=?`, gameID).
		Scan(&optOut, &reqReason); err != nil {
		t.Fatalf("query rsvp flags: %v", err)
	}
	if optOut != 1 || reqReason != 0 {
		t.Errorf("expected rsvp_opt_out=1, rsvp_require_reason=0; got %d, %d", optOut, reqReason)
	}
}

// TestUpdateGame_RsvpFlagsPartialUpdate verifies that PUT /api/games/{id} without
// the rsvp_* fields leaves existing DB values untouched (no implicit reset).
func TestUpdateGame_RsvpFlagsPartialUpdate(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	if _, err := db.Exec(`UPDATE games SET rsvp_opt_out=1, rsvp_require_reason=0 WHERE id=?`, gameID); err != nil {
		t.Fatalf("seed rsvp flags: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	body := map[string]any{
		"date":       "2026-06-14",
		"time":       "20:00",
		"opponent":   "FC Test",
		"team_ids":   []int{teamID},
		"event_type": "heim",
	}
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/games/%d", gameID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var optOut, reqReason int
	if err := db.QueryRow(`SELECT rsvp_opt_out, rsvp_require_reason FROM games WHERE id=?`, gameID).
		Scan(&optOut, &reqReason); err != nil {
		t.Fatalf("query rsvp flags: %v", err)
	}
	if optOut != 1 || reqReason != 0 {
		t.Errorf("partial update must preserve flags; expected (1,0), got (%d,%d)", optOut, reqReason)
	}
}

// TestUpdateGame_RsvpFlags_PlayerForbidden verifies that a Spieler cannot change
// rsvp_opt_out / rsvp_require_reason via PUT — the endpoint is gated as a whole.
func TestUpdateGame_RsvpFlags_PlayerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	spielerID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, spielerID, "standard", []string{"spieler"})

	body := map[string]any{
		"rsvp_opt_out":        1,
		"rsvp_require_reason": 0,
	}
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/games/%d", gameID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var optOut, reqReason int
	if err := db.QueryRow(`SELECT rsvp_opt_out, rsvp_require_reason FROM games WHERE id=?`, gameID).
		Scan(&optOut, &reqReason); err != nil {
		t.Fatalf("query rsvp flags: %v", err)
	}
	if optOut != 0 || reqReason != 1 {
		t.Errorf("DB flags must be unchanged on 403; expected defaults (0,1), got (%d,%d)", optOut, reqReason)
	}
}

// optOutFixture creates a season+team+game+kader with N kader members. All
// helpers below need that bedrock setup.
func optOutFixture(t *testing.T, db *sql.DB, kaderSize int, optOut bool) (seasonID, teamID, kaderID, gameID int, memberIDs []int) {
	t.Helper()
	seasonID = testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("activate season: %v", err)
	}
	teamID = testutil.CreateTeam(t, db, "Herren")
	kaderID = testutil.CreateKader(t, db, teamID, seasonID)
	gameID = testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")
	if optOut {
		if _, err := db.Exec(`UPDATE games SET rsvp_opt_out=1 WHERE id=?`, gameID); err != nil {
			t.Fatalf("set rsvp_opt_out: %v", err)
		}
	}
	memberIDs = make([]int, kaderSize)
	for i := 0; i < kaderSize; i++ {
		uid := testutil.CreateUser(t, db, "standard")
		mid := testutil.CreateMember(t, db, uid)
		if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, mid); err != nil {
			t.Fatalf("kader_members insert: %v", err)
		}
		memberIDs[i] = mid
	}
	return
}

// TestListGames_OptOutCountsKaderImplicit verifies that ListGames adds implicit
// confirms for regular kader members when rsvp_opt_out=1.
func TestListGames_OptOutCountsKaderImplicit(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, _, _, _, _ := optOutFixture(t, db, 3, true)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d", seasonID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}
	if c, ok := games[0]["confirmed_count"].(float64); !ok || int(c) != 3 {
		t.Errorf("expected confirmed_count=3 (3 kader members, opt-out), got %v", games[0]["confirmed_count"])
	}
}

// TestListGames_OptOutWithDeclinedNotCounted verifies that an explicit declined
// is removed from the implicit confirm pool and shows up in declined_count.
func TestListGames_OptOutWithDeclinedNotCounted(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, _, _, gameID, memberIDs := optOutFixture(t, db, 3, true)
	// Member 0 explicitly declines.
	declinerUID := 0
	db.QueryRow(`SELECT user_id FROM members WHERE id=?`, memberIDs[0]).Scan(&declinerUID)
	if _, err := db.Exec(
		`INSERT INTO game_responses (game_id, member_id, responded_by, status) VALUES (?,?,?,?)`,
		gameID, memberIDs[0], declinerUID, "declined"); err != nil {
		t.Fatalf("seed response: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d", seasonID), token)
	defer res.Body.Close()
	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	if c, _ := games[0]["confirmed_count"].(float64); int(c) != 2 {
		t.Errorf("expected confirmed_count=2 (3 kader minus 1 declined), got %v", games[0]["confirmed_count"])
	}
	if d, _ := games[0]["declined_count"].(float64); int(d) != 1 {
		t.Errorf("expected declined_count=1, got %v", games[0]["declined_count"])
	}
}

// TestGetParticipants_OptOutMarksKaderConfirmed verifies that GetParticipants
// returns rsvp_status='confirmed' for kader members without a response when opt-out.
func TestGetParticipants_OptOutMarksKaderConfirmed(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, _, gameID, _ := optOutFixture(t, db, 2, true)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", gameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var participants []map[string]any
	json.NewDecoder(res.Body).Decode(&participants)
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}
	for _, p := range participants {
		if p["rsvp_status"] != "confirmed" {
			t.Errorf("opt-out kader member should have rsvp_status=confirmed, got %v", p["rsvp_status"])
		}
	}
}

// TestGetParticipants_OptOutExtendedRemainsNull verifies that extended kader
// members do not get implicit confirm on opt-out.
func TestGetParticipants_OptOutExtendedRemainsNull(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, kaderID, gameID, _ := optOutFixture(t, db, 0, true)

	// Only extended members in this fixture.
	extUserID := testutil.CreateUser(t, db, "standard")
	extMemberID := testutil.CreateMember(t, db, extUserID)
	if _, err := db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, extMemberID); err != nil {
		t.Fatalf("extended insert: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", gameID), token)
	defer res.Body.Close()
	var participants []map[string]any
	json.NewDecoder(res.Body).Decode(&participants)
	if len(participants) != 1 {
		t.Fatalf("expected 1 extended participant, got %d", len(participants))
	}
	if participants[0]["rsvp_status"] != nil {
		t.Errorf("extended member must keep rsvp_status=null even at opt-out, got %v", participants[0]["rsvp_status"])
	}
	if participants[0]["is_extended"] != true {
		t.Errorf("expected is_extended=true, got %v", participants[0]["is_extended"])
	}
}

// TestGetParticipants_NoOptOutBehavesAsBefore verifies that without opt-out
// no implicit-confirm logic kicks in.
func TestGetParticipants_NoOptOutBehavesAsBefore(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, _, gameID, _ := optOutFixture(t, db, 2, false)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", gameID), token)
	defer res.Body.Close()
	var participants []map[string]any
	json.NewDecoder(res.Body).Decode(&participants)
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}
	for _, p := range participants {
		if p["rsvp_status"] != nil {
			t.Errorf("non-opt-out: rsvp_status must be null when no response, got %v", p["rsvp_status"])
		}
	}
}

// TestGetGame_ReturnsCounts verifies that the game detail endpoint exposes
// confirmed_count / declined_count / maybe_count, opt-out-aware.
func TestGetGame_ReturnsCounts(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, _, gameID, _ := optOutFixture(t, db, 4, true)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", gameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	game, _ := body["game"].(map[string]any)
	if c, _ := game["confirmed_count"].(float64); int(c) != 4 {
		t.Errorf("expected confirmed_count=4 (4 kader, opt-out), got %v", game["confirmed_count"])
	}
	if _, ok := game["declined_count"]; !ok {
		t.Errorf("response must include declined_count field")
	}
	if _, ok := game["maybe_count"]; !ok {
		t.Errorf("response must include maybe_count field")
	}
}


// helper: insert a team with custom age_class+gender (avoids "Erwachsene"/"mixed" fixture default)
func mkTeamCustom(t *testing.T, db *sql.DB, name, ageClass, gender string) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO teams (name, age_class, gender) VALUES (?, ?, ?)`, name, ageClass, gender)
	if err != nil {
		t.Fatalf("mkTeamCustom: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func mkKaderCustom(t *testing.T, db *sql.DB, seasonID, teamID int, ageClass, gender string, teamNumber int) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, ageClass, gender, teamID, teamNumber)
	if err != nil {
		t.Fatalf("mkKaderCustom: %v", err)
	}
}

// TestListGames_DoppelheimspielDisplayCSV verifies that a game with two teams of the same
// age_class+gender exposes team_display_short_csv and team_display_long_csv with both teams sorted.
func TestListGames_DoppelheimspielDisplayCSV(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	t1 := mkTeamCustom(t, db, "TS B1", "B-Jugend", "m")
	t2 := mkTeamCustom(t, db, "TS B2", "B-Jugend", "m")
	mkKaderCustom(t, db, seasonID, t1, "B-Jugend", "m", 1)
	mkKaderCustom(t, db, seasonID, t2, "B-Jugend", "m", 2)

	gameID := testutil.CreateGame(t, db, seasonID, t1, "2026-02-15")
	db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, t2)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d", seasonID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}
	g := games[0]
	if shortCSV, _ := g["team_display_short_csv"].(string); shortCSV != "mB1, mB2" {
		t.Errorf("expected team_display_short_csv 'mB1, mB2', got %q", shortCSV)
	}
	if longCSV, _ := g["team_display_long_csv"].(string); longCSV != "B-Jugend 1 männlich, B-Jugend 2 männlich" {
		t.Errorf("expected team_display_long_csv 'B-Jugend 1 männlich, B-Jugend 2 männlich', got %q", longCSV)
	}
	teams, _ := g["teams"].([]any)
	if len(teams) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(teams))
	}
	for _, tm := range teams {
		m := tm.(map[string]any)
		if _, ok := m["display_short"]; !ok {
			t.Errorf("team item missing display_short field")
		}
		if _, ok := m["display_long"]; !ok {
			t.Errorf("team item missing display_long field")
		}
	}
}

// TestGetGame_DisplayFields verifies GET /api/games/{id} returns display_short/long per team
// and the aggregate CSV fields.
func TestGetGame_DisplayFields(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := mkTeamCustom(t, db, "Team A", "A-Jugend", "m")
	mkKaderCustom(t, db, seasonID, teamID, "A-Jugend", "m", 1)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-03-10")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", gameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	game, _ := body["game"].(map[string]any)

	if csv, _ := game["team_display_short_csv"].(string); csv != "mA" {
		t.Errorf("expected team_display_short_csv 'mA', got %q", csv)
	}
	if csv, _ := game["team_display_long_csv"].(string); csv != "A-Jugend männlich" {
		t.Errorf("expected team_display_long_csv 'A-Jugend männlich', got %q", csv)
	}
	teams, _ := game["teams"].([]any)
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}
	tm := teams[0].(map[string]any)
	if s, _ := tm["display_short"].(string); s != "mA" {
		t.Errorf("expected display_short 'mA', got %q", s)
	}
	if l, _ := tm["display_long"].(string); l != "A-Jugend männlich" {
		t.Errorf("expected display_long 'A-Jugend männlich', got %q", l)
	}
}

// TestListMyGames_DisplayCSV verifies team_display_short_csv/long_csv are populated for /api/games/my.
func TestListMyGames_DisplayCSV(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	t1 := mkTeamCustom(t, db, "TS B1", "B-Jugend", "m")
	t2 := mkTeamCustom(t, db, "TS B2", "B-Jugend", "m")
	k1 := mkKaderCustomReturn(t, db, seasonID, t1, "B-Jugend", "m", 1)
	mkKaderCustom(t, db, seasonID, t2, "B-Jugend", "m", 2)

	gameID := testutil.CreateGame(t, db, seasonID, t1, "2026-02-20")
	db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, t2)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, k1, memberID)

	srv := testServer(t, db)
	token := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv, "/api/games/my", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var gameList []map[string]any
	json.NewDecoder(res.Body).Decode(&gameList)
	if len(gameList) != 1 {
		t.Fatalf("expected 1 game, got %d", len(gameList))
	}
	g := gameList[0]
	if shortCSV, _ := g["team_display_short_csv"].(string); shortCSV != "mB1, mB2" {
		t.Errorf("expected team_display_short_csv 'mB1, mB2', got %q", shortCSV)
	}
	if longCSV, _ := g["team_display_long_csv"].(string); longCSV != "B-Jugend 1 männlich, B-Jugend 2 männlich" {
		t.Errorf("expected team_display_long_csv 'B-Jugend 1 männlich, B-Jugend 2 männlich', got %q", longCSV)
	}
}

// helper: insert kader and return ID
func mkKaderCustomReturn(t *testing.T, db *sql.DB, seasonID, teamID int, ageClass, gender string, teamNumber int) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, ageClass, gender, teamID, teamNumber)
	if err != nil {
		t.Fatalf("mkKaderCustomReturn: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}
