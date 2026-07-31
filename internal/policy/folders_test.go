package policy_test

import (
	"database/sql"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/policy"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func mkFolder(t *testing.T, db *sql.DB, name string, parentID *int, createdBy int) int {
	t.Helper()
	var parentVal any
	if parentID != nil {
		parentVal = *parentID
	}
	res, err := db.Exec(`INSERT INTO file_folders (name, parent_id, created_by) VALUES (?, ?, ?)`,
		name, parentVal, createdBy)
	if err != nil {
		t.Fatalf("mkFolder: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func mkPerm(t *testing.T, db *sql.DB, folderID int, pt, pr string, canRead, canWrite int) {
	t.Helper()
	var ref any
	if pr != "" {
		ref = pr
	}
	_, err := db.Exec(
		`INSERT INTO folder_permissions (folder_id, principal_type, principal_ref, can_read, can_write) VALUES (?, ?, ?, ?, ?)`,
		folderID, pt, ref, canRead, canWrite)
	if err != nil {
		t.Fatalf("mkPerm: %v", err)
	}
}

func folderSpielerP(userID int) *policy.Principal {
	return &policy.Principal{UserID: userID, Role: "standard", ClubFunctions: []string{"spieler"}}
}

// TestCanReadFolder_NoACL: Spieler without any ACL entry is denied.
func TestCanReadFolder_NoACL(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folderID := mkFolder(t, db, "private", nil, adminID)

	p := folderSpielerP(userID)
	if policy.CanReadFolder(db, p, folderID) {
		t.Error("spieler without ACL should be denied")
	}
}

// TestCanReadFolder_WithACL: Spieler granted explicit read access is allowed.
func TestCanReadFolder_WithACL(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folderID := mkFolder(t, db, "team-docs", nil, adminID)
	mkPerm(t, db, folderID, "user", strconv.Itoa(userID), 1, 0)

	p := folderSpielerP(userID)
	if !policy.CanReadFolder(db, p, folderID) {
		t.Error("spieler with explicit read ACL should be allowed")
	}
}

// TestCanReadFolder_ClubFunction: club_function ACL grants access to all members with that function.
func TestCanReadFolder_ClubFunction(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folderID := mkFolder(t, db, "spieler-docs", nil, adminID)
	mkPerm(t, db, folderID, "club_function", "spieler", 1, 0)

	p := folderSpielerP(userID)
	if !policy.CanReadFolder(db, p, folderID) {
		t.Error("spieler with club_function ACL should be allowed")
	}
}

// TestCanReadFolder_Admin: admin always has read access regardless of ACL.
func TestCanReadFolder_Admin(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	folderID := mkFolder(t, db, "secret", nil, adminID)

	p := &policy.Principal{UserID: adminID, Role: "admin", ClubFunctions: []string{}}
	if !policy.CanReadFolder(db, p, folderID) {
		t.Error("admin should always have read access")
	}
}

// ─── Owner precedence ────────────────────────────────────────────────────────

func plainP(userID int) *policy.Principal {
	return &policy.Principal{UserID: userID, Role: "standard", ClubFunctions: []string{}}
}

func assertAccess(t *testing.T, db *sql.DB, p *policy.Principal, folderID int, wantRead, wantWrite bool, msg string) {
	t.Helper()
	cr, cw, err := policy.FolderAccess(db, p, folderID)
	if err != nil {
		t.Fatalf("%s: FolderAccess: %v", msg, err)
	}
	if cr != wantRead || cw != wantWrite {
		t.Errorf("%s: got read=%v write=%v, want read=%v write=%v", msg, cr, cw, wantRead, wantWrite)
	}
}

// TestFolderAccess_OwnerKeepsRightsAfterGrantingToOthers is the regression test for
// the reported bug: granting the first ACL entry on a fresh folder used to cut the
// creator's inherited rights, locking them out of their own folder.
func TestFolderAccess_OwnerKeepsRightsAfterGrantingToOthers(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	ownerID := testutil.CreateUser(t, db, "standard")
	otherID := testutil.CreateUser(t, db, "standard")

	parentID := mkFolder(t, db, "Trainer", nil, adminID)
	mkPerm(t, db, parentID, "club_function", "trainer", 1, 1)
	childID := mkFolder(t, db, "Florians-Ordner", &parentID, ownerID)

	owner := &policy.Principal{UserID: ownerID, Role: "standard", ClubFunctions: []string{"trainer"}}
	assertAccess(t, db, owner, childID, true, true, "before granting (inherited from parent)")

	mkPerm(t, db, childID, "user", strconv.Itoa(otherID), 1, 1)

	assertAccess(t, db, owner, childID, true, true, "after granting to a third party")
}

// TestFolderAccess_OwnerRightsSpanSubtree: ownership of an ancestor is enough, so a
// subfolder created by someone else inside the owner's folder stays reachable.
func TestFolderAccess_OwnerRightsSpanSubtree(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	ownerID := testutil.CreateUser(t, db, "standard")
	otherID := testutil.CreateUser(t, db, "standard")

	rootID := mkFolder(t, db, "Trainer", nil, adminID)
	mkPerm(t, db, rootID, "everyone", "", 1, 1)
	xID := mkFolder(t, db, "X", &rootID, ownerID)
	yID := mkFolder(t, db, "X-Y", &xID, otherID)
	mkPerm(t, db, yID, "user", strconv.Itoa(otherID), 1, 1)

	assertAccess(t, db, plainP(ownerID), yID, true, true, "owner of ancestor X on subfolder X/Y")
}

// TestFolderAccess_OwnerWithoutClubFunction: the owner right is absolute — it does
// not depend on role, club function or any matching ACL entry.
func TestFolderAccess_OwnerWithoutClubFunction(t *testing.T) {
	db := testutil.NewDB(t)
	ownerID := testutil.CreateUser(t, db, "standard")

	folderID := mkFolder(t, db, "eigener", nil, ownerID)
	mkPerm(t, db, folderID, "club_function", "vorstand", 1, 1)

	assertAccess(t, db, plainP(ownerID), folderID, true, true, "owner without any matching principal")
}

// TestFolderAccess_NonOwnerNoAccess guards the short-circuit against being too
// broad: someone who created nothing in the path gets nothing.
func TestFolderAccess_NonOwnerNoAccess(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	ownerID := testutil.CreateUser(t, db, "standard")
	strangerID := testutil.CreateUser(t, db, "standard")

	folderID := mkFolder(t, db, "fremd", nil, adminID)
	mkPerm(t, db, folderID, "user", strconv.Itoa(ownerID), 1, 1)

	assertAccess(t, db, plainP(strangerID), folderID, false, false, "stranger on a foreign folder")
}

// ─── Team principals ─────────────────────────────────────────────────────────

// mkTeamKader creates a team plus a kader in the given season.
func mkTeamKader(t *testing.T, db *sql.DB, name string, seasonID int) (teamID, kaderID int) {
	t.Helper()
	teamID = testutil.CreateTeam(t, db, name)
	kaderID = testutil.CreateKader(t, db, teamID, seasonID)
	return teamID, kaderID
}

// mkTeamFolder creates an admin-owned folder carrying a single team-typed ACL entry.
// Admin ownership matters: a folder owned by the principal under test would be
// granted through owner precedence and mask the team resolution entirely.
func mkTeamFolder(t *testing.T, db *sql.DB, adminID, teamID int, principalType string) int {
	t.Helper()
	folderID := mkFolder(t, db, "team-docs", nil, adminID)
	mkPerm(t, db, folderID, principalType, strconv.Itoa(teamID), 1, 1)
	return folderID
}

func TestFolderAccess_TeamPlayerMatches(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID, kaderID := mkTeamKader(t, db, "mA", seasonID)
	testutil.AddKaderMember(t, db, kaderID, memberID)

	folderID := mkTeamFolder(t, db, adminID, teamID, "team")
	assertAccess(t, db, plainP(userID), folderID, true, true, "player of the team")
}

func TestFolderAccess_TeamTrainerMatches(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID, kaderID := mkTeamKader(t, db, "mA", seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, memberID)

	folderID := mkTeamFolder(t, db, adminID, teamID, "team")
	assertAccess(t, db, plainP(userID), folderID, true, true, "trainer of the team")
}

func TestFolderAccess_TeamExtendedMemberMatches(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID, kaderID := mkTeamKader(t, db, "mA", seasonID)
	testutil.AddExtendedKaderMember(t, db, kaderID, memberID)

	folderID := mkTeamFolder(t, db, adminID, teamID, "team")
	assertAccess(t, db, plainP(userID), folderID, true, true, "extended-squad member of the team")
}

// TestFolderAccess_TeamParentNotMatchedByTeam: 'team' and 'team_parents' are
// separately controllable, so a parent must not slip in through the team entry.
func TestFolderAccess_TeamParentNotMatchedByTeam(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	childUserID := testutil.CreateUser(t, db, "standard")
	parentUserID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, childUserID)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID, kaderID := mkTeamKader(t, db, "mA", seasonID)
	testutil.AddKaderMember(t, db, kaderID, memberID)
	testutil.AddFamilyLink(t, db, parentUserID, memberID)

	folderID := mkTeamFolder(t, db, adminID, teamID, "team")
	assertAccess(t, db, plainP(parentUserID), folderID, false, false, "parent against a 'team' entry")
}

func TestFolderAccess_TeamParentsMatches(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	childUserID := testutil.CreateUser(t, db, "standard")
	parentUserID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, childUserID)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID, kaderID := mkTeamKader(t, db, "mA", seasonID)
	testutil.AddKaderMember(t, db, kaderID, memberID)
	testutil.AddFamilyLink(t, db, parentUserID, memberID)

	folderID := mkTeamFolder(t, db, adminID, teamID, "team_parents")
	assertAccess(t, db, plainP(parentUserID), folderID, true, true, "parent against a 'team_parents' entry")
}

