package attendance_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

func testServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	return prodserver.New(t, db)
}

// pastDate / futureDate liefern fix benannte Daten relativ zum Test-„Heute"
// (2026-06-28). Wir verwenden datierte Konstanten statt time.Now(), damit die
// Tests deterministisch bleiben.
const (
	pastDate1    = "2026-04-15"
	pastDate2    = "2026-05-20"
	futureDate1  = "2027-01-15"
	clubFnTrainr = "trainer"
	clubFnSL     = "sportliche_leitung"
)

// makeTrainer registriert einen User als Trainer des Teams für die aktive Saison.
func makeTrainer(t *testing.T, db *sql.DB, teamID, seasonID int) (userID, kaderID int) {
	t.Helper()
	userID = testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		memberID, clubFnTrainr); err != nil {
		t.Fatalf("club function: %v", err)
	}
	kaderID = testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, memberID)
	return userID, kaderID
}

func addKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("kader_members: %v", err)
	}
}

func addExtendedMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("kader_extended_members: %v", err)
	}
}

func recordTrainingAttendance(t *testing.T, db *sql.DB, sessionID, memberID, present int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO training_attendances (training_id, member_id, present) VALUES (?, ?, ?)`,
		sessionID, memberID, present); err != nil {
		t.Fatalf("training_attendances: %v", err)
	}
}

func recordGameAttendance(t *testing.T, db *sql.DB, gameID, memberID, present int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO game_attendances (game_id, member_id, present) VALUES (?, ?, ?)`,
		gameID, memberID, present); err != nil {
		t.Fatalf("game_attendances: %v", err)
	}
}

