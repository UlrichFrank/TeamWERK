package duties_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func recvWithin(ch chan string, d time.Duration) (string, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return "", false
	}
}

// TestDutiesMutation_ScopedToTeamAndStaff verifies that creating a team-bound
// duty slot (POST /api/duty-slots with team_id) delivers the "duties" event to
// the slot's team (trainer + player) and club-wide staff, but not to a player of
// a different team — mirroring the /api/duty-board team filter.
func TestDutiesMutation_ScopedToTeamAndStaff(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	kaderB := testutil.CreateKader(t, db, teamB, season)
	dtID := createDutyType(t, db, "Aufbau", 2.0)

	trainerU := testutil.CreateUser(t, db, "standard")
	trainerM := testutil.CreateMember(t, db, trainerU)
	testutil.AddClubFunction(t, db, trainerM, "trainer")
	testutil.AddKaderTrainer(t, db, kaderA, trainerM)

	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	testutil.AddKaderMember(t, db, kaderA, playerM)

	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	foreignU := testutil.CreateUser(t, db, "standard")
	foreignM := testutil.CreateMember(t, db, foreignU)
	testutil.AddKaderMember(t, db, kaderB, foreignM)

	sharedHub := hub.NewHub()
	h := duties.NewHandler(db, testutil.TestConfig(), sharedHub)
	srv := testServer(t, h)

	trainerCh := sharedHub.SubscribeUser(trainerU)
	playerCh := sharedHub.SubscribeUser(playerU)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	foreignCh := sharedHub.SubscribeUser(foreignU)

	token := testutil.Token(t, vorstandU, "standard", []string{"vorstand"})
	body := map[string]any{
		"event_name":   "Aufbau Heimspiel",
		"event_date":   "2026-06-14",
		"duty_type_id": dtID,
		"slots_total":  2,
		"team_id":      teamA,
		"season_id":    season,
	}
	res := testutil.Post(t, srv, "/api/duty-slots", token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("CreateSlot: expected 201, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"trainer":  trainerCh,
		"player":   playerCh,
		"vorstand": vorstandCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "duties" {
			t.Errorf("%s stream must receive 'duties', got %q ok=%v", name, ev, ok)
		}
	}
	if ev, ok := recvWithin(foreignCh, 300*time.Millisecond); ok {
		t.Errorf("team-foreign player must NOT receive duties event, got %q", ev)
	}
}
