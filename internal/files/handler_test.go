package files

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// ── checkAntiEscalation: a grant may not exceed the caller's own rights ───────

func TestCheckAntiEscalation_AdminGrantsAnything(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	folder := mkFolder(t, db, "Root", nil, adminID)
	// No permission rows for the caller at all — admin bypasses regardless.
	ok, err := checkAntiEscalation(db, &auth.Claims{UserID: adminID, Role: "admin"}, folder, true, true)
	if err != nil {
		t.Fatalf("checkAntiEscalation: %v", err)
	}
	if !ok {
		t.Error("admin must be allowed to grant read+write on any folder")
	}
}

func TestCheckAntiEscalation_ReadWriteCallerGrantsReadWrite(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := mkFolder(t, db, "Root", nil, adminID)
	mkPerm(t, db, folder, "user", itoa(userID), 1, 1) // caller holds read+write

	ok, err := checkAntiEscalation(db, stdClaims(userID), folder, true, true)
	if err != nil {
		t.Fatalf("checkAntiEscalation: %v", err)
	}
	if !ok {
		t.Error("a read+write manager must be allowed to grant read+write")
	}
}

func TestCheckAntiEscalation_NoWriteCannotManage(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := mkFolder(t, db, "Root", nil, adminID)
	mkPerm(t, db, folder, "user", itoa(userID), 1, 0) // read-only caller

	ok, err := checkAntiEscalation(db, stdClaims(userID), folder, true, false)
	if err != nil {
		t.Fatalf("checkAntiEscalation: %v", err)
	}
	if ok {
		t.Error("a read-only caller must not be able to manage permissions (needs write)")
	}
}

// TestCheckAntiEscalation_WriteOnlyCannotGrantRead nails the closed gap: a caller who holds
// write but not read on a folder may still manage permissions, but must NOT be able to hand
// out read access they don't themselves have.
func TestCheckAntiEscalation_WriteOnlyCannotGrantRead(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := mkFolder(t, db, "Root", nil, adminID)
	mkPerm(t, db, folder, "user", itoa(userID), 0, 1) // write without read

	// Granting a read right the caller lacks → denied.
	ok, err := checkAntiEscalation(db, stdClaims(userID), folder, true, false)
	if err != nil {
		t.Fatalf("checkAntiEscalation: %v", err)
	}
	if ok {
		t.Error("write-without-read caller must not grant read access they don't hold")
	}

	// Granting only write (which the caller holds) → allowed.
	ok, err = checkAntiEscalation(db, stdClaims(userID), folder, false, true)
	if err != nil {
		t.Fatalf("checkAntiEscalation: %v", err)
	}
	if !ok {
		t.Error("write-without-read caller may still grant write they hold")
	}
}

// ── Route-level authz: folder/file CRUD + download-token ─────────────────────

func filesRouteServer(t *testing.T, h *Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/folders", h.CreateFolder)
		r.Delete("/api/folders/{id}", h.DeleteFolder)
		r.Post("/api/folders/{folderId}/files", h.UploadFile)
		r.Post("/api/folders/{id}/permissions", h.AddPermission)
		r.Delete("/api/folders/{id}/permissions/{permId}", h.DeletePermission)
		r.Get("/api/files/{id}/download-token", h.HandleDownloadToken)
	})
}

func TestCreateFolder_NoWriteForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	parent := testutil.CreateFolder(t, db, "Root", 0, adminID)
	testutil.SetFolderPermission(t, db, parent, "user", itoa(userID), true, false) // read only

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.Post(t, srv, "/api/folders", tok, map[string]any{"name": "Sub", "parent_id": parent})
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 creating subfolder without can_write, got %d", res.StatusCode)
	}
}

func TestCreateFolder_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	parent := testutil.CreateFolder(t, db, "Root", 0, adminID)
	testutil.SetFolderPermission(t, db, parent, "user", itoa(userID), true, true)

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.Post(t, srv, "/api/folders", tok, map[string]any{"name": "Sub", "parent_id": parent})
	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 creating subfolder with can_write, got %d", res.StatusCode)
	}
}

