package games_test

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

// TestCreateGame_NotifiesWithGamesCategory — ein neues Spiel löst notify.Send
// mit Kategorie "games" aus (Auto-Duty-Regen kann zusätzlich "duties" senden —
// waitForCategory sucht gezielt "games").
func TestCreateGame_NotifiesWithGamesCategory(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	admin := testutil.CreateUser(t, db, "admin")

	cat := captureNotifyCategory(t)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, admin, "admin", nil)

	res := testutil.Post(t, srv, "/api/games", tok, map[string]any{
		"date":                  "2026-03-20",
		"time":                  "19:00",
		"opponent":              "FC Gegner",
		"team_ids":              []int{teamID},
		"event_type":            "heim",
		"season_id":             seasonID,
		"rsvp_default_players":  "none",
		"rsvp_default_extended": "none",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", res.StatusCode)
	}
	waitForCategory(t, cat, "games")
}
