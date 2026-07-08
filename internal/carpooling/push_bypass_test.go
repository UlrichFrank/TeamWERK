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

// requestPairing baut das Szenario (biete-Zeile eines Empfängers + suche-Zeile
// des Admins), stellt die Anfrage und gibt einen Push-Empfänger-Kanal zurück.
func requestPairing(t *testing.T, disablePref bool) (recipient int, pushes chan []int) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-03-20")

	recipient = testutil.CreateUser(t, db, "standard")
	admin := testutil.CreateUser(t, db, "admin")
	if disablePref {
		testutil.CreateNotificationPreference(t, db, recipient, "carpooling", false, false)
	}

	var bieteID, sucheID int
	r, err := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt) VALUES (?, ?, 'biete', 3, 'P')`, gameID, recipient)
	if err != nil {
		t.Fatalf("seed biete: %v", err)
	}
	id, _ := r.LastInsertId()
	bieteID = int(id)
	r, err = db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, admin)
	if err != nil {
		t.Fatalf("seed suche: %v", err)
	}
	id, _ = r.LastInsertId()
	sucheID = int(id)

	pushes = make(chan []int, 8)
	orig := push.SendToUsers
	push.SendToUsers = func(_ *sql.DB, _ *appconfig.Config, uids []int, _, _, _ string) { pushes <- uids }
	t.Cleanup(func() { push.SendToUsers = orig })

	srv := prodserver.New(t, db)
	tok := testutil.Token(t, admin, "admin", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", tok, map[string]any{"bieteId": bieteID, "sucheId": sucheID})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}
	return recipient, pushes
}

// TestRequestPairing_RespectsCarpoolingOptOut — der angefragte Nutzer mit
// 'carpooling'=aus erhält die Mitfahranfrage NICHT mehr (war Bug, jetzt
// konsistent mit Confirm/Reject).
func TestRequestPairing_RespectsCarpoolingOptOut(t *testing.T) {
	recipient, pushes := requestPairing(t, true)
	deadline := time.After(400 * time.Millisecond)
	for {
		select {
		case uids := <-pushes:
			for _, u := range uids {
				if u == recipient {
					t.Fatalf("Empfänger %d trotz Opt-out gepusht", recipient)
				}
			}
		case <-deadline:
			return
		}
	}
}

// TestRequestPairing_DefaultSends — ohne Opt-out geht die Anfrage-Push raus.
func TestRequestPairing_DefaultSends(t *testing.T) {
	recipient, pushes := requestPairing(t, false)
	deadline := time.After(2 * time.Second)
	for {
		select {
		case uids := <-pushes:
			for _, u := range uids {
				if u == recipient {
					return
				}
			}
		case <-deadline:
			t.Fatalf("push an Empfänger %d nicht erhalten (Default)", recipient)
		}
	}
}
