package carpooling_test

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

// TestUpsert_NotifiesWithCarpoolingCategory — eine neue Mitfahrgelegenheit löst
// (in einer Goroutine) notify.Send mit Kategorie "carpooling" aus.
func TestUpsert_NotifiesWithCarpoolingCategory(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-03-20")
	admin := testutil.CreateUser(t, db, "admin")

	// Ein Gegenpart ('biete') existiert bereits — sonst kehrt notifyOpposite
	// mangels Empfänger früh zurück und ruft notify.Send gar nicht auf.
	other := testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(
		`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt) VALUES (?, ?, 'biete', 3, 'Parkplatz')`,
		gameID, other); err != nil {
		t.Fatalf("seed biete: %v", err)
	}

	cat := captureNotifyCategory(t)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, admin, "admin", nil)

	res := testutil.Post(t, srv, "/api/mitfahrgelegenheiten", tok, map[string]any{
		"gameId":     gameID,
		"typ":        "suche",
		"plaetze":    1, // 'suche' verlangt plaetze >= 1
		"treffpunkt": "Halle",
		"notiz":      "",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 2xx", res.StatusCode)
	}
	waitForCategory(t, cat, "carpooling")
}
