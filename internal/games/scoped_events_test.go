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
