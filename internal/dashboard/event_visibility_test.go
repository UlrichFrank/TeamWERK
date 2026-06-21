package dashboard_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/dashboard"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestDashboard_NaechsteTermine_Filter: Dashboard zeigt nur Events der Teams,
// in denen der User selbst Mitglied ist — Fremd-Team-Spiele tauchen nicht auf.
// Dies ist Konsequenz der bestehenden user_accessible_teams-Filterung; der
// event-team-visibility-Change kodifiziert das als verbindliche Invariante.
func TestDashboard_NaechsteTermine_Filter(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)

	playerUID := testutil.CreateUser(t, db, "standard")
	playerMID := testutil.CreateMember(t, db, playerUID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, playerMID)

	// Ein Spiel pro Team morgen — beide würden ohne Filter als „nächste Termine"
	// zählen, weil Datum identisch ist.
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	ownGameID := testutil.CreateGame(t, db, seasonID, teamA, tomorrow)
	otherGameID := testutil.CreateGame(t, db, seasonID, teamB, tomorrow)

	h := dashboard.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, playerUID, "standard", nil)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	var events []map[string]any
	json.Unmarshal(body["meineTermine"], &events)

	for _, e := range events {
		id := int(e["id"].(float64))
		if id == otherGameID {
			t.Errorf("Dashboard zeigt fremdes Team-Spiel %d (Team B), das der Spieler nicht sehen darf", otherGameID)
		}
	}

	// Sanity: das eigene Spiel ist drin (sofern Tests nicht 0 Events liefern).
	foundOwn := false
	for _, e := range events {
		if int(e["id"].(float64)) == ownGameID {
			foundOwn = true
		}
	}
	if !foundOwn {
		t.Errorf("erwartet eigenes Spiel %d in meineTermine, got %v", ownGameID, events)
	}
}
