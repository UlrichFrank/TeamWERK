package trainings_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

func testServer(t *testing.T, h *trainings.Handler) *httptest.Server {
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/training-sessions", h.ListSessions)
		r.Get("/api/training-sessions/{id}", h.GetSession)
		r.Post("/api/training-sessions/{id}/respond", h.Respond)
		r.Get("/api/training-sessions/{id}/attendances", h.GetAttendances)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("trainer", "sportliche_leitung"))
			r.Post("/api/training-series", h.CreateSeries)
			r.Put("/api/training-series/{id}", h.UpdateSeries)
			r.Delete("/api/training-series/{id}", h.DeleteSeries)
			r.Get("/api/training-series/{id}/unavailabilities", h.ListSeriesUnavailabilities)
			r.Post("/api/training-series/{id}/unavailabilities", h.CreateSeriesUnavailability)
			r.Delete("/api/training-series/{id}/unavailabilities/{uid}", h.DeleteSeriesUnavailability)
			r.Post("/api/training-sessions/{id}/attendances", h.SaveAttendances)
			r.Delete("/api/training-sessions/{id}/attendance-tracking", h.ResetAttendanceTracking)
			r.Post("/api/training-sessions", h.CreateSession)
			r.Put("/api/training-sessions/{id}", h.UpdateSession)
			r.Delete("/api/training-sessions/{id}", h.DeleteSession)
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Put("/api/trainings/{id}/note", h.UpdateTrainingNote)
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

	var sessionsResp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.NewDecoder(res.Body).Decode(&sessionsResp)
	sessions := sessionsResp.Items
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

	var sessionsResp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.NewDecoder(res.Body).Decode(&sessionsResp)
	sessions := sessionsResp.Items
	res.Body.Close()

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

// TestListSessions_VorstandSeesAll verifies that a user with only the vorstand
// club function (no trainer membership) sees sessions from all teams.
func TestListSessions_VorstandSeesAll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	testutil.CreateTrainingSession(t, db, teamA, seasonID, "2026-03-10")
	testutil.CreateTrainingSession(t, db, teamB, seasonID, "2026-03-10")

	userID := testutil.CreateUser(t, db, "standard")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-03-01&to=2026-03-31", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var sessionsResp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.NewDecoder(res.Body).Decode(&sessionsResp)
	sessions := sessionsResp.Items
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
		"team_id":     teamID,
		"season_id":   seasonID,
		"name":        "Dienstags-Training",
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
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-05-31 12:00"))) // vor Cutoff
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
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-05-31 12:00"))) // vor Cutoff
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

