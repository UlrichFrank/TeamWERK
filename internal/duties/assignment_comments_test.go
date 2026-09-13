package duties_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── TC-DAC01 ─────────────────────────────────────────────────────────────────

// TestSetAssignmentComment_HappyPath verifies that the owner of an assignment
// can set a comment and it is persisted.
func TestSetAssignmentComment_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")

	var assignID int
	if err := db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, userID).Scan(&assignID); err != nil {
		t.Fatalf("lookup assignment id: %v", err)
	}

	eh := hub.NewHub()
	h := duties.NewHandler(db, testutil.TestConfig(), eh)
	srv := testServer(t, h)

	ch := eh.SubscribeUser(userID)
	defer eh.Unsubscribe(ch)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Put(t, srv, "/api/duty-assignments/"+itoa(assignID)+"/comment", token,
		map[string]any{"body": "Marmorkuchen"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body string
	if err := db.QueryRow(`SELECT body FROM duty_assignment_comments WHERE assignment_id=?`,
		assignID).Scan(&body); err != nil {
		t.Fatalf("select comment: %v", err)
	}
	if body != "Marmorkuchen" {
		t.Errorf("expected body %q, got %q", "Marmorkuchen", body)
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

// ── TC-DAC02 ─────────────────────────────────────────────────────────────────

// TestSetAssignmentComment_Upsert verifies that a second PUT overwrites the
// existing comment instead of creating a second row (UNIQUE(assignment_id)).
func TestSetAssignmentComment_Upsert(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")

	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, userID).Scan(&assignID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, userID, "spieler", nil)

	res1 := testutil.Put(t, srv, "/api/duty-assignments/"+itoa(assignID)+"/comment", token,
		map[string]any{"body": "Marmorkuchen"})
	res1.Body.Close()
	res2 := testutil.Put(t, srv, "/api/duty-assignments/"+itoa(assignID)+"/comment", token,
		map[string]any{"body": "Käsekuchen"})
	res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on second PUT, got %d", res2.StatusCode)
	}

	if got := countRows(t, db, "duty_assignment_comments", "assignment_id=?", assignID); got != 1 {
		t.Errorf("expected exactly 1 comment row, got %d", got)
	}
	var body string
	db.QueryRow(`SELECT body FROM duty_assignment_comments WHERE assignment_id=?`, assignID).Scan(&body)
	if body != "Käsekuchen" {
		t.Errorf("expected overwritten body %q, got %q", "Käsekuchen", body)
	}
}

// ── TC-DAC03 ─────────────────────────────────────────────────────────────────

// TestSetAssignmentComment_ForProxyChild verifies that a parent account may
// set a comment on behalf of a linked proxy child's own assignment.
func TestSetAssignmentComment_ForProxyChild(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childUserID := testutil.CreateUser(t, db, "standard")
	db.Exec(`UPDATE users SET can_login=0 WHERE id=?`, childUserID)
	childMemberID := testutil.CreateMember(t, db, childUserID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUserID, childMemberID)

	insertDutyAssignment(t, db, slotID, childUserID, "assigned")
	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, childUserID).Scan(&assignID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, parentUserID, "elternteil", nil)
	res := testutil.Put(t, srv, "/api/duty-assignments/"+itoa(assignID)+"/comment", token,
		map[string]any{"body": "Zitronenkuchen"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_assignment_comments", "assignment_id=?", assignID); got != 1 {
		t.Errorf("expected comment persisted for child's assignment, got %d rows", got)
	}
}

// ── TC-DAC04 ─────────────────────────────────────────────────────────────────

// TestSetAssignmentComment_ForeignForbidden verifies that a user unrelated to
// the assignment (not owner, not parent) gets 403 and the comment is
// unchanged — no admin/vorstand bypass.
func TestSetAssignmentComment_ForeignForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	ownerUserID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, ownerUserID, "assigned")
	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, ownerUserID).Scan(&assignID)

	// Even an admin has no bypass.
	adminUserID := testutil.CreateUser(t, db, "admin")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Put(t, srv, "/api/duty-assignments/"+itoa(assignID)+"/comment", token,
		map[string]any{"body": "Fremder Kommentar"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_assignment_comments", "assignment_id=?", assignID); got != 0 {
		t.Errorf("expected no comment to be created, got %d rows", got)
	}
}

