package games_test

import (
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestSaveLineup_TrainerFremdesTeam403: ein Trainer, dessen Kader an dem Spiel
// nicht beteiligt ist, darf die Aufstellung nicht speichern. Vor dem Objekt-Gate
// reichte die bloße Vereinsfunktion `trainer` — jeder Trainer konnte die
// Aufstellung jedes Spiels im Verein überschreiben.
func TestSaveLineup_TrainerFremdesTeam403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-05-01")

	// Trainer gehört zu Team B, das Spiel gehört Team A.
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderB, trainerMemberID)
	db.Exec(`INSERT INTO trainer_memberships (member_id, team_id, season_id) VALUES (?, ?, ?)`,
		trainerMemberID, teamB, seasonID)

	// Spieler von Team A, den der fremde Trainer aufstellen möchte.
	playerMemberID := testutil.CreateMember(t, db, 0)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, "/api/games/"+itoa(gameID)+"/lineup", token,
		map[string]any{"member_ids": []int{playerMemberID}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for trainer of an uninvolved team, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM game_lineup WHERE game_id=?`, gameID).Scan(&count)
	if count != 0 {
		t.Errorf("expected no lineup rows, got %d", count)
	}
}

// TestSaveLineup_TrainerEigenesTeamOK: der Trainer eines beteiligten Teams
// speichert wie bisher (204, Zeilen in game_lineup).
func TestSaveLineup_TrainerEigenesTeamOK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)

	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)
	db.Exec(`INSERT INTO trainer_memberships (member_id, team_id, season_id) VALUES (?, ?, ?)`,
		trainerMemberID, teamID, seasonID)

	playerMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, playerMemberID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})
	res := testutil.Post(t, srv, "/api/games/"+itoa(gameID)+"/lineup", token,
		map[string]any{"member_ids": []int{playerMemberID}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM game_lineup WHERE game_id=? AND member_id=?`,
		gameID, playerMemberID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 lineup row, got %d", count)
	}
}

// TestSaveLineup_AdminUndSportlicheLeitungVereinsweit: admin und sportliche
// Leitung bleiben ohne Team-Bezug erlaubt (gleiche Regel wie bei der
// Anwesenheitserfassung, canRecordGameAttendance).
func TestSaveLineup_AdminUndSportlicheLeitungVereinsweit(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")

	playerMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, playerMemberID)

	srv := testServer(t, db)

	adminUserID := testutil.CreateUser(t, db, "admin")
	res := testutil.Post(t, srv, "/api/games/"+itoa(gameID)+"/lineup",
		testutil.Token(t, adminUserID, "admin", nil),
		map[string]any{"member_ids": []int{playerMemberID}})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("admin: expected 204, got %d", res.StatusCode)
	}

	slUserID := testutil.CreateUser(t, db, "standard")
	res = testutil.Post(t, srv, "/api/games/"+itoa(gameID)+"/lineup",
		testutil.Token(t, slUserID, "standard", []string{"sportliche_leitung"}),
		map[string]any{"member_ids": []int{playerMemberID}})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("sportliche_leitung: expected 204, got %d", res.StatusCode)
	}
}
