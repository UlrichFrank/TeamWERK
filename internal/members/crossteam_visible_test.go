package members_test

import (
	"database/sql"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func crossTeamVisible(t *testing.T, db *sql.DB, memberID int) int {
	t.Helper()
	var v int
	if err := db.QueryRow(`SELECT cross_team_visible FROM members WHERE id=?`, memberID).Scan(&v); err != nil {
		t.Fatalf("read cross_team_visible: %v", err)
	}
	return v
}

// TestUpdateMember_CrossTeamVisible_EigenesMember — der eingeloggte Nutzer
// kann den Toggle auf seinem eigenen Member direkt setzen (kein Draft, kein
// Vorstand nötig).
func TestUpdateMember_CrossTeamVisible_EigenesMember(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)

	srv := newMembersServer(t, db)
	token := testutil.Token(t, uid, "standard", nil)

	path := "/api/members/" + strconv.Itoa(mid) + "/cross-team-visible"
	res := testutil.Do(t, srv, http.MethodPut, path, token, map[string]any{"cross_team_visible": true})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := crossTeamVisible(t, db, mid); got != 1 {
		t.Errorf("cross_team_visible: expected 1, got %d", got)
	}

	// Zurücksetzen.
	res2 := testutil.Do(t, srv, http.MethodPut, path, token, map[string]any{"cross_team_visible": false})
	res2.Body.Close()
	if got := crossTeamVisible(t, db, mid); got != 0 {
		t.Errorf("after unset: expected 0, got %d", got)
	}
}

// TestUpdateMember_CrossTeamVisible_EigenesKindAlsElternteil — Elternteil
// (ohne eigenes Member) kann den Toggle für sein Kind setzen.
func TestUpdateMember_CrossTeamVisible_EigenesKindAlsElternteil(t *testing.T) {
	db := testutil.NewDB(t)
	parentUID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUID, childMemberID); err != nil {
		t.Fatalf("family_links: %v", err)
	}

	srv := newMembersServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUID, "standard", nil, true)

	path := "/api/members/" + strconv.Itoa(childMemberID) + "/cross-team-visible"
	res := testutil.Do(t, srv, http.MethodPut, path, token, map[string]any{"cross_team_visible": true})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := crossTeamVisible(t, db, childMemberID); got != 1 {
		t.Errorf("child cross_team_visible: expected 1, got %d", got)
	}
}

// TestUpdateMember_CrossTeamVisible_Fremd_403 — ein Standard-Nutzer ohne
// Eigentumsverhältnis (weder eigenes Member noch Elternteil) bekommt 403.
func TestUpdateMember_CrossTeamVisible_Fremd_403(t *testing.T) {
	db := testutil.NewDB(t)
	// Opfer-Member, das niemandem im Caller-Sinne gehört.
	victimUID := testutil.CreateUser(t, db, "standard")
	victimMID := testutil.CreateMember(t, db, victimUID)

	// Caller hat keinen Bezug zu victimMID.
	callerUID := testutil.CreateUser(t, db, "standard")

	srv := newMembersServer(t, db)
	token := testutil.Token(t, callerUID, "standard", nil)

	path := "/api/members/" + strconv.Itoa(victimMID) + "/cross-team-visible"
	res := testutil.Do(t, srv, http.MethodPut, path, token, map[string]any{"cross_team_visible": true})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	if got := crossTeamVisible(t, db, victimMID); got != 0 {
		t.Errorf("Fremd-Setzen darf nicht durchgehen: expected 0, got %d", got)
	}
}
