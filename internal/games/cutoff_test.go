package games_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func berlinTime(t *testing.T, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	tm, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	if err != nil {
		t.Fatalf("ParseInLocation %s: %v", value, err)
	}
	return tm
}

func fixedNow(tm time.Time) func() time.Time { return func() time.Time { return tm } }

// cutoffServer mounts a minimal router with the game RSVP endpoints behind
// the auth middleware so that JWT-based handlers behave like in production.
func cutoffServer(t *testing.T, h *games.Handler) *httptest.Server {
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/games", h.ListGames)
		r.Get("/api/games/{id}", h.GetGame)
		r.Get("/api/games/my", h.ListMyGames)
		r.Post("/api/games/{id}/respond", h.RespondToGame)
	})
}

// setupCutoffGame creates a game on 2026-06-15 18:00 Europe/Berlin and a
// kader linked to teamID so that regular members (spieler/eltern) pass the
// UserCanSeeGame visibility check. Returns DB, gameID, teamID, seasonID,
// kaderID.
func setupCutoffGame(t *testing.T) (db *sql.DB, gameID, teamID, seasonID, kaderID int) {
	t.Helper()
	db = testutil.NewDB(t)
	seasonID = testutil.CreateSeason(t, db, "2025/26")
	teamID = testutil.CreateTeam(t, db, "Team A")
	gameID = testutil.CreateGame(t, db, seasonID, teamID, "2026-06-15")
	kaderID = testutil.CreateKader(t, db, teamID, seasonID)
	return
}

func addClubFunction(t *testing.T, db *sql.DB, memberID int, fn string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		memberID, fn); err != nil {
		t.Fatalf("addClubFunction: %v", err)
	}
}

func newGamesHandler(t *testing.T, db *sql.DB, nowAt time.Time) *games.Handler {
	t.Helper()
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(nowAt))
	return h
}

// Spieler 24 h vor Spielbeginn → 204.
func TestRespondToGame_Cutoff_PlayerBefore_OK(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-14 18:00")) // T-24h
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// Spieler 12 h vor Spielbeginn → 422 rsvp_locked.
func TestRespondToGame_Cutoff_PlayerAfter_422(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-15 06:00")) // T-12h
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
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
	if got, _ := body["locks_at"].(string); got == "" {
		t.Errorf("expected locks_at to be set, got empty")
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM game_responses WHERE game_id=? AND member_id=?`,
		gameID, mID).Scan(&n)
	if n != 0 {
		t.Errorf("expected no response row, got %d", n)
	}
}

// Spieler ändert confirmed → declined 12 h vor Spiel → 422, alter Status bleibt.
func TestRespondToGame_Cutoff_StatusChange_422(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, responded_at)
	         VALUES (?, ?, ?, 'confirmed', datetime('now'))`, gameID, mID, uID)

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-15 06:00"))
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "declined"})
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
	var status string
	db.QueryRow(`SELECT status FROM game_responses WHERE game_id=? AND member_id=?`,
		gameID, mID).Scan(&status)
	if status != "confirmed" {
		t.Errorf("expected status to remain 'confirmed', got %q", status)
	}
}

// Eltern für Kind 12 h vor Spiel → 422.
func TestRespondToGame_Cutoff_ParentAfter_422(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, childMemberID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUserID, childMemberID)

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-15 06:00"))
	srv := cutoffServer(t, h)

	token := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "declined", "member_id": childMemberID})
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
}

// Trainer 12 h vor Spiel → 204 (Override).
func TestRespondToGame_Cutoff_TrainerAfter_OK(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMID := testutil.CreateMember(t, db, trainerUserID)
	addClubFunction(t, db, trainerMID, "trainer") // für event_visibility-Bypass
	targetMember := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, targetMember)

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-15 06:00"))
	srv := cutoffServer(t, h)

	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "declined", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// sportliche_leitung 12 h vor Spiel → 204.