// ── TC-DAC05 ─────────────────────────────────────────────────────────────────

// TestSetAssignmentComment_NotFound verifies 404 for an unknown assignment id.
func TestSetAssignmentComment_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Put(t, srv, "/api/duty-assignments/999999/comment", token,
		map[string]any{"body": "x"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

// ── TC-DAC06 ─────────────────────────────────────────────────────────────────

// TestSetAssignmentComment_Unauthenticated verifies 401 without a token.
func TestSetAssignmentComment_Unauthenticated(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, userID).Scan(&assignID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	res := testutil.Put(t, srv, "/api/duty-assignments/"+itoa(assignID)+"/comment", "",
		map[string]any{"body": "x"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}

// ── TC-DAC07 ─────────────────────────────────────────────────────────────────

// TestSetAssignmentComment_EmptyBody verifies that an empty/whitespace-only
// body is rejected with 400 and does not touch an existing comment.
func TestSetAssignmentComment_EmptyBody(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, userID).Scan(&assignID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, userID, "spieler", nil)

	res := testutil.Put(t, srv, "/api/duty-assignments/"+itoa(assignID)+"/comment", token,
		map[string]any{"body": "   "})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_assignment_comments", "assignment_id=?", assignID); got != 0 {
		t.Errorf("expected no comment row, got %d", got)
	}
}

// ── TC-DAC08 ─────────────────────────────────────────────────────────────────

// TestSetAssignmentComment_TooLong verifies the 280-byte cap is enforced.
func TestSetAssignmentComment_TooLong(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, userID).Scan(&assignID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, userID, "spieler", nil)

	res := testutil.Put(t, srv, "/api/duty-assignments/"+itoa(assignID)+"/comment", token,
		map[string]any{"body": strings.Repeat("a", 281)})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", res.StatusCode)
	}
}

// ── TC-DAC09 ─────────────────────────────────────────────────────────────────

// TestDeleteAssignmentComment_HappyPath verifies that the owner can delete
// their own comment explicitly (independent of unclaiming).
func TestDeleteAssignmentComment_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, userID).Scan(&assignID)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Kuchen')`, assignID)

	eh := hub.NewHub()
	h := duties.NewHandler(db, testutil.TestConfig(), eh)
	srv := testServer(t, h)
	ch := eh.SubscribeUser(userID)
	defer eh.Unsubscribe(ch)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-assignments/"+itoa(assignID)+"/comment", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_assignment_comments", "assignment_id=?", assignID); got != 0 {
		t.Errorf("expected comment deleted, got %d rows", got)
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

// ── TC-DAC10 ─────────────────────────────────────────────────────────────────

// TestDeleteAssignmentComment_ForeignForbidden verifies 403 for a caller who
// is neither the owner nor their parent.
func TestDeleteAssignmentComment_ForeignForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	ownerUserID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, ownerUserID, "assigned")
	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, ownerUserID).Scan(&assignID)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Kuchen')`, assignID)

	otherUserID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, otherUserID, "spieler", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-assignments/"+itoa(assignID)+"/comment", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_assignment_comments", "assignment_id=?", assignID); got != 1 {
		t.Errorf("expected comment untouched, got %d rows", got)
	}
}

// ── TC-DAC11 ─────────────────────────────────────────────────────────────────

// TestDeleteAssignmentComment_NotFound verifies 404 for an unknown assignment id.
func TestDeleteAssignmentComment_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-assignments/999999/comment", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

// ── TC-DAC12 ─────────────────────────────────────────────────────────────────

