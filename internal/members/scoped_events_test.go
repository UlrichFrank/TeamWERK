package members_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// recvWithin reads one event from ch within d, or returns ("", false).
func recvWithin(ch chan string, d time.Duration) (string, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return "", false
	}
}

// TestMembersMutation_ScopedToVorstand verifies that a members mutation
// (PUT /api/members/{id}/status) delivers the "members" live-update event to the
// finance group (admin + vorstand + kassierer) but NOT to a plain player's
// stream — the player may not read member data, so must not be asked to reload.
func TestMembersMutation_ScopedToVorstand(t *testing.T) {
	db := testutil.NewDB(t)

	// Vorstand performs the mutation and must receive the event.
	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")
	vorstandTok := testutil.Token(t, vorstandU, "standard", []string{"vorstand"})

	// Admin and kassierer also in the finance group.
	adminU := testutil.CreateUser(t, db, "admin")
	kassiererU := testutil.CreateUser(t, db, "standard")
	kassiererM := testutil.CreateMember(t, db, kassiererU)
	testutil.AddClubFunction(t, db, kassiererM, "kassierer")

	// Plain player — must NOT receive the event.
	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	testutil.AddClubFunction(t, db, playerM, "spieler")

	// The member whose status we mutate.
	targetM := testutil.CreateMember(t, db, 0)

	srv, sharedHub := prodserver.NewWithHub(t, db)

	// Subscribe each user's per-user stream directly on the shared hub (this is
	// what /api/events does under the hood).
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	adminCh := sharedHub.SubscribeUser(adminU)
	kassiererCh := sharedHub.SubscribeUser(kassiererU)
	playerCh := sharedHub.SubscribeUser(playerU)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/members/"+strconv.Itoa(targetM)+"/status", vorstandTok,
		map[string]string{"status": "pausiert"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("UpdateStatus: expected 204, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"vorstand":  vorstandCh,
		"admin":     adminCh,
		"kassierer": kassiererCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "members" {
			t.Errorf("%s stream must receive 'members', got %q ok=%v", name, ev, ok)
		}
	}
	if ev, ok := recvWithin(playerCh, 300*time.Millisecond); ok {
		t.Errorf("plain player must NOT receive members event, got %q", ev)
	}
}
