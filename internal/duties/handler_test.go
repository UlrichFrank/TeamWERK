package duties_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── local helpers ─────────────────────────────────────────────────────────────

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func insertDutyAssignment(t *testing.T, db *sql.DB, slotID, userID int, status string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO duty_assignments (duty_slot_id, user_id, status) VALUES (?, ?, ?)`,
		slotID, userID, status); err != nil {
		t.Fatalf("insertDutyAssignment: %v", err)
	}
}

func slotsFilled(t *testing.T, db *sql.DB, slotID int) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT slots_filled FROM duty_slots WHERE id=?`, slotID).Scan(&n); err != nil {
		t.Fatalf("slotsFilled: %v", err)
	}
	return n
}

func countRows(t *testing.T, db *sql.DB, table, where string, args ...any) int {
	t.Helper()
	var n int
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, table, where)
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("countRows(%s): %v", q, err)
	}
	return n
}

// createDutyType inserts a duty type and returns its ID.
func createDutyType(t *testing.T, db *sql.DB, name string, hoursValue float64) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO duty_types (name, hours_value) VALUES (?, ?)`, name, hoursValue)
	if err != nil {
		t.Fatalf("createDutyType: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// createDutySlot inserts a duty slot with slots_total=2, slots_filled=0 and
// returns its ID. Pass gameID=0 to leave game_id NULL.
func createDutySlot(t *testing.T, db *sql.DB, dutyTypeID, seasonID, teamID, gameID int, date string) int {
	t.Helper()
	var gameArg any
	if gameID > 0 {
		gameArg = gameID
	}
	res, err := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, slots_filled, team_id, season_id, game_id)
		 VALUES (?, ?, ?, 2, 0, ?, ?, ?)`,
		"Testdienst", date, dutyTypeID, teamID, seasonID, gameArg)
	if err != nil {
		t.Fatalf("createDutySlot: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// addPlayerMembership inserts a kader row (if needed) and links the member.
// This is the underlying mechanism behind the player_memberships VIEW.
func addPlayerMembership(t *testing.T, db *sql.DB, memberID, teamID, seasonID int) {
	t.Helper()
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("addPlayerMembership kader_members: %v", err)
	}
}

// testServer wires up a duties handler on a fresh httptest.Server.
func testServer(t *testing.T, h *duties.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/duty-board", h.Board)
		r.Post("/api/duty-board/{slotId}/claim", h.Claim)
		r.Delete("/api/duty-board/{slotId}/claim", h.Unclaim)
		r.Get("/api/duty-accounts", h.Accounts)
		r.Post("/api/duty-slots", h.CreateSlot)
		r.Put("/api/duty-slots/{id}", h.UpdateSlot)
		r.Delete("/api/duty-slots/{id}", h.DeleteSlot)
		r.Get("/api/duty-slots/{id}/assignments", h.ListAssignments)
		r.Post("/api/duty-assignments/{id}/fulfill", h.Fulfill)
		r.Post("/api/duty-assignments/{id}/cash-substitute", h.CashSubstitute)
		r.Get("/api/duty-types/{id}/instruction", h.GetInstruction)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Get("/api/duty-types", h.ListTypes)
		})
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand"))
			r.Put("/api/duty-types/{id}/instruction", h.SetInstruction)
		})
	})
}

// ── TC-D01 ────────────────────────────────────────────────────────────────────

// TestClaim_FreeSlot verifies that claiming an open slot succeeds with 204,
// increments slots_filled to 1, creates a duty_assignment with status=assigned,
// and ensures a duty_accounts row exists for the user.
func TestClaim_FreeSlot(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := slotsFilled(t, db, slotID); got != 1 {
		t.Errorf("expected slots_filled=1, got %d", got)
	}
	if got := countRows(t, db, "duty_assignments",
		"duty_slot_id=? AND user_id=? AND status='assigned'", slotID, userID); got != 1 {
		t.Errorf("expected 1 duty_assignment with status=assigned, got %d", got)
	}
	if got := countRows(t, db, "duty_accounts",
		"user_id=? AND season_id=?", userID, seasonID); got != 1 {
		t.Errorf("expected duty_accounts row, got %d", got)
	}
}

// ── TC-D02 ────────────────────────────────────────────────────────────────────

// TestClaim_FullSlot verifies that claiming a slot that is already full
// returns 409 Conflict.
func TestClaim_FullSlot(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	// Slot has slots_total=2 by default; set slots_filled=2 to make it full.
	if _, err := db.Exec(`UPDATE duty_slots SET slots_total=1, slots_filled=1 WHERE id=?`, slotID); err != nil {
		t.Fatalf("set slot full: %v", err)
	}

	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", res.StatusCode)
	}
}

// ── TC-D03 ────────────────────────────────────────────────────────────────────

// TestClaim_Duplicate verifies that claiming a slot the user has already
// claimed returns 409 Conflict (UNIQUE constraint violation).
func TestClaim_Duplicate(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	// Manually reflect the fill count so the slot-full check does not fire first.
	db.Exec(`UPDATE duty_slots SET slots_filled=1 WHERE id=?`, slotID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", res.StatusCode)
	}
}

// ── TC-D04 ────────────────────────────────────────────────────────────────────

// TestUnclaim_Pending verifies that unclaiming an own assignment with
// status=assigned succeeds with 204, decrements slots_filled, and removes
// the assignment row.
func TestUnclaim_Pending(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")

	insertDutyAssignment(t, db, slotID, userID, "assigned")
	db.Exec(`UPDATE duty_slots SET slots_filled=1 WHERE id=?`, slotID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := slotsFilled(t, db, slotID); got != 0 {
		t.Errorf("expected slots_filled=0 after unclaim, got %d", got)
	}
	if got := countRows(t, db, "duty_assignments",
		"duty_slot_id=? AND user_id=?", slotID, userID); got != 0 {
		t.Errorf("expected assignment deleted, got %d rows", got)
	}
}

// ── TC-D05 ────────────────────────────────────────────────────────────────────

// TestUnclaim_Fulfilled verifies that attempting to unclaim a fulfilled
// assignment returns 409 Conflict.
func TestUnclaim_Fulfilled(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")

	insertDutyAssignment(t, db, slotID, userID, "fulfilled")
	db.Exec(`UPDATE duty_slots SET slots_filled=1 WHERE id=?`, slotID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", res.StatusCode)
	}
}

// ── TC-D06 ────────────────────────────────────────────────────────────────────

// TestUnclaim_NotFound verifies that unclaiming a slot the user never claimed
// returns 404.
func TestUnclaim_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

// ── TC-D07 ────────────────────────────────────────────────────────────────────

// TestClaim_ForProxyChild verifies that a parent user can claim a duty slot
// on behalf of a proxy child (can_login=0) that is linked via family_links.
// The resulting assignment must be owned by the child's user ID.
func TestClaim_ForProxyChild(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	parentUserID := testutil.CreateUser(t, db, "standard")

	// Create proxy child: user with can_login=0.
	childUserID := testutil.CreateUser(t, db, "standard")
	db.Exec(`UPDATE users SET can_login=0 WHERE id=?`, childUserID)

	childMemberID := testutil.CreateMember(t, db, childUserID)

	// Link parent to child member via family_links.
	if _, err := db.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUserID, childMemberID); err != nil {
		t.Fatalf("insert family_links: %v", err)
	}

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, parentUserID, "elternteil", nil)
	body := map[string]any{"user_id": childUserID}
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, body)
	res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	// Assignment must belong to the child, not the parent.
	if got := countRows(t, db, "duty_assignments",
		"duty_slot_id=? AND user_id=?", slotID, childUserID); got != 1 {
		t.Errorf("expected assignment for child user %d, got %d rows", childUserID, got)
	}
	if got := countRows(t, db, "duty_assignments",
		"duty_slot_id=? AND user_id=?", slotID, parentUserID); got != 0 {
		t.Errorf("expected no assignment for parent user, got %d rows", got)
	}
}

