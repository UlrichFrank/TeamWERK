package members_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// TestDeleteMember_BroadcastsMembers verifies that deleting a member emits the
// "members" live-update event so the finance group's MembersPage reloads, and
// that a teammate of the deleted player (who saw them on the roster) also gets
// it — the audience is resolved BEFORE the row is removed.
func TestDeleteMember_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)

	// Vorstand performs the delete (finance group).
	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")
	vorstandTok := testutil.Token(t, vorstandU, "standard", []string{"vorstand"})

	// Target player on the team + a teammate who sees them on the roster.
	targetU := testutil.CreateUser(t, db, "standard")
	targetM := testutil.CreateMember(t, db, targetU)
	testutil.AddClubFunction(t, db, targetM, "spieler")
	testutil.AddKaderMember(t, db, kader, targetM)

	teammateU := testutil.CreateUser(t, db, "standard")
	teammateM := testutil.CreateMember(t, db, teammateU)
	testutil.AddClubFunction(t, db, teammateM, "spieler")
	testutil.AddKaderMember(t, db, kader, teammateM)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	teammateCh := sharedHub.SubscribeUser(teammateU)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/members/"+strconv.Itoa(targetM), vorstandTok, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DeleteMember: expected 204, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"vorstand": vorstandCh,
		"teammate": teammateCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "members" {
			t.Errorf("%s stream must receive 'members', got %q ok=%v", name, ev, ok)
		}
	}
}

// TestAddChildPhone_BroadcastsMembers verifies that a parent adding a phone to a
// proxy-account child emits "members", reaching the finance group and the parent
// (extraUserID) — the child's profile/contact data changed.
func TestAddChildPhone_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)

	// Finance group recipient.
	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	// Parent with a proxy-account child.
	parentU := testutil.CreateUser(t, db, "standard")
	childU := testutil.CreateUser(t, db, "standard")
	childM := testutil.CreateMember(t, db, childU)
	testutil.AddFamilyLink(t, db, parentU, childM)
	parentTok := testutil.TokenWithIsParent(t, parentU, "standard", nil, true)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	parentCh := sharedHub.SubscribeUser(parentU)

	res := testutil.Do(t, srv, http.MethodPost,
		"/api/profile/kind/"+strconv.Itoa(childM)+"/phones", parentTok,
		map[string]any{"label": "Mobil", "number": "0170123456", "sort_order": 0})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		t.Fatalf("AddChildPhone: expected 200/201, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"vorstand": vorstandCh,
		"parent":   parentCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "members" {
			t.Errorf("%s stream must receive 'members', got %q ok=%v", name, ev, ok)
		}
	}
}

// TestUpdateVisibility_ReachesTeammate verifies that a self-service visibility
// change emits "members" and reaches a non-finance teammate — visibility governs
// which contact fields the roster shows, so those views must reload. It must NOT
// reach a player on a different team.
func TestUpdateVisibility_ReachesTeammate(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	kaderB := testutil.CreateKader(t, db, teamB, season)

	// The user changing their own visibility, on team A.
	selfU := testutil.CreateUser(t, db, "standard")
	selfM := testutil.CreateMember(t, db, selfU)
	testutil.AddClubFunction(t, db, selfM, "spieler")
	testutil.AddKaderMember(t, db, kaderA, selfM)
	selfTok := testutil.Token(t, selfU, "standard", []string{"spieler"})

	// Teammate on team A — must receive the event.
	teammateU := testutil.CreateUser(t, db, "standard")
	teammateM := testutil.CreateMember(t, db, teammateU)
	testutil.AddClubFunction(t, db, teammateM, "spieler")
	testutil.AddKaderMember(t, db, kaderA, teammateM)

	// Player on team B — must NOT receive team A's member event.
	foreignU := testutil.CreateUser(t, db, "standard")
	foreignM := testutil.CreateMember(t, db, foreignU)
	testutil.AddClubFunction(t, db, foreignM, "spieler")
	testutil.AddKaderMember(t, db, kaderB, foreignM)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	selfCh := sharedHub.SubscribeUser(selfU)
	teammateCh := sharedHub.SubscribeUser(teammateU)
	foreignCh := sharedHub.SubscribeUser(foreignU)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/profile/visibility", selfTok,
		map[string]any{"phones_visible": true, "address_visible": false,
			"photo_visible": true, "email_visible": false, "whatsapp_visible": false})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("UpdateVisibility: expected 204, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"self":     selfCh,
		"teammate": teammateCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "members" {
			t.Errorf("%s stream must receive 'members', got %q ok=%v", name, ev, ok)
		}
	}
	if ev, ok := recvWithin(foreignCh, 300*time.Millisecond); ok {
		t.Errorf("team-foreign player must NOT receive members event, got %q", ev)
	}
}

// TestAcceptChangeRequest_BroadcastsMembers verifies that accepting a member
// change draft emits "members" to the finance group (whose list shows the
// pending-draft flag that just cleared) and to the affected member's own user.
func TestAcceptChangeRequest_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)

	// Vorstand accepts the draft (finance group).
	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")
	vorstandTok := testutil.Token(t, vorstandU, "standard", []string{"vorstand"})

	// Target member with a linked (non-finance) user + a pending "name" draft,
	// created through the proven CreateOrUpdateDraft path so old_value is well-formed.
	targetU := testutil.CreateUser(t, db, "standard")
	targetM := testutil.CreateMember(t, db, targetU)
	dh := members.NewHandler(db, hub.NewHub())
	draft, err := dh.CreateOrUpdateDraft(targetM, targetU, members.ChangeRequest{
		FieldName: "name",
		NewValue:  json.RawMessage(`{"first_name":"New","last_name":"Name"}`),
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateDraft: %v", err)
	}
	draftID := int64(draft.ID)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	vorstandCh := sharedHub.SubscribeUser(vorstandU)
	targetCh := sharedHub.SubscribeUser(targetU)

	res := testutil.Do(t, srv, http.MethodPost,
		"/api/members/"+strconv.Itoa(targetM)+"/change-drafts/"+strconv.FormatInt(draftID, 10)+"/accept",
		vorstandTok, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("AcceptChangeRequest: expected 200, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"vorstand": vorstandCh,
		"target":   targetCh,
	} {
		if ev, ok := recvWithin(ch, time.Second); !ok || ev != "members" {
			t.Errorf("%s stream must receive 'members', got %q ok=%v", name, ev, ok)
		}
	}
}
