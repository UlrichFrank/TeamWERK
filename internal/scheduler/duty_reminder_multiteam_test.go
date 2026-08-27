package scheduler

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// addPlayerToTeam legt einen User mit Vereinsfunktion 'spieler' im Kader des Teams
// (aktive Saison) an und gibt die User-ID zurück.
func addPlayerToTeam(t *testing.T, db *sql.DB, seasonID, teamID int) int {
	t.Helper()
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, memberID); err != nil {
		t.Fatalf("insert club function: %v", err)
	}
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}
	return userID
}

// createTeamlessSlot legt einen Dienst-Slot OHNE team_id an (optional mit game_id)
// und gibt die Slot-ID zurück. testutil.CreateDutySlot kann das nicht: es schreibt
// die übergebene Team-ID roh, aus 0 würde team_id=0 statt NULL.
func createTeamlessSlot(t *testing.T, db *sql.DB, dutyTypeID, seasonID, gameID int) int {
	t.Helper()
	var gameArg any
	if gameID > 0 {
		gameArg = gameID
	}
	res, err := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, slots_filled, team_id, season_id, game_id)
		 VALUES (?, ?, ?, 3, 0, NULL, ?, ?)`,
		"Turnier", "2026-07-01", dutyTypeID, seasonID, gameArg)
	if err != nil {
		t.Fatalf("insert teamless duty_slot: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// Ein Slot ohne team_id an einem Termin mit mehreren Kadern adressiert genau die Teams
// dieses Termins — nicht nur eines und nicht den ganzen Verein. Vor dem Fix fiel der
// fehlende team_id auf "alle Rollenträger vereinsweit" zurück.
func TestEligibleUsers_TeamlosSlotAmMehrTeamSpiel(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	teamB := testutil.CreateTeam(t, db, "B-Jugend")
	teamC := testutil.CreateTeam(t, db, "Unbeteiligte C-Jugend")

	playerA := addPlayerToTeam(t, db, seasonID, teamA)
	playerB := addPlayerToTeam(t, db, seasonID, teamB)
	playerC := addPlayerToTeam(t, db, seasonID, teamC)

	gameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-07-01")
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamB); err != nil {
		t.Fatalf("link second team: %v", err)
	}

	dutyTypeID := createDutyTypeWithTarget(t, db, "Hallendienst", "spieler")
	slotID := createTeamlessSlot(t, db, dutyTypeID, seasonID, gameID)

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{
		id:         slotID,
		targetRole: "spieler",
		gameID:     sql.NullInt64{Int64: int64(gameID), Valid: true},
	})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if !containsUserID(users, playerA) {
		t.Errorf("Spieler aus Team A (%d) fehlt in den Empfängern: %+v", playerA, users)
	}
	if !containsUserID(users, playerB) {
		t.Errorf("Spieler aus Team B (%d) fehlt in den Empfängern: %+v", playerB, users)
	}
	if containsUserID(users, playerC) {
		t.Errorf("Spieler aus unbeteiligtem Team C (%d) darf nicht benachrichtigt werden: %+v", playerC, users)
	}
}

// Dieselbe Auflösung für den Eltern-Pfad: Eltern beider beteiligten Teams werden
// erinnert, Eltern eines unbeteiligten Teams nicht.
func TestEligibleUsers_TeamlosSlotAmMehrTeamSpiel_Eltern(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	teamB := testutil.CreateTeam(t, db, "B-Jugend")
	teamC := testutil.CreateTeam(t, db, "Unbeteiligte C-Jugend")

	addParent := func(teamID int) int {
		parentUserID := testutil.CreateUser(t, db, "standard")
		childMemberID := testutil.CreateMember(t, db, 0)
		if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID); err != nil {
			t.Fatalf("insert family_link: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, childMemberID); err != nil {
			t.Fatalf("insert club function: %v", err)
		}
		kaderID := testutil.CreateKader(t, db, teamID, seasonID)
		if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID); err != nil {
			t.Fatalf("insert kader_member: %v", err)
		}
		return parentUserID
	}
	parentA, parentB, parentC := addParent(teamA), addParent(teamB), addParent(teamC)

	gameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-07-01")
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamB); err != nil {
		t.Fatalf("link second team: %v", err)
	}

	dutyTypeID := createDutyTypeWithTarget(t, db, "Kuchenbacken", "elternteil")
	slotID := createTeamlessSlot(t, db, dutyTypeID, seasonID, gameID)

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{
		id:         slotID,
		targetRole: "elternteil",
		gameID:     sql.NullInt64{Int64: int64(gameID), Valid: true},
	})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if !containsUserID(users, parentA) || !containsUserID(users, parentB) {
		t.Errorf("Eltern beider beteiligten Teams erwartet (A=%d, B=%d), got %+v", parentA, parentB, users)
	}
	if containsUserID(users, parentC) {
		t.Errorf("Elternteil aus unbeteiligtem Team C (%d) darf nicht benachrichtigt werden: %+v", parentC, users)
	}
}

// Regressionsschutz für den unveränderten Fall: ohne Team UND ohne Spiel (z.B.
// Vereinsfest) bleibt die Erinnerung bewusst vereinsweit.
func TestEligibleUsers_SlotOhneTeamUndOhneSpielBleibtVereinsweit(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	playerA := addPlayerToTeam(t, db, seasonID, teamA)

	dutyTypeID := createDutyTypeWithTarget(t, db, "Vereinsfest", "spieler")
	slotID := createTeamlessSlot(t, db, dutyTypeID, seasonID, 0)

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{id: slotID, targetRole: "spieler"})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if !containsUserID(users, playerA) {
		t.Errorf("vereinsweiter Slot muss Spieler %d erreichen, got %+v", playerA, users)
	}
}