// ── TC-D08 ────────────────────────────────────────────────────────────────────

// TestClaim_ForeignUserForbidden verifies that claiming on behalf of a user
// who is not a family-linked proxy child of the caller returns 403 Forbidden.
func TestClaim_ForeignUserForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	callerUserID := testutil.CreateUser(t, db, "standard")
	foreignUserID := testutil.CreateUser(t, db, "standard")
	// No family_links entry — foreignUser is unrelated.

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, callerUserID, "spieler", nil)
	body := map[string]any{"user_id": foreignUserID}
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, body)
	res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// ── TC-D09 ────────────────────────────────────────────────────────────────────

// TestBoard_AdminSeesAll verifies that an admin user sees all duty slots in
// the active season regardless of which team they belong to.
func TestBoard_AdminSeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	teamC := testutil.CreateTeam(t, db, "Team C")
	dtID := createDutyType(t, db, "Kasse", 1.0)

	createDutySlot(t, db, dtID, seasonID, teamA, 0, "2026-06-14")
	createDutySlot(t, db, dtID, seasonID, teamB, 0, "2026-06-15")
	createDutySlot(t, db, dtID, seasonID, teamC, 0, "2026-06-16")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	// Count total slots across all groups.
	total := 0
	for _, g := range groups {
		if slots, ok := g["slots"].([]any); ok {
			total += len(slots)
		}
	}
	if total != 3 {
		t.Errorf("admin: expected 3 slots total, got %d", total)
	}
}

// TestBoard_VorstandSeesAllTeams verifies that a user with the Vereinsfunktion
// 'vorstand' (system role 'standard') sees all duty slots of all teams in the
// active season — same scope as an admin, even when not a member of any team.
func TestBoard_VorstandSeesAllTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	teamC := testutil.CreateTeam(t, db, "Team C")
	dtID := createDutyType(t, db, "Kasse", 1.0)

	createDutySlot(t, db, dtID, seasonID, teamA, 0, "2026-06-14")
	createDutySlot(t, db, dtID, seasonID, teamB, 0, "2026-06-15")
	createDutySlot(t, db, dtID, seasonID, teamC, 0, "2026-06-16")

	// Vorstand user: standard role, no player_memberships, but has the
	// vorstand club function via member_club_functions.
	vorstandUserID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, vorstandUserID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'vorstand')`, memberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	total := 0
	for _, g := range groups {
		if slots, ok := g["slots"].([]any); ok {
			total += len(slots)
		}
	}
	if total != 3 {
		t.Errorf("vorstand: expected 3 slots across all teams, got %d", total)
	}
}

// groupTeamIDs extracts the team_ids array of a board group as []int.
func groupTeamIDs(t *testing.T, grp map[string]any) []int {
	t.Helper()
	raw, ok := grp["team_ids"].([]any)
	if !ok {
		t.Fatalf("expected team_ids array in group, got %T (%v)", grp["team_ids"], grp["team_ids"])
	}
	ids := make([]int, 0, len(raw))
	for _, v := range raw {
		f, ok := v.(float64)
		if !ok {
			t.Fatalf("expected numeric team id, got %T (%v)", v, v)
		}
		ids = append(ids, int(f))
	}
	return ids
}

// TestBoard_GamelessSlotCarriesSlotTeam verifies that a game-less handslot
// exposes its own ds.team_id via the team_ids array — required by the frontend
// team filter.
func TestBoard_GamelessSlotCarriesSlotTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kasse", 1.0)
	createDutySlot(t, db, dtID, seasonID, teamA, 0, "2026-06-14")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	ids := groupTeamIDs(t, groups[0])
	if len(ids) != 1 || ids[0] != teamA {
		t.Errorf("expected team_ids=[%d], got %v", teamA, ids)
	}
}

// TestBoard_GamelessSlotWithoutTeamHasEmptyTeamIDs verifies that a game-less
// slot without a team yields an empty team_ids array (never null), so the
// frontend .includes() filter needs no null guard.
func TestBoard_GamelessSlotWithoutTeamHasEmptyTeamIDs(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	dtID := createDutyType(t, db, "Vereinsfest", 4.0)
	// team_id NULL, game_id NULL → game-loser Dienst ohne Team.
	if _, err := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, slots_filled, team_id, season_id, game_id)
		 VALUES (?, ?, ?, 2, 0, NULL, ?, NULL)`,
		"Sommerfest", "2026-06-14", dtID, seasonID); err != nil {
		t.Fatalf("insert slot: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if ids := groupTeamIDs(t, groups[0]); len(ids) != 0 {
		t.Errorf("expected empty team_ids, got %v", ids)
	}
}

// TestBoard_GameGroupCarriesTerminTeams verifies that a game-based group derives
// its teams from game_teams — including ALL teams of a multi-team fixture, not
// just the slot's ds.team_id.
func TestBoard_GameGroupCarriesTerminTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	gameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-06-14")
	// second team on the same fixture
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamB); err != nil {
		t.Fatalf("game_teams teamB: %v", err)
	}
	dtID := createDutyType(t, db, "Kasse", 1.0)
	// slot carries only teamA, but the group must expose both A and B.
	createDutySlot(t, db, dtID, seasonID, teamA, gameID, "2026-06-14")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	ids := groupTeamIDs(t, groups[0])
	if len(ids) != 2 || !containsInt(ids, teamA) || !containsInt(ids, teamB) {
		t.Errorf("expected team_ids to contain %d and %d, got %v", teamA, teamB, ids)
	}
	names, ok := groups[0]["team_names"].([]any)
	if !ok || len(names) != len(ids) {
		t.Errorf("expected team_names positionally aligned with team_ids, got %v", groups[0]["team_names"])
	}
}