// TestListSlotComments_HappyPath verifies that any authenticated user with
// board access can read all comments of a slot, including comments from
// people other than themselves and without being assigned themselves.
func TestListSlotComments_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userA, "assigned")
	insertDutyAssignment(t, db, slotID, userB, "assigned")
	var assignA, assignB int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`, slotID, userA).Scan(&assignA)
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`, slotID, userB).Scan(&assignB)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Marmorkuchen')`, assignA)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Käsekuchen')`, assignB)

	// A third, uninvolved user reads — universal read access.
	readerUserID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, readerUserID, "spieler", nil)
	res := testutil.Get(t, srv, "/api/duty-slots/"+itoa(slotID)+"/comments", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var comments []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&comments); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
}

// ── TC-DAC13 ─────────────────────────────────────────────────────────────────

// TestListSlotComments_NotFound verifies 404 for an unknown slot id.
func TestListSlotComments_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Get(t, srv, "/api/duty-slots/999999/comments", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

// ── TC-DAC14 ─────────────────────────────────────────────────────────────────

// TestAssignmentComment_CascadeOnUnclaim verifies that unclaiming a slot
// (DELETE /api/duty-board/{slotId}/claim) automatically removes the caller's
// own comment via ON DELETE CASCADE.
func TestAssignmentComment_CascadeOnUnclaim(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	db.Exec(`UPDATE duty_slots SET slots_filled=1 WHERE id=?`, slotID)
	var assignID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, userID).Scan(&assignID)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Kuchen')`, assignID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, userID, "spieler", nil)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_assignment_comments", "assignment_id=?", assignID); got != 0 {
		t.Errorf("expected comment cascaded away with the assignment, got %d rows", got)
	}
}

// ── TC-DAC15 ─────────────────────────────────────────────────────────────────

// TestAssignmentComment_CascadeOnSlotDelete verifies that deleting a slot
// (as happens from the calendar modal's edit dialog) removes the comments of
// all its assignments via ON DELETE CASCADE.
func TestAssignmentComment_CascadeOnSlotDelete(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userA, "assigned")
	insertDutyAssignment(t, db, slotID, userB, "assigned")
	var assignA, assignB int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`, slotID, userA).Scan(&assignA)
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`, slotID, userB).Scan(&assignB)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Marmorkuchen')`, assignA)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Käsekuchen')`, assignB)

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-slots/"+itoa(slotID), token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		t.Fatalf("expected slot delete to succeed, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_assignment_comments", "assignment_id IN (?,?)", assignA, assignB); got != 0 {
		t.Errorf("expected comments cascaded away with slot deletion, got %d rows", got)
	}
}

// ── TC-DAC16 ─────────────────────────────────────────────────────────────────

// TestDutyBoard_CommentCount verifies comment_count in the board response:
// zero without comments, correct count when some assignments are commented.
func TestDutyBoard_CommentCount(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kuchen", 1.0)
	slotWithout := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")
	slotWith := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-15")

	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")
	userC := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotWith, userA, "assigned")
	insertDutyAssignment(t, db, slotWith, userB, "assigned")
	insertDutyAssignment(t, db, slotWith, userC, "assigned")
	var assignA, assignB int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`, slotWith, userA).Scan(&assignA)
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`, slotWith, userB).Scan(&assignB)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Marmorkuchen')`, assignA)
	db.Exec(`INSERT INTO duty_assignment_comments (assignment_id, body) VALUES (?, 'Käsekuchen')`, assignB)

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, "/api/duty-board", token)
	defer res.Body.Close()
	var groups []map[string]any
	json.NewDecoder(res.Body).Decode(&groups)

	counts := map[int]float64{}
	for _, g := range groups {
		slots, _ := g["slots"].([]any)
		for _, s := range slots {
			slot, _ := s.(map[string]any)
			id := int(slot["id"].(float64))
			counts[id] = slot["comment_count"].(float64)
		}
	}
	if counts[slotWithout] != 0 {
		t.Errorf("slotWithout: expected comment_count=0, got %v", counts[slotWithout])
	}
	if counts[slotWith] != 2 {
		t.Errorf("slotWith: expected comment_count=2, got %v", counts[slotWith])
	}
}
