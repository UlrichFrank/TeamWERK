package trainings_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

type rsvpAttendance struct {
	MemberID      int     `json:"member_id"`
	IsExtended    bool    `json:"is_extended"`
	IsTrainer     bool    `json:"is_trainer"`
	RSVPStatus    *string `json:"rsvp_status"`
	RSVPIsDefault bool    `json:"rsvp_is_default"`
}

// fetchAttendances calls GET /attendances and indexes the result by member ID.
func fetchAttendances(t *testing.T, srv *httptest.Server, sessionID int, token string) map[int]rsvpAttendance {
	t.Helper()
	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("attendances: expected 200, got %d", res.StatusCode)
	}
	var items []rsvpAttendance
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()
	byID := map[int]rsvpAttendance{}
	for _, it := range items {
		byID[it.MemberID] = it
	}
	return byID
}

// 4.1 Happy-Path: rsvp_default_players='declined' → Spieler ohne Response bekommt
// rsvp_status='declined', rsvp_is_default=true.
func TestRsvpDefault_PlayersDeclined_VirtualDeclined(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")
	db.Exec(`UPDATE training_sessions SET rsvp_default_players='declined' WHERE id=?`, sessionID)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	playerID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, playerID)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	token := testutil.Token(t, adminUserID, "admin", nil)

	byID := fetchAttendances(t, srv, sessionID, token)
	got := byID[playerID]
	if got.RSVPStatus == nil || *got.RSVPStatus != "declined" {
		t.Fatalf("expected rsvp_status=declined, got %v", got.RSVPStatus)
	}
	if !got.RSVPIsDefault {
		t.Errorf("expected rsvp_is_default=true for virtual default")
	}
}

// 4.2 Erweiterter Kader unabhängig: players='confirmed', extended='none' →
// Stammkader auto-confirmed (is_default), Erweiterter Kader null.
func TestRsvpDefault_ExtendedIndependent(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")
	db.Exec(`UPDATE training_sessions SET rsvp_default_players='confirmed', rsvp_default_extended='none' WHERE id=?`, sessionID)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	playerID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, playerID)
	extID := testutil.CreateMember(t, db, 0)
	testutil.AddExtendedKaderMember(t, db, kaderID, extID)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	token := testutil.Token(t, adminUserID, "admin", nil)

	byID := fetchAttendances(t, srv, sessionID, token)
	if p := byID[playerID]; p.RSVPStatus == nil || *p.RSVPStatus != "confirmed" || !p.RSVPIsDefault {
		t.Errorf("Stammkader should be confirmed (default), got %v is_default=%v", p.RSVPStatus, p.RSVPIsDefault)
	}
	if e := byID[extID]; e.RSVPStatus != nil {
		t.Errorf("Erweiterter Kader with extended='none' should have null rsvp, got %v", *e.RSVPStatus)
	}
}

// 4.3 Serie → Session-Copy: Serie mit players='declined' anlegen, generierte
// Session erbt die Werte.
func TestRsvpDefault_SeriesCopiesToSessions(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	// Trainer, damit hasTeamAccess greift.
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	srv := testServer(t, trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})

	body := map[string]any{
		"team_id":              teamID,
		"season_id":            seasonID,
		"name":                 "Serie",
		"day_of_week":          2,
		"start_time":           "18:00",
		"end_time":             "20:00",
		"valid_from":           "2026-01-05",
		"valid_until":          "2026-01-20",
		"rsvp_default_players": "declined",
	}
	res := testutil.Do(t, srv, http.MethodPost, "/api/training-series", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("CreateSeries: expected 201, got %d", res.StatusCode)
	}

	var players, extended string
	err := db.QueryRow(`SELECT rsvp_default_players, rsvp_default_extended FROM training_sessions WHERE series_id IS NOT NULL LIMIT 1`).
		Scan(&players, &extended)
	if err != nil {
		t.Fatalf("query generated session: %v", err)
	}
	if players != "declined" || extended != "none" {
		t.Errorf("session should inherit (declined,none), got (%s,%s)", players, extended)
	}
}

