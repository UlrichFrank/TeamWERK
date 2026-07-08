package upload_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// pngBytes is a minimal valid PNG magic-byte prefix; persistMultipartFile sniffs
// the first bytes and accepts it as image/png (no full decode).
var pngBytes = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x00}

// recvEvent reads one event from ch within d, or returns ("", false).
func recvEvent(ch chan string, d time.Duration) (string, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return "", false
	}
}

// multipartPhoto returns a ready-to-send *http.Request carrying a valid PNG in
// the "file" form field.
func multipartPhoto(t *testing.T, url, token string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "photo.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(pngBytes); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", token)
	return req
}

// TestUploadMemberPhoto_BroadcastsMembers verifies the upload route broadcasts
// the "members" event to the finance group AND the affected member's audience
// (its team roster + own linked user), matching how member fields already do.
func TestUploadMemberPhoto_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	// Admin performs the upload (member-photo route is admin-only) and is in the
	// finance group, so must receive the event.
	adminU := testutil.CreateUser(t, db, "admin")
	adminTok := testutil.Token(t, adminU, "admin", nil)

	// Target member: a player on team A with its own (non-finance) linked user.
	targetU := testutil.CreateUser(t, db, "standard")
	targetM := testutil.CreateMember(t, db, targetU)
	testutil.AddClubFunction(t, db, targetM, "spieler")
	testutil.AddKaderMember(t, db, kaderA, targetM)

	// Teammate on team A (non-finance) — sees the roster, must receive the event.
	teammateU := testutil.CreateUser(t, db, "standard")
	teammateM := testutil.CreateMember(t, db, teammateU)
	testutil.AddClubFunction(t, db, teammateM, "spieler")
	testutil.AddKaderMember(t, db, kaderA, teammateM)

	srv, sharedHub := prodserver.NewWithHub(t, db)

	adminCh := sharedHub.SubscribeUser(adminU)
	ownerCh := sharedHub.SubscribeUser(targetU)
	teammateCh := sharedHub.SubscribeUser(teammateU)

	req := multipartPhoto(t, srv.URL+"/api/upload/member-photo/"+strconv.Itoa(targetM), adminTok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("UploadMemberPhoto: expected 200, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"admin":    adminCh,
		"owner":    ownerCh,
		"teammate": teammateCh,
	} {
		if ev, ok := recvEvent(ch, time.Second); !ok || ev != "members" {
			t.Errorf("%s stream must receive 'members', got %q ok=%v", name, ev, ok)
		}
	}
}

// TestDeleteMemberPhoto_BroadcastsMembers verifies the delete route broadcasts
// "members" to the finance group and the affected member's audience.
func TestDeleteMemberPhoto_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	adminU := testutil.CreateUser(t, db, "admin")
	adminTok := testutil.Token(t, adminU, "admin", nil)

	targetU := testutil.CreateUser(t, db, "standard")
	targetM := testutil.CreateMember(t, db, targetU)
	testutil.AddClubFunction(t, db, targetM, "spieler")
	testutil.AddKaderMember(t, db, kaderA, targetM)

	srv, sharedHub := prodserver.NewWithHub(t, db)

	adminCh := sharedHub.SubscribeUser(adminU)
	ownerCh := sharedHub.SubscribeUser(targetU)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/upload/member-photo/"+strconv.Itoa(targetM), adminTok, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DeleteMemberPhoto: expected 204, got %d", res.StatusCode)
	}

	for name, ch := range map[string]chan string{
		"admin": adminCh,
		"owner": ownerCh,
	} {
		if ev, ok := recvEvent(ch, time.Second); !ok || ev != "members" {
			t.Errorf("%s stream must receive 'members', got %q ok=%v", name, ev, ok)
		}
	}
}

// TestUploadUserPhoto_BroadcastsMembers verifies a user photo upload broadcasts
// "members" reaching at least the acting user (whose member row surfaces the
// photo on rosters/profiles).
func TestUploadUserPhoto_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)

	userU := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, userU)
	userTok := testutil.Token(t, userU, "standard", nil)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	userCh := sharedHub.SubscribeUser(userU)

	req := multipartPhoto(t, srv.URL+"/api/upload/user-photo", userTok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("UploadUserPhoto: expected 200, got %d", res.StatusCode)
	}

	if ev, ok := recvEvent(userCh, time.Second); !ok || ev != "members" {
		t.Errorf("acting user stream must receive 'members', got %q ok=%v", ev, ok)
	}
}

// TestDeleteUserPhoto_BroadcastsMembers verifies the user photo delete route
// broadcasts "members" reaching the acting user.
func TestDeleteUserPhoto_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)

	userU := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, userU)
	userTok := testutil.Token(t, userU, "standard", nil)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	userCh := sharedHub.SubscribeUser(userU)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/upload/user-photo", userTok, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DeleteUserPhoto: expected 204, got %d", res.StatusCode)
	}

	if ev, ok := recvEvent(userCh, time.Second); !ok || ev != "members" {
		t.Errorf("acting user stream must receive 'members', got %q ok=%v", ev, ok)
	}
}

// TestUploadChildPhoto_BroadcastsMembers verifies a parent uploading a child's
// photo broadcasts "members" reaching the parent (and finance group).
func TestUploadChildPhoto_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)

	parentU := testutil.CreateUser(t, db, "standard")
	childM := testutil.CreateMember(t, db, 0)
	testutil.AddFamilyLink(t, db, parentU, childM)
	parentTok := testutil.TokenWithIsParent(t, parentU, "standard", nil, true)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	parentCh := sharedHub.SubscribeUser(parentU)

	req := multipartPhoto(t, srv.URL+"/api/profile/kind/"+strconv.Itoa(childM)+"/photo", parentTok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("UploadChildPhoto: expected 200, got %d", res.StatusCode)
	}

	if ev, ok := recvEvent(parentCh, time.Second); !ok || ev != "members" {
		t.Errorf("parent stream must receive 'members', got %q ok=%v", ev, ok)
	}
}

// TestDeleteChildPhoto_BroadcastsMembers verifies a parent deleting a child's
// photo broadcasts "members" reaching the parent.
func TestDeleteChildPhoto_BroadcastsMembers(t *testing.T) {
	db := testutil.NewDB(t)

	parentU := testutil.CreateUser(t, db, "standard")
	childM := testutil.CreateMember(t, db, 0)
	testutil.AddFamilyLink(t, db, parentU, childM)
	parentTok := testutil.TokenWithIsParent(t, parentU, "standard", nil, true)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	parentCh := sharedHub.SubscribeUser(parentU)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/profile/kind/"+strconv.Itoa(childM)+"/photo", parentTok, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DeleteChildPhoto: expected 204, got %d", res.StatusCode)
	}

	if ev, ok := recvEvent(parentCh, time.Second); !ok || ev != "members" {
		t.Errorf("parent stream must receive 'members', got %q ok=%v", ev, ok)
	}
}
