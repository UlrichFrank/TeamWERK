package games_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/eventlog"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestSaveLineup_NotifiesAddedAndRemovedPlayers: Aufstellungs-Änderungen lösen
// eine Push-Meldung an genau die betroffenen Spieler (und ihre Eltern via
// family_links) aus — nicht an die ganze Mannschaft (das deckt schon die
// Termin-Meldung selbst ab). Die erste Aufstellung markiert den aufgenommenen
// Spieler als "added", die zweite erzeugt zusätzlich "removed" für den
// rausgefallenen. Der Versand läuft über notify.SendAsync (Push
// Notifications-Gotcha), deshalb pollt der Test den Event-Log statt synchron
// zu prüfen.
func TestSaveLineup_NotifiesAddedAndRemovedPlayers(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	playerAUserID := testutil.CreateUser(t, db, "standard")
	playerAMemberID := testutil.CreateMember(t, db, playerAUserID)
	testutil.AddKaderMember(t, db, kaderID, playerAMemberID)
	parentUserID := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentUserID, playerAMemberID)

	playerBUserID := testutil.CreateUser(t, db, "standard")
	playerBMemberID := testutil.CreateMember(t, db, playerBUserID)
	testutil.AddKaderMember(t, db, kaderID, playerBMemberID)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	adminToken := testutil.Token(t, adminUserID, "admin", nil)

	// Erste Aufstellung: nur Spieler A -> "added" an A + Elternteil.
	res := testutil.Post(t, srv, "/api/games/"+itoa(gameID)+"/lineup", adminToken,
		map[string]any{"member_ids": []int{playerAMemberID}})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	waitForEventLogRow(t, db, playerAUserID, "In die Aufstellung aufgenommen")
	waitForEventLogRow(t, db, parentUserID, "In die Aufstellung aufgenommen")

	// Zweite Aufstellung: Spieler A raus, Spieler B rein.
	res = testutil.Post(t, srv, "/api/games/"+itoa(gameID)+"/lineup", adminToken,
		map[string]any{"member_ids": []int{playerBMemberID}})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	waitForEventLogRow(t, db, playerBUserID, "In die Aufstellung aufgenommen")
	waitForEventLogRow(t, db, playerAUserID, "Aus der Aufstellung genommen")
	waitForEventLogRow(t, db, parentUserID, "Aus der Aufstellung genommen")

	// Spieler B hat keinen Elternteil verloren und darf keine
	// Entfernungs-Meldung bekommen (er wurde ja gerade erst aufgenommen).
	events, err := eventlog.ListForUser(context.Background(), db, playerBUserID, 100)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	for _, e := range events {
		if e.Title == "Aus der Aufstellung genommen" {
			t.Errorf("Spieler B sollte keine Entfernungs-Meldung bekommen, hat aber: %+v", e)
		}
	}
}

// waitForEventLogRow pollt den Event-Log eines Nutzers, bis eine Zeile mit dem
// erwarteten Titel erscheint (notify.SendAsync liefert asynchron) oder ein
// Timeout erreicht ist.
func waitForEventLogRow(t *testing.T, db *sql.DB, userID int, title string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		events, err := eventlog.ListForUser(context.Background(), db, userID, 100)
		if err != nil {
			t.Fatalf("ListForUser: %v", err)
		}
		for _, e := range events {
			if e.Title == title {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("kein Event-Log-Eintrag %q für Nutzer %d innerhalb der Frist", title, userID)
}
