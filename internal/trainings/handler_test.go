package trainings_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

func testServer(t *testing.T, h *trainings.Handler) *httptest.Server {
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/training-sessions", h.ListSessions)
		r.Post("/api/training-sessions/{id}/respond", h.Respond)
		r.Get("/api/training-sessions/{id}/attendances", h.GetAttendances)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("trainer", "sportliche_leitung"))
			r.Post("/api/training-series", h.CreateSeries)
			r.Delete("/api/training-series/{id}", h.DeleteSeries)
			r.Post("/api/training-sessions/{id}/attendances", h.SaveAttendances)
			r.Post("/api/training-sessions", h.CreateSession)
			r.Put("/api/training-sessions/{id}", h.UpdateSession)
		})
	})
}

func newHandler(t *testing.T) (*trainings.Handler, *httptest.Server) {
	t.Helper()
	db := testutil.NewDB(t)
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	return h, srv
}

// TestListSessions_FilterByTeam verifies that a trainer only sees sessions for their own team.
func TestListSessions_FilterByTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID)

	testutil.CreateTrainingSession(t, db, teamA, seasonID, "2026-03-10")
	testutil.CreateTrainingSession(t, db, teamB, seasonID, "2026-03-10")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-03-01&to=2026-03-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var sessions []map[string]any
	json.NewDecoder(res.Body).Decode(&sessions)
	res.Body.Close()

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if int(sessions[0]["team_id"].(float64)) != teamA {
		t.Errorf("expected team_id %d, got %v", teamA, sessions[0]["team_id"])
	}
}

// TestListSessions_AdminSeesAll verifies that an admin sees sessions from all teams.
func TestListSessions_AdminSeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	testutil.CreateTrainingSession(t, db, teamA, seasonID, "2026-03-10")
	testutil.CreateTrainingSession(t, db, teamB, seasonID, "2026-03-10")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-03-01&to=2026-03-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var sessions []map[string]any
	json.NewDecoder(res.Body).Decode(&sessions)
	res.Body.Close()

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

// TestListSessions_Unauthenticated verifies that requests without a token are rejected.
func TestListSessions_Unauthenticated(t *testing.T) {
	_, srv := newHandler(t)
	res := testutil.Get(t, srv, "/api/training-sessions", "")
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}

// TestCreateSeries_GeneratesSessions verifies that creating a series generates one session per matching weekday.
func TestCreateSeries_GeneratesSessions(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminUserID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	body := map[string]any{
		"team_id":   teamID,
		"season_id": seasonID,
		"name":      "Dienstags-Training",
		"day_of_week": 1, // Tuesday (0=Mon, 1=Tue, …)
		"start_time":  "18:00",
		"end_time":    "20:00",
		"valid_from":  "2026-01-06", // first Tuesday
		"valid_until": "2026-01-27", // last Tuesday — 4 Tuesdays total
	}
	res := testutil.Post(t, srv, "/api/training-series", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE team_id=? AND season_id=?`,
		teamID, seasonID).Scan(&count)
	if count != 4 {
		t.Errorf("expected 4 sessions, got %d", count)
	}
}

// TestCreateSeries_WrongTeam_Forbidden verifies that a trainer cannot create a series for a team they don't manage.
func TestCreateSeries_WrongTeam_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	body := map[string]any{
		"team_id":     teamB, // team B — no kader access
		"season_id":   seasonID,
		"name":        "Test",
		"day_of_week": 2,
		"start_time":  "18:00",
		"end_time":    "20:00",
		"valid_from":  "2026-01-06",
		"valid_until": "2026-01-27",
	}
	res := testutil.Post(t, srv, "/api/training-series", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// TestRespond_SavesRSVP verifies that a spieler can submit an RSVP for a session.
func TestRespond_SavesRSVP(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-01")

	spielerUserID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, spielerUserID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, spielerUserID, "standard", []string{"spieler"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var status string
	err := db.QueryRow(`SELECT status FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, memberID).Scan(&status)
	if err != nil {
		t.Fatalf("no RSVP record found: %v", err)
	}
	if status != "confirmed" {
		t.Errorf("expected status 'confirmed', got %q", status)
	}
}

