package games_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Tests für die event-team-visibility-Regel an den HTTP-Routen.

// TestListGames_FilterEigeneTeams: ein Standard-Spieler in Team A sieht nur
// Spiele mit Team A in `game_teams`, nicht Spiele anderer Teams.
func TestListGames_FilterEigeneTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)

	ownGameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-05-01")
	otherGameID := testutil.CreateGame(t, db, seasonID, teamB, "2026-05-02")

	playerUID := testutil.CreateUser(t, db, "standard")
	playerMID := testutil.CreateMember(t, db, playerUID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, playerMID)

	srv := testServer(t, db)
	token := testutil.Token(t, playerUID, "standard", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d", seasonID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	if len(games) != 1 {
		t.Fatalf("Spieler in Team A: erwartet 1 Spiel, got %d", len(games))
	}
	if int(games[0]["id"].(float64)) != ownGameID {
		t.Errorf("erwartet ownGameID=%d, got %v (other=%d)", ownGameID, games[0]["id"], otherGameID)
	}
}

// TestListGames_ElternSiehtTeamsDerKinder: Elternteil (kein eigenes Member)
// eines Kindes in Team A sieht Team-A-Spiele.
func TestListGames_ElternSiehtTeamsDerKinder(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)

	ownGameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-05-01")
	testutil.CreateGame(t, db, seasonID, teamB, "2026-05-02")

	childMID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, childMID)

	parentUID := testutil.CreateUser(t, db, "standard")
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUID, childMID)

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUID, "standard", nil, true)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d", seasonID), token)
	defer res.Body.Close()
	var games []map[string]any
	json.NewDecoder(res.Body).Decode(&games)
	if len(games) != 1 {
		t.Fatalf("Elternteil: erwartet 1 Spiel (Kind-Team A), got %d", len(games))
	}
	if int(games[0]["id"].(float64)) != ownGameID {
		t.Errorf("erwartet Kind-Team-Spiel, got %v", games[0]["id"])
	}
}

// TestGetGame_FremdEvent_404: Direkter ID-Zugriff auf fremdes Game → 404.
func TestGetGame_FremdEvent_404(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)

	playerUID := testutil.CreateUser(t, db, "standard")
	playerMID := testutil.CreateMember(t, db, playerUID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, playerMID)

	otherGameID := testutil.CreateGame(t, db, seasonID, teamB, "2026-05-02")

	srv := testServer(t, db)
	token := testutil.Token(t, playerUID, "standard", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", otherGameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Fremd-Game: erwartet 404, got %d", res.StatusCode)
	}
}

// TestGetGame_EigenesEvent_200: Spieler in Team A öffnet eigenes Game → 200.
func TestGetGame_EigenesEvent_200(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)

	playerUID := testutil.CreateUser(t, db, "standard")
	playerMID := testutil.CreateMember(t, db, playerUID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, playerMID)

	gameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-05-01")

	srv := testServer(t, db)
	token := testutil.Token(t, playerUID, "standard", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", gameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Eigenes Game: erwartet 200, got %d", res.StatusCode)
	}
}

// TestGetParticipants_FremdEvent_404: 404 statt 200 + leerer Liste.
func TestGetParticipants_FremdEvent_404(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)

	playerUID := testutil.CreateUser(t, db, "standard")
	playerMID := testutil.CreateMember(t, db, playerUID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, playerMID)

	otherGameID := testutil.CreateGame(t, db, seasonID, teamB, "2026-05-02")

	srv := testServer(t, db)
	token := testutil.Token(t, playerUID, "standard", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", otherGameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("/participants Fremd-Game: erwartet 404, got %d", res.StatusCode)
	}
}
