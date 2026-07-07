package games_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

func recvWithin(ch chan string, d time.Duration) (string, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return "", false
	}
}

// TestGamesMutation_ScopedToTeamAndStaff verifies that a game mutation
// (POST /api/games/{id}/attendances → "attendance-changed") reaches the streams
// of the game's team (trainer + player), a club-wide vorstand, and a parent of a
// team player — but NOT a player of a different team.
func TestGamesMutation_ScopedToTeamAndStaff(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderB := testutil.CreateKader(t, db, teamB, season)

	// Game bound to team A, in the past so attendance can be recorded.
	gameID := testutil.CreateGame(t, db, season, teamA, "2026-06-14")

	// Trainer of team A performs the mutation (also creates kader A).
	trainerU := makeTrainer(t, db, teamA, season)
	kaderA := kaderOf(t, db, teamA, season)

	// Player on team A + a parent of that player.
	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	addKaderMember(t, db, kaderA, playerM)
	parentU := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentU, playerM)

	// Club-wide vorstand — must receive team events.
	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	// Player on team B — must NOT receive team A's event.
	foreignU := testutil.CreateUser(t, db, "standard")
	foreignM := testutil.CreateMember(t, db, foreignU)
	addKaderMember(t, db, kaderB, foreignM)

	srv, sharedHub := prodserver.NewWithHub(t, db)

	trainerCh := sharedHub.SubscribeUser(trainerU)
	playerCh := sharedHub.SubscribeUser(playerU)
	parentCh := sharedHub.SubscribeUser(parentU)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	foreignCh := sharedHub.SubscribeUser(foreignU)

	token := testutil.Token(t, trainerU, "standard", []string{"trainer"})
	body := []map[string]any{{"member_id": playerM, "present": true}}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/attendances", gameID), token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("SaveAttendances: expected 204, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"trainer":  trainerCh,
		"player":   playerCh,
		"parent":   parentCh,
		"vorstand": vorstandCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "attendance-changed" {
			t.Errorf("%s stream must receive 'attendance-changed', got %q ok=%v", name, ev, ok)
		}
	}
	if ev, ok := recvWithin(foreignCh, 300*time.Millisecond); ok {
		t.Errorf("team-foreign player must NOT receive game event, got %q", ev)
	}
}

// TestUpdateGame_TeamReassign_NotifiesOldTeam verifies that re-assigning a game
// from team A to team B (PUT /api/games/{id} with team_ids=[B]) still delivers
// the "games" live-update to a player of the REMOVED team A. Regression guard:
// broadcastGame resolves the audience from the (already rewritten) game_teams, so
// without capturing the old teams before the rewrite, team A would silently keep
// a game it no longer owns.
func TestUpdateGame_TeamReassign_NotifiesOldTeam(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	kaderB := testutil.CreateKader(t, db, teamB, season)

	// Game currently bound to team A.
	gameID := testutil.CreateGame(t, db, season, teamA, "2026-06-14")

	// Player on team A (the team being removed) — must still be notified.
	oldPlayerU := testutil.CreateUser(t, db, "standard")
	oldPlayerM := testutil.CreateMember(t, db, oldPlayerU)
	addKaderMember(t, db, kaderA, oldPlayerM)

	// Player on team B (the new team) — must be notified about the new game.
	newPlayerU := testutil.CreateUser(t, db, "standard")
	newPlayerM := testutil.CreateMember(t, db, newPlayerU)
	addKaderMember(t, db, kaderB, newPlayerM)

	// Admin performs the mutation (bypasses RequireClubFunction on the route).
	adminU := testutil.CreateUser(t, db, "admin")

	srv, sharedHub := prodserver.NewWithHub(t, db)

	oldPlayerCh := sharedHub.SubscribeUser(oldPlayerU)
	newPlayerCh := sharedHub.SubscribeUser(newPlayerU)

	token := testutil.Token(t, adminU, "admin", nil)
	body := map[string]any{
		"date": "2026-06-14", "time": "16:00",
		"opponent": "FC Umgehängt", "team_ids": []int{teamB},
		"event_type": "heim",
	}
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/games/%d", gameID), token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("UpdateGame: expected 200, got %d", res.StatusCode)
	}

	// Sanity: the game is now owned by team B only.
	var teamCount, hasB int
	if err := db.QueryRow(`SELECT COUNT(*) FROM game_teams WHERE game_id=?`, gameID).Scan(&teamCount); err != nil {
		t.Fatalf("count game_teams: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM game_teams WHERE game_id=? AND team_id=?`, gameID, teamB).Scan(&hasB); err != nil {
		t.Fatalf("check team B: %v", err)
	}
	if teamCount != 1 || hasB != 1 {
		t.Fatalf("expected game bound only to team B, got count=%d hasB=%d", teamCount, hasB)
	}

	// The REMOVED team A player must receive the "games" event.
	if ev, ok := recvWithin(oldPlayerCh, time.Second); !ok || ev != "games" {
		t.Errorf("old-team player must receive 'games' after re-assign, got %q ok=%v", ev, ok)
	}
	// The NEW team B player must also receive it.
	if ev, ok := recvWithin(newPlayerCh, time.Second); !ok || ev != "games" {
		t.Errorf("new-team player must receive 'games' after re-assign, got %q ok=%v", ev, ok)
	}
}

// TestRegenerateDaySlots_BroadcastsDuties verifies that regenerating a day's
// duty slots (POST /api/games/regenerate-day) emits a global "duties" event so
// the Dienstbörse (useLiveUpdates('duties')) refreshes. A day-scoped regen can
// touch slots of multiple teams, hence a global Broadcast is used. Guards the
// CLAUDE.md Broadcast hard rule for the regen route.
func TestRegenerateDaySlots_BroadcastsDuties(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")

	// A trainer of team A may hit the regen route (vorstand/trainer/sL tier).
	trainerU := makeTrainer(t, db, teamA, season)

	srv, sharedHub := prodserver.NewWithHub(t, db)

	// Global subscribe — Broadcast reaches both global clients and per-user streams.
	ch := sharedHub.Subscribe()
	defer sharedHub.Unsubscribe(ch)

	token := testutil.Token(t, trainerU, "standard", []string{"trainer"})
	res := testutil.Post(t, srv,
		fmt.Sprintf("/api/games/regenerate-day?date=2026-06-14&season_id=%d", season),
		token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("RegenerateDaySlots: expected 200, got %d", res.StatusCode)
	}

	if ev, ok := recvWithin(ch, time.Second); !ok || ev != "duties" {
		t.Errorf("must receive 'duties' event after RegenerateDaySlots, got %q ok=%v", ev, ok)
	}
}
