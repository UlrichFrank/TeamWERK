package carpooling_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// TestRequestPairing_BypassesPushPref nagelt fest: RequestPairing pusht die
// Mitfahranfrage an den Gegenpart über push.SendToUsers OHNE FilterByPushPref →
// der Empfänger erhält sie auch bei push_enabled=0.
//
// OFFENE DESIGN-FRAGE (nicht hier entschieden): gewollter Bypass oder Bug?
func TestRequestPairing_BypassesPushPref(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-03-20")

	bietER := testutil.CreateUser(t, db, "standard") // Empfänger der Anfrage
	admin := testutil.CreateUser(t, db, "admin")     // Sucher/Anfragender
	testutil.CreateNotificationPreference(t, db, bietER, "carpooling", false, false)

	var bieteID, sucheID int
	if r, err := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt) VALUES (?, ?, 'biete', 3, 'P')`, gameID, bietER); err != nil {
		t.Fatalf("seed biete: %v", err)
	} else {
		id, _ := r.LastInsertId()
		bieteID = int(id)
	}
	if r, err := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, admin); err != nil {
		t.Fatalf("seed suche: %v", err)
	} else {
		id, _ := r.LastInsertId()
		sucheID = int(id)
	}

	ch := make(chan []int, 8)
	orig := push.SendToUsers
	push.SendToUsers = func(_ *sql.DB, _ *appconfig.Config, uids []int, _, _, _ string) { ch <- uids }
	t.Cleanup(func() { push.SendToUsers = orig })

	srv := prodserver.New(t, db)
	tok := testutil.Token(t, admin, "admin", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", tok, map[string]any{
		"bieteId": bieteID,
		"sucheId": sucheID,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case uids := <-ch:
			for _, u := range uids {
				if u == bietER {
					return
				}
			}
		case <-deadline:
			t.Fatalf("push an Empfänger %d nicht erhalten", bietER)
		}
	}
}
