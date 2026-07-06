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

// TestListSessions_MyReason_Populated_When_RespondedWithReason: eigene RSVP mit
// Grund → my_reason im /api/training-sessions-Response gesetzt.
func TestListSessions_MyReason_Populated_When_RespondedWithReason(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-01")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, 'declined', 'Klavierstunde', CURRENT_TIMESTAMP)`,
		sessionID, memberID, userID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-04-01&to=2026-05-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var list []map[string]any
	json.NewDecoder(res.Body).Decode(&list)
	res.Body.Close()
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	if list[0]["my_reason"] != "Klavierstunde" {
		t.Errorf("expected my_reason=\"Klavierstunde\", got %v", list[0]["my_reason"])
	}
}

// TestListSessions_MyReason_Absent_When_DefaultRsvp: nur Default-RSVP → my_reason
// fehlt (omitempty).
func TestListSessions_MyReason_Absent_When_DefaultRsvp(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-01")
	db.Exec(`UPDATE training_sessions SET rsvp_default_players='confirmed' WHERE id=?`, sessionID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-04-01&to=2026-05-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var list []map[string]any
	json.NewDecoder(res.Body).Decode(&list)
	res.Body.Close()
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	if list[0]["my_rsvp"] != "confirmed" {
		t.Errorf("expected my_rsvp=confirmed (default), got %v", list[0]["my_rsvp"])
	}
	if _, present := list[0]["my_reason"]; present {
		t.Errorf("expected my_reason absent for default RSVP, got %v", list[0]["my_reason"])
	}
}

// TestListSessions_ChildrenReason_ForParent: Kind hat mit Grund abgesagt →
// children_rsvp[i].reason im Eltern-Response gesetzt.
func TestListSessions_ChildrenReason_ForParent(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-01")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, 'declined', 'Krank', CURRENT_TIMESTAMP)`,
		sessionID, childMemberID, parentUserID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-04-01&to=2026-05-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var list []map[string]any
	json.NewDecoder(res.Body).Decode(&list)
	res.Body.Close()

	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	children, _ := list[0]["children_rsvp"].([]any)
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	child := children[0].(map[string]any)
	if child["reason"] != "Krank" {
		t.Errorf("expected child reason=\"Krank\", got %v", child["reason"])
	}
}

// TestGetTrainingAttendances_Reason_Trainer_SeesAll: Trainer sieht alle Reasons.
func TestGetTrainingAttendances_Reason_Trainer_SeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-01")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	// Zwei fremde Mitglieder mit Reason.
	other1 := testutil.CreateMember(t, db, 0)
	other2 := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other1)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other2)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'Reason1', CURRENT_TIMESTAMP)`, sessionID, other1)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'Reason2', CURRENT_TIMESTAMP)`, sessionID, other2)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	reasonsSeen := map[string]bool{}
	for _, it := range items {
		if it["reason"] != nil {
			reasonsSeen[it["reason"].(string)] = true
		}
	}
	if !reasonsSeen["Reason1"] || !reasonsSeen["Reason2"] {
		t.Errorf("trainer should see both reasons, got %v", reasonsSeen)
	}
}

// TestGetTrainingAttendances_Reason_Member_SeesOwn: Regulärer Spieler (nicht
// Trainer) sieht nur seine eigene Reason; fremde Reasons sind null.
func TestGetTrainingAttendances_Reason_Member_SeesOwn(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-01")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, 'declined', 'MeineReason', CURRENT_TIMESTAMP)`, sessionID, memberID, userID)

	other := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'FremdReason', CURRENT_TIMESTAMP)`, sessionID, other)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	var ownReason, foreignReason any
	for _, it := range items {
		mid := int(it["member_id"].(float64))
		switch mid {
		case memberID:
			ownReason = it["reason"]
		case other:
			foreignReason = it["reason"]
		}
	}
	if ownReason != "MeineReason" {
		t.Errorf("member should see own reason, got %v", ownReason)
	}
	if foreignReason != nil {
		t.Errorf("member should NOT see foreign reason, got %v", foreignReason)
	}
}

// TestGetTrainingAttendances_Reason_Parent_SeesChild: Elternteil sieht die
// Reason des verlinkten Kindes, aber keine fremden Reasons.
func TestGetTrainingAttendances_Reason_Parent_SeesChild(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-01")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, 'declined', 'KindReason', CURRENT_TIMESTAMP)`,
		sessionID, childMemberID, parentUserID)

	other := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'FremdReason', CURRENT_TIMESTAMP)`, sessionID, other)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	var childReason, foreignReason any
	for _, it := range items {
		mid := int(it["member_id"].(float64))
		switch mid {
		case childMemberID:
			childReason = it["reason"]
		case other:
			foreignReason = it["reason"]
		}
	}
	if childReason != "KindReason" {
		t.Errorf("parent should see child reason, got %v", childReason)
	}
	if foreignReason != nil {
		t.Errorf("parent should NOT see foreign reason, got %v", foreignReason)
	}
}

// TestGetTrainingAttendances_Reason_Foreigner_Hidden: Nutzer mit Team-Access
// (Extended-Kader-Membership) aber kein Trainer, kein Kind-Eintrag, keine eigene
// Response → sieht fremde Reasons als null. `user_accessible_teams` ist eine
// VIEW, deshalb kommt Team-Access nur über echte Kader-Membership zustande.
func TestGetTrainingAttendances_Reason_Foreigner_Hidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-01")

	other := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'FremdReason', CURRENT_TIMESTAMP)`, sessionID, other)

	// Fremder Nutzer ist im Extended-Kader (Team-Access), aber selbst nicht
	// respondiert, kein Trainer, kein Elternteil.
	foreignUserID := testutil.CreateUser(t, db, "standard")
	foreignMemberID := testutil.CreateMember(t, db, foreignUserID)
	testutil.AddExtendedKaderMember(t, db, kaderID, foreignMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, foreignUserID, "standard", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	for _, it := range items {
		mid := int(it["member_id"].(float64))
		if mid == other && it["reason"] != nil {
			t.Errorf("extended-kader foreigner should NOT see foreign reason, got %v", it["reason"])
		}
	}
}
