package carpooling_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/carpooling"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func testServerHTTP(t *testing.T, h *carpooling.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/mitfahrgelegenheiten", h.List)
	})
}

// createMultiTeamGame inserts a generic event linked to two teams.
func createMultiTeamGame(t *testing.T, db *sql.DB, seasonID int, teamIDA, teamIDB int, date string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, event_type, is_home) VALUES (?, ?, ?, ?, ?, ?)`,
		seasonID, "Team-Event", date, "10:00", "generisch", 0)
	if err != nil {
		t.Fatalf("createMultiTeamGame: %v", err)
	}
	gameID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?), (?, ?)`,
		gameID, teamIDA, gameID, teamIDB); err != nil {
		t.Fatalf("createMultiTeamGame game_teams: %v", err)
	}
	return int(gameID)
}

// TestList_HeimspielTeamIDs verifies that a single-team home game in the
// carpooling list response carries a teamIds array with exactly one element
// (the team's ID) and a time field with the game's anstosszeit.
func TestList_HeimspielTeamIDs(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	adminID := testutil.CreateUser(t, db, "admin")
	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServerHTTP(t, h)

	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Get(t, srv, "/api/mitfahrgelegenheiten", token)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body struct {
		Games []struct {
			Game struct {
				ID      int    `json:"id"`
				Time    string `json:"time"`
				TeamIDs []int  `json:"teamIds"`
			} `json:"game"`
		} `json:"games"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(body.Games))
	}
	g := body.Games[0].Game
	if g.ID != gameID {
		t.Errorf("expected game id %d, got %d", gameID, g.ID)
	}
	if g.Time != "18:00" {
		t.Errorf("expected time 18:00, got %q", g.Time)
	}
	if len(g.TeamIDs) != 1 || g.TeamIDs[0] != teamID {
		t.Errorf("expected teamIds=[%d], got %v", teamID, g.TeamIDs)
	}
}

// TestList_MultiTeamGenericEventTeamIDs verifies that a generic event linked
// to multiple teams returns all team IDs in the teamIds array, sorted ascending.
func TestList_MultiTeamGenericEventTeamIDs(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	gameID := createMultiTeamGame(t, db, seasonID, teamB, teamA, "2099-12-31")

	adminID := testutil.CreateUser(t, db, "admin")
	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServerHTTP(t, h)

	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Get(t, srv, "/api/mitfahrgelegenheiten", token)
	defer res.Body.Close()

	var body struct {
		Games []struct {
			Game struct {
				ID      int   `json:"id"`
				TeamIDs []int `json:"teamIds"`
			} `json:"game"`
		} `json:"games"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(body.Games))
	}
	g := body.Games[0].Game
	if g.ID != gameID {
		t.Errorf("expected game id %d, got %d", gameID, g.ID)
	}
	if len(g.TeamIDs) != 2 {
		t.Fatalf("expected 2 teamIds, got %v", g.TeamIDs)
	}
	// Sorted ascending — see parseTeamIDs
	if g.TeamIDs[0] >= g.TeamIDs[1] {
		t.Errorf("expected teamIds sorted ascending, got %v", g.TeamIDs)
	}
	if !(g.TeamIDs[0] == teamA && g.TeamIDs[1] == teamB) &&
		!(g.TeamIDs[0] == teamB && g.TeamIDs[1] == teamA) {
		t.Errorf("teamIds %v does not contain both teamA=%d and teamB=%d", g.TeamIDs, teamA, teamB)
	}
}
