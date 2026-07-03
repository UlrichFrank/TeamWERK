package games_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type trainerGameAttendanceItem struct {
	MemberID   int     `json:"member_id"`
	IsExtended bool    `json:"is_extended"`
	IsTrainer  bool    `json:"is_trainer"`
	RSVPStatus *string `json:"rsvp_status"`
	Present    *bool   `json:"present"`
}

// setupTrainerGame baut ein Spiel mit Trainer + Spieler auf und liefert die IDs.
func setupTrainerGame(t *testing.T) (db *sql.DB, gameID, teamID, seasonID, trainerMemberID, trainerUserID int) {
	t.Helper()
	db = testutil.NewDB(t)
	seasonID = testutil.CreateSeason(t, db, "2025/26")
	teamID = testutil.CreateTeam(t, db, "Team A")
	gameID = testutil.CreateGame(t, db, seasonID, teamID, "2026-06-14")

	trainerUserID = testutil.CreateUser(t, db, "standard")
	trainerMemberID = testutil.CreateMember(t, db, trainerUserID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`, trainerMemberID, "trainer")

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)
	return
}

// Trainer erscheint mit is_trainer=true, default confirmed, kein Attendance.
func TestGetGameAttendances_Trainer_DefaultConfirmed(t *testing.T) {
	db, gameID, _, _, trainerMemberID, trainerUserID := setupTrainerGame(t)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []trainerGameAttendanceItem
	json.NewDecoder(res.Body).Decode(&items)

	found := false
	for _, item := range items {
		if item.MemberID == trainerMemberID {
			found = true
			if !item.IsTrainer {
				t.Errorf("trainer row should have is_trainer=true")
			}
			if item.RSVPStatus == nil || *item.RSVPStatus != "confirmed" {
				t.Errorf("trainer default rsvp_status should be 'confirmed', got %v", item.RSVPStatus)
			}
			if item.Present != nil {
				t.Errorf("trainer present should be nil, got %v", item.Present)
			}
		}
	}
	if !found {
		t.Errorf("trainer member %d missing from attendance list", trainerMemberID)
	}

	// Kein Default-INSERT.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM game_responses WHERE game_id=? AND member_id=?`,
		gameID, trainerMemberID).Scan(&n)
	if n != 0 {
		t.Errorf("expected no game_responses row for trainer default, got %d", n)
	}
}

// Explizite Absage überschreibt Default.
func TestGetGameAttendances_Trainer_ExplicitDeclineOverrides(t *testing.T) {
	db, gameID, _, _, trainerMemberID, trainerUserID := setupTrainerGame(t)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
	         VALUES (?, ?, ?, 'declined', 'Verletzt', CURRENT_TIMESTAMP)`, gameID, trainerMemberID, trainerUserID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token)
	defer res.Body.Close()

	var items []trainerGameAttendanceItem
	json.NewDecoder(res.Body).Decode(&items)
	for _, item := range items {
		if item.MemberID == trainerMemberID {
			if item.RSVPStatus == nil || *item.RSVPStatus != "declined" {
				t.Errorf("expected declined, got %v", item.RSVPStatus)
			}
		}
	}
}

// Header-Zähler ignoriert Trainer-Zusagen bei Games.
func TestGetGame_ConfirmedCount_ExcludesTrainer(t *testing.T) {
	db, gameID, teamID, seasonID, trainerMemberID, trainerUserID := setupTrainerGame(t)

	// Spieler im Stammkader mit confirmed-Antwort.
	playerUserID := testutil.CreateUser(t, db, "standard")
	playerMemberID := testutil.CreateMember(t, db, playerUserID)
	kaderID := kaderOf(t, db, teamID, seasonID)
	addKaderMember(t, db, kaderID, playerMemberID)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, responded_at)
	         VALUES (?, ?, ?, 'confirmed', CURRENT_TIMESTAMP)`, gameID, playerMemberID, playerUserID)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, responded_at)
	         VALUES (?, ?, ?, 'confirmed', CURRENT_TIMESTAMP)`, gameID, trainerMemberID, trainerUserID)

	// Admin ruft das Detail ab.
	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", gameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Game struct {
			ConfirmedCount int `json:"confirmed_count"`
		} `json:"game"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.Game.ConfirmedCount != 1 {
		t.Errorf("expected confirmed_count=1 (only player), got %d", body.Game.ConfirmedCount)
	}
}

// SaveAttendances lehnt Trainer-Ziel-Member mit HTTP 400 ab.
func TestSaveGameAttendances_TrainerRejected(t *testing.T) {
	db, gameID, teamID, seasonID, trainerMemberID, trainerUserID := setupTrainerGame(t)
	_ = teamID
	_ = seasonID

	srv := testServer(t, db)
	// Trainer selbst darf Anwesenheit erfassen — auch er darf keine Anwesenheit für sich setzen.
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	body := []map[string]any{{"member_id": trainerMemberID, "present": true}}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for trainer as attendance target, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM game_attendances WHERE game_id=? AND member_id=?`,
		gameID, trainerMemberID).Scan(&n)
	if n != 0 {
		t.Errorf("expected no game_attendances row for trainer, got %d", n)
	}
}
