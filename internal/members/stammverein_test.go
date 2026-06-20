package members_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// memberDetail deckt die für den Stammverein relevanten Felder der GetMember-Antwort ab.
type memberDetail struct {
	HomeClubID   *int    `json:"home_club_id"`
	HomeClubName *string `json:"home_club_name"`
}

func TestUpdate_AssignHomeClub(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})
	memberID := testutil.CreateMember(t, database, 0)
	srv := newMembersServer(t, database)

	// Stammverein 8 ("TV Cannstatt 1846") aus dem Seed zuweisen.
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", memberID), tok,
		map[string]any{"first_name": "Test", "last_name": "Spieler", "status": "aktiv", "home_club_id": 8})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT: status %d, want 204", res.StatusCode)
	}

	got := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d", memberID), tok)
	if got.StatusCode != http.StatusOK {
		t.Fatalf("GET: status %d, want 200", got.StatusCode)
	}
	var d memberDetail
	json.NewDecoder(got.Body).Decode(&d)
	got.Body.Close()
	if d.HomeClubID == nil || *d.HomeClubID != 8 {
		t.Fatalf("home_club_id = %v, want 8", d.HomeClubID)
	}
	if d.HomeClubName == nil || *d.HomeClubName != "TV Cannstatt 1846" {
		t.Errorf("home_club_name = %v, want 'TV Cannstatt 1846'", d.HomeClubName)
	}
}

func TestUpdate_RemoveHomeClub(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})
	memberID := testutil.CreateMember(t, database, 0)
	database.Exec(`UPDATE members SET home_club_id=8 WHERE id=?`, memberID)
	srv := newMembersServer(t, database)

	// home_club_id: null → Zuordnung entfernen.
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", memberID), tok,
		map[string]any{"first_name": "Test", "last_name": "Spieler", "status": "aktiv", "home_club_id": nil})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT: status %d, want 204", res.StatusCode)
	}

	got := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d", memberID), tok)
	var d memberDetail
	json.NewDecoder(got.Body).Decode(&d)
	got.Body.Close()
	if d.HomeClubID != nil {
		t.Errorf("home_club_id = %v, want nil (entfernt)", *d.HomeClubID)
	}
}

func TestUpdate_InvalidHomeClubID(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})
	memberID := testutil.CreateMember(t, database, 0)
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", memberID), tok,
		map[string]any{"first_name": "Test", "last_name": "Spieler", "status": "aktiv", "home_club_id": 9999})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("PUT mit ungültiger home_club_id: status %d, want 400", res.StatusCode)
	}
}
