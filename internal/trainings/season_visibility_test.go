package trainings_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// TestSessions_ParentAltSeasonKader_DoesNotLeak reproduziert den Bug, bei dem
// ein Elternteil in der aktiven Saison Trainings eines Teams sah, in dem sein
// Kind nur in einer *vergangenen* Saison stand.
//
// Setup: Kind war in Saison 1 im Kader von Team A und ist in Saison 2 im Kader
// von Team B. Ein Training in Team A/Saison 2 darf für den Elternteil NICHT
// sichtbar sein.
func TestSessions_ParentAltSeasonKader_DoesNotLeak(t *testing.T) {
	db := testutil.NewDB(t)
	oldSeason := testutil.CreateSeason(t, db, "2025/26") // wird durch neue Saison inaktiv
	newSeason := testutil.CreateSeason(t, db, "2026/27") // is_active=1

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")

	// Alt-Saison-Zugehörigkeit des Kindes zu Team A (regulär):
	kaderA_old := testutil.CreateKader(t, db, teamA, oldSeason)
	// Aktuelle Saison-Zugehörigkeit zu Team B:
	kaderB_new := testutil.CreateKader(t, db, teamB, newSeason)
	// Team A hat in der aktiven Saison einen eigenen Kader ohne unser Kind:
	testutil.CreateKader(t, db, teamA, newSeason)

	sessionTeamA_new := testutil.CreateTrainingSession(t, db, teamA, newSeason, "2026-10-15")
	sessionTeamB_new := testutil.CreateTrainingSession(t, db, teamB, newSeason, "2026-10-16")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	// Kind Alt-Saison: Team A regulär
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA_old, childMemberID)
	// Kind aktuelle Saison: Team B regulär
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderB_new, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/training-sessions?from=2025-01-01&to=2027-12-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp struct {
		Items []sessionWithChildren `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()

	seen := map[int]bool{}
	for _, s := range resp.Items {
		seen[s.ID] = true
	}
	if seen[sessionTeamA_new] {
		t.Errorf("Bug: Training %d (Team A, aktuelle Saison) darf Elternteil nicht sehen — Kind ist nur in Alt-Saison in Team A", sessionTeamA_new)
	}
	if !seen[sessionTeamB_new] {
		t.Errorf("Training %d (Team B, aktuelle Saison) muss Elternteil sehen — Kind ist dort regulär", sessionTeamB_new)
	}
}

// TestSessions_OwnAltSeasonKader_DoesNotLeak: analog für die eigene
// Membership (User statt Elternteil).
func TestSessions_OwnAltSeasonKader_DoesNotLeak(t *testing.T) {
	db := testutil.NewDB(t)
	oldSeason := testutil.CreateSeason(t, db, "2025/26")
	newSeason := testutil.CreateSeason(t, db, "2026/27")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA_old := testutil.CreateKader(t, db, teamA, oldSeason)
	kaderB_new := testutil.CreateKader(t, db, teamB, newSeason)
	testutil.CreateKader(t, db, teamA, newSeason)

	sessionTeamA_new := testutil.CreateTrainingSession(t, db, teamA, newSeason, "2026-10-15")
	sessionTeamB_new := testutil.CreateTrainingSession(t, db, teamB, newSeason, "2026-10-16")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA_old, memberID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderB_new, memberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, userID, "standard", nil, false)

	res := testutil.Get(t, srv, "/api/training-sessions?from=2025-01-01&to=2027-12-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp struct {
		Items []sessionWithChildren `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()

	seen := map[int]bool{}
	for _, s := range resp.Items {
		seen[s.ID] = true
	}
	if seen[sessionTeamA_new] {
		t.Errorf("Bug: Training %d (Team A, aktuelle Saison) darf User nicht sehen — User ist nur in Alt-Saison in Team A", sessionTeamA_new)
	}
	if !seen[sessionTeamB_new] {
		t.Errorf("Training %d (Team B, aktuelle Saison) muss User sehen", sessionTeamB_new)
	}
}
