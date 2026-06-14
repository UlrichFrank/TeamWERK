package testutil

import (
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

var counter atomic.Int64

func nextID() int64 { return counter.Add(1) }

// CreateUser inserts a user with the given system role and returns its ID.
func CreateUser(t *testing.T, database *sql.DB, role string) int {
	t.Helper()
	email := fmt.Sprintf("user%d@test.local", nextID())
	hash, err := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("CreateUser bcrypt: %v", err)
	}
	res, err := database.Exec(`INSERT INTO users (email, password, role) VALUES (?, ?, ?)`,
		email, string(hash), role)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreateTeam inserts a team and returns its ID.
func CreateTeam(t *testing.T, database *sql.DB, name string) int {
	t.Helper()
	res, err := database.Exec(
		`INSERT INTO teams (name, age_class, gender) VALUES (?, ?, ?)`,
		name, "Erwachsene", "mixed")
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreateSeason inserts an active season, deactivating any previous active season.
func CreateSeason(t *testing.T, database *sql.DB, name string) int {
	t.Helper()
	database.Exec(`UPDATE seasons SET is_active=0`)
	res, err := database.Exec(
		`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES (?, ?, ?, 1)`,
		name, "2025-09-01", "2026-06-30")
	if err != nil {
		t.Fatalf("CreateSeason: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreateMember inserts a member linked to the given userID and returns its ID.
// Pass userID=0 to create a member without a user link (NULL).
func CreateMember(t *testing.T, database *sql.DB, userID int) int {
	t.Helper()
	n := nextID()
	var userArg any
	if userID > 0 {
		userArg = userID
	}
	res, err := database.Exec(
		`INSERT INTO members (first_name, last_name, status, user_id) VALUES (?, ?, ?, ?)`,
		"Test", fmt.Sprintf("Member%d", n), "aktiv", userArg)
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreateKader inserts a kader for the given team and season, returns its ID.
func CreateKader(t *testing.T, database *sql.DB, teamID, seasonID int) int {
	t.Helper()
	var maxNum int
	database.QueryRow(
		`SELECT COALESCE(MAX(team_number), 0) FROM kader WHERE season_id=? AND age_class=? AND gender=?`,
		seasonID, "Erwachsene", "mixed").Scan(&maxNum)
	res, err := database.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "Erwachsene", "mixed", teamID, maxNum+1)
	if err != nil {
		t.Fatalf("CreateKader: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// AddKaderTrainer links a member to a kader as trainer.
func AddKaderTrainer(t *testing.T, database *sql.DB, kaderID, memberID int) {
	t.Helper()
	_, err := database.Exec(`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID)
	if err != nil {
		t.Fatalf("AddKaderTrainer: %v", err)
	}
}

// CreateTrainingSeries inserts a minimal training series and returns its ID.
func CreateTrainingSeries(t *testing.T, database *sql.DB, teamID, seasonID, createdByUserID int) int {
	t.Helper()
	res, err := database.Exec(
		`INSERT INTO training_series (team_id, season_id, name, day_of_week, start_time, end_time, valid_from, valid_until, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		teamID, seasonID, "Test Series", 2, "18:00", "20:00", "2025-10-01", "2026-06-30", createdByUserID)
	if err != nil {
		t.Fatalf("CreateTrainingSeries: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreateTrainingSession inserts a single training session and returns its ID.
// date must be in "2006-01-02" format.
func CreateTrainingSession(t *testing.T, database *sql.DB, teamID, seasonID int, date string) int {
	t.Helper()
	res, err := database.Exec(
		`INSERT INTO training_sessions (team_id, season_id, date, start_time, end_time, title)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		teamID, seasonID, date, "18:00", "20:00", "Test Session")
	if err != nil {
		t.Fatalf("CreateTrainingSession: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreateGame inserts a game and its team link, returns the game ID.
func CreateGame(t *testing.T, database *sql.DB, seasonID, teamID int, date string) int {
	t.Helper()
	res, err := database.Exec(
		`INSERT INTO games (season_id, opponent, date, time, event_type, is_home) VALUES (?, ?, ?, ?, ?, ?)`,
		seasonID, "Test Opponent", date, "18:00", "heim", 1)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	gameID, _ := res.LastInsertId()
	_, err = database.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamID)
	if err != nil {
		t.Fatalf("CreateGame game_teams: %v", err)
	}
	return int(gameID)
}

// CreateDutyType inserts a duty type and returns its ID.
func CreateDutyType(t *testing.T, database *sql.DB, name string, hoursValue float64) int {
	t.Helper()
	res, err := database.Exec(`INSERT INTO duty_types (name, hours_value) VALUES (?, ?)`, name, hoursValue)
	if err != nil {
		t.Fatalf("CreateDutyType: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreateDutySlot inserts a duty slot (slots_total=2, slots_filled=0) and returns its ID.
// Pass gameID=0 to omit the game FK (NULL).
func CreateDutySlot(t *testing.T, database *sql.DB, dutyTypeID, seasonID, teamID, gameID int, date string) int {
	t.Helper()
	var gameArg any
	if gameID > 0 {
		gameArg = gameID
	}
	res, err := database.Exec(
		`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, slots_filled, team_id, season_id, game_id)
		 VALUES (?, ?, ?, 2, 0, ?, ?, ?)`,
		"Testdienst", date, dutyTypeID, teamID, seasonID, gameArg)
	if err != nil {
		t.Fatalf("CreateDutySlot: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreateInvitationToken inserts an invitation token and returns the plain-text token.
// Pass a past time for expiresAt to create an expired token.
func CreateInvitationToken(t *testing.T, database *sql.DB, email, role string, expiresAt time.Time) string {
	t.Helper()
	plain, hash, err := auth.GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("CreateInvitationToken generate: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO invitation_tokens (email, role, token, expires_at, first_name, last_name) VALUES (?, ?, ?, ?, ?, ?)`,
		email, role, hash, expiresAt, "Test", "User"); err != nil {
		t.Fatalf("CreateInvitationToken insert: %v", err)
	}
	return plain
}

// CreatePasswordResetToken inserts a password-reset token and returns the plain-text token.
func CreatePasswordResetToken(t *testing.T, database *sql.DB, userID int, expiresAt time.Time) string {
	t.Helper()
	plain, hash, err := auth.GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("CreatePasswordResetToken generate: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES (?, ?, ?)`,
		userID, hash, expiresAt); err != nil {
		t.Fatalf("CreatePasswordResetToken insert: %v", err)
	}
	return plain
}

// CreateRefreshToken inserts a hashed refresh token for the user and returns the plain token.
func CreateRefreshToken(t *testing.T, database *sql.DB, userID int) string {
	t.Helper()
	plain, hash, err := auth.GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("CreateRefreshToken generate: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		userID, hash, time.Now().Add(7*24*time.Hour)); err != nil {
		t.Fatalf("CreateRefreshToken insert: %v", err)
	}
	return plain
}

// CreatePlayerMembership links a member to a team for the given season.
func CreatePlayerMembership(t *testing.T, database *sql.DB, memberID, teamID, seasonID int) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT OR IGNORE INTO player_memberships (member_id, team_id, season_id) VALUES (?, ?, ?)`,
		memberID, teamID, seasonID); err != nil {
		t.Fatalf("CreatePlayerMembership: %v", err)
	}
}

// AddExtendedKaderMember adds a member to the extended kader (kader_extended_members).
func AddExtendedKaderMember(t *testing.T, database *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT OR IGNORE INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("AddExtendedKaderMember: %v", err)
	}
}