// TestRespond_UpdatesExistingRSVP verifies that a second RSVP overwrites the first without creating a duplicate.
func TestRespond_UpdatesExistingRSVP(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-01")

	spielerUserID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, spielerUserID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, spielerUserID, "standard", []string{"spieler"})
	path := fmt.Sprintf("/api/training-sessions/%d/respond", sessionID)

	r1 := testutil.Post(t, srv, path, token, map[string]any{"status": "confirmed"})
	r1.Body.Close()
	r2 := testutil.Post(t, srv, path, token, map[string]any{"status": "declined"})
	r2.Body.Close()

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, memberID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 RSVP record, got %d", count)
	}

	var status string
	db.QueryRow(`SELECT status FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, memberID).Scan(&status)
	if status != "declined" {
		t.Errorf("expected status 'declined', got %q", status)
	}
}

// TestSaveAttendances_TrainerOK verifies that an admin can save attendances for a past session.
func TestSaveAttendances_TrainerOK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2025-01-10") // past date

	adminUserID := testutil.CreateUser(t, db, "admin")
	memberID := testutil.CreateMember(t, db, adminUserID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	body := []map[string]any{{"member_id": memberID, "present": true}}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", res.StatusCode)
	}
}

// TestSaveAttendances_PlayerForbidden verifies that a user without trainer club function cannot save attendances.
func TestSaveAttendances_PlayerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2025-01-10")

	spielerUserID := testutil.CreateUser(t, db, "standard")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, spielerUserID, "standard", nil)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token,
		[]map[string]any{})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// TC-T-EXT01: GetAttendances returns saved attendance records.
func TestGetAttendances_ReadsBack(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2025-10-10")

	adminUserID := testutil.CreateUser(t, db, "admin")
	memberID := testutil.CreateMember(t, db, adminUserID)

	// Link member to team via kader (player_memberships is a view over kader_members).
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	// Save attendance: present=true.
	saveRes := testutil.Post(t, srv,
		fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token,
		[]map[string]any{{"member_id": memberID, "present": true}})
	saveRes.Body.Close()
	if saveRes.StatusCode != http.StatusNoContent {
		t.Fatalf("save attendances: expected 204, got %d", saveRes.StatusCode)
	}

	// Read back.
	getRes := testutil.Get(t, srv,
		fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get attendances: expected 200, got %d", getRes.StatusCode)
	}
	var items []struct {
		MemberID int   `json:"member_id"`
		Present  *bool `json:"present"` // nullable pointer
	}
	json.NewDecoder(getRes.Body).Decode(&items)
	getRes.Body.Close()

	var found bool
	for _, a := range items {
		if a.MemberID == memberID && a.Present != nil && *a.Present {
			found = true
		}
	}
	if !found {
		t.Errorf("expected attendance for member %d with present=true (got %d items)", memberID, len(items))
	}
}

// TC-T-EXT02: Elternteil antwortet für verknüpftes Kind.
func TestRespond_ParentForChild(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-01")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	// Issue a token with role=elternteil and isParent=true.
	tok, err := auth.IssueAccessToken(testutil.TestJWTSecret, parentUserID, "parent@test.local", "elternteil", nil, true)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	token := "Bearer " + tok

	res := testutil.Post(t, srv,
		fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "confirmed", "member_id": childMemberID})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var status string
	err = db.QueryRow(`SELECT status FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, childMemberID).Scan(&status)
	if err != nil {
		t.Fatalf("no response record found: %v", err)
	}
	if status != "confirmed" {
		t.Errorf("expected status 'confirmed', got %q", status)
	}
}

// ── CreateSession / UpdateSession / DeleteSeries ──────────────────────────────