// TestBoard_GenericEventCarriesTerminTeams verifies that a generic event, whose
// slots have ds.team_id=NULL, still exposes the team(s) of its Termin via
// game_teams.
func TestBoard_GenericEventCarriesTerminTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	// generic event as a game row with its game_teams link
	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, event_type, is_home) VALUES (?, ?, ?, ?, 'generisch', 0)`,
		seasonID, "Vereinsfest", "2026-06-14", "10:00")
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	gameID64, _ := res.LastInsertId()
	gameID := int(gameID64)
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamA); err != nil {
		t.Fatalf("game_teams: %v", err)
	}
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	// slot with team_id NULL but linked to the generic event
	if _, err := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, slots_filled, team_id, season_id, game_id)
		 VALUES (?, ?, ?, 2, 0, NULL, ?, ?)`,
		"Aufbau", "2026-06-14", dtID, seasonID, gameID); err != nil {
		t.Fatalf("insert slot: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	got := testutil.Get(t, srv, "/api/duty-board", token)
	if got.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", got.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(got.Body).Decode(&groups)
	got.Body.Close()

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	ids := groupTeamIDs(t, groups[0])
	if len(ids) != 1 || ids[0] != teamA {
		t.Errorf("expected team_ids=[%d] from game_teams despite NULL slot team_id, got %v", teamA, ids)
	}
}

// TestBoard_GameIDNullGroupHasGenericEventType verifies that game-less duty
// groups (e.g. Vereinsfest aufbau) carry event_type="generisch" in the response
// so the frontend "Sonstiges" pill can include them.
func TestBoard_GameIDNullGroupHasGenericEventType(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Vereinsfest", 4.0)
	// gameID=0 → game_id NULL, i.e. game-loser Vereinsdienst.
	createDutySlot(t, db, dtID, seasonID, teamA, 0, "2026-06-14")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0]["game_id"] != nil {
		t.Errorf("expected game_id=nil, got %v", groups[0]["game_id"])
	}
	if et, _ := groups[0]["event_type"].(string); et != "generisch" {
		t.Errorf("expected event_type=\"generisch\", got %q", et)
	}
}

// ── TC-D10 ────────────────────────────────────────────────────────────────────

// TestBoard_UserSeesOwnTeam verifies that a normal user only sees duty slots
// for the team their member profile is registered in via player_memberships.
func TestBoard_UserSeesOwnTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	dtID := createDutyType(t, db, "Aufbau", 2.0)

	slotA := createDutySlot(t, db, dtID, seasonID, teamA, 0, "2026-06-14")
	createDutySlot(t, db, dtID, seasonID, teamB, 0, "2026-06-15")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	addPlayerMembership(t, db, memberID, teamA, seasonID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	// Only Team A's slot must appear.
	total := 0
	for _, g := range groups {
		if slots, ok := g["slots"].([]any); ok {
			for _, rawSlot := range slots {
				s := rawSlot.(map[string]any)
				slotIDFloat, _ := s["id"].(float64)
				if int(slotIDFloat) != slotA {
					t.Errorf("unexpected slot id %v in board", slotIDFloat)
				}
				total++
			}
		}
	}
	if total != 1 {
		t.Errorf("expected 1 slot (team A), got %d", total)
	}
}

// ── TC-D11 ────────────────────────────────────────────────────────────────────

// TestBoard_AudienceElternVisible verifies that a slot with audiences=["eltern"]
// is visible to a user whose linked child plays in the slot's team.
func TestBoard_AudienceElternVisible(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Elterndienst", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["eltern"]' WHERE id=?`, slotID)

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	// Parent has a family_links entry AND the child is in the slot's team.
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUserID, childMemberID)
	addPlayerMembership(t, db, childMemberID, teamID, seasonID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, parentUserID, "elternteil", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	total := 0
	for _, g := range groups {
		if slots, ok := g["slots"].([]any); ok {
			total += len(slots)
		}
	}
	if total != 1 {
		t.Errorf("expected 1 eltern slot visible to parent, got %d", total)
	}
}

// ── TC-D12 ────────────────────────────────────────────────────────────────────

// TestBoard_AudienceElternHidden verifies that a slot with audiences=["eltern"]
// is NOT visible to a user without any family_links entry.
func TestBoard_AudienceElternHidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Elterndienst", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["eltern"]' WHERE id=?`, slotID)

	// Regular player with no family_links entry.
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	addPlayerMembership(t, db, memberID, teamID, seasonID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	total := 0
	for _, g := range groups {
		if slots, ok := g["slots"].([]any); ok {
			total += len(slots)
		}
	}
	if total != 0 {
		t.Errorf("expected 0 eltern slots for user without family_links, got %d", total)
	}
}

