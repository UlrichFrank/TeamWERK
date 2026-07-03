package games_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// seedGameWithKader creates an active season, team, kader and game and returns IDs.
func seedGameWithKader(t *testing.T, db *sql.DB) (seasonID, teamID, kaderID, gameID int) {
	t.Helper()
	seasonID = testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("activate season: %v", err)
	}
	teamID = testutil.CreateTeam(t, db, "Herren")
	kaderID = testutil.CreateKader(t, db, teamID, seasonID)
	gameID = testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")
	return
}

// firstGameMyRSVP fetches GET /api/games/my for the given user and returns the
// my_rsvp value of the (single expected) game.
func firstGameMyRSVP(t *testing.T, db *sql.DB, userID int) any {
	t.Helper()
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/games/my", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("games/my: expected 200, got %d", res.StatusCode)
	}
	var list []map[string]any
	json.NewDecoder(res.Body).Decode(&list)
	res.Body.Close()
	if len(list) != 1 {
		t.Fatalf("expected 1 game, got %d", len(list))
	}
	return list[0]["my_rsvp"]
}

// 4.9 Happy-Path (my_rsvp): extended-only member with rsvp_default_extended='declined'
// gets my_rsvp='declined'.
func TestGameRsvpDefault_ExtendedDeclined_MyRSVP(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, kaderID, gameID := seedGameWithKader(t, db)
	db.Exec(`UPDATE games SET rsvp_default_extended='declined' WHERE id=?`, gameID)

	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, mid)

	if got := firstGameMyRSVP(t, db, uid); got != "declined" {
		t.Errorf("expected my_rsvp=declined, got %v", got)
	}
}

// 4.9 extended='confirmed' auto-confirms an extended-only member (MODIFIED req scenario).
func TestGameRsvpDefault_ExtendedConfirmed_MyRSVP(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, kaderID, gameID := seedGameWithKader(t, db)
	db.Exec(`UPDATE games SET rsvp_default_extended='confirmed' WHERE id=?`, gameID)

	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, mid)

	if got := firstGameMyRSVP(t, db, uid); got != "confirmed" {
		t.Errorf("expected my_rsvp=confirmed, got %v", got)
	}
}

// 4.9 Konfliktsperre: POST /api/games mit declined + rsvp_require_reason=1 → 400, kein Spiel.
func TestGameRsvpDefault_CreateConflictRejected(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	body := map[string]any{
		"date":                 "2026-02-01",
		"time":                 "18:00",
		"opponent":             "FC Test",
		"team_ids":             []int{teamID},
		"event_type":           "heim",
		"season_id":            seasonID,
		"rsvp_default_players": "declined",
		"rsvp_require_reason":  1,
	}
	res := testutil.Do(t, srv, http.MethodPost, "/api/games", token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var payload map[string]any
	json.NewDecoder(res.Body).Decode(&payload)
	if payload["error"] != "invalid_rsvp_settings" {
		t.Errorf("expected error=invalid_rsvp_settings, got %v", payload["error"])
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM games`).Scan(&count)
	if count != 0 {
		t.Errorf("no game must be created on conflict, got %d", count)
	}
}

// 4.9 Konfliktsperre: PUT /api/games/{id} mit extended='declined' + require_reason=1 → 400, unverändert.
func TestGameRsvpDefault_UpdateConflictRejected(t *testing.T) {
	db := testutil.NewDB(t)
	_, teamID, _, gameID := seedGameWithKader(t, db)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	body := map[string]any{
		"date":                  "2026-01-15",
		"time":                  "18:00",
		"opponent":              "FC Test",
		"team_ids":              []int{teamID},
		"event_type":            "heim",
		"rsvp_default_extended": "declined",
		"rsvp_require_reason":   1,
	}
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/games/%d", gameID), token, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}

	var defExtended string
	db.QueryRow(`SELECT rsvp_default_extended FROM games WHERE id=?`, gameID).Scan(&defExtended)
	if defExtended != "none" {
		t.Errorf("game must be unchanged on 400, got extended=%s", defExtended)
	}
}

// 4.9 Header-Zähler: players='confirmed', 3 Kader-Spieler, 0 Responses → confirmed_count=3.
func TestGameRsvpDefault_HeaderCount_PlayersConfirmed(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, kaderID, gameID := seedGameWithKader(t, db)
	db.Exec(`UPDATE games SET rsvp_default_players='confirmed' WHERE id=?`, gameID)
	for i := 0; i < 3; i++ {
		mid := testutil.CreateMember(t, db, 0)
		db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, mid)
	}

	confirmed, declined := gameCounts(t, db, gameID)
	if confirmed != 3 || declined != 0 {
		t.Errorf("expected confirmed=3 declined=0, got %d/%d", confirmed, declined)
	}
}

// 4.9 Header-Zähler: extended='declined', 2 Erweiterte, 0 Responses → declined_count=2.
func TestGameRsvpDefault_HeaderCount_ExtendedDeclined(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, kaderID, gameID := seedGameWithKader(t, db)
	db.Exec(`UPDATE games SET rsvp_default_extended='declined' WHERE id=?`, gameID)
	for i := 0; i < 2; i++ {
		mid := testutil.CreateMember(t, db, 0)
		db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, mid)
	}

	confirmed, declined := gameCounts(t, db, gameID)
	if declined != 2 || confirmed != 0 {
		t.Errorf("expected declined=2 confirmed=0, got confirmed=%d declined=%d", confirmed, declined)
	}
}

// 4.9 Trainer bleibt aus den Zählern ausgeschlossen, auch bei players='declined'.
func TestGameRsvpDefault_TrainerNotCounted(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, kaderID, gameID := seedGameWithKader(t, db)
	db.Exec(`UPDATE games SET rsvp_default_players='declined' WHERE id=?`, gameID)
	trainerMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	confirmed, declined := gameCounts(t, db, gameID)
	if confirmed != 0 || declined != 0 {
		t.Errorf("trainer must not be counted, got confirmed=%d declined=%d", confirmed, declined)
	}
}

// gameCounts fetches confirmed_count/declined_count via GET /api/games/{id}.
func gameCounts(t *testing.T, db *sql.DB, gameID int) (confirmed, declined int) {
	t.Helper()
	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", gameID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GetGame: expected 200, got %d", res.StatusCode)
	}
	var payload struct {
		Game struct {
			ConfirmedCount int `json:"confirmed_count"`
			DeclinedCount  int `json:"declined_count"`
		} `json:"game"`
	}
	json.NewDecoder(res.Body).Decode(&payload)
	res.Body.Close()
	return payload.Game.ConfirmedCount, payload.Game.DeclinedCount
}
