package files

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── test helpers ──────────────────────────────────────────────────────────────

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

func stdClaims(userID int) *auth.Claims {
	return &auth.Claims{UserID: userID, Role: "standard", ClubFunctions: []string{}}
}

// ── resolveAccess: nearest-ancestor-wins ─────────────────────────────────────

func TestResolveAccess_NearestAncestorWins(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")

	parent := mkFolder(t, db, "Root", nil, adminID)
	mkPerm(t, db, parent, "everyone", "", 1, 0)

	child := mkFolder(t, db, "Vorstand-Intern", &parent, adminID)
	mkPerm(t, db, child, "club_function", "vorstand", 1, 0)

	cr, cw, err := resolveAccess(db, stdClaims(userID), child)
	if err != nil {
		t.Fatalf("resolveAccess: %v", err)
	}
	if cr || cw {
		t.Errorf("standard user must not access restricted subfolder: canRead=%v canWrite=%v", cr, cw)
	}
}

func TestResolveAccess_InheritFromParent(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")

	parent := mkFolder(t, db, "Root", nil, adminID)
	mkPerm(t, db, parent, "everyone", "", 1, 0)

	child := mkFolder(t, db, "Allgemein", &parent, adminID)
	// no permissions on child — should inherit from parent

	cr, _, err := resolveAccess(db, stdClaims(userID), child)
	if err != nil {
		t.Fatalf("resolveAccess: %v", err)
	}
	if !cr {
		t.Error("standard user should inherit read from parent folder")
	}
}

func TestResolveAccess_NoRulesAnywhere(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")

	folder := mkFolder(t, db, "Orphan", nil, adminID)

	cr, cw, err := resolveAccess(db, stdClaims(userID), folder)
	if err != nil {
		t.Fatalf("resolveAccess: %v", err)
	}
	if cr || cw {
		t.Error("folder with no permissions should not be accessible")
	}
}

// ── resolveAccess: family context ────────────────────────────────────────────

func TestResolveAccess_FamilyContext_ClubFunction(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	parentUserID := testutil.CreateUser(t, db, "standard")

	// member with club_function=spieler (no own user account required for this test)
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`, childMemberID, "spieler")
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	folder := mkFolder(t, db, "Spieler-Bereich", nil, adminID)
	mkPerm(t, db, folder, "club_function", "spieler", 1, 0)

	cr, _, err := resolveAccess(db, stdClaims(parentUserID), folder)
	if err != nil {
		t.Fatalf("resolveAccess: %v", err)
	}
	if !cr {
		t.Error("Elternteil should inherit read via child's club_function=spieler")
	}
}

func TestResolveAccess_FamilyContext_UserID(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	parentUserID := testutil.CreateUser(t, db, "standard")
	childUserID := testutil.CreateUser(t, db, "standard")

	childMemberID := testutil.CreateMember(t, db, childUserID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID)

	folder := mkFolder(t, db, "Kind-Akte", nil, adminID)
	mkPerm(t, db, folder, "user", itoa(childUserID), 1, 0)

	cr, _, err := resolveAccess(db, stdClaims(parentUserID), folder)
	if err != nil {
		t.Fatalf("resolveAccess: %v", err)
	}
	if !cr {
		t.Error("Elternteil should inherit read via child's user_id in folder permission")
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// ── FolderContents: restricted subfolder ─────────────────────────────────────

func TestFolderContents_RestrictedSubfolder(t *testing.T) {
	db := testutil.NewDB(t)
	tmpDir := t.TempDir()
	h := NewHandler(db, tmpDir, "test-secret")

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")

	parent := mkFolder(t, db, "Root", nil, adminID)
	mkPerm(t, db, parent, "everyone", "", 1, 0)

	restricted := mkFolder(t, db, "Nur-Vorstand", &parent, adminID)
	mkPerm(t, db, restricted, "club_function", "vorstand", 1, 1)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/folders/{id}/contents", h.FolderContents)
	})

	tok := testutil.Token(t, userID, "standard", []string{})
	res := testutil.Get(t, srv, "/api/folders/"+itoa(restricted)+"/contents", tok)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

// ── ListPermissions: display_name ────────────────────────────────────────────

func TestListPermissions_DisplayName(t *testing.T) {
	db := testutil.NewDB(t)
	tmpDir := t.TempDir()
	h := NewHandler(db, tmpDir, "test-secret")

	adminID := testutil.CreateUser(t, db, "admin")

	// Ensure the admin user has a recognizable name
	db.Exec(`UPDATE users SET first_name = 'Max', last_name = 'Mustermann' WHERE id = ?`, adminID)

	folder := mkFolder(t, db, "Test", nil, adminID)
	mkPerm(t, db, folder, "everyone", "", 1, 0)
	mkPerm(t, db, folder, "user", itoa(adminID), 1, 0)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/folders/{id}/permissions", h.ListPermissions)
	})

	tok := testutil.Token(t, adminID, "admin", []string{})
	res := testutil.Get(t, srv, "/api/folders/"+itoa(folder)+"/permissions", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var perms []struct {
		PrincipalType string `json:"principal_type"`
		DisplayName   string `json:"display_name"`
	}
	json.NewDecoder(res.Body).Decode(&perms)
	res.Body.Close()

	for _, p := range perms {
		if p.PrincipalType == "user" {
			if p.DisplayName == "" {
				t.Error("display_name must be set for user permission entries")
			}
			return
		}
	}
	t.Error("no user-type permission found in response")
}
