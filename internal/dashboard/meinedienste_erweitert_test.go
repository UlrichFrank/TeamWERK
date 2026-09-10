package dashboard_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/dashboard"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestDashboard_MeineDienste_ErweiterterKaderZaehltNicht: „Meine Dienste" wählt
// das nächste Spiel nur aus Stammkader-/Trainer-Teams. Ein früheres Spiel des
// Teams, in dem der Spieler nur im erweiterten Kader steht, darf nicht
// erscheinen — dort gibt es keine Dienstpflicht (deckungsgleich mit /dienste).
func TestDashboard_MeineDienste_ErweiterterKaderZaehltNicht(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	stammTeam := testutil.CreateTeam(t, db, "mC1")
	erwTeam := testutil.CreateTeam(t, db, "mB1")
	stammKader := testutil.CreateKader(t, db, stammTeam, seasonID)
	erwKader := testutil.CreateKader(t, db, erwTeam, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, stammKader, memberID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, erwKader, memberID)

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	nextWeek := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	erwGame := testutil.CreateGame(t, db, seasonID, erwTeam, tomorrow)
	stammGame := testutil.CreateGame(t, db, seasonID, stammTeam, nextWeek)

	dt := testutil.CreateDutyType(t, db, "Kampfgericht", 1.0)
	testutil.CreateDutySlot(t, db, dt, seasonID, erwTeam, erwGame, tomorrow)
	testutil.CreateDutySlot(t, db, dt, seasonID, stammTeam, stammGame, nextWeek)

	srv := testServer(t, dashboard.NewHandler(db))
	token := testutil.Token(t, userID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body struct {
		MeineDienste struct {
			NextGame *struct {
				ID int `json:"id"`
			} `json:"nextGame"`
		} `json:"meineDienste"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	if body.MeineDienste.NextGame == nil {
		t.Fatalf("expected nextGame for Stammkader team, got nil")
	}
	if got := body.MeineDienste.NextGame.ID; got != stammGame {
		t.Errorf("expected nextGame=%d (Stammkader), got %d (extended-kader game is %d)", stammGame, got, erwGame)
	}
}

// TestDashboard_MeineDienste_NurErweiterterKader_KeinSpiel: wer ausschließlich
// im erweiterten Kader steht, bekommt kein „nächstes Spiel mit Diensten".
func TestDashboard_MeineDienste_NurErweiterterKader_KeinSpiel(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "mB1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	parentID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentID, childMemberID)

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, tomorrow)
	dt := testutil.CreateDutyType(t, db, "Kampfgericht", 1.0)
	testutil.CreateDutySlot(t, db, dt, seasonID, teamID, gameID, tomorrow)

	srv := testServer(t, dashboard.NewHandler(db))
	token := testutil.TokenWithIsParent(t, parentID, "standard", nil, true)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	var md map[string]any
	json.Unmarshal(body["meineDienste"], &md)
	if md["nextGame"] != nil {
		t.Errorf("expected no nextGame for parent of extended-only child, got %v", md["nextGame"])
	}
}
