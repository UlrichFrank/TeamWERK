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
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
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
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
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