// TestBoard_AudienceElternTeamScoped verifies that a parent does NOT match
// the 'eltern' audience on slots of a team where their child does not play —
// even if the parent themselves is visible on the board through another
// channel (here: as trainer of the slot's team).
func TestBoard_AudienceElternTeamScoped(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	dtID := createDutyType(t, db, "Elterndienst", 1.0)
	slotA := createDutySlot(t, db, dtID, seasonID, teamA, 0, "2026-06-14")
	db.Exec(`UPDATE duty_slots SET audiences='["eltern"]' WHERE id=?`, slotA)

	// Parent is trainer of team A (so the slot is visible via team source)
	parentUserID := testutil.CreateUser(t, db, "standard")
	parentMemberID := testutil.CreateMember(t, db, parentUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, parentMemberID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'trainer')`, parentMemberID)

	// Parent's child plays in team B, not in team A
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)
	addPlayerMembership(t, db, childMemberID, teamB, seasonID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	// Default (audience filter active): trainer audience does NOT match ['eltern'],
	// and the eltern audience does NOT match because the child plays in team B not A.
	token := testutil.Token(t, parentUserID, "standard", []string{"trainer"})
	if n := boardSlotCount(t, srv, "", token); n != 0 {
		t.Errorf("eltern slot should be hidden when child does not play in slot's team, got %d", n)
	}
	// audience=all reveals the slot via the team source
	if n := boardSlotCount(t, srv, "?audience=all", token); n != 1 {
		t.Errorf("with audience=all the slot should be visible via trainer team source, got %d", n)
	}
}

// ── TC-D13 ────────────────────────────────────────────────────────────────────

// boardSlotCount runs GET /api/duty-board with the given query string and token
// and returns the total number of slot entries in the response.
func boardSlotCount(t *testing.T, srv *httptest.Server, query, token string) int {
	t.Helper()
	res := testutil.Get(t, srv, "/api/duty-board"+query, token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()
	total := 0
	for _, g := range groups {
		if slots, ok := g["slots"].([]any); ok {
			total += len(slots)
		}
	}
	return total
}

// TestDutyBoard_TrainerSeesOwnTeam verifies that a trainer linked to a team
// via kader_trainers (but not as a player) sees that team's duty slots.
func TestDutyBoard_TrainerSeesOwnTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Hallendienst", 2.0)
	createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	// audience=all so the audience filter does not hide the slot
	if n := boardSlotCount(t, srv, "?audience=all", token); n != 1 {
		t.Errorf("trainer should see 1 slot of own team, got %d", n)
	}
}

// TestDutyBoard_TrainerDoesNotSeeOtherTeams verifies that a trainer does not
// see slots of teams they neither train nor play in.
func TestDutyBoard_TrainerDoesNotSeeOtherTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	dtID := createDutyType(t, db, "Hallendienst", 2.0)
	createDutySlot(t, db, dtID, seasonID, teamA, 0, "2026-06-14")
	createDutySlot(t, db, dtID, seasonID, teamB, 0, "2026-06-14")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	if n := boardSlotCount(t, srv, "?audience=all", token); n != 1 {
		t.Errorf("trainer should see only Team A slot (1), got %d", n)
	}
}

// TestDutyBoard_TrainerAudienceFilterDefault verifies that without
// ?audience=all, a trainer only sees slots whose audience matches their
// function (or is NULL), and NOT slots restricted to other audiences.
func TestDutyBoard_TrainerAudienceFilterDefault(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Dienst", 1.0)
	matchSlot := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	otherSlot := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	nullSlot := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["trainer"]' WHERE id=?`, matchSlot)
	db.Exec(`UPDATE duty_slots SET audiences='["spieler"]' WHERE id=?`, otherSlot)
	// nullSlot keeps audiences=NULL

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'trainer')`, trainerMemberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	// No ?audience param → filter active, trainer sees matchSlot + nullSlot but not otherSlot
	if n := boardSlotCount(t, srv, "", token); n != 2 {
		t.Errorf("trainer should see 2 slots (audience=trainer + NULL), got %d", n)
	}
	_ = nullSlot
}

// TestDutyBoard_TrainerAudienceAll verifies that ?audience=all reveals all
// slots of the trainer's teams regardless of audience.
func TestDutyBoard_TrainerAudienceAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Dienst", 1.0)
	a := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	b := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["trainer"]' WHERE id=?`, a)
	db.Exec(`UPDATE duty_slots SET audiences='["spieler"]' WHERE id=?`, b)

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	if n := boardSlotCount(t, srv, "?audience=all", token); n != 2 {
		t.Errorf("trainer with audience=all should see 2 slots, got %d", n)
	}
}

// TestDutyBoard_VorstandAudienceFilterDefault verifies that a vorstand
// (Vereinsfunktion vorstand) sees only audience-matching slots by default
// and all slots with ?audience=all.
func TestDutyBoard_VorstandAudienceFilterDefault(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Dienst", 1.0)
	matchSlot := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	otherSlot := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["vorstand"]' WHERE id=?`, matchSlot)
	db.Exec(`UPDATE duty_slots SET audiences='["spieler"]' WHERE id=?`, otherSlot)

	vorstandUserID := testutil.CreateUser(t, db, "standard")
	vorstandMemberID := testutil.CreateMember(t, db, vorstandUserID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'vorstand')`, vorstandMemberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})

	if n := boardSlotCount(t, srv, "", token); n != 1 {
		t.Errorf("vorstand default should see 1 audience-matching slot, got %d", n)
	}
	if n := boardSlotCount(t, srv, "?audience=all", token); n != 2 {
		t.Errorf("vorstand with audience=all should see 2 slots, got %d", n)
	}
}

// TestDutyBoard_SpielerAudienceAllIgnored verifies that a non-privileged
// player cannot disable the audience filter via ?audience=all.
func TestDutyBoard_SpielerAudienceAllIgnored(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Dienst", 1.0)
	matchSlot := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	hiddenSlot := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["spieler"]' WHERE id=?`, matchSlot)
	db.Exec(`UPDATE duty_slots SET audiences='["vorstand"]' WHERE id=?`, hiddenSlot)

	playerUserID := testutil.CreateUser(t, db, "standard")
	playerMemberID := testutil.CreateMember(t, db, playerUserID)
	addPlayerMembership(t, db, playerMemberID, teamID, seasonID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, playerMemberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, playerUserID, "standard", []string{"spieler"})
	if n := boardSlotCount(t, srv, "?audience=all", token); n != 1 {
		t.Errorf("player should see only audience-matching slot even with audience=all, got %d", n)
	}
}

// TestDutyBoard_AdminAudienceBypass verifies that admin role always sees
// every slot regardless of audience and query param.
func TestDutyBoard_AdminAudienceBypass(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Dienst", 1.0)
	a := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	b := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["spieler"]' WHERE id=?`, a)
	db.Exec(`UPDATE duty_slots SET audiences='["eltern"]' WHERE id=?`, b)

	adminUserID := testutil.CreateUser(t, db, "admin")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	if n := boardSlotCount(t, srv, "", token); n != 2 {
		t.Errorf("admin should see all slots without param, got %d", n)
	}
	if n := boardSlotCount(t, srv, "?audience=all", token); n != 2 {
		t.Errorf("admin should see all slots with audience=all, got %d", n)
	}
}

// ── TC-D14 ────────────────────────────────────────────────────────────────────

// TestBoard_ViewMine verifies that GET /api/duty-board?view=mine returns only
// the slots the requesting user has claimed.
func TestBoard_ViewMine(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)

	userID := testutil.CreateUser(t, db, "admin")

	var slotIDs []int
	for i := range 5 {
		date := fmt.Sprintf("2026-06-%02d", 14+i)
		slotIDs = append(slotIDs, createDutySlot(t, db, dtID, seasonID, teamID, 0, date))
	}
	// Claim exactly 2 of the 5 slots.
	insertDutyAssignment(t, db, slotIDs[0], userID, "assigned")
	insertDutyAssignment(t, db, slotIDs[2], userID, "assigned")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-board?view=mine", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)
	res.Body.Close()

	total := 0
	for _, g := range groups {
		if slots, ok := g["slots"].([]any); ok {
			total += len(slots)
		}
	}
	if total != 2 {
		t.Errorf("view=mine: expected 2 claimed slots, got %d", total)
	}
}

