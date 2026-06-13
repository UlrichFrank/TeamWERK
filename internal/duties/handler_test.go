package duties_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── local helpers ─────────────────────────────────────────────────────────────

func itoa(n int) string { return fmt.Sprintf("%d", n) }

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
// is visible to a user who has a family_links entry (i.e. is a parent).
func TestBoard_AudienceElternVisible(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Elterndienst", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["eltern"]' WHERE id=?`, slotID)

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	// Parent has a family_links entry → can see eltern-audience slots.
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUserID, childMemberID)

	// Also add the parent's member to teamID so the slot is visible in the board.
	parentMemberID := testutil.CreateMember(t, db, parentUserID)
	addPlayerMembership(t, db, parentMemberID, teamID, seasonID)

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

// ── TC-D13 ────────────────────────────────────────────────────────────────────

// TestBoard_TrainerBypassesAudience verifies that a user with the club
// function 'trainer' in member_club_functions bypasses the audience filter
// and can see eltern-restricted slots.
func TestBoard_TrainerBypassesAudience(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Elterndienst", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	db.Exec(`UPDATE duty_slots SET audiences='["eltern"]' WHERE id=?`, slotID)

	// Trainer user: has member_club_functions.function = 'trainer'.
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	addPlayerMembership(t, db, trainerMemberID, teamID, seasonID)

	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'trainer')`,
		trainerMemberID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "spieler", nil)
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
		t.Errorf("trainer should bypass audience filter and see 1 slot, got %d", total)
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
		"event_name":  "Aufbau Heimspiel",
		"event_date":  "2026-06-14",
		"duty_type_id": dtID,
		"slots_total": 2,
		"team_id":     teamID,
		"season_id":   seasonID,
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
