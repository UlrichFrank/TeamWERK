package games_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestListMyGames_MyReason_Populated_When_RespondedWithReason: eigene RSVP mit
// Grund → my_reason im /api/games/my-Response gesetzt.
func TestListMyGames_MyReason_Populated_When_RespondedWithReason(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, 'declined', 'Arbeit bis 20h', CURRENT_TIMESTAMP)`,
		gameID, memberID, userID)

	srv := testServer(t, db)
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var list []map[string]any
	json.NewDecoder(res.Body).Decode(&list)
	res.Body.Close()
	if len(list) != 1 {
		t.Fatalf("expected 1 game, got %d", len(list))
	}
	if list[0]["my_reason"] != "Arbeit bis 20h" {
		t.Errorf("expected my_reason=\"Arbeit bis 20h\", got %v", list[0]["my_reason"])
	}
}

// TestListMyGames_MyReason_Absent_When_DefaultRsvp: nur Default-RSVP, keine
// explizite Antwort → my_reason fehlt (omitempty), auch wenn eine leere Reason-
// Row existierte.
func TestListMyGames_MyReason_Absent_When_DefaultRsvp(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")
	db.Exec(`UPDATE games SET rsvp_default_players='confirmed' WHERE id=?`, gameID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	srv := testServer(t, db)
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var list []map[string]any
	json.NewDecoder(res.Body).Decode(&list)
	res.Body.Close()
	if len(list) != 1 {
		t.Fatalf("expected 1 game, got %d", len(list))
	}
	if list[0]["my_rsvp"] != "confirmed" {
		t.Errorf("expected my_rsvp=confirmed (default), got %v", list[0]["my_rsvp"])
	}
	if _, present := list[0]["my_reason"]; present {
		t.Errorf("expected my_reason absent for default RSVP, got %v", list[0]["my_reason"])
	}
}

// TestListMyGames_ChildrenReason_ForParent: Kind hat mit Grund abgesagt →
// children_rsvp[i].reason im Elternteil-Response gesetzt.
func TestListMyGames_ChildrenReason_ForParent(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, 'declined', 'Krank', CURRENT_TIMESTAMP)`,
		gameID, childMemberID, parentUserID)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)
	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var list []gameWithChildren
	json.NewDecoder(res.Body).Decode(&list)
	res.Body.Close()

	var found *gameWithChildren
	for i := range list {
		if list[i].ID == gameID {
			found = &list[i]
		}
	}
	if found == nil {
		t.Fatalf("game %d not visible to parent", gameID)
	}
	if len(found.ChildrenRSVP) != 1 {
		t.Fatalf("expected 1 child rsvp, got %d", len(found.ChildrenRSVP))
	}
	// Access via raw map because gameChildRSVP was defined without Reason field.
	// Re-decode into a broader type just to check the reason field.
	var listRaw []map[string]any
	res2 := testutil.Get(t, srv, "/api/games/my", token)
	json.NewDecoder(res2.Body).Decode(&listRaw)
	res2.Body.Close()
	children, _ := listRaw[0]["children_rsvp"].([]any)
	if len(children) != 1 {
		t.Fatalf("expected 1 child in raw list, got %d", len(children))
	}
	child := children[0].(map[string]any)
	if child["reason"] != "Krank" {
		t.Errorf("expected child reason=\"Krank\", got %v", child["reason"])
	}
}

// TestListMyGames_ChildrenReason_OmittedWhenEmpty: Kind ohne explizite Antwort
// oder ohne Grund → reason-Feld fehlt im JSON.
func TestListMyGames_ChildrenReason_OmittedWhenEmpty(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)
	// Kein game_responses-Eintrag → Kind ohne explizite Antwort.

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)
	res := testutil.Get(t, srv, "/api/games/my", token)
	var listRaw []map[string]any
	json.NewDecoder(res.Body).Decode(&listRaw)
	res.Body.Close()

	if len(listRaw) != 1 {
		t.Fatalf("expected 1 game, got %d", len(listRaw))
	}
	children, _ := listRaw[0]["children_rsvp"].([]any)
	if len(children) != 1 {
		t.Fatalf("expected 1 child in list, got %d", len(children))
	}
	child := children[0].(map[string]any)
	if _, present := child["reason"]; present {
		t.Errorf("expected reason absent for child without explicit response, got %v", child["reason"])
	}
	_ = gameID
}

