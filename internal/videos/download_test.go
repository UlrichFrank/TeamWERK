package videos

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newDownloadServer(t *testing.T, h *Handler) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Use(auth.Middleware(testutil.TestJWTSecret))
	r.Get("/api/videos/{id}", h.Get)
	r.Get("/api/videos/{id}/download", h.Download)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// setupDownloadFixture mirrors setupStreamFixture (stream_http_test.go): a
// ready video with real (fake) HLS files on disk plus a player of the team who
// may view it.
func setupDownloadFixture(t *testing.T, reserved uint64) (*Handler, int, string) {
	t.Helper()
	db := testutil.NewDB(t)
	root := t.TempDir()
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VideoStorageDir: root, VideoReservedBytes: reserved}
	h := NewHandler(db, nil, cfg)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, team, season, uploader, "ready")

	playerUser := testutil.CreateUser(t, db, "standard")
	playerMember := testutil.CreateMember(t, db, playerUser)
	addKaderMember(t, db, kader, playerMember)

	writeFakeHLS(t, root, vid)

	tok, err := auth.IssueAccessToken(testutil.TestJWTSecret, playerUser, "p@test.local", "standard", []string{"spieler"}, false)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	return h, vid, "Bearer " + tok
}

// countingRemux wraps a remuxFunc and counts how often it was invoked.
type countingRemux struct {
	fn    remuxFunc
	calls int
}

func (c *countingRemux) run(ctx context.Context, listFile, outPath string) error {
	c.calls++
	return c.fn(ctx, listFile, outPath)
}

var fakeRemuxContent = []byte("fake mp4 bytes")

func fakeRemuxSuccess(_ context.Context, _, outPath string) error {
	return os.WriteFile(outPath, fakeRemuxContent, 0o644)
}

func fakeRemuxFailure(_ context.Context, _, _ string) error {
	return errors.New("ffmpeg exploded")
}

func doGetBearer(t *testing.T, url, bearer string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", bearer)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return res
}

func TestDownload_HappyPath(t *testing.T) {
	h, vid, bearer := setupDownloadFixture(t, 1024)
	remux := &countingRemux{fn: fakeRemuxSuccess}
	h.remux = remux.run
	srv := newDownloadServer(t, h)

	res := doGetBearer(t, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/download", bearer)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "video/mp4" {
		t.Errorf("Content-Type = %q, want video/mp4", got)
	}
	cd := res.Header.Get("Content-Disposition")
	if cd == "" || !strings.Contains(cd, "attachment") || !strings.Contains(cd, ".mp4") {
		t.Errorf("Content-Disposition = %q, want attachment with .mp4 filename", cd)
	}
	if got := res.Header.Get("Content-Length"); got != strconv.Itoa(len(fakeRemuxContent)) {
		t.Errorf("Content-Length = %q, want %d", got, len(fakeRemuxContent))
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != string(fakeRemuxContent) {
		t.Errorf("body = %q, want %q", body, fakeRemuxContent)
	}
	if remux.calls != 1 {
		t.Errorf("remux calls = %d, want 1", remux.calls)
	}

	// Temp-Dateien wurden aufgeräumt (kein dauerhaftes Caching).
	assertTempDirEmpty(t, h.cfg.VideoStorageDir)
}

func TestDownload_ForbiddenUserGetsSameStatusAsGet(t *testing.T) {
	h, vid, _ := setupDownloadFixture(t, 1024)
	h.remux = fakeRemuxSuccess
	srv := newDownloadServer(t, h)

	outsider := testutil.CreateUser(t, h.db, "standard")
	tok, err := auth.IssueAccessToken(testutil.TestJWTSecret, outsider, "o@test.local", "standard", []string{"spieler"}, false)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	bearer := "Bearer " + tok

	getRes := doGetBearer(t, srv.URL+"/api/videos/"+strconv.Itoa(vid), bearer)
	defer getRes.Body.Close()

	dlRes := doGetBearer(t, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/download", bearer)
	defer dlRes.Body.Close()

	if dlRes.StatusCode != getRes.StatusCode {
		t.Errorf("download status = %d, want same as GET /api/videos/{id} (%d)", dlRes.StatusCode, getRes.StatusCode)
	}
	if dlRes.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (existence not disclosed)", dlRes.StatusCode)
	}
}

func TestDownload_NotReadyIsConflict(t *testing.T) {
	db := testutil.NewDB(t)
	root := t.TempDir()
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VideoStorageDir: root, VideoReservedBytes: 1024}
	h := NewHandler(db, nil, cfg)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, team, season, uploader, "processing")

	playerUser := testutil.CreateUser(t, db, "standard")
	playerMember := testutil.CreateMember(t, db, playerUser)
	addKaderMember(t, db, kader, playerMember)
	tok, err := auth.IssueAccessToken(testutil.TestJWTSecret, playerUser, "p@test.local", "standard", []string{"spieler"}, false)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	remux := &countingRemux{fn: fakeRemuxSuccess}
	h.remux = remux.run
	srv := newDownloadServer(t, h)

	res := doGetBearer(t, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/download", "Bearer "+tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
	if remux.calls != 0 {
		t.Errorf("remux calls = %d, want 0 (not ready → no remux attempt)", remux.calls)
	}
}

func TestDownload_RemuxFailureIsInternalError(t *testing.T) {
	h, vid, bearer := setupDownloadFixture(t, 1024)
	h.remux = fakeRemuxFailure
	srv := newDownloadServer(t, h)

	res := doGetBearer(t, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/download", bearer)
	defer res.Body.Close()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct == "video/mp4" {
		t.Error("Content-Type is video/mp4 despite remux failure — partial response leaked")
	}

	assertTempDirEmpty(t, h.cfg.VideoStorageDir)
}

func TestDownload_InsufficientStorage(t *testing.T) {
	// Riesige Reserve → RequireFreeBytes schlägt immer fehl → HTTP 507, kein Remux.
	h, vid, bearer := setupDownloadFixture(t, 1<<62)
	remux := &countingRemux{fn: fakeRemuxSuccess}
	h.remux = remux.run
	srv := newDownloadServer(t, h)

	res := doGetBearer(t, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/download", bearer)
	defer res.Body.Close()
	if res.StatusCode != http.StatusInsufficientStorage {
		t.Fatalf("status = %d, want 507", res.StatusCode)
	}
	if remux.calls != 0 {
		t.Errorf("remux calls = %d, want 0 (disk guard trips before remux)", remux.calls)
	}
}

// assertTempDirEmpty verifies that DownloadTempDir(root) contains no leftover
// files after a request — "kein dauerhaftes Caching" (video-download spec).
func assertTempDirEmpty(t *testing.T, root string) {
	t.Helper()
	dir := DownloadTempDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read temp dir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("temp dir %s not empty: %v", dir, names)
	}
}