// TestSaveAttendances_TrainerInBatch_Skipped verifies the regression fix: a
// bulk save batch containing a trainer roster row alongside a player skips the
// trainer (no training_attendances row) without failing the request, so the
// player's attendance persists and the response is 204.
func TestSaveAttendances_TrainerInBatch_Skipped(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2025-01-10") // past
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// Trainer of the team (kader_trainer, not a player).
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		trainerMemberID, "trainer"); err != nil {
		t.Fatalf("club function: %v", err)
	}
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	// Player (kader member; player_memberships is a view over kader_members).
	playerMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, playerMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	// Batch as the frontend sends it: player + trainer together.
	body := []map[string]any{
		{"member_id": playerMemberID, "present": true},
		{"member_id": trainerMemberID, "present": false},
	}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	// Player attendance persisted.
	var present int
	if err := db.QueryRow(
		`SELECT present FROM training_attendances WHERE training_id=? AND member_id=?`,
		sessionID, playerMemberID).Scan(&present); err != nil {
		t.Fatalf("player row not persisted: %v", err)
	}
	if present != 1 {
		t.Errorf("expected player present=1, got %d", present)
	}
	// Trainer entry skipped — no row written.
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM training_attendances WHERE training_id=? AND member_id=?`,
		sessionID, trainerMemberID).Scan(&n); err != nil {
		t.Fatalf("count trainer rows: %v", err)
	}
	if n != 0 {
		t.Errorf("expected no attendance row for trainer, got %d", n)
	}
}

// TestSaveAttendances_SetsAttendanceTrackedFlag verifies that a successful
// player-upsert flips training_sessions.attendance_tracked from 0 → 1.
func TestSaveAttendances_SetsAttendanceTrackedFlag(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2025-01-10")
	adminUserID := testutil.CreateUser(t, db, "admin")
	memberID := testutil.CreateMember(t, db, adminUserID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	// Vor Save: 0
	var tracked int
	db.QueryRow(`SELECT attendance_tracked FROM training_sessions WHERE id=?`, sessionID).Scan(&tracked)
	if tracked != 0 {
		t.Fatalf("pre-save attendance_tracked expected 0, got %d", tracked)
	}
	body := []map[string]any{{"member_id": memberID, "present": true}}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	db.QueryRow(`SELECT attendance_tracked FROM training_sessions WHERE id=?`, sessionID).Scan(&tracked)
	if tracked != 1 {
		t.Errorf("post-save attendance_tracked expected 1, got %d", tracked)
	}
}

// TestSaveAttendances_TrainerOnlyBatch_DoesNotSetFlag verifies that a batch
// containing only trainer-only entries (all skipped) does NOT flip the flag.
func TestSaveAttendances_TrainerOnlyBatch_DoesNotSetFlag(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2025-01-10")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		trainerMemberID, "trainer"); err != nil {
		t.Fatalf("club function: %v", err)
	}
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	body := []map[string]any{{"member_id": trainerMemberID, "present": false}}
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var tracked int
	db.QueryRow(`SELECT attendance_tracked FROM training_sessions WHERE id=?`, sessionID).Scan(&tracked)
	if tracked != 0 {
		t.Errorf("trainer-only batch must not set attendance_tracked; got %d", tracked)
	}
}

// TestResetAttendanceTracking_ClearsFlagButKeepsRows verifies that the DELETE
// route sets attendance_tracked=0 while leaving training_attendances rows
// untouched (Undo-Semantik: erneuter Save reaktiviert die Rows).
func TestResetAttendanceTracking_ClearsFlagButKeepsRows(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2025-01-10")
	adminUserID := testutil.CreateUser(t, db, "admin")
	memberID := testutil.CreateMember(t, db, adminUserID)
	testutil.RecordTrainingAttendance(t, db, sessionID, memberID, true)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Delete(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendance-tracking", sessionID), token)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var tracked int
	db.QueryRow(`SELECT attendance_tracked FROM training_sessions WHERE id=?`, sessionID).Scan(&tracked)
	if tracked != 0 {
		t.Errorf("attendance_tracked expected 0 after reset, got %d", tracked)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_attendances WHERE training_id=? AND member_id=?`, sessionID, memberID).Scan(&n)
	if n != 1 {
		t.Errorf("row must remain after reset (Undo), got %d", n)
	}
	// Idempotent: nochmal Reset → weiter 204, weiter 0.
	res2 := testutil.Delete(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendance-tracking", sessionID), token)
	res2.Body.Close()
	if res2.StatusCode != http.StatusNoContent {
		t.Errorf("second reset expected 204, got %d", res2.StatusCode)
	}
}

// TestResetAttendanceTracking_UnknownSession_404 verifies 404 for a
// non-existing session ID.
func TestResetAttendanceTracking_UnknownSession_404(t *testing.T) {
	db := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, db, "admin")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Delete(t, srv, "/api/training-sessions/99999/attendance-tracking", token)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

