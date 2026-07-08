package duties_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// captureNotifyCategory überschreibt den notify.Send-Seam und liefert einen
// Kanal der übergebenen Kategorien.
func captureNotifyCategory(t *testing.T) chan string {
	t.Helper()
	ch := make(chan string, 16)
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, _ []int, category, _, _, _ string) {
		ch <- category
	}
	t.Cleanup(func() { notify.Send = orig })
	return ch
}

func waitForCategory(t *testing.T, ch chan string, want string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case c := <-ch:
			if c == want {
				return
			}
		case <-deadline:
			t.Fatalf("notify.Send mit Kategorie %q nicht erhalten", want)
		}
	}
}

// TestCreateSlot_NotifiesWithDutiesCategory — ein neuer Dienst-Slot löst
// notify.Send mit Kategorie "duties" aus.
func TestCreateSlot_NotifiesWithDutiesCategory(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	dutyTypeID := testutil.CreateDutyType(t, db, "Kasse", 1.0)
	admin := testutil.CreateUser(t, db, "admin")

	cat := captureNotifyCategory(t)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, admin, "admin", nil)

	res := testutil.Post(t, srv, "/api/duty-slots", tok, map[string]any{
		"event_name":   "Heimspiel",
		"event_date":   "2026-03-20",
		"event_time":   "18:00",
		"duty_type_id": dutyTypeID,
		"role_desc":    "Kasse",
		"slots_total":  2,
		"season_id":    seasonID,
		"team_id":      teamID,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", res.StatusCode)
	}
	waitForCategory(t, cat, "duties")
}
