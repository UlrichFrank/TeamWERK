package config_test

import (
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

// TestSettingsMutation_StaysGlobal verifies that the club-wide "settings" topic
// remains global: a settings mutation (PUT /api/club) reaches EVERY connected
// per-user stream, including a plain player without any club function — the
// regression guard that global topics were not accidentally scoped.
func TestSettingsMutation_StaysGlobal(t *testing.T) {
	db := testutil.NewDB(t)

	// Vorstand performs the mutation.
	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")
	vorstandTok := testutil.Token(t, vorstandU, "standard", []string{"vorstand"})

	// Plain player without any function — must STILL receive the global event.
	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	testutil.AddClubFunction(t, db, playerM, "spieler")

	adminU := testutil.CreateUser(t, db, "admin")

	srv, sharedHub := prodserver.NewWithHub(t, db)

	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	playerCh := sharedHub.SubscribeUser(playerU)
	adminCh := sharedHub.SubscribeUser(adminU)

	res := testutil.Do(t, srv, http.MethodPut, "/api/club", vorstandTok,
		map[string]any{"name": "Team Stuttgart", "logo_url": "", "address": "Musterstr. 1"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("UpdateClub: expected 204, got %d", res.StatusCode)
	}

	// Every connected stream — including the plain player — must receive it.
	for name, ch := range map[string]chan string{
		"vorstand": vorstandCh,
		"player":   playerCh,
		"admin":    adminCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "settings" {
			t.Errorf("%s stream must receive global 'settings', got %q ok=%v", name, ev, ok)
		}
	}
}
