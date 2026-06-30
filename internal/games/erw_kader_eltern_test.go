package games_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type gameChildRSVP struct {
	MemberID int     `json:"member_id"`
	Name     string  `json:"name"`
	RSVP     *string `json:"rsvp"`
}

type gameWithChildren struct {
	ID           int             `json:"id"`
	ChildrenRSVP []gameChildRSVP `json:"children_rsvp"`
}

// TestMyGames_ParentExtendedChild_InChildrenRSVP: a parent whose child is only in
// the extended kader sees that child in children_rsvp on /api/games/my.
func TestMyGames_ParentExtendedChild_InChildrenRSVP(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddExtendedKaderMember(t, db, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var gamesList []gameWithChildren
	json.NewDecoder(res.Body).Decode(&gamesList)
	res.Body.Close()

	var found *gameWithChildren
	for i := range gamesList {
		if gamesList[i].ID == gameID {
			found = &gamesList[i]
		}
	}
	if found == nil {
		t.Fatalf("game %d not visible to parent of extended-kader child", gameID)
	}
	if len(found.ChildrenRSVP) != 1 || found.ChildrenRSVP[0].MemberID != childMemberID {
		t.Fatalf("expected children_rsvp to contain extended child %d, got %+v", childMemberID, found.ChildrenRSVP)
	}
	if found.ChildrenRSVP[0].RSVP != nil {
		t.Errorf("expected rsvp=null (no response yet), got %v", *found.ChildrenRSVP[0].RSVP)
	}
}

// TestMyGames_ExtendedChild_NoAutoConfirm: under rsvp_opt_out a regular-kader child is
// auto-confirmed in children_rsvp but an extended-kader child is not.
func TestMyGames_ExtendedChild_NoAutoConfirm(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")
	db.Exec(`UPDATE games SET rsvp_opt_out=1 WHERE id=?`, gameID)

	parentUserID := testutil.CreateUser(t, db, "standard")
	regularChild := testutil.CreateMember(t, db, 0)
	extendedChild := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, regularChild)
	testutil.AddExtendedKaderMember(t, db, kaderID, extendedChild)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, regularChild)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, extendedChild)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var gamesList []gameWithChildren
	json.NewDecoder(res.Body).Decode(&gamesList)
	res.Body.Close()

	byMember := map[int]*string{}
	seen := map[int]bool{}
	for i := range gamesList {
		if gamesList[i].ID != gameID {
			continue
		}
		for _, c := range gamesList[i].ChildrenRSVP {
			byMember[c.MemberID] = c.RSVP
			seen[c.MemberID] = true
		}
	}
	if rsvp := byMember[regularChild]; !seen[regularChild] || rsvp == nil || *rsvp != "confirmed" {
		t.Errorf("regular child: expected auto-confirmed, got %v (present=%v)", gderef(byMember[regularChild]), seen[regularChild])
	}
	if rsvp := byMember[extendedChild]; !seen[extendedChild] || rsvp != nil {
		t.Errorf("extended child: expected rsvp=null (no auto-confirm), got %v (present=%v)", gderef(byMember[extendedChild]), seen[extendedChild])
	}
}

// TestGameRespond_ParentForExtendedChild_OK: a parent can submit a game response for a
// child that is only in the extended kader.
func TestGameRespond_ParentForExtendedChild_OK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddExtendedKaderMember(t, db, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Post(t, srv,
		fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "confirmed", "member_id": childMemberID})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM game_responses WHERE game_id=? AND member_id=?`,
		gameID, childMemberID).Scan(&status); err != nil {
		t.Fatalf("no game response record for extended child: %v", err)
	}
	if status != "confirmed" {
		t.Errorf("expected status 'confirmed', got %q", status)
	}
}

// TestTeams_ParentExtendedChild_TeamListed: GET /api/teams returns the team of an
// extended-only child for its parent (team filter on /termine).
func TestTeams_ParentExtendedChild_TeamListed(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddExtendedKaderMember(t, db, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/teams", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var teams []map[string]any
	json.NewDecoder(res.Body).Decode(&teams)
	res.Body.Close()

	if !containsTeam(teams, teamID) {
		t.Errorf("expected team %d in /api/teams for parent of extended-kader child, got %+v", teamID, teams)
	}
}

// TestTeams_ParentNoKader_TeamNotListed: a parent whose child is in no kader does not
// see the team (no over-visibility from the user_accessible_teams switch).
func TestTeams_ParentNoKader_TeamNotListed(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	testutil.CreateKader(t, db, teamID, seasonID) // team has a kader, but child is not in it

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/teams", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var teams []map[string]any
	json.NewDecoder(res.Body).Decode(&teams)
	res.Body.Close()

	if containsTeam(teams, teamID) {
		t.Errorf("expected team %d NOT listed for parent without kader bond, got %+v", teamID, teams)
	}
}

func containsTeam(teams []map[string]any, teamID int) bool {
	for _, tm := range teams {
		if id, ok := tm["id"].(float64); ok && int(id) == teamID {
			return true
		}
	}
	return false
}

func gderef(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
