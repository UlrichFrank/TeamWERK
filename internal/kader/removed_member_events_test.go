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

// TestUpdateKader_RemovedMemberNotified verifies Fund C (1): a player who is
// removed from the roster via PUT /api/admin/kader/{id} (members_remove) still
// receives the "kader" event, even though they are no longer part of the team
// audience after the delete. Without the removed-member audience they would keep
// showing on a roster they left.
func TestUpdateKader_RemovedMemberNotified(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	// Player that will be removed. Owns a linked user so we can watch their stream.
	removedU := testutil.CreateUser(t, db, "standard")
	removedM := testutil.CreateMember(t, db, removedU)
	testutil.AddKaderMember(t, db, kaderA, removedM)
	// Parent of the removed player must also learn about the removal.
	removedParentU := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, removedParentU, removedM)

	// A player that stays on the roster (still in the team audience afterwards).
	stayU := testutil.CreateUser(t, db, "standard")
	stayM := testutil.CreateMember(t, db, stayU)
	testutil.AddKaderMember(t, db, kaderA, stayM)

	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	sharedHub := hub.NewHub()
	h := kader.NewHandler(db, sharedHub)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/admin/kader/{id}", h.UpdateKader)
	})

	removedCh := sharedHub.SubscribeUser(removedU)
	removedParentCh := sharedHub.SubscribeUser(removedParentU)
	stayCh := sharedHub.SubscribeUser(stayU)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)

	token := testutil.Token(t, vorstandU, "standard", []string{"vorstand"})
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/admin/kader/%d", kaderA), token,
		map[string]any{"members_remove": []int{removedM}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("UpdateKader: expected 204, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"removed player": removedCh,
		"removed parent": removedParentCh,
		"staying player": stayCh,
		"vorstand":       vorstandCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "kader" {
			t.Errorf("%s stream must receive 'kader', got %q ok=%v", name, ev, ok)
		}
	}
}

// TestUpdateKader_AgeClassSwitch_NotifiesOldTeam verifies Fund C (2): an
// age-class change repoints kader.team_id to a different team; the OLD team must
// still be notified (its roster shrank), not only the new one.
func TestUpdateKader_AgeClassSwitch_NotifiesOldTeam(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	oldTeam := testutil.CreateTeam(t, db, "Old Team")
	k := testutil.CreateKader(t, db, oldTeam, season)

	// Player on the OLD team's roster. The age-class switch moves the kader to a
	// newly-ensured team, so after the update this player is no longer in the new
	// team audience — but their old-team subscription must still fire.
	oldPlayerU := testutil.CreateUser(t, db, "standard")
	oldPlayerM := testutil.CreateMember(t, db, oldPlayerU)
	testutil.AddKaderMember(t, db, k, oldPlayerM)

	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	sharedHub := hub.NewHub()
	h := kader.NewHandler(db, sharedHub)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/admin/kader/{id}", h.UpdateKader)
	})

	oldPlayerCh := sharedHub.SubscribeUser(oldPlayerU)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)

	// Switch the age class → ensureTeam creates/points to a different team.
	newAgeClass := "A-Jugend"
	token := testutil.Token(t, vorstandU, "standard", []string{"vorstand"})
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/admin/kader/%d", k), token,
		map[string]any{"age_class": newAgeClass})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("UpdateKader: expected 204, got %d", res.StatusCode)
	}

	// Confirm the team actually moved (guards against the test passing trivially).
	var teamID int
	if err := db.QueryRow(`SELECT team_id FROM kader WHERE id=?`, k).Scan(&teamID); err != nil {
		t.Fatalf("read team_id: %v", err)
	}
	if teamID == oldTeam {
		t.Fatalf("age-class switch did not repoint team_id (still %d)", oldTeam)
	}

	for name, ch := range map[string]chan string{
		"old-team player": oldPlayerCh,
		"vorstand":        vorstandCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "kader" {
			t.Errorf("%s stream must receive 'kader', got %q ok=%v", name, ev, ok)
		}
	}
}