// recordExcusedTrainingResponse legt einen auto-declined Response mit
// absence_id an, der als ENTSCHULDIGT zählt.
func recordExcusedTrainingResponse(t *testing.T, db *sql.DB, sessionID, memberID, respondedBy int) {
	t.Helper()
	res, err := db.Exec(`INSERT INTO member_absences (member_id, type, start_date, end_date, created_by) VALUES (?, 'vacation', '2026-04-01', '2026-04-30', ?)`,
		memberID, respondedBy)
	if err != nil {
		t.Fatalf("member_absences: %v", err)
	}
	absenceID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, absence_id) VALUES (?, ?, ?, 'declined', ?)`,
		sessionID, memberID, respondedBy, absenceID); err != nil {
		t.Fatalf("training_responses: %v", err)
	}
}

func setSessionCancelled(t *testing.T, db *sql.DB, sessionID int) {
	t.Helper()
	if _, err := db.Exec(`UPDATE training_sessions SET status='cancelled' WHERE id=?`, sessionID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
}

// ---------- TeamStats ----------

func TestGetTeamStats_HappyPath_CountsThreePillars(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, kaderID := makeTrainer(t, db, teamID, seasonID)

	player := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, player)

	ts1 := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)
	ts2 := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate2)
	recordTrainingAttendance(t, db, ts1, player, 1) // present
	recordTrainingAttendance(t, db, ts2, player, 0) // missed

	ts3 := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-25")
	recordExcusedTrainingResponse(t, db, ts3, player, trainerUserID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{clubFnTrainr})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-stats", teamID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	regular := body["regular_members"].([]any)
	if len(regular) != 1 {
		t.Fatalf("expected 1 regular member, got %d", len(regular))
	}
	m := regular[0].(map[string]any)
	if int(m["training_present"].(float64)) != 1 {
		t.Errorf("training_present=1 expected, got %v", m["training_present"])
	}
	if int(m["training_missed"].(float64)) != 1 {
		t.Errorf("training_missed=1 expected, got %v", m["training_missed"])
	}
	if int(m["training_excused"].(float64)) != 1 {
		t.Errorf("training_excused=1 expected, got %v", m["training_excused"])
	}
}

func TestGetTeamStats_CancelledSessionsIgnored(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, kaderID := makeTrainer(t, db, teamID, seasonID)

	player := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, player)
	tsCancelled := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)
	recordTrainingAttendance(t, db, tsCancelled, player, 1)
	setSessionCancelled(t, db, tsCancelled)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{clubFnTrainr})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-stats", teamID), token)
	defer res.Body.Close()
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	m := body["regular_members"].([]any)[0].(map[string]any)
	if int(m["training_present"].(float64)) != 0 {
		t.Errorf("cancelled session must not count, got training_present=%v", m["training_present"])
	}
}

func TestGetTeamStats_ExtendedSeparate(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	_, kaderID := makeTrainer(t, db, teamID, seasonID)
	regular := testutil.CreateMember(t, db, 0)
	extended := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, regular)
	addExtendedMember(t, db, kaderID, extended)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-stats", teamID), token)
	defer res.Body.Close()
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	if len(body["regular_members"].([]any)) != 1 {
		t.Errorf("expected 1 regular member, got %v", body["regular_members"])
	}
	if len(body["extended_members"].([]any)) != 1 {
		t.Errorf("expected 1 extended member, got %v", body["extended_members"])
	}
}

func TestGetTeamStats_DualMemberOnlyRegular(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	_, kaderID := makeTrainer(t, db, teamID, seasonID)
	dual := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, dual)
	addExtendedMember(t, db, kaderID, dual)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-stats", teamID), token)
	defer res.Body.Close()
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	if len(body["regular_members"].([]any)) != 1 {
		t.Errorf("dual member should appear once in regular_members")
	}
	if len(body["extended_members"].([]any)) != 0 {
		t.Errorf("dual member must not appear in extended_members, got %v", body["extended_members"])
	}
}

func TestGetTeamStats_SpielerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	user := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, user, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-stats", teamID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestGetTeamStats_TeamNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	user := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, user, "admin", nil)
	res := testutil.Get(t, srv, "/api/teams/99999/attendance-stats", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

func TestGetTeamStats_Unauthenticated(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	srv := testServer(t, db)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-stats", teamID), "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

// ---------- MemberStats ----------

func TestGetMemberStats_OwnSelf_OK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	user := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, user)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	addKaderMember(t, db, kaderID, memberID)
	ts := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)
	recordTrainingAttendance(t, db, ts, memberID, 1)

	srv := testServer(t, db)
	token := testutil.Token(t, user, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d/attendance-stats", memberID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	counts := body["counts"].(map[string]any)
	if int(counts["training_present"].(float64)) != 1 {
		t.Errorf("training_present=1 expected, got %v", counts["training_present"])
	}
	events := body["events"].([]any)
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestGetMemberStats_ParentOfLinkedChild_OK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	parentUser := testutil.CreateUser(t, db, "standard")
	childMember := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUser, childMember); err != nil {
		t.Fatalf("family_links: %v", err)
	}
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	addKaderMember(t, db, kaderID, childMember)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUser, "standard", nil, true)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d/attendance-stats", childMember), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestGetMemberStats_TrainerOfMembersTeam_OK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, kaderID := makeTrainer(t, db, teamID, seasonID)
	playerMember := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, playerMember)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{clubFnTrainr})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d/attendance-stats", playerMember), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestGetMemberStats_Fremder_403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	someMember := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, someMember)

	strangerUser := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, strangerUser, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d/attendance-stats", someMember), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestGetMemberStats_CancelledEventCategorized(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	user := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, user)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	addKaderMember(t, db, kaderID, memberID)
	tsCancelled := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)
	setSessionCancelled(t, db, tsCancelled)

	srv := testServer(t, db)
	token := testutil.Token(t, user, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d/attendance-stats", memberID), token)
	defer res.Body.Close()
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	events := body["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].(map[string]any)["category"].(string) != "cancelled" {
		t.Errorf("expected category=cancelled, got %v", events[0])
	}
	counts := body["counts"].(map[string]any)
	if int(counts["training_present"].(float64)) != 0 ||
		int(counts["training_missed"].(float64)) != 0 ||
		int(counts["training_excused"].(float64)) != 0 {
		t.Errorf("cancelled must not increase any counter, got %v", counts)
	}
}

func TestGetMemberStats_MemberNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	user := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, user, "admin", nil)
	res := testutil.Get(t, srv, "/api/members/99999/attendance-stats", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

// ---------- TeamOpen ----------

func TestGetTeamOpen_ShowsPastUnrecorded(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, _ := makeTrainer(t, db, teamID, seasonID)
	testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{clubFnTrainr})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-open", teamID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	if len(items) != 1 {
		t.Fatalf("expected 1 open item, got %d", len(items))
	}
}

func TestGetTeamOpen_HidesFutureAndRecorded(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, _ := makeTrainer(t, db, teamID, seasonID)
	playerMember := testutil.CreateMember(t, db, 0)

	// past training mit attendance → nicht in liste
	tsRecorded := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)
	recordTrainingAttendance(t, db, tsRecorded, playerMember, 1)
	// future training → nicht in liste
	testutil.CreateTrainingSession(t, db, teamID, seasonID, futureDate1)
	// past game mit attendance → nicht in liste
	gameRecorded := testutil.CreateGame(t, db, seasonID, teamID, pastDate2)
	recordGameAttendance(t, db, gameRecorded, playerMember, 1)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{clubFnTrainr})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-open", teamID), token)
	defer res.Body.Close()
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	if len(items) != 0 {
		t.Errorf("expected 0 open items, got %d: %v", len(items), items)
	}
}

func TestGetTeamOpen_CancelledNotShown(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, _ := makeTrainer(t, db, teamID, seasonID)
	tsCancelled := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)
	setSessionCancelled(t, db, tsCancelled)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{clubFnTrainr})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-open", teamID), token)
	defer res.Body.Close()
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	if len(items) != 0 {
		t.Errorf("cancelled session must not appear, got %d items", len(items))
	}
}

func TestGetTeamOpen_SpielerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	user := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, user, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-open", teamID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// SportlicheLeitung darf jede Team-Statistik sehen.
func TestGetTeamStats_SportlicheLeitung_OK(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	user := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, user, "standard", []string{clubFnSL})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-stats", teamID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
