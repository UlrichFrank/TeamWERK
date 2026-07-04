package kader_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/kader"
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

// TestKaderMutation_ScopedToTeamAndStaff verifies that a kader mutation
// (PUT /api/admin/kader/{id}) delivers the "kader" event to the kader's team
// (trainer + player + parent) and club-wide staff, but not to a player of a
// different team.
func TestKaderMutation_ScopedToTeamAndStaff(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	kaderB := testutil.CreateKader(t, db, teamB, season)

	trainerU := testutil.CreateUser(t, db, "standard")
	trainerM := testutil.CreateMember(t, db, trainerU)
	testutil.AddClubFunction(t, db, trainerM, "trainer")
	testutil.AddKaderTrainer(t, db, kaderA, trainerM)

	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	testutil.AddKaderMember(t, db, kaderA, playerM)
	parentU := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentU, playerM)

	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	foreignU := testutil.CreateUser(t, db, "standard")
	foreignM := testutil.CreateMember(t, db, foreignU)
	testutil.AddKaderMember(t, db, kaderB, foreignM)

	sharedHub := hub.NewHub()
	h := kader.NewHandler(db, sharedHub)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/admin/kader/{id}", h.UpdateKader)
	})

	trainerCh := sharedHub.SubscribeUser(trainerU)
	playerCh := sharedHub.SubscribeUser(playerU)
	parentCh := sharedHub.SubscribeUser(parentU)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	foreignCh := sharedHub.SubscribeUser(foreignU)

	// A member to add to kader A to make the mutation non-trivial.
	newMember := testutil.CreateMember(t, db, 0)
	token := testutil.Token(t, vorstandU, "standard", []string{"vorstand"})
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/admin/kader/%d", kaderA), token,
		map[string]any{"members_add": []int{newMember}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("UpdateKader: expected 204, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"trainer":  trainerCh,
		"player":   playerCh,
		"parent":   parentCh,
		"vorstand": vorstandCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "kader" {
			t.Errorf("%s stream must receive 'kader', got %q ok=%v", name, ev, ok)
		}
	}
	if ev, ok := recvWithin(foreignCh, 300*time.Millisecond); ok {
		t.Errorf("team-foreign player must NOT receive kader event, got %q", ev)
	}
}
