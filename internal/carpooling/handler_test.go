package carpooling_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func testFullServer(t *testing.T, h *carpooling.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/mitfahrgelegenheiten", h.List)
		r.Post("/api/mitfahrgelegenheiten", h.Upsert)
		r.Delete("/api/mitfahrgelegenheiten/{id}", h.Delete)
		r.Post("/api/mitfahrt-paarungen", h.RequestPairing)
		r.Post("/api/mitfahrt-paarungen/{id}/confirm", h.ConfirmPairing)
		r.Post("/api/mitfahrt-paarungen/{id}/reject", h.RejectPairing)
	})
}

// createFamilyLink creates a family_link between parentUserID and the member linked to childUserID.
func createFamilyLink(t *testing.T, db *sql.DB, parentUserID, childMemberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID); err != nil {
		t.Fatalf("createFamilyLink: %v", err)
	}
}

// setupParentChild creates a parent user, a child user with a member record, links them, and returns their IDs.
func setupParentChild(t *testing.T, db *sql.DB) (parentID, childUserID, childMemberID int) {
	t.Helper()
	parentID = testutil.CreateUser(t, db, "standard")
	childUserID = testutil.CreateUser(t, db, "standard")
	childMemberID = testutil.CreateMember(t, db, childUserID)
	createFamilyLink(t, db, parentID, childMemberID)
	return parentID, childUserID, childMemberID
}

// TestElternteil_UpsertFuerKind verifies a parent can create a carpooling entry for their child.
func TestElternteil_UpsertFuerKind(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID, childUserID, childMemberID := setupParentChild(t, db)
	// Kind ins Kader des Spiel-Teams — sonst greift event-team-visibility
	// und Eltern sieht das Game nicht.
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrgelegenheiten", token, map[string]any{
		"gameId":    gameID,
		"typ":       "suche",
		"forUserId": childUserID,
		"plaetze":   1,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	// Verify entry was created with child's user_id
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'suche'`, gameID, childUserID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 entry for child, got %d", count)
	}
}

// TestElternteil_UpsertFremdeUserId verifies 403 when forUserId is not a child.
// Der Parent ist eigenes Member im Team (Team-Visibility), versucht aber für
// einen Fremden zu schreiben → 403 (Ownership-Verstoß, nicht 404).
func TestElternteil_UpsertFremdeUserId(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID := testutil.CreateUser(t, db, "standard")
	// Parent in Team-Kader, damit event-team-visibility 200 erlaubt — der 403
	// muss aus dem Ownership-Check kommen, nicht aus fehlender Sichtbarkeit.
	parentMemberID := testutil.CreateMember(t, db, parentID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, parentMemberID)

	otherID := testutil.CreateUser(t, db, "standard")
	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrgelegenheiten", token, map[string]any{
		"gameId":    gameID,
		"typ":       "suche",
		"forUserId": otherID,
		"plaetze":   1,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// TestElternteil_DeleteKindEintrag verifies a parent can delete their child's entry.
func TestElternteil_DeleteKindEintrag(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID, childUserID, _ := setupParentChild(t, db)
	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	// Child creates entry directly in DB
	res2, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, childUserID)
	entryID, _ := res2.LastInsertId()

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/mitfahrgelegenheiten/"+itoa(int(entryID)), token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// TestElternteil_DeleteFremderEintrag verifies 403 when parent tries to delete unrelated entry.
func TestElternteil_DeleteFremderEintrag(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID := testutil.CreateUser(t, db, "standard")
	otherID := testutil.CreateUser(t, db, "standard")
	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	res2, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, otherID)
	entryID, _ := res2.LastInsertId()

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/mitfahrgelegenheiten/"+itoa(int(entryID)), token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// TestElternteil_PaarungsanfrageKind verifies a parent can request a pairing for their child's suche entry.
func TestElternteil_PaarungsanfrageKind(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID, childUserID, _ := setupParentChild(t, db)
	bieterID := testutil.CreateUser(t, db, "standard")
	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	// Bieter creates biete entry
	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()

	// Kind creates suche entry
	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, childUserID)
	sucheID, _ := sucheRes.LastInsertId()

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{
		"bieteId": bieteID,
		"sucheId": sucheID,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// TestElternteil_PaarungsanfrageKeinBezug verifies 403 when no relation to either entry.
func TestElternteil_PaarungsanfrageKeinBezug(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID := testutil.CreateUser(t, db, "standard")
	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")
	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, userA)
	bieteID, _ := bieteRes.LastInsertId()
	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, userB)
	sucheID, _ := sucheRes.LastInsertId()

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{
		"bieteId": bieteID,
		"sucheId": sucheID,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// TestElternteil_ConfirmPaarungFuerKind verifies a parent can confirm a pairing for their child.
func TestElternteil_ConfirmPaarungFuerKind(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID, childUserID, _ := setupParentChild(t, db)
	sucherID := testutil.CreateUser(t, db, "standard")
	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	// Kind ist Bieter
	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, childUserID)
	bieteID, _ := bieteRes.LastInsertId()
	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, sucherID)
	sucheID, _ := sucheRes.LastInsertId()

	// Sucher initiiert Paarungsanfrage (initiiert_von='suche')
	paarRes, _ := db.Exec(`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von) VALUES (?, ?, 'suche')`, bieteID, sucheID)
	paarID, _ := paarRes.LastInsertId()

	// Elternteil bestätigt im Namen des Kindes (Kind wäre die Gegenseite als Bieter)
	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen/"+itoa(int(paarID))+"/confirm", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

// TestElternteil_IsOwnFuerKindEintrag verifies isOwn=true for a child's entry when parent lists.
func TestElternteil_IsOwnFuerKindEintrag(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID, childUserID, childMemberID := setupParentChild(t, db)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)
	db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, childUserID)

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServerHTTP(t, h)

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Get(t, srv, "/api/mitfahrgelegenheiten", token)
	defer res.Body.Close()

	var body struct {
		Games []struct {
			Suche []struct {
				IsOwn bool `json:"isOwn"`
			} `json:"suche"`
		} `json:"games"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if len(body.Games) == 0 || len(body.Games[0].Suche) == 0 {
		t.Fatal("expected at least one suche entry")
	}
	if !body.Games[0].Suche[0].IsOwn {
		t.Error("expected isOwn=true for child entry when viewed by parent")
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
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
	matchesPair := (g.TeamIDs[0] == teamA && g.TeamIDs[1] == teamB) ||
		(g.TeamIDs[0] == teamB && g.TeamIDs[1] == teamA)
	if !matchesPair {
		t.Errorf("teamIds %v does not contain both teamA=%d and teamB=%d", g.TeamIDs, teamA, teamB)
	}
}
