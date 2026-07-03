package trainings_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// childRSVPResp mirrors the children_rsvp entries in the /api/training-sessions response.
type childRSVPResp struct {
	MemberID int     `json:"member_id"`
	Name     string  `json:"name"`
	RSVP     *string `json:"rsvp"`
}

type sessionWithChildren struct {
	ID           int             `json:"id"`
	ChildrenRSVP []childRSVPResp `json:"children_rsvp"`
}

// TestSessions_ParentExtendedChild_InChildrenRSVP: a parent whose child is only in
// the extended kader sees that child in children_rsvp on the session list.
func TestSessions_ParentExtendedChild_InChildrenRSVP(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-07-15")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddExtendedKaderMember(t, db, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/training-sessions?from=2025-01-01&to=2027-12-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var sessions []sessionWithChildren
	json.NewDecoder(res.Body).Decode(&sessions)
	res.Body.Close()

	var found *sessionWithChildren
	for i := range sessions {
		if sessions[i].ID == sessionID {
			found = &sessions[i]
		}
	}
	if found == nil {
		t.Fatalf("session %d not visible to parent of extended-kader child", sessionID)
	}
	if len(found.ChildrenRSVP) != 1 || found.ChildrenRSVP[0].MemberID != childMemberID {
		t.Fatalf("expected children_rsvp to contain extended child %d, got %+v", childMemberID, found.ChildrenRSVP)
	}
	if found.ChildrenRSVP[0].RSVP != nil {
		t.Errorf("expected rsvp=null (no response yet), got %v", *found.ChildrenRSVP[0].RSVP)
	}
}

// TestSessions_ExtendedChild_NoAutoConfirm: with rsvp_default_players='confirmed'
// (extended stays 'none'), a regular-kader child is auto-confirmed but an
// extended-kader child is not (must respond explicitly).
func TestSessions_ExtendedChild_NoAutoConfirm(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-07-15")
	db.Exec(`UPDATE training_sessions SET rsvp_default_players='confirmed' WHERE id=?`, sessionID)

	parentUserID := testutil.CreateUser(t, db, "standard")
	regularChild := testutil.CreateMember(t, db, 0)
	extendedChild := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, regularChild)
	testutil.AddExtendedKaderMember(t, db, kaderID, extendedChild)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, regularChild)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, extendedChild)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/training-sessions?from=2025-01-01&to=2027-12-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var sessions []sessionWithChildren
	json.NewDecoder(res.Body).Decode(&sessions)
	res.Body.Close()

	byMember := map[int]*string{}
	for i := range sessions {
		if sessions[i].ID != sessionID {
			continue
		}
		for _, c := range sessions[i].ChildrenRSVP {
			byMember[c.MemberID] = c.RSVP
		}
	}
	if rsvp, ok := byMember[regularChild]; !ok || rsvp == nil || *rsvp != "confirmed" {
		t.Errorf("regular child: expected auto-confirmed, got %v (present=%v)", deref(byMember[regularChild]), ok)
	}
	if rsvp, ok := byMember[extendedChild]; !ok || rsvp != nil {
		t.Errorf("extended child: expected rsvp=null (no auto-confirm), got %v (present=%v)", deref(byMember[extendedChild]), ok)
	}
}

// TestSessions_ChildInBothKaders_SingleEntry: a child in both regular and extended
// kader of the same team appears exactly once (regular wins).
func TestSessions_ChildInBothKaders_SingleEntry(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-07-15")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)
	testutil.AddExtendedKaderMember(t, db, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/training-sessions?from=2025-01-01&to=2027-12-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var sessions []sessionWithChildren
	json.NewDecoder(res.Body).Decode(&sessions)
	res.Body.Close()

	var count int
	for i := range sessions {
		if sessions[i].ID != sessionID {
			continue
		}
		for _, c := range sessions[i].ChildrenRSVP {
			if c.MemberID == childMemberID {
				count++
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 children_rsvp entry for member in both kaders, got %d", count)
	}
}

// TestRespond_ParentForExtendedChild_OK: a parent can submit a response for a child
// that is only in the extended kader.
func TestRespond_ParentForExtendedChild_OK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-07-15")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddExtendedKaderMember(t, db, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Post(t, srv,
		fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "confirmed", "member_id": childMemberID})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, childMemberID).Scan(&status); err != nil {
		t.Fatalf("no response record for extended child: %v", err)
	}
	if status != "confirmed" {
		t.Errorf("expected status 'confirmed', got %q", status)
	}
}

func deref(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
