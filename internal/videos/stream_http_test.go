package videos

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

const testStreamSecret = "test-stream-secret"

// newStreamServer builds a router mirroring the production mounting: /play behind
// JWT auth, the HLS subtree behind the stream-token middleware (no JWT). It points
// the handler's storage dir at a temp dir and writes fake HLS files for video vid.
func newStreamServer(t *testing.T, h *Handler) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(testutil.TestJWTSecret))
		r.Get("/api/videos/{id}/play", h.Play)
	})
	r.Route("/api/videos/{id}/hls", func(r chi.Router) {
		r.Use(h.StreamTokenMiddleware)
		r.Get("/master.m3u8", h.ServeMaster)
		r.Get("/{rendition}/{segment}", h.ServeRenditionFile)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// writeFakeHLS creates a minimal but realistic HLS tree for video id under root.
func writeFakeHLS(t *testing.T, root string, id int) {
	t.Helper()
	procDir := ProcessedDir(root, id)
	for _, rend := range []string{"720p", "360p"} {
		dir := RenditionDir(root, id, rend)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		idx := "#EXTM3U\n#EXT-X-VERSION:3\n#EXTINF:10.0,\nseg_001.ts\n#EXT-X-ENDLIST\n"
		if err := os.WriteFile(filepath.Join(dir, "index.m3u8"), []byte(idx), 0o644); err != nil {
			t.Fatalf("write index: %v", err)
		}
		// 4096 bytes of fake segment data (enough to exercise Range).
		seg := make([]byte, 4096)
		for i := range seg {
			seg[i] = byte(i % 251)
		}
		if err := os.WriteFile(filepath.Join(dir, "seg_001.ts"), seg, 0o644); err != nil {
			t.Fatalf("write seg: %v", err)
		}
	}
	master := "#EXTM3U\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1280x720\n720p/index.m3u8\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=640x360\n360p/index.m3u8\n"
	if err := os.WriteFile(filepath.Join(procDir, "master.m3u8"), []byte(master), 0o644); err != nil {
		t.Fatalf("write master: %v", err)
	}
}

// setupStreamFixture creates a team/season/video plus a player user who can view it,
// returns the handler (storage at tmp), the video id, and a Bearer token for the player.
func setupStreamFixture(t *testing.T) (*Handler, int, string) {
	t.Helper()
	db := testutil.NewDB(t)
	root := t.TempDir()
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VideoStreamSecret: testStreamSecret, VideoStorageDir: root}
	h := NewHandler(db, nil, cfg)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, team, season, uploader, "ready")

	// A player of the team → may view.
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

func TestPlay_ReturnsTokenAndMasterURL(t *testing.T) {
	h, vid, bearer := setupStreamFixture(t)
	srv := newStreamServer(t, h)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/play", nil)
	req.Header.Set("Authorization", bearer)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("play status %d, want 200", res.StatusCode)
	}
	var body struct {
		Token     string `json:"token"`
		MasterURL string `json:"master_url"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.Token == "" {
		t.Error("empty token")
	}
	want := "/api/videos/" + strconv.Itoa(vid) + "/hls/master.m3u8"
	if body.MasterURL != want {
		t.Errorf("master_url = %q, want %q", body.MasterURL, want)
	}
	// The returned token must verify against the video.
	if _, err := h.Verify(body.Token, vid); err != nil {
		t.Errorf("returned token does not verify: %v", err)
	}
}

func TestPlay_ForbiddenForOutsider(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)

	// An unrelated standard user with no team membership.
	outsider, err := auth.IssueAccessToken(testutil.TestJWTSecret, 99999, "x@test.local", "standard", []string{}, false)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/play", nil)
	req.Header.Set("Authorization", "Bearer "+outsider)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("play request: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("outsider play status %d, want 403", res.StatusCode)
	}
}

// signFor mints a valid stream token via the handler.
func signFor(t *testing.T, h *Handler, vid, uid int) string {
	t.Helper()
	tok, err := h.Sign(vid, uid)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return tok
}

func TestMaster_ValidToken_RewritesRenditionURLs(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)
	tok := signFor(t, h, vid, 7)

	res, err := http.Get(srv.URL + "/api/videos/" + strconv.Itoa(vid) + "/hls/master.m3u8?st=" + tok)
	if err != nil {
		t.Fatalf("get master: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("master status %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/vnd.apple.mpegurl" {
		t.Errorf("master content-type = %q", ct)
	}
	body, _ := io.ReadAll(res.Body)
	s := string(body)
	if !strings.Contains(s, "720p/index.m3u8?st="+tok) {
		t.Errorf("720p rendition URL not rewritten with token:\n%s", s)
	}
	if !strings.Contains(s, "360p/index.m3u8?st="+tok) {
		t.Errorf("360p rendition URL not rewritten with token:\n%s", s)
	}
}

func TestHLS_MissingToken_403(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)

	res, err := http.Get(srv.URL + "/api/videos/" + strconv.Itoa(vid) + "/hls/master.m3u8")
	if err != nil {
		t.Fatalf("get master: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("missing token status %d, want 403", res.StatusCode)
	}
}

func TestHLS_WrongVidToken_403(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)
	// Token minted for a different video id.
	tok := signFor(t, h, vid+1000, 7)

	res, err := http.Get(srv.URL + "/api/videos/" + strconv.Itoa(vid) + "/hls/master.m3u8?st=" + tok)
	if err != nil {
		t.Fatalf("get master: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("wrong-vid token status %d, want 403", res.StatusCode)
	}
}

func TestSegment_ValidToken_200(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)
	tok := signFor(t, h, vid, 7)

	res, err := http.Get(srv.URL + "/api/videos/" + strconv.Itoa(vid) + "/hls/720p/seg_001.ts?st=" + tok)
	if err != nil {
		t.Fatalf("get segment: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("segment status %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "video/mp2t" {
		t.Errorf("segment content-type = %q, want video/mp2t", ct)
	}
}

func TestSegment_RangeRequest_206(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)
	tok := signFor(t, h, vid, 7)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/hls/720p/seg_001.ts?st="+tok, nil)
	req.Header.Set("Range", "bytes=0-99")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("range get: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusPartialContent {
		t.Fatalf("range status %d, want 206", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if len(body) != 100 {
		t.Errorf("range body len %d, want 100", len(body))
	}
}

func TestRenditionIndex_ValidToken_200(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)
	tok := signFor(t, h, vid, 7)

	res, err := http.Get(srv.URL + "/api/videos/" + strconv.Itoa(vid) + "/hls/720p/index.m3u8?st=" + tok)
	if err != nil {
		t.Fatalf("get index: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("index status %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/vnd.apple.mpegurl" {
		t.Errorf("index content-type = %q", ct)
	}
}

func TestSegment_PathTraversal_Rejected(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)
	tok := signFor(t, h, vid, 7)

	// Disallowed rendition and segment names must 404 (no traversal).
	bad := []string{
		"/api/videos/" + strconv.Itoa(vid) + "/hls/720p/..%2f..%2fmaster.m3u8?st=" + tok,
		"/api/videos/" + strconv.Itoa(vid) + "/hls/evil/seg_001.ts?st=" + tok,
		"/api/videos/" + strconv.Itoa(vid) + "/hls/720p/secret.txt?st=" + tok,
	}
	for _, u := range bad {
		res, _ := http.Get(srv.URL + u)
		if res.StatusCode == http.StatusOK {
			t.Errorf("traversal %q returned 200, want non-200", u)
		}
		res.Body.Close()
	}
}
