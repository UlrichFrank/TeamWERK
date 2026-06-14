package scheduler

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Spieler-Auflösung: User mit Vereinsfunktion 'spieler' im aktiven Kader des Teams
// MUSS in der Empfängerliste auftauchen. Users.role spielt keine Rolle (war historisch
// 'spieler', heute 'standard').
func TestEligibleUsers_SpielerViaClubFunction(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	// Vereinsfunktion 'spieler'.
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, memberID); err != nil {
		t.Fatalf("insert club function: %v", err)
	}
	// In aktiven Saison-Kader aufnehmen.
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}

	// Slot mit target_role='spieler' für das Team.
	dutyTypeID := createDutyTypeWithTarget(t, db, "Hallendienst", "spieler")
	slotID := testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{
		id:         slotID,
		targetRole: "spieler",
		teamID:     sql.NullInt64{Int64: int64(teamID), Valid: true},
	})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if !containsUserID(users, userID) {
		t.Errorf("expected user %d (with spieler function in active kader) in recipients, got %+v", userID, users)
	}
}

// Eltern-Auflösung: User mit family_link zu einem Member mit Vereinsfunktion 'spieler'
// im aktiven Kader MUSS in der Empfängerliste auftauchen.
func TestEligibleUsers_ElternteilViaFamilyLinks(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

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

	dutyTypeID := createDutyTypeWithTarget(t, db, "Kuchenbacken", "elternteil")
	slotID := testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{
		id:         slotID,
		targetRole: "elternteil",
		teamID:     sql.NullInt64{Int64: int64(teamID), Valid: true},
	})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if !containsUserID(users, parentUserID) {
		t.Errorf("expected parent user %d in recipients, got %+v", parentUserID, users)
	}
}

// Negativfall: User mit role='standard' und ohne member_club_functions wird NICHT
// als Spieler-Empfänger gefunden (alte Fehlbehauptung „role='spieler'" gilt nicht mehr).
func TestEligibleUsers_SpielerSkipsUserWithoutClubFunction(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	// User+Member ohne Vereinsfunktion, aber im Kader.
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}

	dutyTypeID := createDutyTypeWithTarget(t, db, "Hallendienst", "spieler")
	slotID := testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{
		id:         slotID,
		targetRole: "spieler",
		teamID:     sql.NullInt64{Int64: int64(teamID), Valid: true},
	})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if containsUserID(users, userID) {
		t.Errorf("user without 'spieler' club function should NOT be a recipient, but was: %+v", users)
	}
}

func createDutyTypeWithTarget(t *testing.T, db *sql.DB, name, target string) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO duty_types (name, hours_value, target_role) VALUES (?, 1.0, ?)`, name, target)
	if err != nil {
		t.Fatalf("create duty_type: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func containsUserID(users []reminderUser, id int) bool {
	for _, u := range users {
		if u.id == id {
			return true
		}
	}
	return false
}
