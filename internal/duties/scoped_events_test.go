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

// TestClaim_BroadcastsToTeam verifies that claiming an open slot (POST
// /api/duty-board/{slotId}/claim) sends the "duties" event to the slot's team
// audience — a teammate of the claiming user must be notified so foreign
// duty-board views refresh. Guards the CLAUDE.md Broadcast hard rule.
func TestClaim_BroadcastsToTeam(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, season, teamA, 0, "2026-06-14")

	claimU := testutil.CreateUser(t, db, "standard")
	claimM := testutil.CreateMember(t, db, claimU)
	testutil.AddKaderMember(t, db, kaderA, claimM)

	teammateU := testutil.CreateUser(t, db, "standard")
	teammateM := testutil.CreateMember(t, db, teammateU)
	testutil.AddKaderMember(t, db, kaderA, teammateM)

	sharedHub := hub.NewHub()
	h := duties.NewHandler(db, testutil.TestConfig(), sharedHub)
	srv := testServer(t, h)

	teammateCh := sharedHub.SubscribeUser(teammateU)

	token := testutil.Token(t, claimU, "spieler", nil)
	res := testutil.Post(t, srv, "/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("Claim: expected 204, got %d", res.StatusCode)
	}

	if ev, ok := recvWithin(teammateCh, time.Second); !ok || ev != "duties" {
		t.Errorf("teammate must receive 'duties' event after claim, got %q ok=%v", ev, ok)
	}
}

// TestUnclaim_BroadcastsToActingUser verifies that releasing an own slot
// (DELETE /api/duty-board/{slotId}/claim) sends the "duties" event to the
// acting user themselves. The acting user is passed as an extra recipient
// because after unclaiming they may drop out of the slot's team audience, yet
// their own view must still refresh. Guards the CLAUDE.md Broadcast hard rule.
func TestUnclaim_BroadcastsToActingUser(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, season, teamA, 0, "2026-06-14")

	// Acting user has no team membership — only reachable via extraUserIDs.
	userID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, userID, "assigned")
	db.Exec(`UPDATE duty_slots SET slots_filled=1 WHERE id=?`, slotID)

	sharedHub := hub.NewHub()
	h := duties.NewHandler(db, testutil.TestConfig(), sharedHub)
	srv := testServer(t, h)

	userCh := sharedHub.SubscribeUser(userID)

	token := testutil.Token(t, userID, "spieler", nil)
	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/duty-board/"+itoa(slotID)+"/claim", token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("Unclaim: expected 204, got %d", res.StatusCode)
	}

	if ev, ok := recvWithin(userCh, time.Second); !ok || ev != "duties" {
		t.Errorf("acting user must receive 'duties' event after unclaim, got %q ok=%v", ev, ok)
	}
}
