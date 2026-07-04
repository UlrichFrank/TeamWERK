package trainings_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

func recvWithin(ch chan string, d time.Duration) (string, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return "", false
	}
}

// TestTrainingsMutation_ScopedToTeamAndStaff verifies that a training-session
// mutation (POST /api/training-sessions/{id}/attendances → "trainings") reaches
// the team's trainer + player + a club-wide vorstand, but NOT a player of a
// different team.
func TestTrainingsMutation_ScopedToTeamAndStaff(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	kaderB := testutil.CreateKader(t, db, teamB, season)

	sessionID := testutil.CreateTrainingSession(t, db, teamA, season, "2025-01-10")

	// Trainer of team A performs the mutation.
	trainerU := testutil.CreateUser(t, db, "standard")
	trainerM := testutil.CreateMember(t, db, trainerU)
	testutil.AddClubFunction(t, db, trainerM, "trainer")
	testutil.AddKaderTrainer(t, db, kaderA, trainerM)

	// Player on team A.
	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	testutil.AddKaderMember(t, db, kaderA, playerM)

	// Club-wide vorstand.
	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	// Player on team B — must NOT receive team A's event.
	foreignU := testutil.CreateUser(t, db, "standard")
	foreignM := testutil.CreateMember(t, db, foreignU)
	testutil.AddKaderMember(t, db, kaderB, foreignM)

	sharedHub := hub.NewHub()
	h := trainings.NewHandler(db, testutil.TestConfig(), sharedHub)
	srv := testServer(t, h)

	trainerCh := sharedHub.SubscribeUser(trainerU)
	playerCh := sharedHub.SubscribeUser(playerU)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	foreignCh := sharedHub.SubscribeUser(foreignU)

	token := testutil.Token(t, trainerU, "standard", []string{"trainer"})
	body := []map[string]any{{"member_id": playerM, "present": true}}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("SaveAttendances: expected 204, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"trainer":  trainerCh,
		"player":   playerCh,
		"vorstand": vorstandCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "trainings" {
			t.Errorf("%s stream must receive 'trainings', got %q ok=%v", name, ev, ok)
		}
	}
	if ev, ok := recvWithin(foreignCh, 300*time.Millisecond); ok {
		t.Errorf("team-foreign player must NOT receive training event, got %q", ev)
	}
}
