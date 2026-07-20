package games_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// makeTrainer creates a user with role=standard, a linked member with the
// trainer club function, and registers him as kader trainer for (teamID,
// seasonID). Returns the user_id.
func makeTrainer(t *testing.T, db *sql.DB, teamID, seasonID int) int {
	t.Helper()
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		memberID, "trainer"); err != nil {
		t.Fatalf("club function: %v", err)
	}
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, memberID)
	return userID
}

// addKaderMember adds memberID to the given kader as a regular kader member.
func addKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("kader_members: %v", err)
	}
}

// addExtendedKaderMember adds memberID to the given kader as extended.
func addExtendedKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("kader_extended_members: %v", err)
	}
}

// kaderOf returns the (single) kader id for the given team+season.
func kaderOf(t *testing.T, db *sql.DB, teamID, seasonID int) int {
	t.Helper()
	var id int
	if err := db.QueryRow(
		`SELECT id FROM kader WHERE team_id=? AND season_id=?`, teamID, seasonID).
		Scan(&id); err != nil {
		t.Fatalf("kaderOf: %v", err)
	}
	return id
}

// TestSaveGameAttendances_HappyPath verifies that a trainer of the game's
// team can record attendance for a past game, the rows are persisted and
// the response is 204.
func TestSaveGameAttendances_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")
	trainerUserID := makeTrainer(t, db, teamID, seasonID)
	playerMemberID := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderOf(t, db, teamID, seasonID), playerMemberID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	body := []map[string]any{{"member_id": playerMemberID, "present": true}}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token, body)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var present int
	if err := db.QueryRow(
		`SELECT present FROM game_attendances WHERE game_id=? AND member_id=?`,
		gameID, playerMemberID).Scan(&present); err != nil {
		t.Fatalf("row not persisted: %v", err)
	}
	if present != 1 {
		t.Errorf("expected present=1, got %d", present)
	}
}

// TestSaveGameAttendances_FutureGame_422 verifies that a game in the future
// cannot have attendance recorded.
func TestSaveGameAttendances_FutureGame_422(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2027-06-14")
	trainerUserID := makeTrainer(t, db, teamID, seasonID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token, []map[string]any{})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
}

// TestSaveGameAttendances_TrainerOfOtherTeam_403 verifies that a trainer who
// is not assigned to any of the game's teams gets 403.
func TestSaveGameAttendances_TrainerOfOtherTeam_403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	gameTeamID := testutil.CreateTeam(t, db, "Game Team")
	otherTeamID := testutil.CreateTeam(t, db, "Other Team")
	gameID := testutil.CreateGame(t, db, seasonID, gameTeamID, "2026-06-14")
	otherTrainerUserID := makeTrainer(t, db, otherTeamID, seasonID)

	srv := testServer(t, db)
	token := testutil.Token(t, otherTrainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token, []map[string]any{})
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// TestSaveGameAttendances_SportlicheLeitung_Any_OK verifies that sportliche_
// leitung can record attendance for any team's game without being a trainer
// of that team.
func TestSaveGameAttendances_SportlicheLeitung_Any_OK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")
	slUserID := testutil.CreateUser(t, db, "standard")

	srv := testServer(t, db)
	token := testutil.Token(t, slUserID, "standard", []string{"sportliche_leitung"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token, []map[string]any{})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// TestSaveGameAttendances_Unauthenticated_401 verifies that no token yields 401.
func TestSaveGameAttendances_Unauthenticated_401(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	srv := testServer(t, db)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), "", []map[string]any{})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