// ── TC-D15a ───────────────────────────────────────────────────────────────────

// TestAccounts_AdminSeesAll verifies that an admin receives all duty accounts,
// each with a correctly computed balance (soll - ist).
func TestAccounts_AdminSeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")

	user1 := testutil.CreateUser(t, db, "standard")
	user2 := testutil.CreateUser(t, db, "standard")
	user3 := testutil.CreateUser(t, db, "standard")
	adminID := testutil.CreateUser(t, db, "admin")

	db.Exec(`INSERT INTO duty_accounts (user_id, season_id, soll, ist) VALUES (?, ?, 10, 4)`, user1, seasonID)
	db.Exec(`INSERT INTO duty_accounts (user_id, season_id, soll, ist) VALUES (?, ?, 8, 8)`, user2, seasonID)
	db.Exec(`INSERT INTO duty_accounts (user_id, season_id, soll, ist) VALUES (?, ?, 6, 2)`, user3, seasonID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-accounts", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var accounts []map[string]any
	json.NewDecoder(res.Body).Decode(&accounts)
	res.Body.Close()

	if len(accounts) != 3 {
		t.Fatalf("admin: expected 3 accounts, got %d", len(accounts))
	}
	for _, a := range accounts {
		soll := a["soll"].(float64)
		ist := a["ist"].(float64)
		balance := a["balance"].(float64)
		if balance != soll-ist {
			t.Errorf("balance mismatch: soll=%.1f, ist=%.1f, balance=%.1f", soll, ist, balance)
		}
	}
}

// ── TC-D15b ───────────────────────────────────────────────────────────────────

// TestAccounts_UserSeesOwn verifies that a non-admin user only receives their own account.
func TestAccounts_UserSeesOwn(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")

	userID := testutil.CreateUser(t, db, "standard")
	otherID := testutil.CreateUser(t, db, "standard")

	db.Exec(`INSERT INTO duty_accounts (user_id, season_id, soll, ist) VALUES (?, ?, 10, 4)`, userID, seasonID)
	db.Exec(`INSERT INTO duty_accounts (user_id, season_id, soll, ist) VALUES (?, ?, 8, 3)`, otherID, seasonID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Get(t, srv, "/api/duty-accounts", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var accounts []map[string]any
	json.NewDecoder(res.Body).Decode(&accounts)
	res.Body.Close()

	if len(accounts) != 1 {
		t.Fatalf("expected 1 account (own), got %d", len(accounts))
	}
	if int(accounts[0]["user_id"].(float64)) != userID {
		t.Errorf("expected own user_id=%d, got %v", userID, accounts[0]["user_id"])
	}
}

// ── TC-D16 ────────────────────────────────────────────────────────────────────

// TestCreateSlot_IsCustom verifies that POSTing to /api/duty-slots creates a
// slot with is_custom=1.
func TestCreateSlot_IsCustom(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)

	adminID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminID, "admin", nil)
	body := map[string]any{
		"event_name":   "Aufbau Heimspiel",
		"event_date":   "2026-06-14",
		"duty_type_id": dtID,
		"slots_total":  2,
		"team_id":      teamID,
		"season_id":    seasonID,
	}
	res := testutil.Post(t, srv, "/api/duty-slots", token, body)
	res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var isCustom int
	if err := db.QueryRow(`SELECT is_custom FROM duty_slots WHERE team_id=? AND season_id=?`,
		teamID, seasonID).Scan(&isCustom); err != nil {
		t.Fatalf("read is_custom: %v", err)
	}
	if isCustom != 1 {
		t.Errorf("expected is_custom=1, got %d", isCustom)
	}
}

// ── TC-D17 ────────────────────────────────────────────────────────────────────

// TestUpdateSlot_IsCustom verifies that PUTting to /api/duty-slots/{id} on a
// slot that has is_custom=0 flips it to is_custom=1.
func TestUpdateSlot_IsCustom(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	// Confirm slot starts with is_custom=0 (default from CreateDutySlot).
	var isCustomBefore int
	db.QueryRow(`SELECT is_custom FROM duty_slots WHERE id=?`, slotID).Scan(&isCustomBefore)
	if isCustomBefore != 0 {
		t.Fatalf("precondition: expected is_custom=0, got %d", isCustomBefore)
	}

	adminID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminID, "admin", nil)
	body := map[string]any{
		"event_name":  "Aufbau (geändert)",
		"event_date":  "2026-06-14",
		"slots_total": 3,
	}
	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-slots/"+itoa(slotID), token, body)
	res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var isCustomAfter int
	if err := db.QueryRow(`SELECT is_custom FROM duty_slots WHERE id=?`, slotID).Scan(&isCustomAfter); err != nil {
		t.Fatalf("read is_custom: %v", err)
	}
	if isCustomAfter != 1 {
		t.Errorf("expected is_custom=1 after update, got %d", isCustomAfter)
	}
}

// ── TC-D18 ────────────────────────────────────────────────────────────────────

// TestDeleteSlot_WithAssignments verifies that deleting a duty slot that has
// existing assignments succeeds with 204 and that both the slot and its
// assignments are gone (cascade delete).
func TestDeleteSlot_WithAssignments(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	helperID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, helperID, "assigned")

	if got := countRows(t, db, "duty_assignments", "duty_slot_id=?", slotID); got != 1 {
		t.Fatalf("precondition: expected 1 assignment, got %d", got)
	}

	adminID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-slots/"+itoa(slotID), token, nil)
	res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_slots", "id=?", slotID); got != 0 {
		t.Errorf("slot not deleted: got %d rows", got)
	}
	if got := countRows(t, db, "duty_assignments", "duty_slot_id=?", slotID); got != 0 {
		t.Errorf("assignments not cascade-deleted: got %d rows", got)
	}
}

// ── Fulfill / CashSubstitute / ListAssignments ────────────────────────────────