// TestResetAttendanceTracking_ForeignTeam_Forbidden verifies that a trainer
// without access to the session's team gets 403.
func TestResetAttendanceTracking_ForeignTeam_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	sessionID := testutil.CreateTrainingSession(t, db, teamA, seasonID, "2025-01-10")
	// Trainer nur für Team B.
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)
	testutil.AddKaderTrainer(t, db, kaderB, trainerMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Delete(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendance-tracking", sessionID), token)
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
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-05-31 12:00"))) // vor Cutoff
	srv := testServer(t, h)

	// Eltern haben role="standard" und isParent=true (kein eigenes elternteil-Role/Function-Slot).
	tok, err := auth.IssueAccessToken(testutil.TestJWTSecret, parentUserID, "parent@test.local", "standard", nil, true)
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

// TestUpdateSession_RsvpFlagsPersisted verifies that PUT /api/training-sessions/{id}
// with rsvp_default_players and rsvp_require_reason writes both values to DB.
func TestUpdateSession_RsvpFlagsPersisted(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-08-05")
	adminUserID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	body := map[string]any{
		"team_id":               teamID,
		"season_id":             seasonID,
		"title":                 "Test",
		"date":                  "2026-08-05",
		"start_time":            "18:00",
		"end_time":              "20:00",
		"rsvp_default_players":  "confirmed",
		"rsvp_default_extended": "declined",
		"rsvp_require_reason":   0,
	}
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/training-sessions/%d", sessionID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var defPlayers, defExtended string
	var reqReason int
	if err := db.QueryRow(`SELECT rsvp_default_players, rsvp_default_extended, rsvp_require_reason FROM training_sessions WHERE id=?`, sessionID).
		Scan(&defPlayers, &defExtended, &reqReason); err != nil {
		t.Fatalf("query rsvp flags: %v", err)
	}
	if defPlayers != "confirmed" || defExtended != "declined" || reqReason != 0 {
		t.Errorf("expected (confirmed,declined,0), got (%s,%s,%d)", defPlayers, defExtended, reqReason)
	}
}

// TestUpdateSession_RsvpFlagsPartialUpdate verifies that PUT without the rsvp_* fields
// leaves the existing DB values untouched.
func TestUpdateSession_RsvpFlagsPartialUpdate(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-08-05")

	if _, err := db.Exec(`UPDATE training_sessions SET rsvp_default_players='confirmed', rsvp_require_reason=0 WHERE id=?`, sessionID); err != nil {
		t.Fatalf("seed rsvp flags: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	body := map[string]any{
		"team_id":    teamID,
		"season_id":  seasonID,
		"title":      "Test",
		"date":       "2026-08-05",
		"start_time": "19:00",
		"end_time":   "21:00",
	}
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/training-sessions/%d", sessionID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var defPlayers string
	var reqReason int
	if err := db.QueryRow(`SELECT rsvp_default_players, rsvp_require_reason FROM training_sessions WHERE id=?`, sessionID).
		Scan(&defPlayers, &reqReason); err != nil {
		t.Fatalf("query rsvp flags: %v", err)
	}
	if defPlayers != "confirmed" || reqReason != 0 {
		t.Errorf("partial update must preserve flags; expected (confirmed,0), got (%s,%d)", defPlayers, reqReason)
	}
}

// TestUpdateSession_RsvpFlags_PlayerForbidden verifies that a Spieler cannot
// reach UpdateSession at all — the endpoint is gated by trainer/sportliche_leitung.
func TestUpdateSession_RsvpFlags_PlayerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-08-05")

	spielerID := testutil.CreateUser(t, db, "standard")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, spielerID, "standard", []string{"spieler"})

	body := map[string]any{
		"rsvp_default_players": "confirmed",
		"rsvp_require_reason":  0,
	}
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/training-sessions/%d", sessionID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var defPlayers string
	var reqReason int
	if err := db.QueryRow(`SELECT rsvp_default_players, rsvp_require_reason FROM training_sessions WHERE id=?`, sessionID).
		Scan(&defPlayers, &reqReason); err != nil {
		t.Fatalf("query rsvp flags: %v", err)
	}
	if defPlayers != "none" || reqReason != 1 {
		t.Errorf("DB flags must be unchanged on 403; expected (none,1), got (%s,%d)", defPlayers, reqReason)
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
	var sessionsResp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.NewDecoder(res.Body).Decode(&sessionsResp)
	sessions := sessionsResp.Items
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

// TestGetAttendances_OptOut_NotAppliedToExtended verifies that rsvp_default_players='confirmed'
// auto-confirm applies only to primary kader members, never to extended kader members
// (whose default stays 'none').
func TestGetAttendances_OptOut_NotAppliedToExtended(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")
	db.Exec(`UPDATE training_sessions SET rsvp_default_players = 'confirmed' WHERE id = ?`, sessionID)

	adminUserID := testutil.CreateUser(t, db, "admin")
	primaryMemberID := testutil.CreateMember(t, db, 0)
	extMemberID := testutil.CreateMember(t, db, 0)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, primaryMemberID)
	testutil.AddExtendedKaderMember(t, db, kaderID, extMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []struct {
		MemberID   int     `json:"member_id"`
		IsExtended bool    `json:"is_extended"`
		RSVPStatus *string `json:"rsvp_status"`
	}
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	byID := map[int]*string{}
	for _, item := range items {
		rsvp := item.RSVPStatus
		byID[item.MemberID] = rsvp
	}

	if rsvp := byID[primaryMemberID]; rsvp == nil || *rsvp != "confirmed" {
		t.Errorf("primary kader member should be auto-confirmed via rsvp_default_players, got %v", rsvp)
	}
	if rsvp := byID[extMemberID]; rsvp != nil {
		t.Errorf("extended kader member should NOT be auto-confirmed, got %v", *rsvp)
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
	var sessionsResp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.NewDecoder(res.Body).Decode(&sessionsResp)
	sessions := sessionsResp.Items
	res.Body.Close()

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for player without kader, got %d", len(sessions))
	}
}

// --- event-notes: PUT /api/trainings/{id}/note -----------------------------

// trainerWithSession sets up a trainer-user for teamA and a session, returning
// the db, server, trainer token and the session id.
func trainerWithSession(t *testing.T, sessionDate string) (*sql.DB, *httptest.Server, string, int) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID)
	sessionID := testutil.CreateTrainingSession(t, db, teamA, seasonID, sessionDate)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	return db, srv, token, sessionID
}

func TestTrainings_SetNote_TrainerOwnTeam_Returns200(t *testing.T) {
	db, srv, token, sessionID := trainerWithSession(t, "2026-03-10")

	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/trainings/%d/note", sessionID), token,
		map[string]string{"note": "Halle gesperrt, wir joggen am See"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var note string
	db.QueryRow(`SELECT note FROM training_sessions WHERE id=?`, sessionID).Scan(&note)
	if note != "Halle gesperrt, wir joggen am See" {
		t.Errorf("note not persisted, got %q", note)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM pending_event_notes_push
		WHERE ref_type='training' AND ref_id=?
		  AND notify_after > datetime('now','+4 minutes')
		  AND notify_after <= datetime('now','+5 minutes')`, sessionID).Scan(&n)
	if n != 1 {
		t.Errorf("expected pending row with notify_after≈now+5min, got %d", n)
	}
}

func TestTrainings_SetNote_TrainerOtherTeam_Returns403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	// Trainer is bound to team B but edits a team A session.
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)
	testutil.AddKaderTrainer(t, db, kaderB, trainerMemberID)
	sessionID := testutil.CreateTrainingSession(t, db, teamA, seasonID, "2026-03-10")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})

	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/trainings/%d/note", sessionID), token,
		map[string]string{"note": "darf ich nicht"})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var note string
	db.QueryRow(`SELECT note FROM training_sessions WHERE id=?`, sessionID).Scan(&note)
	if note != "" {
		t.Errorf("note should be unchanged, got %q", note)
	}
}

func TestTrainings_SetNote_TooLong_Returns400(t *testing.T) {
	db, srv, token, sessionID := trainerWithSession(t, "2026-03-10")

	long := ""
	for i := 0; i < 201; i++ {
		long += "x"
	}
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/trainings/%d/note", sessionID), token,
		map[string]string{"note": long})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}

	var note string
	db.QueryRow(`SELECT note FROM training_sessions WHERE id=?`, sessionID).Scan(&note)
	if note != "" {
		t.Errorf("note should be unchanged, got %q", note)
	}
}

func TestTrainings_SetNote_SecondEditResetsTimer(t *testing.T) {
	db, srv, token, sessionID := trainerWithSession(t, "2026-03-10")

	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/trainings/%d/note", sessionID), token,
		map[string]string{"note": "erst"})
	res.Body.Close()

	// Simulate the timer having almost elapsed.
	if _, err := db.Exec(`UPDATE pending_event_notes_push
		SET notify_after = datetime('now','-10 minutes')
		WHERE ref_type='training' AND ref_id=?`, sessionID); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	res = testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/trainings/%d/note", sessionID), token,
		map[string]string{"note": "korrigiert"})
	res.Body.Close()

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM pending_event_notes_push
		WHERE ref_type='training' AND ref_id=?
		  AND note_text='korrigiert'
		  AND notify_after > datetime('now','+4 minutes')`, sessionID).Scan(&n)
	if n != 1 {
		t.Errorf("expected notify_after reset to now+5min with new text, got %d", n)
	}
}

func TestTrainings_SetNote_EmptyDeletesPending(t *testing.T) {
	db, srv, token, sessionID := trainerWithSession(t, "2026-03-10")

	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/trainings/%d/note", sessionID), token,
		map[string]string{"note": "vorhanden"})
	res.Body.Close()

	res = testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/trainings/%d/note", sessionID), token,
		map[string]string{"note": ""})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM pending_event_notes_push
		WHERE ref_type='training' AND ref_id=?`, sessionID).Scan(&n)
	if n != 0 {
		t.Errorf("expected pending row deleted on empty note, got %d", n)
	}
}

// ── RSVP-Cutoff (2 h vor Session-Beginn) ──────────────────────────────────────

func berlinTime(t *testing.T, layout, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	tm, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		t.Fatalf("ParseInLocation %s: %v", value, err)
	}
	return tm
}

// fixedNow returns a clock that always returns the given time.
func fixedNow(tm time.Time) func() time.Time { return func() time.Time { return tm } }

func setupCutoffSession(t *testing.T) (db *sql.DB, sessionID, teamID, seasonID int) {
	t.Helper()
	db = testutil.NewDB(t)
	seasonID = testutil.CreateSeason(t, db, "2025/26")
	teamID = testutil.CreateTeam(t, db, "Team A")
	// Fixture: date=2026-06-15, start_time=18:00, Europe/Berlin (Sommerzeit).
	sessionID = testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-15")
	return
}

// Spieler darf 3 Stunden vor Beginn antworten (vor Cutoff).
func TestRespond_Cutoff_PlayerBefore_OK(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	spielerUserID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, spielerUserID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 15:00"))) // T-3h
	srv := testServer(t, h)

	token := testutil.Token(t, spielerUserID, "standard", []string{"spieler"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "declined"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// Spieler sagt 30 min vor Beginn ab → 422 rsvp_locked.
func TestRespond_Cutoff_PlayerAfter_422(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	spielerUserID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, spielerUserID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 17:30"))) // T-30min
	srv := testServer(t, h)

	token := testutil.Token(t, spielerUserID, "standard", []string{"spieler"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "declined"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	if body["error"] != "rsvp_locked" {
		t.Errorf("expected error=rsvp_locked, got %v", body["error"])
	}
	if body["locks_at"] == nil || body["locks_at"] == "" {
		t.Errorf("expected locks_at to be set, got %v", body["locks_at"])
	}

	// Keine Zeile in DB geschrieben.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, memberID).Scan(&n)
	if n != 0 {
		t.Errorf("expected no response row, got %d", n)
	}
}

// Spieler ändert bestehende confirmed-Antwort nach Cutoff → 422, alter Status bleibt.
func TestRespond_Cutoff_PlayerStatusChange_422(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	spielerUserID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, spielerUserID)
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, responded_at)
	         VALUES (?, ?, ?, 'confirmed', CURRENT_TIMESTAMP)`, sessionID, memberID, spielerUserID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 17:30")))
	srv := testServer(t, h)

	token := testutil.Token(t, spielerUserID, "standard", []string{"spieler"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "declined"})
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
	var status string
	db.QueryRow(`SELECT status FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, memberID).Scan(&status)
	if status != "confirmed" {
		t.Errorf("expected status to remain 'confirmed', got %q", status)
	}
}

// Eltern für Kind nach Cutoff → 422.
func TestRespond_Cutoff_ParentAfter_422(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 17:30")))
	srv := testServer(t, h)

	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "declined", "member_id": childMemberID})
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
}

// Trainer nach Cutoff → 204.
func TestRespond_Cutoff_TrainerAfter_OK(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	trainerUserID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, trainerUserID)
	targetMember := testutil.CreateMember(t, db, 0)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 17:30")))
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "declined", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// sportliche_leitung nach Cutoff → 204.
func TestRespond_Cutoff_SportlicheLeitung_OK(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	slUserID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, slUserID)
	targetMember := testutil.CreateMember(t, db, 0)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 17:30")))
	srv := testServer(t, h)

	token := testutil.Token(t, slUserID, "standard", []string{"sportliche_leitung"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "confirmed", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// Vorstand 5 min nach Beginn → 204.
func TestRespond_Cutoff_VorstandAfterStart_OK(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	vUserID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, vUserID)
	targetMember := testutil.CreateMember(t, db, 0)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 18:05"))) // 5 min nach Beginn
	srv := testServer(t, h)

	token := testutil.Token(t, vUserID, "standard", []string{"vorstand"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "declined", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// Admin (Rolle) ohne Funktion nach Cutoff → 204.
func TestRespond_Cutoff_AdminAfter_OK(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	adminUserID := testutil.CreateUser(t, db, "admin")
	testutil.CreateMember(t, db, adminUserID)
	targetMember := testutil.CreateMember(t, db, 0)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 17:30")))
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "confirmed", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// Kassierer ohne weitere Funktion nach Cutoff → 422.
// Kassierer ist weder Staff (kein Cutoff-Override) noch Elternteil/Owner des Zielmitglieds →
// Antwort für ein fremdes Mitglied ist 403 (Ownership-Gate, VOR dem Cutoff). Früher erwartete
// dieser Test 422 — das kodierte das inzwischen behobene Broken-Access-Control-Loch (jeder
// durfte für jedes Mitglied antworten, nur der Cutoff bremste).
func TestRespond_Cutoff_KassiererForeignMember_403(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	kUserID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, kUserID)
	targetMember := testutil.CreateMember(t, db, 0)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 17:30")))
	srv := testServer(t, h)

	token := testutil.Token(t, kUserID, "standard", []string{"kassierer"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "declined", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 (kein Ownership), got %d", res.StatusCode)
	}
}

// Absence-Lock hat Vorrang: 403 vor Cutoff-Check.
func TestRespond_Cutoff_AbsenceLockTakesPrecedence_403(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	spielerUserID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, spielerUserID)

	// Absence anlegen, dann Response mit absence_id setzen (so wie der Server es bei CreateSession tut).
	res, err := db.Exec(`INSERT INTO member_absences (member_id, type, start_date, end_date, created_by)
	                     VALUES (?, 'vacation', '2026-06-14', '2026-06-20', ?)`, memberID, spielerUserID)
	if err != nil {
		t.Fatalf("insert absence: %v", err)
	}
	absID, _ := res.LastInsertId()
	db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, responded_at, absence_id)
	         VALUES (?, ?, ?, 'declined', CURRENT_TIMESTAMP, ?)`, sessionID, memberID, spielerUserID, absID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	// 3 Stunden vor Beginn — Cutoff würde greifen nicht, aber Absence-Lock kommt zuerst.
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-15 15:00")))
	srv := testServer(t, h)

	token := testutil.Token(t, spielerUserID, "standard", []string{"spieler"})
	httpRes := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "confirmed"})
	httpRes.Body.Close()
	if httpRes.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 (absence lock), got %d", httpRes.StatusCode)
	}
}