// TestSaveGameAttendances_NotFound_404 verifies that a non-existent game id
// yields 404 (handler checks game existence first).
func TestSaveGameAttendances_NotFound_404(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID := makeTrainer(t, db, teamID, seasonID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, "/api/games/99999/attendances", token, []map[string]any{})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

// TestGetGameAttendances_HappyPath verifies that the list includes kader
// members with their RSVP status, present (nullable) and is_extended flag.
func TestGetGameAttendances_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	regular := testutil.CreateMember(t, db, 0)
	extended := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, regular)
	addExtendedKaderMember(t, db, kaderID, extended)

	// record attendance for the regular member only
	if _, err := db.Exec(
		`INSERT INTO game_attendances (game_id, member_id, present) VALUES (?, ?, 1)`,
		gameID, regular); err != nil {
		t.Fatalf("seed attendance: %v", err)
	}

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		trainerMemberID, "trainer"); err != nil {
		t.Fatalf("club function: %v", err)
	}
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Trainer erscheint jetzt zusätzlich in der Liste (is_trainer=true), also 3 Items.
	if len(items) != 3 {
		t.Fatalf("expected 3 items (trainer + regular + extended), got %d", len(items))
	}
	byID := map[int]map[string]any{}
	for _, it := range items {
		id := int(it["member_id"].(float64))
		byID[id] = it
	}
	if byID[regular]["is_extended"].(bool) != false {
		t.Errorf("regular member should have is_extended=false")
	}
	if byID[regular]["present"] == nil {
		t.Errorf("regular member should have present set")
	} else if byID[regular]["present"].(bool) != true {
		t.Errorf("expected present=true for regular, got %v", byID[regular]["present"])
	}
	if byID[extended]["is_extended"].(bool) != true {
		t.Errorf("extended member should have is_extended=true")
	}
	if byID[extended]["present"] != nil {
		t.Errorf("extended member without attendance row should have present=nil, got %v", byID[extended]["present"])
	}
	if byID[trainerMemberID]["is_trainer"].(bool) != true {
		t.Errorf("trainer member should have is_trainer=true")
	}
	if byID[trainerMemberID]["present"] != nil {
		t.Errorf("trainer present should be nil, got %v", byID[trainerMemberID]["present"])
	}
}

// TestGetGameAttendances_DedupOverlap verifies that a member in both regular
// and extended kader is listed once with is_extended=false.
func TestGetGameAttendances_DedupOverlap(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	dualMember := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, dualMember)
	addExtendedKaderMember(t, db, kaderID, dualMember)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0]["is_extended"].(bool) != false {
		t.Errorf("dual member should be reported as is_extended=false")
	}
}