// TC: Fulfill setzt status='fulfilled'; duty_accounts.ist bleibt unverändert.
func TestFulfill_SetsStatusFulfilled(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	var assignmentID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`, slotID, userID).Scan(&assignmentID)

	// Seed duty_accounts so we can verify ist stays unchanged.
	db.Exec(`INSERT INTO duty_accounts (user_id, season_id, soll, ist) VALUES (?, ?, 10, 0)`, userID, seasonID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	trainerID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, trainerID, "admin", nil)

	res := testutil.Post(t, srv,
		"/api/duty-assignments/"+itoa(assignmentID)+"/fulfill", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var status string
	db.QueryRow(`SELECT status FROM duty_assignments WHERE id=?`, assignmentID).Scan(&status)
	if status != "fulfilled" {
		t.Errorf("expected status='fulfilled', got %q", status)
	}
	// Invariante: Fulfill aktualisiert duty_accounts.ist NICHT direkt.
	var ist float64
	db.QueryRow(`SELECT ist FROM duty_accounts WHERE user_id=? AND season_id=?`, userID, seasonID).Scan(&ist)
	if ist != 0 {
		t.Errorf("duty_accounts.ist must remain 0 after Fulfill (updated separately); got %v", ist)
	}
}

// TC: CashSubstitute setzt status='cash_substitute' und cash_amount.
func TestCashSubstitute_SetsStatusAndAmount(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyType(t, db, "Kasse", 1.0)
	slotID := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	var assignmentID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`, slotID, userID).Scan(&assignmentID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	trainerID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, trainerID, "admin", nil)

	res := testutil.Post(t, srv,
		"/api/duty-assignments/"+itoa(assignmentID)+"/cash-substitute", token,
		map[string]float64{"amount": 15.0})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var status string
	var cashAmount float64
	db.QueryRow(`SELECT status, COALESCE(cash_amount,0) FROM duty_assignments WHERE id=?`, assignmentID).Scan(&status, &cashAmount)
	if status != "cash_substitute" {
		t.Errorf("expected status='cash_substitute', got %q", status)
	}
	if cashAmount != 15.0 {
		t.Errorf("expected cash_amount=15.0, got %v", cashAmount)
	}
}

// TC: ListAssignments gibt alle Zuweisungen eines Slots zurück.
func TestListAssignments_ReturnsAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")

	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userA, "assigned")
	insertDutyAssignment(t, db, slotID, userB, "fulfilled")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	trainerID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, trainerID, "admin", nil)

	res := testutil.Get(t, srv, "/api/duty-slots/"+itoa(slotID)+"/assignments", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	if len(items) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(items))
	}
	for _, item := range items {
		if item["user_name"] == nil || item["status"] == nil {
			t.Errorf("assignment missing user_name or status: %v", item)
		}
	}
}

