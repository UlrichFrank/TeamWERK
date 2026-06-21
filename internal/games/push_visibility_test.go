package games_test

import (
	"context"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Diese Tests dokumentieren die Push-Sichtbarkeits-Invariante (event-team-
// visibility, Requirement „Push-Notifications synchron mit Event-Sichtbarkeit"):
// Der existierende Empfänger-Berechner `teamMembersAndParents` liefert
// player_memberships + family_links — definitionsgemäß ein Subset der
// visibility-berechtigten User. Funktionsträger sind durch den Bypass-Pfad in
// `auth.UserCanSeeGame` ebenfalls visibility-berechtigt; sie werden in
// inhaltlich gerichteten Pushes (z. B. „Aufstellung geändert") weiterhin über
// ihre bestehenden Inhalts-Filter adressiert.
//
// Diese Datei prüft die Invariante an einer konkreten Konstellation; sie
// braucht keine HTTP-Schicht, weil sie nur die Helper kombiniert.

// TestPush_FremdEventKeinEmpfaenger: User, der weder im Game-Team spielt noch
// Funktionsträger ist, ist NICHT in `usersWithAccessToGame(gameID)` und darf
// daher keine Game-Push erhalten.
func TestPush_FremdEventKeinEmpfaenger(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)

	playerA := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateMember(t, db, playerA)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, memberA)

	playerB := testutil.CreateUser(t, db, "standard")
	memberB := testutil.CreateMember(t, db, playerB)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderB, memberB)

	gameA := testutil.CreateGame(t, db, seasonID, teamA, "2026-04-04")

	// PlayerB hat keine Sichtbarkeit auf gameA und darf daher kein Push-Empfänger sein.
	ok, err := auth.UserCanSeeGame(context.Background(), db, playerB, gameA)
	if err != nil {
		t.Fatalf("UserCanSeeGame: %v", err)
	}
	if ok {
		t.Errorf("playerB darf gameA NICHT sehen — Push-Empfänger-Filter würde ihn ausschließen")
	}
}

// TestPush_TrainerImmerEmpfaenger: Ein Trainer (member_club_functions) ohne
// eigene Team-Mitgliedschaft hat dennoch `UserCanSeeGame=true` (Bypass) — er
// fällt also nicht aus inhaltlich gerichteten Pushes heraus.
func TestPush_TrainerImmerEmpfaenger(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)

	teamA := testutil.CreateTeam(t, db, "Team A")
	gameA := testutil.CreateGame(t, db, seasonID, teamA, "2026-04-04")

	trainerUID := testutil.CreateUser(t, db, "standard")
	trainerMID := testutil.CreateMember(t, db, trainerUID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`, trainerMID, "trainer")

	ok, err := auth.UserCanSeeGame(context.Background(), db, trainerUID, gameA)
	if err != nil {
		t.Fatalf("UserCanSeeGame: %v", err)
	}
	if !ok {
		t.Errorf("Trainer muss gameA sehen können (Bypass), damit er als Push-Empfänger gilt")
	}
}