// DST: Sommerzeit-Session 2026-06-15 18:00 Berlin = 16:00Z; locks_at = 14:00Z.
func TestListSessions_RsvpLocksAt_Summer(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	uID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, uID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, uID, "admin", nil)
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-06-01&to=2026-06-30", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var listResp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.NewDecoder(res.Body).Decode(&listResp)
	items := listResp.Items
	var found map[string]any
	for _, it := range items {
		if int(it["id"].(float64)) == sessionID {
			found = it
			break
		}
	}
	if found == nil {
		t.Fatalf("session %d not in response", sessionID)
	}
	got, _ := found["rsvp_locks_at"].(string)
	if got != "2026-06-15T14:00:00Z" {
		t.Errorf("expected rsvp_locks_at=2026-06-15T14:00:00Z (summer), got %q", got)
	}
}

// DST: Winterzeit-Session 2026-01-15 18:00 Berlin = 17:00Z; locks_at = 15:00Z.
func TestListSessions_RsvpLocksAt_Winter(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-01-15")

	uID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, uID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, uID, "admin", nil)
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-01-31", token)
	defer res.Body.Close()
	var listResp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.NewDecoder(res.Body).Decode(&listResp)
	items := listResp.Items
	var got string
	for _, it := range items {
		if int(it["id"].(float64)) == sessionID {
			got, _ = it["rsvp_locks_at"].(string)
			break
		}
	}
	if got != "2026-01-15T15:00:00Z" {
		t.Errorf("expected rsvp_locks_at=2026-01-15T15:00:00Z (winter), got %q", got)
	}
}