// TestGetGameAttendances_Reason_Trainer_SeesAll: Trainer sieht alle Reasons.
// (Der Endpoint ist von Haus aus nur für admin/sportliche_leitung/Trainer der
// beteiligten Teams erreichbar — canRecordGameAttendance —, deshalb ist dies
// der einzig realistisch erreichbare Positiv-Fall.)
func TestGetGameAttendances_Reason_Trainer_SeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	// Trainer, der über trainer_memberships mit der aktiven Saison verlinkt ist.
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)
	db.Exec(`INSERT INTO trainer_memberships (member_id, team_id, season_id) VALUES (?, ?, ?)`,
		trainerMemberID, teamID, seasonID)

	// Zwei fremde Mitglieder mit Reason.
	other1 := testutil.CreateMember(t, db, 0)
	other2 := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other1)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other2)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'Reason1', CURRENT_TIMESTAMP)`, gameID, other1)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'Reason2', CURRENT_TIMESTAMP)`, gameID, other2)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	url := "/api/games/" + itoa(gameID) + "/attendances"
	res := testutil.Get(t, srv, url, token)
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

// TestGetGameAttendances_Reason_SportlicheLeitung_HidesForeignReason:
// Regressionstest gegen den historischen Leak. Ein sportliche_leitung-Nutzer
// (kein trainer, kein admin, keine eigene Zeile) erreicht den Endpoint über
// canRecordGameAttendance, darf aber fremde Reasons nicht sehen.
func TestGetGameAttendances_Reason_SportlicheLeitung_HidesForeignReason(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	slUserID := testutil.CreateUser(t, db, "standard")

	other := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'FremdReason', CURRENT_TIMESTAMP)`, gameID, other)

	srv := testServer(t, db)
	token := testutil.Token(t, slUserID, "standard", []string{"sportliche_leitung"})
	url := "/api/games/" + itoa(gameID) + "/attendances"
	res := testutil.Get(t, srv, url, token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	for _, it := range items {
		mid := int(it["member_id"].(float64))
		if mid == other && it["reason"] != nil {
			t.Errorf("sportliche_leitung (not trainer, not own, not parent) should NOT see foreign reason, got %v", it["reason"])
		}
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// --- GetParticipants reason-Sichtbarkeit ---
//
// Diese Tests decken die 2026-07 behobene Regression ab: GetParticipants lieferte
// reason=null für alle Teilnehmer, weil gr.reason im SQL fehlte.

// TestGetParticipants_Reason_Trainer_SeesAll: Trainer sieht die Absagegründe
// aller Kader-Mitglieder im participants-Endpoint.
func TestGetParticipants_Reason_Trainer_SeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddClubFunction(t, db, trainerMemberID, "trainer")

	other1 := testutil.CreateMember(t, db, 0)
	other2 := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other1)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other2)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'Urlaub', CURRENT_TIMESTAMP)`, gameID, other1)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'Krank', CURRENT_TIMESTAMP)`, gameID, other2)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/games/"+itoa(gameID)+"/participants", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()

	reasonsSeen := map[string]bool{}
	for _, p := range resp.Items {
		if p["reason"] != nil {
			reasonsSeen[p["reason"].(string)] = true
		}
	}
	if !reasonsSeen["Urlaub"] || !reasonsSeen["Krank"] {
		t.Errorf("trainer should see all reasons, got %v", reasonsSeen)
	}
}

// TestGetParticipants_Reason_Member_SeesOwnOnly: Regulärer Spieler sieht nur
// seinen eigenen Absagegrund, fremde Reasons bleiben null.
func TestGetParticipants_Reason_Member_SeesOwnOnly(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, 'declined', 'MeineReason', CURRENT_TIMESTAMP)`, gameID, memberID, userID)

	other := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'FremdReason', CURRENT_TIMESTAMP)`, gameID, other)

	srv := testServer(t, db)
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/games/"+itoa(gameID)+"/participants", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()

	var ownReason, foreignReason any
	for _, p := range resp.Items {
		mid := int(p["member_id"].(float64))
		switch mid {
		case memberID:
			ownReason = p["reason"]
		case other:
			foreignReason = p["reason"]
		}
	}
	if ownReason != "MeineReason" {
		t.Errorf("member should see own reason, got %v", ownReason)
	}
	if foreignReason != nil {
		t.Errorf("member should NOT see foreign reason, got %v", foreignReason)
	}
}

// TestGetParticipants_Reason_Parent_SeesChild: Elternteil sieht den Absagegrund
// des verlinkten Kindes, aber keine fremden Reasons.
func TestGetParticipants_Reason_Parent_SeesChild(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, 'declined', 'KindReason', CURRENT_TIMESTAMP)`, gameID, childMemberID, parentUserID)

	other := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, other)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, 1, 'declined', 'FremdReason', CURRENT_TIMESTAMP)`, gameID, other)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)
	res := testutil.Get(t, srv, "/api/games/"+itoa(gameID)+"/participants", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()

	var childReason, foreignReason any
	for _, p := range resp.Items {
		mid := int(p["member_id"].(float64))
		switch mid {
		case childMemberID:
			childReason = p["reason"]
		case other:
			foreignReason = p["reason"]
		}
	}
	if childReason != "KindReason" {
		t.Errorf("parent should see child reason, got %v", childReason)
	}
	if foreignReason != nil {
		t.Errorf("parent should NOT see foreign reason, got %v", foreignReason)
	}
}