// TC: Admin legt Einzelsitzung an → 201, Session in DB.
func TestCreateSession_AdminOK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminUserID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	body := map[string]any{
		"team_id":    teamID,
		"season_id":  seasonID,
		"title":      "Zusatztraining",
		"date":       "2026-08-05",
		"start_time": "18:00",
		"end_time":   "20:00",
	}
	res := testutil.Post(t, srv, "/api/training-sessions", token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE team_id=? AND date='2026-08-05'`, teamID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 session in DB, got %d", count)
	}
}

// TC: UpdateSession ändert start_time in DB.
func TestUpdateSession_ChangesTime(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-08-05")
	adminUserID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	body := map[string]any{
		"team_id":    teamID,
		"season_id":  seasonID,
		"title":      "Geändertes Training",
		"date":       "2026-08-05",
		"start_time": "19:00",
		"end_time":   "21:00",
	}
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/training-sessions/%d", sessionID), token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var startTime string
	db.QueryRow(`SELECT start_time FROM training_sessions WHERE id=?`, sessionID).Scan(&startTime)
	if startTime != "19:00" {
		t.Errorf("expected start_time='19:00', got %q", startTime)
	}
}

// TC: DeleteSeries mit scope=all löscht Serie + Sessions + Responses kaskadierend.
func TestDeleteSeries_CascadesSessionsAndResponses(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminUserID := testutil.CreateUser(t, db, "admin")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminUserID)

	// Drei Sessions zur Serie verknüpfen.
	for _, date := range []string{"2026-09-01", "2026-09-08", "2026-09-15"} {
		db.Exec(`INSERT INTO training_sessions (team_id, season_id, date, start_time, end_time, title, series_id)
		         VALUES (?, ?, ?, '18:00', '20:00', 'Test', ?)`,
			teamID, seasonID, date, seriesID)
	}
	var sessionCount int
	db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE series_id=?`, seriesID).Scan(&sessionCount)
	if sessionCount != 3 {
		t.Fatalf("setup: expected 3 sessions, got %d", sessionCount)
	}

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Do(t, srv, http.MethodDelete,
		fmt.Sprintf("/api/training-series/%d?scope=all", seriesID), token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var seriesRemaining, sessionsRemaining int
	db.QueryRow(`SELECT COUNT(*) FROM training_series WHERE id=?`, seriesID).Scan(&seriesRemaining)
	db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE series_id=?`, seriesID).Scan(&sessionsRemaining)
	if seriesRemaining != 0 {
		t.Error("training_series should be deleted")
	}
	if sessionsRemaining != 0 {
		t.Errorf("all training_sessions should be deleted, got %d remaining", sessionsRemaining)
	}
}

// TestListSessions_ExtendedKaderPlayerSeesTeam verifies that a player in the extended kader
// can see training sessions of their team.
func TestListSessions_ExtendedKaderPlayerSeesTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	otherTeamID := testutil.CreateTeam(t, db, "Team B")

	playerUserID := testutil.CreateUser(t, db, "standard")
	playerMemberID := testutil.CreateMember(t, db, playerUserID)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddExtendedKaderMember(t, db, kaderID, playerMemberID)

	testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")
	testutil.CreateTrainingSession(t, db, otherTeamID, seasonID, "2026-09-10")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, playerUserID, "standard", nil)

	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-09-01&to=2026-09-30", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var sessions []map[string]any
	json.NewDecoder(res.Body).Decode(&sessions)
	res.Body.Close()

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session for extended kader player, got %d", len(sessions))
	}
	if int(sessions[0]["team_id"].(float64)) != teamID {
		t.Errorf("expected team_id %d, got %v", teamID, sessions[0]["team_id"])
	}
}

// TestGetAttendances_ExtendedKaderPlayerAppears verifies that a player in the extended kader
// appears in the attendance list.
func TestGetAttendances_ExtendedKaderPlayerAppears(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")

	adminUserID := testutil.CreateUser(t, db, "admin")
	extPlayerUserID := testutil.CreateUser(t, db, "standard")
	extPlayerMemberID := testutil.CreateMember(t, db, extPlayerUserID)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddExtendedKaderMember(t, db, kaderID, extPlayerMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	found := false
	for _, item := range items {
		if int(item["member_id"].(float64)) == extPlayerMemberID {
			found = true
		}
	}
	if !found {
		t.Errorf("expected extended kader member %d in attendance list (got %d items)", extPlayerMemberID, len(items))
	}
}

// TestGetAttendances_IsExtended verifies that is_extended is set correctly:
// primary kader → false, extended-only → true, overlap → appears once with false.
func TestGetAttendances_IsExtended(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")

	adminUserID := testutil.CreateUser(t, db, "admin")
	primaryMemberID := testutil.CreateMember(t, db, 0)
	extOnlyMemberID := testutil.CreateMember(t, db, 0)
	bothMemberID := testutil.CreateMember(t, db, 0)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	// primary kader members
	db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, primaryMemberID)
	db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, bothMemberID)
	// extended kader members
	testutil.AddExtendedKaderMember(t, db, kaderID, extOnlyMemberID)
	testutil.AddExtendedKaderMember(t, db, kaderID, bothMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []struct {
		MemberID   int  `json:"member_id"`
		IsExtended bool `json:"is_extended"`
	}
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	byID := map[int]bool{}
	countByID := map[int]int{}
	for _, item := range items {
		byID[item.MemberID] = item.IsExtended
		countByID[item.MemberID]++
	}

	if byID[primaryMemberID] != false {
		t.Errorf("primary kader member should have is_extended=false")
	}
	if byID[extOnlyMemberID] != true {
		t.Errorf("extended-only member should have is_extended=true")
	}
	if byID[bothMemberID] != false {
		t.Errorf("member in both kadere should have is_extended=false (primary wins)")
	}
	if countByID[bothMemberID] != 1 {
		t.Errorf("member in both kadere should appear exactly once, got %d", countByID[bothMemberID])
	}
}

// TestListSessions_NoKaderPlayerSeesNothing verifies that a player without any kader
// membership cannot see any training sessions.
func TestListSessions_NoKaderPlayerSeesNothing(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	playerUserID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, playerUserID)

	testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, playerUserID, "standard", nil)

	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-09-01&to=2026-09-30", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var sessions []map[string]any
	json.NewDecoder(res.Body).Decode(&sessions)
	res.Body.Close()

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for player without kader, got %d", len(sessions))
	}
}
