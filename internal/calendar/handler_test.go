package calendar_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

func readBody(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// setupCalendarFixture creates a user with a linked member in a kader for a team,
// a game for that team, and a duty slot assigned to the user.
// Returns: userID, userToken, gameID, dutySlotID.
func setupCalendarFixture(t *testing.T) (*httptest.Server, int, string, int, int) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-08-15")
	dutyTypeID := testutil.CreateDutyType(t, db, "Aufwärmer", 1.0)
	dutySlotID := testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, gameID, "2026-08-15")
	db.Exec(`INSERT INTO duty_assignments (duty_slot_id, user_id, status) VALUES (?, ?, 'assigned')`,
		dutySlotID, userID)
	userToken := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)
	return srv, userID, userToken, gameID, dutySlotID
}

func postToken(t *testing.T, srv *httptest.Server, userToken string, body map[string]any) map[string]any {
	t.Helper()
	res := testutil.Post(t, srv, "/api/calendar/token", userToken, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/calendar/token: expected 200, got %d", res.StatusCode)
	}
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return out
}

func allTogglesOn() map[string]any {
	return map[string]any{
		"include_heim": true, "include_auswaerts": true, "include_training": true,
		"include_generisch": true, "include_duty": true,
	}
}

// TestCalendarFeed_ValidToken checks HTTP 200, text/calendar Content-Type, and VCALENDAR wrapper.
func TestCalendarFeed_ValidToken(t *testing.T) {
	srv, _, userToken, _, _ := setupCalendarFixture(t)
	tok := postToken(t, srv, userToken, allTogglesOn())
	calToken := tok["token"].(string)

	res := testutil.Get(t, srv, "/api/calendar/feed/"+calToken, "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	ct := res.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/calendar") {
		t.Errorf("expected text/calendar Content-Type, got %q", ct)
	}
	body := readBody(t, res.Body)
	if !strings.Contains(body, "BEGIN:VCALENDAR") {
		t.Error("feed missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(body, "END:VCALENDAR") {
		t.Error("feed missing END:VCALENDAR")
	}
}

// TestCalendarFeed_GameAndDutyHaveRealDates is a regression guard: the
// modernc.org/sqlite driver returns DATE columns as ISO timestamps
// ("2026-08-15T00:00:00Z"), so parseDT must normalize them. A regression
// produces DTSTART of the Go zero-time (00010101...), which calendar clients
// place in year 1 — the feed "imports but shows nothing".
func TestCalendarFeed_GameAndDutyHaveRealDates(t *testing.T) {
	srv, _, userToken, _, _ := setupCalendarFixture(t)
	tok := postToken(t, srv, userToken, allTogglesOn())
	calToken := tok["token"].(string)

	res := testutil.Get(t, srv, "/api/calendar/feed/"+calToken, "")
	defer res.Body.Close()
	body := readBody(t, res.Body)

	if !strings.Contains(body, "UID:game-") {
		t.Errorf("feed must contain the user's game, body:\n%s", body)
	}
	if !strings.Contains(body, "UID:duty-") {
		t.Errorf("feed must contain the user's duty, body:\n%s", body)
	}
	// Game date is 2026-08-15 at 18:00 → must carry the real date, not year 1.
	if !strings.Contains(body, "DTSTART;TZID=Europe/Berlin:20260815T180000") {
		t.Errorf("game DTSTART has wrong date (parseDT regression?), body:\n%s", body)
	}
	if strings.Contains(body, "00010101T000000") {
		t.Errorf("feed contains Go zero-time DTSTART — date parsing failed, body:\n%s", body)
	}
}

// TestCalendarFeed_CalNameHasFirstName verifies the calendar name carries the
// account's first name so families can subscribe to several feeds and tell them
// apart. The name comes from users.first_name (a member is not always present).
func TestCalendarFeed_CalNameHasFirstName(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	db.Exec(`UPDATE users SET first_name = ? WHERE id = ?`, "Anna", userID)
	userToken := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)

	tok := postToken(t, srv, userToken, allTogglesOn())
	calToken := tok["token"].(string)

	res := testutil.Get(t, srv, "/api/calendar/feed/"+calToken, "")
	defer res.Body.Close()
	body := readBody(t, res.Body)
	if !strings.Contains(body, "X-WR-CALNAME:TeamWERK – Anna") {
		t.Errorf("calendar name must include the account first name, body:\n%s", body)
	}
}

// TestCalendarFeed_InvalidToken returns 404.
func TestCalendarFeed_InvalidToken(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	res := testutil.Get(t, srv, "/api/calendar/feed/nonexistent-token-xyz", "")
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

// TestCalendarFeed_IncludeTrainingFalse verifies training_sessions are excluded.
func TestCalendarFeed_IncludeTrainingFalse(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	// Insert a training session for the user's team.
	db.Exec(`INSERT INTO training_sessions (team_id, season_id, date, start_time, end_time, location, status)
	         VALUES (?, ?, ?, ?, ?, ?, 'active')`,
		teamID, seasonID, "2026-08-15", "18:00", "20:00", "Halle X")

	userToken := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)

	tok := postToken(t, srv, userToken, map[string]any{
		"include_heim": true, "include_auswaerts": true, "include_training": false,
		"include_generisch": true, "include_duty": true,
	})
	calToken := tok["token"].(string)

	res := testutil.Get(t, srv, "/api/calendar/feed/"+calToken, "")
	defer res.Body.Close()
	if strings.Contains(readBody(t, res.Body), "UID:training-") {
		t.Error("feed must not contain training_sessions when include_training=false")
	}
}