// Detail-Response enthält rsvp_locks_at.
func TestGetSession_RsvpLocksAt(t *testing.T) {
	db, sessionID, _, _ := setupCutoffSession(t)
	uID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, uID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/training-sessions/{id}", h.GetSession)
	})

	token := testutil.Token(t, uID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d", sessionID), token)
	defer res.Body.Close()
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	got, _ := body["rsvp_locks_at"].(string)
	if got != "2026-06-15T14:00:00Z" {
		t.Errorf("expected rsvp_locks_at=2026-06-15T14:00:00Z, got %q", got)
	}
}

// TestSaveAttendances_ForeignTeamTrainerForbidden nagelt die team-scoped Recording-Authz fest
// (Welle 1): ein Trainer von Team A darf keine Anwesenheiten für eine Session von Team B
// speichern — der Router-Gate (RequireClubFunction) lässt jeden Trainer durch, erst
// hasTeamAccess im Handler blockt das fremde Team.
func TestSaveAttendances_ForeignTeamTrainerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	sessionB := testutil.CreateTrainingSession(t, db, teamB, seasonID, "2025-01-10") // past, Team B

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID) // trainer of Team A only

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionB), token,
		[]map[string]any{})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for trainer of foreign team, got %d", res.StatusCode)
	}
}