// TC-SEC-D01: Concurrent claim of last slot — exactly one succeeds, no overclaim.
func TestClaimDutySlot_NoConcurrentOverclaim(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26") // CreateSeason activates the season
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyType(t, db, "Kassierer", 2.0)

	// Slot with slots_total=1 — only one claim can succeed.
	res, _ := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, slots_filled, team_id, season_id)
		 VALUES ('Concurrency Test', '2026-06-20', ?, 1, 0, ?, ?)`,
		dutyTypeID, teamID, seasonID)
	slotID, _ := res.LastInsertId()

	user1 := testutil.CreateUser(t, db, "standard")
	user2 := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token1 := testutil.Token(t, user1, "spieler", nil)
	token2 := testutil.Token(t, user2, "spieler", nil)

	var wg sync.WaitGroup
	statuses := make([]int, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		r := testutil.Post(t, srv, "/api/duty-board/"+itoa(int(slotID))+"/claim", token1, nil)
		r.Body.Close()
		statuses[0] = r.StatusCode
	}()
	go func() {
		defer wg.Done()
		r := testutil.Post(t, srv, "/api/duty-board/"+itoa(int(slotID))+"/claim", token2, nil)
		r.Body.Close()
		statuses[1] = r.StatusCode
	}()
	wg.Wait()

	successes := 0
	for _, s := range statuses {
		if s == http.StatusNoContent {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 successful claim, got %d (statuses: %v)", successes, statuses)
	}

	filled := slotsFilled(t, db, int(slotID))
	if filled != 1 {
		t.Errorf("slots_filled should be 1 after one claim, got %d (no overclaim)", filled)
	}

	assignments := countRows(t, db, "duty_assignments", "duty_slot_id=?", slotID)
	if assignments != 1 {
		t.Errorf("expected 1 assignment row, got %d", assignments)
	}
}

// TestListDutyTypes_TrainerCanRead verifies that a user with club_function=trainer
// can read duty types (GET /api/duty-types returns 200).
func TestListDutyTypes_TrainerCanRead(t *testing.T) {
	db := testutil.NewDB(t)
	createDutyType(t, db, "Aufbau", 2.0)

	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/duty-types", token)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for trainer, got %d", res.StatusCode)
	}
}

// CHECK-Constraint auf duty_types.target_role nach Migration 042: 'admin' ist als
// target_role nicht mehr erlaubt (System-Admins sind keine Duty-Zielgruppe).
// Erlaubt sind die Vereinsfunktionen plus 'elternteil'.
func TestDutyType_TargetRole_RejectsAdmin(t *testing.T) {
	db := testutil.NewDB(t)

	_, err := db.Exec(`INSERT INTO duty_types (name, hours_value, target_role) VALUES (?, ?, 'admin')`,
		"Spinnerei", 1.0)
	if err == nil {
		t.Fatal("expected CHECK constraint failure for target_role='admin', got nil")
	}

	// 'elternteil' und Vereinsfunktionen müssen akzeptiert werden.
	for _, target := range []string{"spieler", "elternteil", "trainer", "vorstand", "sportliche_leitung", "vorstand_beisitzer", "kassierer"} {
		if _, err := db.Exec(`INSERT INTO duty_types (name, hours_value, target_role) VALUES (?, ?, ?)`,
			"DT-"+target, 1.0, target); err != nil {
			t.Errorf("target_role=%q should be accepted, got: %v", target, err)
		}
	}
}

// TestListDutyTypes_SpielerForbidden verifies that a plain spieler without
// trainer club function cannot read duty types (GET /api/duty-types returns 403).
func TestListDutyTypes_SpielerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Get(t, srv, "/api/duty-types", token)
	res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for spieler without function, got %d", res.StatusCode)
	}
}

// TestBoard_TeamNameUsesShortForm verifies that the duty-board group header
// uses the team short form ("mA1") rather than the long form ("B-Jugend 1 männlich").
func TestBoard_TeamNameUsesShortForm(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	res, err := db.Exec(`INSERT INTO teams (name, age_class, gender) VALUES (?, ?, ?)`,
		"Team B1", "B-Jugend", "m")
	if err != nil {
		t.Fatalf("insert team: %v", err)
	}
	teamID64, _ := res.LastInsertId()
	teamID := int(teamID64)
	kaderRes, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "B-Jugend", "m", teamID, 1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderID64, _ := kaderRes.LastInsertId()

	dtID := createDutyType(t, db, "Aufbau", 2.0)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-04-10")
	createDutySlot(t, db, dtID, seasonID, teamID, gameID, "2026-04-10")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, int(kaderID64), memberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", []string{"spieler"})
	resp := testutil.Get(t, srv, "/api/duty-board", token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var groups []map[string]any
	json.NewDecoder(resp.Body).Decode(&groups)
	if len(groups) == 0 {
		t.Fatalf("expected at least one group")
	}
	names, ok := groups[0]["team_names"].([]any)
	if !ok || len(names) != 1 {
		t.Fatalf("expected single-element team_names, got %v", groups[0]["team_names"])
	}
	if got, _ := names[0].(string); got != "mB" {
		t.Errorf("expected team_names[0] 'mB' (single team in B-Jugend männlich), got %q", got)
	}
}

// ── Anleitung pro Dienst-Typ ─────────────────────────────────────────────────

func TestPutInstruction_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	dtID := createDutyType(t, db, "Kasse", 2.0)
	userID := testutil.CreateUser(t, db, "standard")
	eh := hub.NewHub()
	h := duties.NewHandler(db, testutil.TestConfig(), eh)
	srv := testServer(t, h)

	// Subscribe before the mutation so the buffered broadcast is captured.
	ch := eh.Subscribe()
	defer eh.Unsubscribe(ch)

	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	body := map[string]any{"markdown": "## Ablauf\n1. Kasse öffnen\n2. Bilanz notieren"}
	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-types/"+itoa(dtID)+"/instruction", token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var reply map[string]string
	if err := json.NewDecoder(res.Body).Decode(&reply); err != nil {
		t.Fatalf("decode reply: %v", err)
	}
	if reply["instruction_updated_at"] == "" {
		t.Errorf("expected non-empty instruction_updated_at in reply")
	}

	var md string
	var updatedAt sql.NullString
	var updatedBy sql.NullInt64
	err := db.QueryRow(`SELECT instruction_md, instruction_updated_at, instruction_updated_by FROM duty_types WHERE id=?`, dtID).
		Scan(&md, &updatedAt, &updatedBy)
	if err != nil {
		t.Fatalf("select duty_types: %v", err)
	}
	if md != "## Ablauf\n1. Kasse öffnen\n2. Bilanz notieren" {
		t.Errorf("instruction_md mismatch, got %q", md)
	}
	if !updatedAt.Valid || updatedAt.String == "" {
		t.Errorf("instruction_updated_at not set")
	}
	if !updatedBy.Valid || int(updatedBy.Int64) != userID {
		t.Errorf("instruction_updated_by=%v, want %d", updatedBy, userID)
	}

	select {
	case ev := <-ch:
		if ev != "duties" {
			t.Errorf("got broadcast %q, want %q", ev, "duties")
		}
	default:
		t.Errorf("no broadcast emitted")
	}
}

func TestPutInstruction_Unauthenticated(t *testing.T) {
	db := testutil.NewDB(t)
	dtID := createDutyType(t, db, "Kasse", 2.0)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	body := map[string]any{"markdown": "x"}
	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-types/"+itoa(dtID)+"/instruction", "", body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}

	var md string
	db.QueryRow(`SELECT instruction_md FROM duty_types WHERE id=?`, dtID).Scan(&md)
	if md != "" {
		t.Errorf("instruction_md changed on unauthenticated request: %q", md)
	}
}

func TestPutInstruction_ForbiddenForStandard(t *testing.T) {
	db := testutil.NewDB(t)
	dtID := createDutyType(t, db, "Kasse", 2.0)
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", []string{"spieler"})
	body := map[string]any{"markdown": "x"}
	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-types/"+itoa(dtID)+"/instruction", token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestPutInstruction_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	body := map[string]any{"markdown": "x"}
	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-types/9999/instruction", token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

func TestPutInstruction_MissingBody(t *testing.T) {
	db := testutil.NewDB(t)
	dtID := createDutyType(t, db, "Kasse", 2.0)
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	// Body without "markdown" field.
	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-types/"+itoa(dtID)+"/instruction", token, map[string]any{"foo": "bar"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestPutInstruction_TooLarge(t *testing.T) {
	db := testutil.NewDB(t)
	dtID := createDutyType(t, db, "Kasse", 2.0)
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	big := make([]byte, 65537)
	for i := range big {
		big[i] = 'x'
	}
	body := map[string]any{"markdown": string(big)}
	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-types/"+itoa(dtID)+"/instruction", token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

// TestDutyTypes_ListOmitsInstructionMd — die Typen-Liste liefert has_instruction
// statt des Markdown-Volltexts; zusätzlich revalidiert die Liste per ETag/304.
func TestDutyTypes_ListOmitsInstructionMd(t *testing.T) {
	db := testutil.NewDB(t)
	dtWith := createDutyType(t, db, "Kasse", 2.0)
	createDutyType(t, db, "Aufbau", 1.0) // ohne Anleitung
	testutil.SetDutyInstruction(t, db, dtWith, "## Foo")
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	res := testutil.Get(t, srv, "/api/duty-types", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Neben den beiden hier angelegten Typen seedet Migration 020 den
	// „Spielbericht"-Typ (ohne Anleitung) — die Liste prüft daher pro Name,
	// nicht positionell und nicht über eine feste Gesamtzahl.
	byName := make(map[string]map[string]any, len(items))
	for _, item := range items {
		if _, ok := item["instruction_md"]; ok {
			t.Errorf("Liste enthält instruction_md für %v — Volltext gehört nur in den Detail-Pfad", item["name"])
		}
		if _, ok := item["has_instruction"].(bool); !ok {
			t.Fatalf("has_instruction fehlt oder ist kein Bool: %v", item)
		}
		name, _ := item["name"].(string)
		byName[name] = item
	}
	for name, wantHas := range map[string]bool{"Kasse": true, "Aufbau": false} {
		item, ok := byName[name]
		if !ok {
			t.Fatalf("Typ %q fehlt in der Liste", name)
		}
		if has := item["has_instruction"].(bool); has != wantHas {
			t.Errorf("has_instruction für %v = %v, want %v", name, has, wantHas)
		}
	}
	if _, ok := byName["Kasse"]["instruction_updated_at"]; !ok {
		t.Errorf("expected instruction_updated_at in response")
	}

	// ETag/304-Revalidierung der Liste.
	etag := res.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("kein ETag gesetzt")
	}
	if cc := res.Header.Get("Cache-Control"); cc != "private, no-cache" {
		t.Errorf("Cache-Control = %q, want private, no-cache", cc)
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/duty-types", nil)
	req.Header.Set("Authorization", token)
	req.Header.Set("If-None-Match", etag)
	res304, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("revalidierter GET: %v", err)
	}
	defer res304.Body.Close()
	if res304.StatusCode != http.StatusNotModified {
		t.Errorf("revalidierter Abruf: status %d, want 304", res304.StatusCode)
	}
}

// TestDutyTypes_DetailKeepsInstructionMd — der Detail-Pfad
// GET /api/duty-types/{id}/instruction liefert den Volltext samt Metadaten;
// unbekannte IDs → 404, ohne Token → 401.
func TestDutyTypes_DetailKeepsInstructionMd(t *testing.T) {
	db := testutil.NewDB(t)
	dtID := createDutyType(t, db, "Kasse", 2.0)
	testutil.SetDutyInstruction(t, db, dtID, "## Foo")
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	// Authenticated-Tier: auch Spieler/Eltern (Board-Link) lesen den Volltext.
	token := testutil.Token(t, userID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/duty-types/"+itoa(dtID)+"/instruction", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var detail map[string]any
	if err := json.NewDecoder(res.Body).Decode(&detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if md, _ := detail["instruction_md"].(string); md != "## Foo" {
		t.Errorf("instruction_md = %q, want '## Foo'", md)
	}
	if detail["name"] != "Kasse" {
		t.Errorf("name = %v, want Kasse", detail["name"])
	}
	if v, ok := detail["instruction_updated_at"].(string); !ok || v == "" {
		t.Errorf("instruction_updated_at fehlt: %v", detail["instruction_updated_at"])
	}
	if _, ok := detail["instruction_updated_by"]; !ok {
		t.Errorf("instruction_updated_by fehlt in der Antwort")
	}

	// Fehlerfall: unbekannter Typ → 404.
	res404 := testutil.Get(t, srv, "/api/duty-types/999999/instruction", token)
	defer res404.Body.Close()
	if res404.StatusCode != http.StatusNotFound {
		t.Errorf("unbekannte ID: status %d, want 404", res404.StatusCode)
	}

	// Fehlerfall: ohne Token → 401.
	resUnauth := testutil.Get(t, srv, "/api/duty-types/"+itoa(dtID)+"/instruction", "")
	defer resUnauth.Body.Close()
	if resUnauth.StatusCode != http.StatusUnauthorized {
		t.Errorf("ohne Token: status %d, want 401", resUnauth.StatusCode)
	}
}

func TestBoard_ExposesHasInstruction(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	// Two types: one with instruction, one without.
	dtWith := createDutyType(t, db, "Kasse", 2.0)
	dtWithout := createDutyType(t, db, "Aufbau", 1.0)
	testutil.SetDutyInstruction(t, db, dtWith, "## Ablauf")
	slotWith := createDutySlot(t, db, dtWith, seasonID, teamID, 0, "2026-06-14")
	slotWithout := createDutySlot(t, db, dtWithout, seasonID, teamID, 0, "2026-06-14")

	userID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-board", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)

	seen := map[int]map[string]any{}
	for _, g := range groups {
		slots, _ := g["slots"].([]any)
		for _, s := range slots {
			m, _ := s.(map[string]any)
			idF, _ := m["id"].(float64)
			seen[int(idF)] = m
		}
	}
	sw, ok := seen[slotWith]
	if !ok {
		t.Fatalf("slot with instruction not returned")
	}
	if v, _ := sw["has_instruction"].(bool); !v {
		t.Errorf("expected has_instruction=true for %q, got %v", "Kasse", v)
	}
	if id, _ := sw["duty_type_id"].(float64); int(id) != dtWith {
		t.Errorf("expected duty_type_id=%d, got %v", dtWith, id)
	}
	swo, ok := seen[slotWithout]
	if !ok {
		t.Fatalf("slot without instruction not returned")
	}
	if v, _ := swo["has_instruction"].(bool); v {
		t.Errorf("expected has_instruction=false for %q, got %v", "Aufbau", v)
	}
	if id, _ := swo["duty_type_id"].(float64); int(id) != dtWithout {
		t.Errorf("expected duty_type_id=%d, got %v", dtWithout, id)
	}
}

// ── Spielbericht-Slot-Guard (assertSlotTakePermitted) ────────────────────────

// matchReportSlot legt einen Spielbericht-Duty-Type + Slot an (Guard matcht per Name).
func matchReportSlot(t *testing.T, db *sql.DB) (seasonID, slotID int) {
	t.Helper()
	seasonID = testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Spielbericht", 0.5)
	slotID = createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	return
}

func TestClaim_MatchReportSlot_NonPressForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	_, slotID := matchReportSlot(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-press user, got %d", res.StatusCode)
	}
	if got := slotsFilled(t, db, slotID); got != 0 {
		t.Errorf("slot must not be claimed, slots_filled=%d", got)
	}
}

func TestClaim_MatchReportSlot_PressTeamOK(t *testing.T) {
	db := testutil.NewDB(t)
	_, slotID := matchReportSlot(t, db)
	userID := testutil.CreateUser(t, db, "presseteam")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "presseteam", nil)
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for presseteam, got %d", res.StatusCode)
	}
}

func TestClaim_MatchReportSlot_AdminOK(t *testing.T) {
	db := testutil.NewDB(t)
	_, slotID := matchReportSlot(t, db)
	userID := testutil.CreateUser(t, db, "admin")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "admin", nil)
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for admin, got %d", res.StatusCode)
	}
}

// TestClaim_MatchReportSlot_ProxyParentForbidden nagelt die Rollenverschiebung fest:
// der Guard wertet die Rolle des HANDELNDEN Elternteils, nicht des Kind-Zielkontos.
// Ein Elternteil ohne presseteam darf einen Spielbericht-Slot auch für ein Kind nicht ziehen.
func TestClaim_MatchReportSlot_ProxyParentForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	_, slotID := matchReportSlot(t, db)

	parentUserID := testutil.CreateUser(t, db, "standard")
	childUserID := testutil.CreateUser(t, db, "standard")
	db.Exec(`UPDATE users SET can_login=0 WHERE id=?`, childUserID)
	childMemberID := testutil.CreateMember(t, db, childUserID)
	if _, err := db.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUserID, childMemberID); err != nil {
		t.Fatalf("insert family_links: %v", err)
	}

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, parentUserID, "standard", nil)
	body := map[string]any{"user_id": childUserID}
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 (parent role counts), got %d", res.StatusCode)
	}
	if got := slotsFilled(t, db, slotID); got != 0 {
		t.Errorf("slot must not be claimed, slots_filled=%d", got)
	}
}

func TestClaim_NonMatchReportSlot_Unaffected(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("guard must not touch non-Spielbericht slot, got %d", res.StatusCode)
	}
}