func TestRespondToGame_Cutoff_SportlicheLeitung_OK(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addClubFunction(t, db, mID, "sportliche_leitung")
	targetMember := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, targetMember)

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-15 06:00"))
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"sportliche_leitung"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "confirmed", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// Vorstand 1 h nach Spielbeginn → 204.
func TestRespondToGame_Cutoff_VorstandAfterStart_OK(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addClubFunction(t, db, mID, "vorstand")
	targetMember := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, targetMember)

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-15 19:00")) // 1 h nach Beginn
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"vorstand"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "declined", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// Admin (Rolle) ohne Funktion 12 h vor Spiel → 204.
func TestRespondToGame_Cutoff_AdminAfter_OK(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "admin")
	testutil.CreateMember(t, db, uID)
	targetMember := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, targetMember)

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-15 06:00"))
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "admin", nil)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "confirmed", "member_id": targetMember})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// Kassierer ohne weitere Funktion 12 h vor Spiel → 422.
func TestRespondToGame_Cutoff_KassiererAfter_422(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addClubFunction(t, db, mID, "kassierer")
	addKaderMember(t, db, kaderID, mID) // damit UserCanSeeGame durchgeht

	h := newGamesHandler(t, db, berlinTime(t, "2026-06-15 06:00"))
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"kassierer"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "declined"})
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
}

// Absence-Lock hat Vorrang vor Cutoff (403, nicht 422).
func TestRespondToGame_Cutoff_AbsenceLockTakesPrecedence_403(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)

	r, err := db.Exec(`INSERT INTO member_absences (member_id, type, start_date, end_date, created_by)
	                   VALUES (?, 'vacation', '2026-06-14', '2026-06-20', ?)`, mID, uID)
	if err != nil {
		t.Fatalf("insert absence: %v", err)
	}
	absID, _ := r.LastInsertId()
	db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, responded_at, absence_id)
	         VALUES (?, ?, ?, 'declined', datetime('now'), ?)`, gameID, mID, uID, absID)

	// Weit vor Cutoff — Absence-Lock kommt zuerst.
	h := newGamesHandler(t, db, berlinTime(t, "2026-06-13 12:00"))
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 (absence lock), got %d", res.StatusCode)
	}
}

// ListMyGames enthält rsvp_locks_at (Sommerzeit: 18:00 Berlin = 16:00Z, -18h = 22:00Z am Vortag).
func TestListMyGames_RsvpLocksAt_Summer(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/games/my", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	var got string
	for _, it := range items {
		if int(it["id"].(float64)) == gameID {
			got, _ = it["rsvp_locks_at"].(string)
			break
		}
	}
	if got != "2026-06-14T22:00:00Z" {
		t.Errorf("expected rsvp_locks_at=2026-06-14T22:00:00Z, got %q", got)
	}
}

// ListGames (Vorstand-Sicht) enthält rsvp_locks_at.
func TestListGames_RsvpLocksAt(t *testing.T) {
	db, gameID, _, seasonID, _ := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "admin")

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d", seasonID), token)
	defer res.Body.Close()
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	var got string
	for _, it := range items {
		if int(it["id"].(float64)) == gameID {
			got, _ = it["rsvp_locks_at"].(string)
			break
		}
	}
	if got != "2026-06-14T22:00:00Z" {
		t.Errorf("expected rsvp_locks_at=2026-06-14T22:00:00Z, got %q", got)
	}
}

// Detail (GetGame) enthält rsvp_locks_at.
func TestGetGame_RsvpLocksAt(t *testing.T) {
	db, gameID, _, _, _ := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "admin")

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := cutoffServer(t, h)

	token := testutil.Token(t, uID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", gameID), token)
	defer res.Body.Close()
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	g, _ := body["game"].(map[string]any)
	got, _ := g["rsvp_locks_at"].(string)
	if got != "2026-06-14T22:00:00Z" {
		t.Errorf("expected rsvp_locks_at=2026-06-14T22:00:00Z, got %q", got)
	}
}
