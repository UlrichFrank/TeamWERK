package upload_test

import (
	"database/sql"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// Nach unified-user-photo (Migration 029) landet jedes hochgeladene Profilbild
// in users.photo_path — auch wenn Eltern oder Admin über einen member-basierten
// Endpoint schreiben. Members ohne verknüpften User-Account bekommen HTTP 409.

func TestUploadChildPhoto_SchreibtUserPhotoPath(t *testing.T) {
	db := testutil.NewDB(t)
	parentU := testutil.CreateUser(t, db, "standard")
	childU := testutil.CreateUser(t, db, "standard")
	childM := testutil.CreateMember(t, db, childU)
	testutil.AddFamilyLink(t, db, parentU, childM)
	parentTok := testutil.TokenWithIsParent(t, parentU, "standard", nil, true)

	srv, _ := prodserver.NewWithHub(t, db)
	req := multipartPhoto(t, srv.URL+"/api/profile/kind/"+strconv.Itoa(childM)+"/photo", parentTok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}

	var photoPath sql.NullString
	if err := db.QueryRow(`SELECT photo_path FROM users WHERE id=?`, childU).Scan(&photoPath); err != nil {
		t.Fatalf("read users.photo_path: %v", err)
	}
	if !photoPath.Valid || photoPath.String == "" {
		t.Errorf("users.photo_path des Kind-Users sollte gesetzt sein, bekam %v", photoPath)
	}
}

func TestUploadChildPhoto_OhneAccount_409(t *testing.T) {
	db := testutil.NewDB(t)
	parentU := testutil.CreateUser(t, db, "standard")
	// Kind OHNE eigenen User-Account.
	childM := testutil.CreateMember(t, db, 0)
	testutil.AddFamilyLink(t, db, parentU, childM)
	parentTok := testutil.TokenWithIsParent(t, parentU, "standard", nil, true)

	srv, _ := prodserver.NewWithHub(t, db)
	req := multipartPhoto(t, srv.URL+"/api/profile/kind/"+strconv.Itoa(childM)+"/photo", parentTok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("erwartet 409 (kein Account), bekam %d", res.StatusCode)
	}
}

func TestDeleteChildPhoto_LoeschtUserPhotoPath(t *testing.T) {
	db := testutil.NewDB(t)
	parentU := testutil.CreateUser(t, db, "standard")
	childU := testutil.CreateUser(t, db, "standard")
	childM := testutil.CreateMember(t, db, childU)
	testutil.AddFamilyLink(t, db, parentU, childM)
	parentTok := testutil.TokenWithIsParent(t, parentU, "standard", nil, true)

	// Foto initial setzen (direkt in DB — Setup, kein UI-Round-Trip).
	if _, err := db.Exec(`UPDATE users SET photo_path='member-photos/preset.jpg' WHERE id=?`, childU); err != nil {
		t.Fatalf("seed photo: %v", err)
	}

	srv, _ := prodserver.NewWithHub(t, db)
	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/profile/kind/"+strconv.Itoa(childM)+"/photo", parentTok, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekam %d", res.StatusCode)
	}

	var photoPath sql.NullString
	db.QueryRow(`SELECT photo_path FROM users WHERE id=?`, childU).Scan(&photoPath)
	if photoPath.Valid && photoPath.String != "" {
		t.Errorf("users.photo_path sollte NULL sein, bekam %q", photoPath.String)
	}
}

func TestUploadMemberPhoto_SchreibtUserPhotoPath(t *testing.T) {
	db := testutil.NewDB(t)
	adminU := testutil.CreateUser(t, db, "admin")
	adminTok := testutil.Token(t, adminU, "admin", nil)
	targetU := testutil.CreateUser(t, db, "standard")
	targetM := testutil.CreateMember(t, db, targetU)

	srv, _ := prodserver.NewWithHub(t, db)
	req := multipartPhoto(t, srv.URL+"/api/upload/member-photo/"+strconv.Itoa(targetM), adminTok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}

	var photoPath sql.NullString
	db.QueryRow(`SELECT photo_path FROM users WHERE id=?`, targetU).Scan(&photoPath)
	if !photoPath.Valid || photoPath.String == "" {
		t.Errorf("users.photo_path des Ziel-Users sollte gesetzt sein, bekam %v", photoPath)
	}
}

func TestUploadMemberPhoto_OhneAccount_409(t *testing.T) {
	db := testutil.NewDB(t)
	adminU := testutil.CreateUser(t, db, "admin")
	adminTok := testutil.Token(t, adminU, "admin", nil)
	// Member ohne verknüpften User.
	targetM := testutil.CreateMember(t, db, 0)

	srv, _ := prodserver.NewWithHub(t, db)
	req := multipartPhoto(t, srv.URL+"/api/upload/member-photo/"+strconv.Itoa(targetM), adminTok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("erwartet 409 (kein Account), bekam %d", res.StatusCode)
	}
}

func TestUploadUserPhoto_UnveraendertesVerhalten(t *testing.T) {
	db := testutil.NewDB(t)
	userU := testutil.CreateUser(t, db, "standard")
	userTok := testutil.Token(t, userU, "standard", nil)

	srv, _ := prodserver.NewWithHub(t, db)
	req := multipartPhoto(t, srv.URL+"/api/upload/user-photo", userTok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}

	var photoPath sql.NullString
	db.QueryRow(`SELECT photo_path FROM users WHERE id=?`, userU).Scan(&photoPath)
	if !photoPath.Valid || photoPath.String == "" {
		t.Errorf("users.photo_path des eigenen Users sollte gesetzt sein, bekam %v", photoPath)
	}
}