func TestDeleteFolder_NoWriteForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := testutil.CreateFolder(t, db, "Docs", 0, adminID)
	testutil.SetFolderPermission(t, db, folder, "user", itoa(userID), true, false) // read only

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.Delete(t, srv, "/api/folders/"+itoa(folder), tok)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 deleting folder without can_write, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM file_folders WHERE id = ?`, folder).Scan(&n)
	if n != 1 {
		t.Error("folder must not be deleted on forbidden request")
	}
}

func TestUploadFile_NoWriteForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := testutil.CreateFolder(t, db, "Docs", 0, adminID)
	testutil.SetFolderPermission(t, db, folder, "user", itoa(userID), true, false) // read only

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.PostMultipart(t, srv, "/api/folders/"+itoa(folder)+"/files", tok, "file", "note.txt", []byte("hello"))
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 uploading without can_write, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM files WHERE folder_id = ?`, folder).Scan(&n)
	if n != 0 {
		t.Error("no file may be persisted on forbidden upload")
	}
}

func TestUploadFile_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := testutil.CreateFolder(t, db, "Docs", 0, adminID)
	testutil.SetFolderPermission(t, db, folder, "user", itoa(userID), true, true)

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.PostMultipart(t, srv, "/api/folders/"+itoa(folder)+"/files", tok, "file", "note.txt", []byte("hello"))
	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 uploading with can_write, got %d", res.StatusCode)
	}
}

// TestAddPermission_EscalationForbidden exercises the read-escalation guard through the HTTP
// route (complements the checkAntiEscalation unit tests): a caller with can_write but not
// can_read must not be able to grant read access via the API.
func TestAddPermission_EscalationForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	otherID := testutil.CreateUser(t, db, "standard")
	folder := testutil.CreateFolder(t, db, "Docs", 0, adminID)
	testutil.SetFolderPermission(t, db, folder, "user", itoa(userID), false, true) // write without read

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.Post(t, srv, "/api/folders/"+itoa(folder)+"/permissions", tok, map[string]any{
		"principal_type": "user", "principal_ref": itoa(otherID), "can_read": true, "can_write": false,
	})
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 granting read without holding read, got %d", res.StatusCode)
	}
}

func TestDeletePermission_NoWriteForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := testutil.CreateFolder(t, db, "Docs", 0, adminID)
	testutil.SetFolderPermission(t, db, folder, "user", itoa(userID), true, false) // read only
	testutil.SetFolderPermission(t, db, folder, "everyone", "", true, false)
	var permID int
	db.QueryRow(`SELECT id FROM folder_permissions WHERE folder_id = ? AND principal_type = 'everyone'`, folder).Scan(&permID)

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.Delete(t, srv, "/api/folders/"+itoa(folder)+"/permissions/"+itoa(permID), tok)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 deleting permission without can_write, got %d", res.StatusCode)
	}
	var pc int
	db.QueryRow(`SELECT COUNT(*) FROM folder_permissions WHERE id=?`, permID).Scan(&pc)
	if pc != 1 {
		t.Error("permission row must survive a forbidden delete")
	}
}

func TestDownloadToken_NoReadForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := testutil.CreateFolder(t, db, "Restricted", 0, adminID)
	// folder has NO permission for userID → resolveAccess returns no read
	fileID := testutil.CreateFile(t, db, folder, adminID, "secret.pdf")

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/files/"+itoa(fileID)+"/download-token", tok)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 download-token without can_read (fail-closed), got %d", res.StatusCode)
	}
}

func TestDownloadToken_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, t.TempDir(), "test-secret")
	srv := filesRouteServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	userID := testutil.CreateUser(t, db, "standard")
	folder := testutil.CreateFolder(t, db, "Shared", 0, adminID)
	testutil.SetFolderPermission(t, db, folder, "user", itoa(userID), true, false)
	fileID := testutil.CreateFile(t, db, folder, adminID, "shared.pdf")

	tok := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/files/"+itoa(fileID)+"/download-token", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 download-token with can_read, got %d", res.StatusCode)
	}
	var body struct {
		Token string `json:"token"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if body.Token == "" {
		t.Error("expected a non-empty download token")
	}
}