// 4.4 Konfliktsperre: PUT mit players='declined' + rsvp_require_reason=1 → HTTP 400,
// keine DB-Änderung.
func TestRsvpDefault_ConflictRejected(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	token := testutil.Token(t, adminUserID, "admin", nil)

	body := map[string]any{
		"team_id":              teamID,
		"season_id":            seasonID,
		"title":                "Test",
		"date":                 "2026-09-10",
		"start_time":           "18:00",
		"end_time":             "20:00",
		"rsvp_default_players": "declined",
		"rsvp_require_reason":  1,
	}
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/training-sessions/%d", sessionID), token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var payload map[string]any
	json.NewDecoder(res.Body).Decode(&payload)
	if payload["error"] != "invalid_rsvp_settings" {
		t.Errorf("expected error=invalid_rsvp_settings, got %v", payload["error"])
	}

	var players string
	db.QueryRow(`SELECT rsvp_default_players FROM training_sessions WHERE id=?`, sessionID).Scan(&players)
	if players != "none" {
		t.Errorf("session must be unchanged on 400, got players=%s", players)
	}
}

// 4.5 Header-Zähler: players='confirmed', 3 Kader-Spieler, 0 Responses →
// confirmed_count=3, declined_count=0.
func TestRsvpDefault_HeaderCount_PlayersConfirmed(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")
	db.Exec(`UPDATE training_sessions SET rsvp_default_players='confirmed' WHERE id=?`, sessionID)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	for i := 0; i < 3; i++ {
		m := testutil.CreateMember(t, db, 0)
		db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, m)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GetSession: expected 200, got %d", res.StatusCode)
	}
	var s struct {
		ConfirmedCount int `json:"confirmed_count"`
		DeclinedCount  int `json:"declined_count"`
	}
	json.NewDecoder(res.Body).Decode(&s)
	res.Body.Close()
	if s.ConfirmedCount != 3 || s.DeclinedCount != 0 {
		t.Errorf("expected confirmed_count=3 declined_count=0, got %d/%d", s.ConfirmedCount, s.DeclinedCount)
	}
}

// 4.6 Header-Zähler: extended='declined', 2 Erweiterte, 0 Responses →
// declined_count=2.
func TestRsvpDefault_HeaderCount_ExtendedDeclined(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")
	db.Exec(`UPDATE training_sessions SET rsvp_default_extended='declined' WHERE id=?`, sessionID)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	for i := 0; i < 2; i++ {
		m := testutil.CreateMember(t, db, 0)
		testutil.AddExtendedKaderMember(t, db, kaderID, m)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d", sessionID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GetSession: expected 200, got %d", res.StatusCode)
	}
	var s struct {
		ConfirmedCount int `json:"confirmed_count"`
		DeclinedCount  int `json:"declined_count"`
	}
	json.NewDecoder(res.Body).Decode(&s)
	res.Body.Close()
	if s.DeclinedCount != 2 || s.ConfirmedCount != 0 {
		t.Errorf("expected declined_count=2 confirmed_count=0, got %d/%d", s.DeclinedCount, s.ConfirmedCount)
	}
}

// 4.7 Trainer bleibt hart-confirmed: players='declined' → Trainer hat weiterhin
// rsvp_status='confirmed' in der Zeile, wird aber nicht im Zähler mitgezählt.
func TestRsvpDefault_TrainerHardConfirmed_NotCounted(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-09-10")
	db.Exec(`UPDATE training_sessions SET rsvp_default_players='declined' WHERE id=?`, sessionID)

	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	trainerMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	token := testutil.Token(t, adminUserID, "admin", nil)

	byID := fetchAttendances(t, srv, sessionID, token)
	tr := byID[trainerMemberID]
	if !tr.IsTrainer {
		t.Fatalf("expected trainer row for member %d", trainerMemberID)
	}
	if tr.RSVPStatus == nil || *tr.RSVPStatus != "confirmed" {
		t.Errorf("trainer must stay hard-confirmed, got %v", tr.RSVPStatus)
	}

	res := testutil.Get(t, srv, fmt.Sprintf("/api/training-sessions/%d", sessionID), token)
	var s struct {
		ConfirmedCount int `json:"confirmed_count"`
		DeclinedCount  int `json:"declined_count"`
	}
	json.NewDecoder(res.Body).Decode(&s)
	res.Body.Close()
	if s.ConfirmedCount != 0 || s.DeclinedCount != 0 {
		t.Errorf("trainer must not be counted, got confirmed=%d declined=%d", s.ConfirmedCount, s.DeclinedCount)
	}
}