// TestCalendarFeed_IncludeTrainingTrue verifies training_sessions appear when toggled on.
func TestCalendarFeed_IncludeTrainingTrue(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team Alpha")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	db.Exec(`INSERT INTO training_sessions (team_id, season_id, date, start_time, end_time, location, status)
	         VALUES (?, ?, ?, ?, ?, ?, 'active')`,
		teamID, seasonID, "2026-08-15", "18:00", "20:00", "Halle X")

	userToken := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)
	tok := postToken(t, srv, userToken, allTogglesOn())
	calToken := tok["token"].(string)

	res := testutil.Get(t, srv, "/api/calendar/feed/"+calToken, "")
	defer res.Body.Close()
	body := readBody(t, res.Body)
	if !strings.Contains(body, "UID:training-") {
		t.Error("feed must contain training_sessions when include_training=true")
	}
	if !strings.Contains(body, "Training: Team Alpha") {
		t.Errorf("feed must contain training summary with team name, body: %s", body)
	}
}

// TestCalendarFeed_IncludeDutyFalse verifies duty events are excluded.
func TestCalendarFeed_IncludeDutyFalse(t *testing.T) {
	srv, _, userToken, _, _ := setupCalendarFixture(t)
	tok := postToken(t, srv, userToken, map[string]any{
		"include_heim": true, "include_auswaerts": true, "include_training": true,
		"include_generisch": true, "include_duty": false,
	})
	calToken := tok["token"].(string)

	res := testutil.Get(t, srv, "/api/calendar/feed/"+calToken, "")
	defer res.Body.Close()
	if strings.Contains(readBody(t, res.Body), "UID:duty-") {
		t.Error("feed must not contain duty events when include_duty=false")
	}
}

// TestCalendarToken_Post_Create creates a token and returns non-empty token value.
func TestCalendarToken_Post_Create(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)

	out := postToken(t, srv, token, allTogglesOn())
	if v, ok := out["token"].(string); !ok || v == "" {
		t.Error("expected non-empty token in response")
	}
}

// TestCalendarToken_Post_Update keeps token value, updates settings.
func TestCalendarToken_Post_Update(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)

	out1 := postToken(t, srv, token, allTogglesOn())
	firstToken := out1["token"].(string)

	out2 := postToken(t, srv, token, map[string]any{
		"include_heim": true, "include_auswaerts": true, "include_training": true,
		"include_generisch": true, "include_duty": false,
	})
	if out2["token"] != firstToken {
		t.Errorf("token changed on update: got %v, want %v", out2["token"], firstToken)
	}
	if out2["include_duty"] != false {
		t.Errorf("expected include_duty=false after update, got %v", out2["include_duty"])
	}
}

// TestCalendarToken_Delete removes the token; feed returns 404 afterwards.
func TestCalendarToken_Delete(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	userToken := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)

	out := postToken(t, srv, userToken, allTogglesOn())
	calToken := out["token"].(string)

	delRes := testutil.Do(t, srv, http.MethodDelete, "/api/calendar/token", userToken, nil)
	delRes.Body.Close()
	if delRes.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE: expected 204, got %d", delRes.StatusCode)
	}

	feedRes := testutil.Get(t, srv, "/api/calendar/feed/"+calToken, "")
	feedRes.Body.Close()
	if feedRes.StatusCode != http.StatusNotFound {
		t.Errorf("after delete: expected 404, got %d", feedRes.StatusCode)
	}
}

// TestCalendarToken_Get_NotFound returns 404 when no token exists for user.
func TestCalendarToken_Get_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)

	res := testutil.Get(t, srv, "/api/calendar/token", token)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

// TestCalendarFeed_TrainingLocation verifies that a training with a venue_id
// shows the venue name and address in the LOCATION field of the iCal feed.
func TestCalendarFeed_TrainingLocation(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team B")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	var venueID int64
	row := db.QueryRow(`INSERT INTO venues (name, street, postal_code, city, is_home_venue) VALUES (?, ?, ?, ?, 1) RETURNING id`,
		"Sporthalle Vaihingen", "Rosenstraße 5", "70563", "Stuttgart")
	row.Scan(&venueID)

	db.Exec(`INSERT INTO training_sessions (team_id, season_id, date, start_time, end_time, venue_id, status)
	         VALUES (?, ?, ?, ?, ?, ?, 'active')`,
		teamID, seasonID, "2026-08-20", "18:00", "20:00", venueID)

	userToken := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)
	tok := postToken(t, srv, userToken, allTogglesOn())
	calToken := tok["token"].(string)

	res := testutil.Get(t, srv, "/api/calendar/feed/"+calToken, "")
	defer res.Body.Close()
	body := readBody(t, res.Body)

	if !strings.Contains(body, "LOCATION:Sporthalle Vaihingen") {
		t.Errorf("training LOCATION missing venue name, body:\n%s", body)
	}
	if !strings.Contains(body, "Rosenstra") {
		t.Errorf("training LOCATION missing street, body:\n%s", body)
	}
}

// TestCalendarToken_Unauthenticated returns 401.
func TestCalendarToken_Unauthenticated(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)

	res := testutil.Get(t, srv, "/api/calendar/token", "")
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}
