package absences_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/absences"
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

// TestAbsencesMutation_ScopedToTeamAndStaff verifies that creating an absence
// for a child on team A (POST /api/absences) delivers the "absences" event to
// the child's team (trainer) and club-wide staff plus the parent who created it,
// but not to a player of a different team.
func TestAbsencesMutation_ScopedToTeamAndStaff(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	kaderB := testutil.CreateKader(t, db, teamB, season)

	// Child on team A + their parent (creator of the absence).
	childM := testutil.CreateMember(t, db, 0)
	testutil.AddKaderMember(t, db, kaderA, childM)
	parentU := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentU, childM)

	// Trainer of team A.
	trainerU := testutil.CreateUser(t, db, "standard")
	trainerM := testutil.CreateMember(t, db, trainerU)
	testutil.AddClubFunction(t, db, trainerM, "trainer")
	testutil.AddKaderTrainer(t, db, kaderA, trainerM)

	// Club-wide vorstand.
	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	// Player on team B — must NOT receive team A's absence event.
	foreignU := testutil.CreateUser(t, db, "standard")
	foreignM := testutil.CreateMember(t, db, foreignU)
	testutil.AddKaderMember(t, db, kaderB, foreignM)

	sharedHub := hub.NewHub()
	h := absences.NewHandler(db, sharedHub)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/absences", h.Create)
	})

	parentCh := sharedHub.SubscribeUser(parentU)
	trainerCh := sharedHub.SubscribeUser(trainerU)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	foreignCh := sharedHub.SubscribeUser(foreignU)

	parentTok := testutil.TokenWithIsParent(t, parentU, "standard", nil, true)
	res := testutil.Do(t, srv, http.MethodPost, "/api/absences", parentTok,
		map[string]any{
			"member_ids": []int{childM},
			"type":       "vacation",
			"start_date": "2026-07-01",
			"end_date":   "2026-07-14",
			"note":       "Urlaub",
		})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Create absence: expected 201, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"parent":   parentCh,
		"trainer":  trainerCh,
		"vorstand": vorstandCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "absences" {
			t.Errorf("%s stream must receive 'absences', got %q ok=%v", name, ev, ok)
		}
	}
	if ev, ok := recvWithin(foreignCh, 300*time.Millisecond); ok {
		t.Errorf("team-foreign player must NOT receive absences event, got %q", ev)
	}
}
