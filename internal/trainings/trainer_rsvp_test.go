package trainings_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// setupTrainerSession baut eine Session mit einem Trainer und liefert die IDs.
func setupTrainerSession(t *testing.T) (db *sql.DB, sessionID, teamID, seasonID, trainerMemberID, trainerUserID int) {
	t.Helper()
	db = testutil.NewDB(t)
	seasonID = testutil.CreateSeason(t, db, "2025/26")
	teamID = testutil.CreateTeam(t, db, "Team A")
	sessionID = testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	trainerUserID = testutil.CreateUser(t, db, "standard")
	trainerMemberID = testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)
	return
}

type trainerAttendanceItem struct {
	MemberID   int     `json:"member_id"`
	IsExtended bool    `json:"is_extended"`
	IsTrainer  bool    `json:"is_trainer"`
	RSVPStatus *string `json:"rsvp_status"`
	Present    *bool   `json:"present"`
	Reason     *string `json:"reason"`
}

// Trainer erscheint mit is_trainer=true und Default-Status confirmed.
func TestGetAttendances_Trainer_DefaultConfirmed(t *testing.T) {
	db, sessionID, _, _, trainerMemberID, _ := setupTrainerSession(t)
	// Session bewusst mit rsvp_default_players='none' — Trainer default confirmed muss dennoch greifen.

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []trainerAttendanceItem
	json.NewDecoder(res.Body).Decode(&items)

	found := false
	for _, item := range items {
		if item.MemberID == trainerMemberID {
			found = true
			if !item.IsTrainer {
				t.Errorf("trainer row should have is_trainer=true")
			}
			if item.IsExtended {
				t.Errorf("trainer row should have is_extended=false")
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

	// Kein Default-INSERT in training_responses.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, trainerMemberID).Scan(&n)
	if n != 0 {
		t.Errorf("expected no training_responses row for trainer default, got %d", n)
	}
}

// Explizite Absage überschreibt Default-confirmed.
func TestGetAttendances_Trainer_ExplicitDeclineOverrides(t *testing.T) {
	db, sessionID, _, _, trainerMemberID, trainerUserID := setupTrainerSession(t)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
	         VALUES (?, ?, ?, 'declined', 'Krank', CURRENT_TIMESTAMP)`, sessionID, trainerMemberID, trainerUserID)

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	defer res.Body.Close()
	var items []trainerAttendanceItem
	json.NewDecoder(res.Body).Decode(&items)

	for _, item := range items {
		if item.MemberID == trainerMemberID {
			if item.RSVPStatus == nil || *item.RSVPStatus != "declined" {
				t.Errorf("expected declined, got %v", item.RSVPStatus)
			}
		}
	}
}

// Header-Zähler ignorieren Trainer-Zusagen.
func TestGetSession_ConfirmedCount_ExcludesTrainer(t *testing.T) {
	db, sessionID, teamID, seasonID, trainerMemberID, trainerUserID := setupTrainerSession(t)

	// Ein Spieler im Stammkader mit confirmed-Antwort.
	playerUserID := testutil.CreateUser(t, db, "standard")
	playerMemberID := testutil.CreateMember(t, db, playerUserID)
	var kaderID int
	db.QueryRow(`SELECT id FROM kader WHERE team_id=? AND season_id=?`, teamID, seasonID).Scan(&kaderID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, playerMemberID)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, responded_at)
	         VALUES (?, ?, ?, 'confirmed', CURRENT_TIMESTAMP)`, sessionID, playerMemberID, playerUserID)
	// Trainer sagt zusätzlich zu.
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, responded_at)
	         VALUES (?, ?, ?, 'confirmed', CURRENT_TIMESTAMP)`, sessionID, trainerMemberID, trainerUserID)

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	// GetSession-Route noch nicht im testServer registriert — direkt eine minimale Route hier bauen.
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/training-sessions/{id}", h.GetSession)
	})
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d", sessionID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		ConfirmedCount int `json:"confirmed_count"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.ConfirmedCount != 1 {
		t.Errorf("expected confirmed_count=1 (only player), got %d", body.ConfirmedCount)
	}
}

// SaveAttendances lehnt Trainer-Ziel-Member mit HTTP 400 ab.
func TestSaveAttendances_TrainerRejected(t *testing.T) {
	db, sessionID, teamID, seasonID, trainerMemberID, _ := setupTrainerSession(t)
	_ = teamID
	_ = seasonID

	// Session ist "heute" oder in Zukunft; SaveAttendances erlaubt nur past/today.
	// Trick: Session-Datum auf gestern setzen.
	db.Exec(`UPDATE training_sessions SET date='2020-01-01' WHERE id=?`, sessionID)

	adminUserID := testutil.CreateUser(t, db, "admin")
	// Admin ist auch trainerLike für hasTeamAccess. Aber die Route ist mit RequireClubFunction geschützt.
	// Wir setzen die Trainer-Funktion im Token — admin darf auch ohne, aber trainer ist expliziter.
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", []string{"trainer"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token,
		[]map[string]any{{"member_id": trainerMemberID, "present": true}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for trainer as attendance target, got %d", res.StatusCode)
	}
	// training_attendances soll leer bleiben.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_attendances WHERE training_id=? AND member_id=?`,
		sessionID, trainerMemberID).Scan(&n)
	if n != 0 {
		t.Errorf("expected no training_attendances row for trainer, got %d", n)
	}
}
