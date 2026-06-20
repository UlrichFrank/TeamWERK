package teams_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func testServer(t *testing.T, h *teams.Handler) *httptest.Server {
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/teams/{id}/roster", h.GetRoster)
		r.Get("/api/teams", h.ListMyTeams)
	})
}

// TestGetRoster_ExtendedPlayers verifies that extended kader members appear in extended_players
// and not in players, and that regular members appear only in players.
func TestGetRoster_ExtendedPlayers(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// Regular player
	regularUserID := testutil.CreateUser(t, db, "standard")
	regularMemberID := testutil.CreateMember(t, db, regularUserID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, regularMemberID)

	// Extended player
	extUserID := testutil.CreateUser(t, db, "standard")
	extMemberID := testutil.CreateMember(t, db, extUserID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, extMemberID)

	h := teams.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, regularUserID, "standard", nil)
	res := testutil.Get(t, srv, "/api/teams/"+strconv.Itoa(teamID)+"/roster", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var roster map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&roster)
	res.Body.Close()

	var players, extended []map[string]any
	json.Unmarshal(roster["players"], &players)
	json.Unmarshal(roster["extended_players"], &extended)

	if len(players) != 1 {
		t.Errorf("expected 1 regular player, got %d", len(players))
	}
	if len(extended) != 1 {
		t.Errorf("expected 1 extended player, got %d", len(extended))
	}

	// extended player must not appear in regular players
	for _, p := range players {
		if int(p["userId"].(float64)) == extUserID {
			t.Error("extended player must not appear in players array")
		}
	}
}

// TestGetRoster_ExtendedPlayerCanAccessRoster verifies that an extended kader member
// can access the roster (not forbidden) after the view update.
func TestGetRoster_ExtendedPlayerCanAccessRoster(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	extUserID := testutil.CreateUser(t, db, "standard")
	extMemberID := testutil.CreateMember(t, db, extUserID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, extMemberID)

	h := teams.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, extUserID, "standard", nil)
	res := testutil.Get(t, srv, "/api/teams/"+strconv.Itoa(teamID)+"/roster", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for extended player, got %d (should not be 403)", res.StatusCode)
	}
}

// TestGetRoster_ExtendedParents verifies that a parent whose child is only in the
// extended kader appears in extended_parents (not parents), while a parent of a
// regular kader child appears in parents. Regression: extended-squad parents were
// previously missing entirely from the Eltern list.
func TestGetRoster_ExtendedParents(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Jugend")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// Regular kader child + parent
	regChildUserID := testutil.CreateUser(t, db, "standard")
	regChildMemberID := testutil.CreateMember(t, db, regChildUserID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, regChildMemberID)
	regParentUserID := testutil.CreateUser(t, db, "standard")
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, regParentUserID, regChildMemberID)

	// Extended kader child + parent
	extChildUserID := testutil.CreateUser(t, db, "standard")
	extChildMemberID := testutil.CreateMember(t, db, extChildUserID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, extChildMemberID)
	extParentUserID := testutil.CreateUser(t, db, "standard")
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, extParentUserID, extChildMemberID)

	h := teams.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, regParentUserID, "standard", nil)
	res := testutil.Get(t, srv, "/api/teams/"+strconv.Itoa(teamID)+"/roster", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var roster map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&roster)
	res.Body.Close()

	var parents, extParents []map[string]any
	json.Unmarshal(roster["parents"], &parents)
	json.Unmarshal(roster["extended_parents"], &extParents)

	if len(parents) != 1 || int(parents[0]["userId"].(float64)) != regParentUserID {
		t.Errorf("expected only regular parent %d in parents, got %v", regParentUserID, parents)
	}
	if len(extParents) != 1 || int(extParents[0]["userId"].(float64)) != extParentUserID {
		t.Errorf("expected only extended parent %d in extended_parents, got %v", extParentUserID, extParents)
	}
}

// TestGetRoster_ParentWithRegularAndExtendedChild verifies that a parent with one
// child in the regular kader and another in the extended kader is listed once under
// parents (regular wins), with both children shown.
func TestGetRoster_ParentWithRegularAndExtendedChild(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Jugend")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	parentUserID := testutil.CreateUser(t, db, "standard")

	regChildUserID := testutil.CreateUser(t, db, "standard")
	regChildMemberID := testutil.CreateMember(t, db, regChildUserID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, regChildMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, regChildMemberID)

	extChildUserID := testutil.CreateUser(t, db, "standard")
	extChildMemberID := testutil.CreateMember(t, db, extChildUserID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, extChildMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, extChildMemberID)

	h := teams.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, parentUserID, "standard", nil)
	res := testutil.Get(t, srv, "/api/teams/"+strconv.Itoa(teamID)+"/roster", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var roster map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&roster)
	res.Body.Close()

	var parents, extParents []map[string]any
	json.Unmarshal(roster["parents"], &parents)
	json.Unmarshal(roster["extended_parents"], &extParents)

	if len(parents) != 1 {
		t.Fatalf("expected 1 parent in parents (regular wins), got %d", len(parents))
	}
	if len(extParents) != 0 {
		t.Errorf("parent must not be duplicated into extended_parents, got %d", len(extParents))
	}
	var children []any
	if c, ok := parents[0]["children"].([]any); ok {
		children = c
	}
	if len(children) != 2 {
		t.Errorf("expected both children (regular + extended) listed, got %v", children)
	}
}

// TestListMyTeams_IsExtended verifies that a user only in kader_extended_members gets isExtended=true.
func TestListMyTeams_IsExtended(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Damen 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	h := teams.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/teams", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var teamList []map[string]any
	json.NewDecoder(res.Body).Decode(&teamList)
	res.Body.Close()

	if len(teamList) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teamList))
	}
	if teamList[0]["isExtended"] != true {
		t.Errorf("expected isExtended=true for extended kader member, got %v", teamList[0]["isExtended"])
	}
}

// TestListMyTeams_IsNotExtended verifies that a primary kader member gets isExtended=false.
func TestListMyTeams_IsNotExtended(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	h := teams.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/teams", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var teamList []map[string]any
	json.NewDecoder(res.Body).Decode(&teamList)
	res.Body.Close()

	if len(teamList) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teamList))
	}
	if teamList[0]["isExtended"] != false {
		t.Errorf("expected isExtended=false for primary kader member, got %v", teamList[0]["isExtended"])
	}
}

// TestGetRoster_NoExtendedPlayers verifies that extended_players is an empty array when none exist.
func TestGetRoster_NoExtendedPlayers(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	h := teams.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/teams/"+strconv.Itoa(teamID)+"/roster", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var roster map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&roster)
	res.Body.Close()

	var extended []map[string]any
	json.Unmarshal(roster["extended_players"], &extended)
	if len(extended) != 0 {
		t.Errorf("expected empty extended_players, got %d", len(extended))
	}
}
