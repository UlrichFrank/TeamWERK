package auth_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// recvEvent blocks up to d for an event on ch.
func recvEvent(ch chan string, d time.Duration) (string, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return "", false
	}
}

// newVorstand creates a standard user with a member row and the "vorstand" club
// function so it is part of the FinanceGroup audience that broadcastFinance
// targets. Returns the user ID and a Vorstand bearer token.
func newVorstand(t *testing.T, db *sql.DB) (int, string) {
	t.Helper()
	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	testutil.AddClubFunction(t, db, mid, "vorstand")
	return uid, testutil.Token(t, uid, "standard", []string{"vorstand"})
}

// TestCreateUser_BroadcastsUsers verifies POST /api/users emits the "users"
// live-update event to the finance group so AdminUsersPage refreshes.
func TestCreateUser_BroadcastsUsers(t *testing.T) {
	db := testutil.NewDB(t)
	actorU, token := newVorstand(t, db)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	ch := sharedHub.SubscribeUser(actorU)
	defer sharedHub.UnsubscribeUser(actorU, ch)

	res := testutil.Post(t, srv, "/api/users", token, map[string]string{
		"email":      "neu@test.local",
		"first_name": "Neu",
		"last_name":  "Nutzer",
		"password":   "einSicheresPasswort1",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("CreateUser: expected 201, got %d", res.StatusCode)
	}

	if ev, ok := recvEvent(ch, time.Second); !ok || ev != "users" {
		t.Errorf("expected 'users' event after CreateUser, got %q ok=%v", ev, ok)
	}
}

// TestUpdateUserRole_BroadcastsUsers verifies PUT /api/users/{id}/role emits the
// "users" event and — because the affected user is passed as an extra recipient —
// reaches the target even when they are not in the finance group themselves.
func TestUpdateUserRole_BroadcastsUsersToTarget(t *testing.T) {
	db := testutil.NewDB(t)
	_, token := newVorstand(t, db)
	targetU := testutil.CreateUser(t, db, "standard")

	srv, sharedHub := prodserver.NewWithHub(t, db)
	// Subscribe the affected target: a plain standard user outside the finance
	// group only receives the event because it is passed as extraUserID.
	ch := sharedHub.SubscribeUser(targetU)
	defer sharedHub.UnsubscribeUser(targetU, ch)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/users/"+itoa(targetU)+"/role", token, map[string]string{"role": "presseteam"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("UpdateUserRole: expected 204, got %d", res.StatusCode)
	}

	if ev, ok := recvEvent(ch, time.Second); !ok || ev != "users" {
		t.Errorf("affected user must receive 'users' event after role change, got %q ok=%v", ev, ok)
	}
}

// TestInvite_BroadcastsUsers verifies POST /api/auth/invite emits the "users"
// event so the invitations list in AdminUsersPage refreshes.
func TestInvite_BroadcastsUsers(t *testing.T) {
	db := testutil.NewDB(t)
	actorU, token := newVorstand(t, db)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	ch := sharedHub.SubscribeUser(actorU)
	defer sharedHub.UnsubscribeUser(actorU, ch)

	res := testutil.Post(t, srv, "/api/auth/invite", token, map[string]string{
		"email": "einladung@test.local",
		"role":  "standard",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("Invite: expected 204, got %d", res.StatusCode)
	}

	if ev, ok := recvEvent(ch, time.Second); !ok || ev != "users" {
		t.Errorf("expected 'users' event after Invite, got %q ok=%v", ev, ok)
	}
}

// TestApproveMembershipRequest_BroadcastsUsers verifies the adult approval path
// (POST /api/membership-requests/{id}/approve) emits the "users" event so the
// pending-request and invitations lists refresh.
func TestApproveMembershipRequest_BroadcastsUsers(t *testing.T) {
	db := testutil.NewDB(t)
	actorU, token := newVorstand(t, db)
	reqID := createMembershipRequest(t, db, "Bewerber", "bewerber@test.local")

	srv, sharedHub := prodserver.NewWithHub(t, db)
	ch := sharedHub.SubscribeUser(actorU)
	defer sharedHub.UnsubscribeUser(actorU, ch)

	res := testutil.Post(t, srv,
		"/api/membership-requests/"+itoa(reqID)+"/approve", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("ApproveMembershipRequest: expected 204, got %d", res.StatusCode)
	}

	if ev, ok := recvEvent(ch, time.Second); !ok || ev != "users" {
		t.Errorf("expected 'users' event after ApproveMembershipRequest, got %q ok=%v", ev, ok)
	}
}

// TestLinkInvitationMember_BroadcastsMembers verifies PUT
// /api/invitations/{id}/member emits the "members" event (in addition to
// "users") so member-facing views refresh when an invitation is linked to a
// member. We assert the "members" event via a dedicated subscriber that only
// keeps the members topic (the hub delivers every topic to a subscriber, so we
// drain until we see the expected one).
func TestLinkInvitationMember_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)
	actorU, token := newVorstand(t, db)

	// An unlinked member and an unused invitation to connect them.
	memberOwner := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, memberOwner)
	// Detach the owner so the member is linkable (LinkInvitationMember rejects
	// members that already have a user_id).
	if _, err := db.Exec(`UPDATE members SET user_id = NULL WHERE id = ?`, memberID); err != nil {
		t.Fatalf("detach member owner: %v", err)
	}
	var invID int
	res := db.QueryRow(`SELECT id FROM invitation_tokens LIMIT 1`)
	if err := res.Scan(&invID); err == sql.ErrNoRows {
		r, err := db.Exec(
			`INSERT INTO invitation_tokens (email, role, token, expires_at) VALUES (?,?,?,?)`,
			"linkme@test.local", "standard", "linktokenhash", time.Now().Add(48*time.Hour))
		if err != nil {
			t.Fatalf("insert invitation: %v", err)
		}
		id64, _ := r.LastInsertId()
		invID = int(id64)
	} else if err != nil {
		t.Fatalf("query invitation: %v", err)
	}

	srv, sharedHub := prodserver.NewWithHub(t, db)
	ch := sharedHub.SubscribeUser(actorU)
	defer sharedHub.UnsubscribeUser(actorU, ch)

	resp := testutil.Do(t, srv, http.MethodPut,
		"/api/invitations/"+itoa(invID)+"/member", token, map[string]int{"member_id": memberID})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("LinkInvitationMember: expected 204, got %d", resp.StatusCode)
	}

	// The handler emits both "users" and "members"; drain the actor's stream and
	// require that "members" is among the delivered events.
	sawMembers := false
	deadline := time.After(time.Second)
	for !sawMembers {
		select {
		case ev := <-ch:
			if ev == "members" {
				sawMembers = true
			}
		case <-deadline:
			t.Fatalf("expected 'members' event after LinkInvitationMember")
		}
	}
}
