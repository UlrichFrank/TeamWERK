package games_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestListMyGames_ParentAltSeasonKader_DoesNotLeak: analog zum Trainings-Bug.
// Ein Elternteil, dessen Kind nur in einer vergangenen Saison im gD-Kader stand,
// darf gD-Spiele der aktuellen Saison nicht sehen.
func TestListMyGames_ParentAltSeasonKader_DoesNotLeak(t *testing.T) {
	db := testutil.NewDB(t)
	oldSeason := testutil.CreateSeason(t, db, "2025/26")
	newSeason := testutil.CreateSeason(t, db, "2026/27")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")

	kaderA_old := testutil.CreateKader(t, db, teamA, oldSeason)
	kaderB_new := testutil.CreateKader(t, db, teamB, newSeason)
	testutil.CreateKader(t, db, teamA, newSeason)

	gameA_new := testutil.CreateGame(t, db, newSeason, teamA, "2026-10-15")
	gameB_new := testutil.CreateGame(t, db, newSeason, teamB, "2026-10-16")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA_old, childMemberID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderB_new, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/games/my?from=2025-01-01&to=2027-12-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	res.Body.Close()

	seen := map[int]bool{}
	for _, g := range games {
		if id, ok := g["id"].(float64); ok {
			seen[int(id)] = true
		}
	}
	if seen[gameA_new] {
		t.Errorf("Bug: Spiel %d (Team A, aktuelle Saison) darf Elternteil nicht sehen — Kind ist nur in Alt-Saison in Team A", gameA_new)
	}
	if !seen[gameB_new] {
		t.Errorf("Spiel %d (Team B, aktuelle Saison) muss Elternteil sehen", gameB_new)
	}
}

// TestListMyGames_OwnAltSeasonKader_DoesNotLeak: analog für eigene Membership.
func TestListMyGames_OwnAltSeasonKader_DoesNotLeak(t *testing.T) {
	db := testutil.NewDB(t)
	oldSeason := testutil.CreateSeason(t, db, "2025/26")
	newSeason := testutil.CreateSeason(t, db, "2026/27")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA_old := testutil.CreateKader(t, db, teamA, oldSeason)
	kaderB_new := testutil.CreateKader(t, db, teamB, newSeason)
	testutil.CreateKader(t, db, teamA, newSeason)

	gameA_new := testutil.CreateGame(t, db, newSeason, teamA, "2026-10-15")
	gameB_new := testutil.CreateGame(t, db, newSeason, teamB, "2026-10-16")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA_old, memberID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderB_new, memberID)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, userID, "standard", nil, false)

	res := testutil.Get(t, srv, "/api/games/my?from=2025-01-01&to=2027-12-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	res.Body.Close()

	seen := map[int]bool{}
	for _, g := range games {
		if id, ok := g["id"].(float64); ok {
			seen[int(id)] = true
		}
	}
	if seen[gameA_new] {
		t.Errorf("Bug: Spiel %d (Team A, aktuelle Saison) darf User nicht sehen — nur Alt-Saison-Zugehörigkeit", gameA_new)
	}
	if !seen[gameB_new] {
		t.Errorf("Spiel %d (Team B, aktuelle Saison) muss User sehen", gameB_new)
	}
}