// TestGetGameAttendances_Spieler_403 verifies that a regular spieler without
// trainer / sL function cannot read the attendance list.
func TestGetGameAttendances_Spieler_403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	spielerUserID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, spielerUserID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token)
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// TestSaveGameAttendances_TrainerInBatch_Skipped verifies the regression fix:
// the bulk save batch sent by the frontend contains a trainer roster row
// alongside a player. The trainer entry must be skipped (no game_attendances
// row) WITHOUT failing the whole request, so the player's attendance persists
// and the response is 204.
func TestSaveGameAttendances_TrainerInBatch_Skipped(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// Trainer of the team (kader_trainer, not a player).
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		trainerMemberID, "trainer"); err != nil {
		t.Fatalf("club function: %v", err)
	}
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	// Player (kader member).
	playerMemberID := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, playerMemberID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	// Batch as the frontend sends it: player + trainer together.
	body := []map[string]any{
		{"member_id": playerMemberID, "present": true},
		{"member_id": trainerMemberID, "present": false},
	}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token, body)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	// Player attendance persisted.
	var present int
	if err := db.QueryRow(
		`SELECT present FROM game_attendances WHERE game_id=? AND member_id=?`,
		gameID, playerMemberID).Scan(&present); err != nil {
		t.Fatalf("player row not persisted: %v", err)
	}
	if present != 1 {
		t.Errorf("expected player present=1, got %d", present)
	}
	// Trainer entry skipped — no row written.
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM game_attendances WHERE game_id=? AND member_id=?`,
		gameID, trainerMemberID).Scan(&n); err != nil {
		t.Fatalf("count trainer rows: %v", err)
	}
	if n != 0 {
		t.Errorf("expected no attendance row for trainer, got %d", n)
	}
}

// TestSaveGameAttendances_SetsAttendanceTrackedFlag verifies that a successful
// player upsert flips games.attendance_tracked from 0 → 1.
func TestSaveGameAttendances_SetsAttendanceTrackedFlag(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")
	trainerUserID := makeTrainer(t, db, teamID, seasonID)
	playerMemberID := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderOf(t, db, teamID, seasonID), playerMemberID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	var tracked int
	db.QueryRow(`SELECT attendance_tracked FROM games WHERE id=?`, gameID).Scan(&tracked)
	if tracked != 0 {
		t.Fatalf("pre-save attendance_tracked expected 0, got %d", tracked)
	}
	body := []map[string]any{{"member_id": playerMemberID, "present": true}}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	db.QueryRow(`SELECT attendance_tracked FROM games WHERE id=?`, gameID).Scan(&tracked)
	if tracked != 1 {
		t.Errorf("post-save attendance_tracked expected 1, got %d", tracked)
	}
}

// TestResetGameAttendanceTracking_ClearsFlagButKeepsRows verifies that DELETE
// /api/games/{id}/attendance-tracking sets the flag to 0 while leaving
// game_attendances rows intact, and that the operation is idempotent.
func TestResetGameAttendanceTracking_ClearsFlagButKeepsRows(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")
	trainerUserID := makeTrainer(t, db, teamID, seasonID)
	playerMemberID := testutil.CreateMember(t, db, 0)
	testutil.RecordGameAttendance(t, db, gameID, playerMemberID, true)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Delete(t, srv, fmt.Sprintf("/api/games/%d/attendance-tracking", gameID), token)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var tracked int
	db.QueryRow(`SELECT attendance_tracked FROM games WHERE id=?`, gameID).Scan(&tracked)
	if tracked != 0 {
		t.Errorf("attendance_tracked expected 0 after reset, got %d", tracked)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM game_attendances WHERE game_id=? AND member_id=?`, gameID, playerMemberID).Scan(&n)
	if n != 1 {
		t.Errorf("row must remain after reset (Undo), got %d", n)
	}
	// Idempotent
	res2 := testutil.Delete(t, srv, fmt.Sprintf("/api/games/%d/attendance-tracking", gameID), token)
	res2.Body.Close()
	if res2.StatusCode != http.StatusNoContent {
		t.Errorf("second reset expected 204, got %d", res2.StatusCode)
	}
}

// TestResetGameAttendanceTracking_ForeignTeam_Forbidden verifies that a
// trainer without access to any team of the game gets 403.
func TestResetGameAttendanceTracking_ForeignTeam_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	gameTeamID := testutil.CreateTeam(t, db, "Game Team")
	otherTeamID := testutil.CreateTeam(t, db, "Other Team")
	gameID := testutil.CreateGame(t, db, seasonID, gameTeamID, "2026-06-14")
	otherTrainerUserID := makeTrainer(t, db, otherTeamID, seasonID)

	srv := testServer(t, db)
	token := testutil.Token(t, otherTrainerUserID, "standard", []string{"trainer"})
	res := testutil.Delete(t, srv, fmt.Sprintf("/api/games/%d/attendance-tracking", gameID), token)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// TestResetGameAttendanceTracking_UnknownGame_404 verifies 404 for a
// non-existing game id.
func TestResetGameAttendanceTracking_UnknownGame_404(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID := makeTrainer(t, db, teamID, seasonID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Delete(t, srv, "/api/games/99999/attendance-tracking", token)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

// TestResetGameAttendanceTracking_Unauthenticated_401 verifies that no token
// yields 401.
func TestResetGameAttendanceTracking_Unauthenticated_401(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	srv := testServer(t, db)
	res := testutil.Delete(t, srv, fmt.Sprintf("/api/games/%d/attendance-tracking", gameID), "")
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}
