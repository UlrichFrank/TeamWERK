package games_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func testServer(t *testing.T, h *games.Handler) *httptest.Server {
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kalender", h.ListGames)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Post("/api/admin/kalender", h.CreateGame)
		})
	})
}

// TestListGames_ReturnsGamesInRange verifies that games in the active season are returned.
func TestListGames_ReturnsGamesInRange(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	testutil.CreateGame(t, db, seasonID, teamID, "2026-01-15")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/kalender?season_id=%d", seasonID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	res.Body.Close()

	if len(games) != 1 {
		t.Errorf("expected 1 game, got %d", len(games))
	}
}

// TestListGames_EmptyRange verifies that an empty list is returned when no games exist.
func TestListGames_EmptyRange(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	res := testutil.Get(t, srv, "/api/kalender", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var result []map[string]any
	json.NewDecoder(res.Body).Decode(&result)
	res.Body.Close()

	if len(result) != 0 {
		t.Errorf("expected 0 games, got %d", len(result))
	}
}

// TestCreateGame_AdminOK verifies that an admin can create a game.
func TestCreateGame_AdminOK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminUserID, "admin", nil)
	body := map[string]any{
		"date":       "2026-06-15",
		"time":       "18:00",
		"opponent":   "FC Test",
		"team_ids":   []int{teamID},
		"event_type": "heim",
		"season_id":  seasonID,
	}
	res := testutil.Post(t, srv, "/api/admin/kalender", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", res.StatusCode)
	}
}

// TestCreateGame_UnauthorizedForbidden verifies that a user without club function cannot create a game.
func TestCreateGame_UnauthorizedForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	userID := testutil.CreateUser(t, db, "standard")
	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	body := map[string]any{
		"date": "2026-06-15", "time": "18:00",
		"opponent": "FC Test", "team_ids": []int{1},
		"event_type": "heim",
	}
	res := testutil.Post(t, srv, "/api/admin/kalender", token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}
