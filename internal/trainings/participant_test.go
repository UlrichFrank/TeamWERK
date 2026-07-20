package trainings_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

func addKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("kader_members: %v", err)
	}
}

func addExtendedKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("kader_extended_members: %v", err)
	}
}

// am_i_participant: Spieler im Stammkader ohne Response bei Default=none
// SOLL true zurückliefern, my_rsvp bleibt null.
func TestListSessions_AmIParticipant_RegularKader_DefaultNone(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-15")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	var found map[string]any
	for _, it := range body.Items {
		if int(it["id"].(float64)) == sessionID {
			found = it
			break
		}
	}
	if found == nil {
		t.Fatalf("session %d not returned", sessionID)
	}
	if p, _ := found["am_i_participant"].(bool); !p {
		t.Errorf("am_i_participant: got %v, want true", found["am_i_participant"])
	}
	if found["my_rsvp"] != nil {
		t.Errorf("my_rsvp: got %v, want nil (Default=none, keine Response)", found["my_rsvp"])
	}
}

// Erweiterter Kader → am_i_participant=true.
func TestListSessions_AmIParticipant_ExtendedKader(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-15")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addExtendedKaderMember(t, db, kaderID, mID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31", token)
	defer res.Body.Close()
	var body struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	for _, it := range body.Items {
		if int(it["id"].(float64)) == sessionID {
			if p, _ := it["am_i_participant"].(bool); !p {
				t.Errorf("am_i_participant: got %v, want true (extended kader)", it["am_i_participant"])
			}
			return
		}
	}
	t.Fatalf("session %d not returned", sessionID)
}

// Trainer sieht am_i_participant=true.
func TestListSessions_AmIParticipant_Trainer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-15")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'trainer')`, mID); err != nil {
		t.Fatalf("insert function: %v", err)
	}
	testutil.AddKaderTrainer(t, db, kaderID, mID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31", token)
	defer res.Body.Close()
	var body struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	for _, it := range body.Items {
		if int(it["id"].(float64)) == sessionID {
			if p, _ := it["am_i_participant"].(bool); !p {
				t.Errorf("am_i_participant für Trainer: got %v, want true", it["am_i_participant"])
			}
			return
		}
	}
	t.Fatalf("session %d not returned", sessionID)
}

// GetSession liefert am_i_participant analog zum List-Endpoint.
func TestGetSession_AmIParticipant_RegularKader(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-15")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d", sessionID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var g map[string]any
	json.NewDecoder(res.Body).Decode(&g)
	if p, _ := g["am_i_participant"].(bool); !p {
		t.Errorf("am_i_participant: got %v, want true", g["am_i_participant"])
	}
}