// TestSaveAttendances_OwnTeamTrainerOK ergänzt den admin-basierten Happy-Path: ein echter,
// dem Team zugeordneter Trainer (nicht admin) darf speichern — beweist, dass hasTeamAccess
// über kader_trainers greift.
func TestSaveAttendances_OwnTeamTrainerOK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	sessionA := testutil.CreateTrainingSession(t, db, teamA, seasonID, "2025-01-10") // past

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionA), token,
		[]map[string]any{})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204 for own-team trainer, got %d", res.StatusCode)
	}
}

// ── DeleteSession — hasTeamAccess (Handler-intern, nicht Tier) ─────────────────

// A1: Trainer des eigenen Teams löscht eine Session → 204; Session verschwindet und
// die pending_event_notes_push-Zeile wird mit aufgeräumt (handler.go DeleteSession).
func TestDeleteSession_OwnTeamTrainer_DeletesAndCleansPending(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID)

	sid := testutil.CreateTrainingSession(t, db, teamA, seasonID, "2025-01-10")

	// Begleitzeilen, die beim Löschen mit weggeräumt werden sollen.
	if _, err := db.Exec(
		`INSERT INTO training_responses (training_id, member_id, responded_by, status, responded_at)
		 VALUES (?, ?, ?, 'confirmed', CURRENT_TIMESTAMP)`, sid, trainerMemberID, trainerUserID); err != nil {
		t.Fatalf("seed training_responses: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO pending_event_notes_push (ref_type, ref_id, note_text, notify_after, updated_by)
		 VALUES ('training', ?, 'x', datetime('now','+5 minutes'), ?)`, sid, trainerUserID); err != nil {
		t.Fatalf("seed pending: %v", err)
	}

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})

	res := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/training-sessions/%d", sid), token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var sessionCount, pendingCount int
	db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE id=?`, sid).Scan(&sessionCount)
	db.QueryRow(`SELECT COUNT(*) FROM pending_event_notes_push WHERE ref_type='training' AND ref_id=?`, sid).Scan(&pendingCount)
	if sessionCount != 0 {
		t.Errorf("session should be deleted, got %d rows", sessionCount)
	}
	if pendingCount != 0 {
		t.Errorf("pending_event_notes_push should be cleaned, got %d rows", pendingCount)
	}
}