func TestFolderAccess_TeamParentsDoesNotMatchPlayer(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID, kaderID := mkTeamKader(t, db, "mA", seasonID)
	testutil.AddKaderMember(t, db, kaderID, memberID)

	folderID := mkTeamFolder(t, db, adminID, teamID, "team_parents")
	assertAccess(t, db, plainP(userID), folderID, false, false, "player against a 'team_parents' entry")
}

func TestFolderAccess_TeamOtherTeamNoAccess(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	grantedTeamID, _ := mkTeamKader(t, db, "mA", seasonID)
	_, otherKaderID := mkTeamKader(t, db, "wB", seasonID)
	testutil.AddKaderMember(t, db, otherKaderID, memberID)

	folderID := mkTeamFolder(t, db, adminID, grantedTeamID, "team")
	assertAccess(t, db, plainP(userID), folderID, false, false, "member of a different team")
}

// TestFolderAccess_TeamInactiveSeasonNoAccess: squad membership is resolved against
// the active season only, so last season's squad loses access automatically.
func TestFolderAccess_TeamInactiveSeasonNoAccess(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	oldSeasonID := testutil.CreateSeason(t, db, "2024/25")
	teamID, oldKaderID := mkTeamKader(t, db, "mA", oldSeasonID)
	testutil.AddKaderMember(t, db, oldKaderID, memberID)

	// Creating the next season deactivates the previous one.
	testutil.CreateSeason(t, db, "2025/26")

	folderID := mkTeamFolder(t, db, adminID, teamID, "team")
	assertAccess(t, db, plainP(userID), folderID, false, false, "squad member of an inactive season")
}

// TestFolderAccess_TeamNoActiveSeasonFailsClosed: without an active season the team
// sets stay empty rather than matching everyone.
func TestFolderAccess_TeamNoActiveSeasonFailsClosed(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID, kaderID := mkTeamKader(t, db, "mA", seasonID)
	testutil.AddKaderMember(t, db, kaderID, memberID)
	if _, err := db.Exec(`UPDATE seasons SET is_active = 0`); err != nil {
		t.Fatalf("deactivate seasons: %v", err)
	}

	folderID := mkTeamFolder(t, db, adminID, teamID, "team")
	assertAccess(t, db, plainP(userID), folderID, false, false, "no active season")
}
