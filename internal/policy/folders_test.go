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