// A2: Trainer nur von Team A darf keine Session von Team B löschen → 403; Session bleibt.
func TestDeleteSession_ForeignTeamTrainer_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID) // trainer of Team A only

	sidB := testutil.CreateTrainingSession(t, db, teamB, seasonID, "2025-01-10")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})

	res := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/training-sessions/%d", sidB), token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE id=?`, sidB).Scan(&count)
	if count != 1 {
		t.Errorf("foreign session must remain, got %d rows", count)
	}
}

// A3: DeleteSession auf unbekannte ID → 404 (admin umgeht den Tier-Gate).
func TestDeleteSession_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-sessions/999999", token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

// ── Respond — Member-Auflösung / parentHasChild (Handler-intern) ──────────────

// B1: Elternteil versucht, für ein fremdes (nicht verknüpftes) Mitglied zu
// antworten → 403 (parentHasChild=false, handler.go im case "elternteil").
// Hinweis: der reale Eltern-Persona ist role="standard"+IsParent (personas_test.go),
// der über den default-Zweig läuft; hier wird role="elternteil" gesetzt, um genau
// den parentHasChild-Zweig zu treffen (siehe Bericht/Auffälligkeit).
func TestRespond_ParentForNonChild_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sid := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-01")

	parentUserID := testutil.CreateUser(t, db, "standard")
	ownChild := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, ownChild); err != nil {
		t.Fatalf("family_links: %v", err)
	}
	strangerMember := testutil.CreateMember(t, db, 0) // NO link to parent

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-05-31 12:00"))) // vor Cutoff
	srv := testServer(t, h)

	// Reale Eltern-Persona: role="standard" + IsParent=true (die synthetische Rolle
	// "elternteil" wird nie ausgestellt). Fremdes Mitglied ohne family_link → 403.
	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sid), token,
		map[string]any{"status": "confirmed", "member_id": strangerMember})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_responses WHERE training_id=? AND member_id=?`, sid, strangerMember).Scan(&n)
	if n != 0 {
		t.Errorf("no response row must be written for stranger, got %d", n)
	}
}

// Isolations-Pin: ein ganz gewöhnlicher standard-Nutzer (keine Funktion, kein Elternteil)
// darf die RSVP eines fremden Mitglieds nicht setzen (403). Entkoppelt vom Cutoff-Timing
// (vor Cutoff) und vom Kassierer-Spezialfall — nagelt das Ownership-Gate direkt fest.
func TestRespond_StandardForeignMember_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sid := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-01")

	userID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, userID) // hat eigenes Mitglied, antwortet aber für ein fremdes
	foreign := testutil.CreateMember(t, db, 0)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-05-31 12:00"))) // vor Cutoff
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sid), token,
		map[string]any{"status": "confirmed", "member_id": foreign})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// B4: Account ohne verknüpften Mitglieds-Datensatz, ohne member_id → 422
// (Respond: own-member-Pfad, memberIDForUser == 0).
func TestRespond_SpielerUnlinkedAccount_422(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sid := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-01")

	userID := testutil.CreateUser(t, db, "standard") // KEIN CreateMember → unverknüpft

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-05-31 12:00")))
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sid), token,
		map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
}

// ── GetAttendances / GetSession — Zugriff & NotFound ──────────────────────────

// C2: Ein Nutzer, der weder Trainer-artig ist noch in user_accessible_teams von
// Team A steht, bekommt bei GET /attendances → 403 (handler.go GetAttendances).
func TestGetAttendances_TrueStranger_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	sid := testutil.CreateTrainingSession(t, db, teamA, seasonID, "2026-06-01")

	strangerUserID := testutil.CreateUser(t, db, "standard") // keine Funktion, kein Kader, kein Member

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, strangerUserID, "standard", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sid), token)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// C1: GET auf unbekannte Session → 404 (handler.go GetSession).
func TestGetSession_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, "/api/training-sessions/999999", token)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

// ── CreateSession / UpdateSession — fremdes Team (Handler-intern hasTeamAccess) ─

// D1: Trainer von Team A legt Session für Team B an → 403 (CreateSession).
func TestCreateSession_ForeignTeam_Forbidden(t *testing.T) {
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
		"team_id":    teamB, // fremdes Team
		"season_id":  seasonID,
		"title":      "Zusatztraining",
		"date":       "2026-08-05",
		"start_time": "18:00",
		"end_time":   "20:00",
	}
	res := testutil.Post(t, srv, "/api/training-sessions", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// D4: Trainer von Team A bearbeitet eine Session von Team B → 403 (UpdateSession).
func TestUpdateSession_ForeignTeam_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMemberID)

	sidB := testutil.CreateTrainingSession(t, db, teamB, seasonID, "2026-08-05")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})

	body := map[string]any{
		"team_id":    teamB,
		"season_id":  seasonID,
		"title":      "Geändert",
		"date":       "2026-08-05",
		"start_time": "19:00",
		"end_time":   "21:00",
	}
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/training-sessions/%d", sidB), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}
