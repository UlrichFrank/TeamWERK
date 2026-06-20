package auth_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func TestUsersPicker_AdminSeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	h := auth.NewHandler(db, testutil.TestConfig(), testutil.TestJWTSecret, nil, "", nil)

	adminID := testutil.CreateUser(t, db, "admin")
	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/users/picker", h.UsersPicker)

	})

	tok := testutil.Token(t, adminID, "admin", []string{})
	res := testutil.Get(t, srv, "/api/users/picker", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var users []struct {
		ID int `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&users)
	res.Body.Close()

	ids := map[int]bool{}
	for _, u := range users {
		ids[u.ID] = true
	}
	if !ids[userA] || !ids[userB] {
		t.Errorf("admin should see all users; missing userA=%v or userB=%v in %v", userA, userB, ids)
	}
}

func TestUsersPicker_SpielerSeesTeamOnly(t *testing.T) {
	db := testutil.NewDB(t)
	h := auth.NewHandler(db, testutil.TestConfig(), testutil.TestJWTSecret, nil, "", nil)

	// Set up two teams with an active season
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active = 1 WHERE id = ?`, seasonID)

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")

	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)

	// spieler in team A
	spielerUserID := testutil.CreateUser(t, db, "standard")
	spielerMemberID := testutil.CreateMember(t, db, spielerUserID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, spielerMemberID)

	// teammate in team A
	teammateUserID := testutil.CreateUser(t, db, "standard")
	teammateMemberID := testutil.CreateMember(t, db, teammateUserID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, teammateMemberID)

	// user in team B — should NOT appear
	otherUserID := testutil.CreateUser(t, db, "standard")
	otherMemberID := testutil.CreateMember(t, db, otherUserID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderB, otherMemberID)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/users/picker", h.UsersPicker)

	})

	tok := testutil.Token(t, spielerUserID, "standard", []string{})
	res := testutil.Get(t, srv, "/api/users/picker", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var users []struct {
		ID int `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&users)
	res.Body.Close()

	ids := map[int]bool{}
	for _, u := range users {
		ids[u.ID] = true
	}
	if !ids[teammateUserID] {
		t.Errorf("spieler should see teammate from same team (id=%d), got %v", teammateUserID, ids)
	}
	if ids[otherUserID] {
		t.Errorf("spieler should NOT see user from other team (id=%d)", otherUserID)
	}
}
