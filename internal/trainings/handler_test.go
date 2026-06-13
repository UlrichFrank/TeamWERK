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
			r.Post("/api/training-sessions/{id}/attendances", h.SaveAttendances)
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
