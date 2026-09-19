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

// --- teams / games (video-upload-eligible-teams) -----------------------------

type eligibleResponse struct {
	GameIDs []int `json:"game_ids"`
	Teams   []struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		UploadWithoutGame bool   `json:"upload_without_game"`
	} `json:"teams"`
	Games []struct {
		ID       int    `json:"id"`
		Opponent string `json:"opponent"`
		SeasonID int    `json:"season_id"`
		TeamIDs  []int  `json:"team_ids"`
	} `json:"games"`
}

func eligibleFull(t *testing.T, srv *httptest.Server, tok string) eligibleResponse {
	t.Helper()
	res := testutil.Get(t, srv, "/api/videos/upload-eligible-games", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var body eligibleResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body
}

func (r eligibleResponse) teamFlag(t *testing.T, id int) (found, withoutGame bool) {
	t.Helper()
	for _, tm := range r.Teams {
		if tm.ID == id {
			return true, tm.UploadWithoutGame
		}
	}
	return false, false
}

func (r eligibleResponse) gameTeamIDs(id int) ([]int, bool) {
	for _, g := range r.Games {
		if g.ID == id {
			return g.TeamIDs, true
		}
	}
	return nil, false
}

// Kern des Changes: das Dienst-Team erscheint auch OHNE jede Kader-Verbindung
// des Nutzers zu dieser Mannschaft — genau der Fall, den GET /api/teams und
// GameVisibilityClause nicht abbilden.
func TestEligibleGames_DutyOnlyUserGetsTeamAndGameWithoutKaderLink(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)
	srv := newEligibleGamesServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "mA2")
	teamB := testutil.CreateTeam(t, db, "mB2")
	testutil.CreateKader(t, db, teamA, season)
	testutil.CreateKader(t, db, teamB, season)
	dutyGame := testutil.CreateGame(t, db, season, teamA, "2026-03-15")
	testutil.CreateGame(t, db, season, teamA, "2026-03-22")
	testutil.CreateGame(t, db, season, teamB, "2026-03-22")

	videoDutyType := createDutyTypeWithVideoFlag(t, db, "Kamera", true)
	slot := testutil.CreateDutySlot(t, db, videoDutyType, season, teamA, dutyGame, "2026-03-15")

	user := testutil.CreateUser(t, db, "standard")
	addDutyAssignment(t, db, slot, user)

	body := eligibleFull(t, srv, testutil.Token(t, user, "standard", []string{"spieler"}))

	if len(body.Teams) != 1 {
		t.Fatalf("teams = %+v, want exactly the duty team", body.Teams)
	}
	if found, without := body.teamFlag(t, teamA); !found || without {
		t.Errorf("team %d: found=%v upload_without_game=%v, want found and false", teamA, found, without)
	}
	if len(body.Games) != 1 {
		t.Fatalf("games = %+v, want exactly the duty game", body.Games)
	}
	ids, ok := body.gameTeamIDs(dutyGame)
	if !ok || len(ids) != 1 || ids[0] != teamA {
		t.Errorf("game %d team_ids = %v (found=%v), want [%d]", dutyGame, ids, ok, teamA)
	}
	if body.Games[0].SeasonID != season {
		t.Errorf("game season_id = %d, want %d", body.Games[0].SeasonID, season)
	}
}

func TestEligibleGames_TrainerTeamAllowsUploadWithoutGame(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)
	srv := newEligibleGamesServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	testutil.CreateKader(t, db, teamB, season)
	gameA := testutil.CreateGame(t, db, season, teamA, "2026-03-15")
	gameB := testutil.CreateGame(t, db, season, teamB, "2026-03-15")

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMember)

	body := eligibleFull(t, srv, testutil.Token(t, trainerUser, "standard", []string{"trainer"}))

	if found, without := body.teamFlag(t, teamA); !found || !without {
		t.Errorf("team A: found=%v upload_without_game=%v, want found and true", found, without)
	}
	if found, _ := body.teamFlag(t, teamB); found {
		t.Errorf("team B must not be listed for a trainer of team A only")
	}
	if _, ok := body.gameTeamIDs(gameA); !ok {
		t.Errorf("game %d of own team missing from games", gameA)
	}
	if _, ok := body.gameTeamIDs(gameB); ok {
		t.Errorf("game %d of foreign team must not be in games", gameB)
	}
}

