package auth_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestRequestMembership_NotifiesWithMembershipCategory — Kategorie-Korrektheit
// (repräsentativ für Gruppe 6): der Beitrittsantrag benachrichtigt Admins über
// notify.Send mit der Kategorie "membership". Nutzt den notify.Send-Seam.
func TestRequestMembership_NotifiesWithMembershipCategory(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateUser(t, db, "admin") // Empfänger — sonst ermittelt der Handler keine adminIDs

	type capture struct {
		category string
		uids     []int
	}
	ch := make(chan capture, 4)
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, userIDs []int, category, _, _, _ string) {
		ch <- capture{category, userIDs}
	}
	t.Cleanup(func() { notify.Send = orig })

	srv := newAuthServer(t, db)
	res := testutil.Post(t, srv, "/api/auth/request-membership", "", map[string]string{
		"first_name": "Max",
		"last_name":  "Muster",
		"email":      "max@test.local",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", res.StatusCode)
	}

	select {
	case c := <-ch:
		if c.category != "membership" {
			t.Fatalf("category = %q, want membership", c.category)
		}
		if len(c.uids) == 0 {
			t.Fatal("keine Empfänger (Admins) ermittelt")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("notify.Send wurde nicht aufgerufen")
	}
}
