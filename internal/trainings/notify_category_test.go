package trainings_test

import (
	"database/sql"
	"net/http"
	"strconv"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

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

// TestUpdateSession_NotifiesWithTrainingsCategory — eine geänderte Trainings-
// einheit löst notify.Send mit Kategorie "trainings" aus.
func TestUpdateSession_NotifiesWithTrainingsCategory(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	admin := testutil.CreateUser(t, db, "admin")

	cat := captureNotifyCategory(t)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, admin, "admin", nil)

	res := testutil.Do(t, srv, http.MethodPut, "/api/training-sessions/"+strconv.Itoa(sessionID), tok, map[string]any{
		"title":      "Einheit",
		"date":       "2026-03-15",
		"start_time": "18:00",
		"end_time":   "19:30",
		"status":     "active",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}
	waitForCategory(t, cat, "trainings")
}