// Trainer von A mit Kamera-Dienst bei B: beide Teams, aber nur A erlaubt den
// Upload ohne Spiel; das Dienst-Spiel von B steht in games.
func TestEligibleGames_TrainerWithDutyAtForeignTeamGetsBoth(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)
	srv := newEligibleGamesServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "mB2")
	teamB := testutil.CreateTeam(t, db, "mA2")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	testutil.CreateKader(t, db, teamB, season)
	gameB := testutil.CreateGame(t, db, season, teamB, "2026-03-15")
	otherB := testutil.CreateGame(t, db, season, teamB, "2026-03-22")

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMember)

	videoDutyType := createDutyTypeWithVideoFlag(t, db, "Kamera", true)
	slot := testutil.CreateDutySlot(t, db, videoDutyType, season, teamB, gameB, "2026-03-15")
	addDutyAssignment(t, db, slot, trainerUser)

	body := eligibleFull(t, srv, testutil.Token(t, trainerUser, "standard", []string{"trainer"}))

	if found, without := body.teamFlag(t, teamA); !found || !without {
		t.Errorf("own team: found=%v upload_without_game=%v, want found and true", found, without)
	}
	if found, without := body.teamFlag(t, teamB); !found || without {
		t.Errorf("duty team: found=%v upload_without_game=%v, want found and false", found, without)
	}
	if ids, ok := body.gameTeamIDs(gameB); !ok || len(ids) != 1 || ids[0] != teamB {
		t.Errorf("duty game team_ids = %v (found=%v), want [%d]", ids, ok, teamB)
	}
	if _, ok := body.gameTeamIDs(otherB); ok {
		t.Errorf("other game of duty team must not be eligible")
	}
	// Reihenfolge: Rollen-Teams zuerst, dann Dienst-Teams.
	if len(body.Teams) != 2 || body.Teams[0].ID != teamA || body.Teams[1].ID != teamB {
		t.Errorf("teams order = %+v, want [own, duty]", body.Teams)
	}
}

func TestEligibleGames_VorstandGetsAllTeamsWithActiveKader(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)
	srv := newEligibleGamesServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	teamNoKader := testutil.CreateTeam(t, db, "Ohne Kader")
	testutil.CreateKader(t, db, teamA, season)
	testutil.CreateKader(t, db, teamB, season)
	game := testutil.CreateGame(t, db, season, teamA, "2026-03-15")

	user := testutil.CreateUser(t, db, "standard")
	body := eligibleFull(t, srv, testutil.Token(t, user, "standard", []string{"vorstand"}))

	for _, id := range []int{teamA, teamB} {
		if found, without := body.teamFlag(t, id); !found || !without {
			t.Errorf("team %d: found=%v upload_without_game=%v, want found and true", id, found, without)
		}
	}
	if found, _ := body.teamFlag(t, teamNoKader); found {
		t.Errorf("team without kader in the active season must not be listed")
	}
	if ids, ok := body.gameTeamIDs(game); !ok || len(ids) != 1 || ids[0] != teamA {
		t.Errorf("game team_ids = %v (found=%v), want [%d]", ids, ok, teamA)
	}
}

// Kader-Zugehörigkeit allein (Spieler, Eltern) schaltet hier nichts frei —
// anders als bei /api/teams, das die Kader-Zugehörigkeit abbildet.
func TestEligibleGames_PlayerWithKaderLinkGetsThreeEmptyLists(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)
	srv := newEligibleGamesServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)
	testutil.CreateGame(t, db, season, team, "2026-03-15")

	user := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateMember(t, db, user)
	testutil.AddKaderMember(t, db, kader, member)

	body := eligibleFull(t, srv, testutil.Token(t, user, "standard", []string{"spieler"}))
	if len(body.GameIDs) != 0 || len(body.Teams) != 0 || len(body.Games) != 0 {
		t.Errorf("want three empty lists, got game_ids=%v teams=%+v games=%+v", body.GameIDs, body.Teams, body.Games)
	}
}
