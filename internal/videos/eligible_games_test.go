package videos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newEligibleGamesServer(t *testing.T, h *Handler) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Use(auth.Middleware(testutil.TestJWTSecret))
	r.Get("/api/videos/upload-eligible-games", h.EligibleGames)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func eligibleGameIDs(t *testing.T, srv *httptest.Server, tok string) []int {
	t.Helper()
	res := testutil.Get(t, srv, "/api/videos/upload-eligible-games", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var body struct {
		GameIDs []int `json:"game_ids"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.GameIDs
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func TestEligibleGames_DutyOnlyUserSeesOnlyAssignedGame(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)
	srv := newEligibleGamesServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	assignedGame := testutil.CreateGame(t, db, season, team, "2026-03-15")
	otherGame := testutil.CreateGame(t, db, season, team, "2026-03-22")

	videoDutyType := createDutyTypeWithVideoFlag(t, db, "Video", true)
	slot := testutil.CreateDutySlot(t, db, videoDutyType, season, team, assignedGame, "2026-03-15")

	user := testutil.CreateUser(t, db, "standard")
	addDutyAssignment(t, db, slot, user)

	tok := testutil.Token(t, user, "standard", []string{"spieler"})
	ids := eligibleGameIDs(t, srv, tok)

	if !containsInt(ids, assignedGame) {
		t.Errorf("game_ids = %v, want to contain assigned game %d", ids, assignedGame)
	}
	if containsInt(ids, otherGame) {
		t.Errorf("game_ids = %v, want NOT to contain unrelated game %d", ids, otherGame)
	}
}

func TestEligibleGames_TrainerSeesAllTeamGames(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)
	srv := newEligibleGamesServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)
	gameOne := testutil.CreateGame(t, db, season, team, "2026-03-15")
	gameTwo := testutil.CreateGame(t, db, season, team, "2026-03-22")

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kader, trainerMember)

	tok := testutil.Token(t, trainerUser, "standard", []string{"trainer"})
	ids := eligibleGameIDs(t, srv, tok)

	if !containsInt(ids, gameOne) || !containsInt(ids, gameTwo) {
		t.Errorf("game_ids = %v, want both %d and %d (trainer sees all team games)", ids, gameOne, gameTwo)
	}
}

func TestEligibleGames_NoPermissionYieldsEmptyList(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)
	srv := newEligibleGamesServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	testutil.CreateGame(t, db, season, team, "2026-03-15")

	user := testutil.CreateUser(t, db, "standard")
	tok := testutil.Token(t, user, "standard", []string{"spieler"})

	// eligibleGameIDs fails the test itself if the status isn't 200 — an empty
	// own permission set is not an error.
	ids := eligibleGameIDs(t, srv, tok)
	if len(ids) != 0 {
		t.Errorf("game_ids = %v, want empty", ids)
	}
}
