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
	"time"

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
		r.Options("/master.m3u8", h.HLSPreflight)
		r.Get("/{rendition}/{segment}", h.ServeRenditionFile)
		r.Options("/{rendition}/{segment}", h.HLSPreflight)
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
		"#EXT-X-VERSION:3\n" +
		"#EXT-X-INDEPENDENT-SEGMENTS\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=2800000,RESOLUTION=1280x720,CODECS=\"avc1.640028,mp4a.40.2\"\n720p/index.m3u8\n"
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

// playAndDecodeExp fährt einen /play-Request und liefert das exp-Claim des
// gelieferten Tokens sowie das fixed-now, das zur Signierung verwendet wurde.
// Der Test kann daraus die effektive TTL ableiten.
func playAndDecodeExp(t *testing.T, srv *httptest.Server, bearer string, vid int) int64 {
	t.Helper()
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
		Token string `json:"token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode play body: %v", err)
	}
	// exp aus dem token-Payload extrahieren (base64url(vid.uid.exp).sig).
	enc, _, ok := strings.Cut(body.Token, ".")
	if !ok {
		t.Fatalf("malformed token %q", body.Token)
	}
	raw, err := b64.DecodeString(enc)
	if err != nil {
		t.Fatalf("decode token payload: %v", err)
	}
	parts := strings.Split(string(raw), ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected payload %q", string(raw))
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		t.Fatalf("parse exp: %v", err)
	}
	return exp
}

// withFixedNow ersetzt die Paket-Uhr für die Dauer des Tests und stellt sie am
// Ende wieder her — nötig um die TTL-Rechnung deterministisch zu prüfen.
func withFixedNow(t *testing.T, fixed time.Time) {
	t.Helper()
	prev := now
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = prev })
}

// TestPlay_TTL_KurzVideo_1h: Videos ≤ 30 min bekommen die Untergrenze 1h.
func TestPlay_TTL_KurzVideo_1h(t *testing.T) {
	h, vid, bearer := setupStreamFixture(t)
	if _, err := h.db.Exec(`UPDATE videos SET duration_sec=? WHERE id=?`, 1200, vid); err != nil {
		t.Fatalf("set duration: %v", err)
	}
	fixed := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)

	srv := newStreamServer(t, h)
	exp := playAndDecodeExp(t, srv, bearer, vid)
	got := exp - fixed.Unix()
	if got != int64((time.Hour).Seconds()) {
		t.Fatalf("kurzvideo TTL = %ds, want 3600s", got)
	}
}

// TestPlay_TTL_LangVideo_DauerPlus30: 90-min-Video → duration + 30min = 2h.
func TestPlay_TTL_LangVideo_DauerPlus30(t *testing.T) {
	h, vid, bearer := setupStreamFixture(t)
	if _, err := h.db.Exec(`UPDATE videos SET duration_sec=? WHERE id=?`, 5400, vid); err != nil {
		t.Fatalf("set duration: %v", err)
	}
	fixed := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)

	srv := newStreamServer(t, h)
	exp := playAndDecodeExp(t, srv, bearer, vid)
	got := exp - fixed.Unix()
	want := int64((5400*time.Second + 30*time.Minute).Seconds())
	if got != want {
		t.Fatalf("langvideo TTL = %ds, want %ds", got, want)
	}
}

// TestPlay_TTL_SehrLangVideo_Cap4h: > 3.5h → auf 4h gedeckelt.
func TestPlay_TTL_SehrLangVideo_Cap4h(t *testing.T) {
	h, vid, bearer := setupStreamFixture(t)
	if _, err := h.db.Exec(`UPDATE videos SET duration_sec=? WHERE id=?`, 100000, vid); err != nil {
		t.Fatalf("set duration: %v", err)
	}
	fixed := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)

	srv := newStreamServer(t, h)
	exp := playAndDecodeExp(t, srv, bearer, vid)
	got := exp - fixed.Unix()
	if got != int64((4 * time.Hour).Seconds()) {
		t.Fatalf("cap TTL = %ds, want 14400s", got)
	}
}

// TestPlay_TTL_LegacyOhneDuration_1h: duration_sec IS NULL → 1h wie vor
// video-tv-streaming, damit Bestandsvideos unverändert weiterlaufen.
func TestPlay_TTL_LegacyOhneDuration_1h(t *testing.T) {
	h, vid, bearer := setupStreamFixture(t)
	// setupStreamFixture setzt duration_sec nicht → bleibt NULL.
	fixed := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)

	srv := newStreamServer(t, h)
	exp := playAndDecodeExp(t, srv, bearer, vid)
	got := exp - fixed.Unix()
	if got != int64((time.Hour).Seconds()) {
		t.Fatalf("legacy TTL = %ds, want 3600s", got)
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
	tok, err := h.Sign(vid, uid, 0)
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
	if strings.Contains(s, "360p/index.m3u8") {
		t.Errorf("360p rendition unexpectedly present (removed in video-tv-streaming):\n%s", s)
	}
	if !strings.Contains(s, "CODECS=") {
		t.Errorf("master.m3u8 missing CODECS attribute (AirPlay needs it):\n%s", s)
	}
}

// TestHLS_MasterCORSHeader: Chromecast-Default-Receiver braucht
// `Access-Control-Allow-Origin: *` auf der Master-Playlist, sonst verweigert er
// das Playback. Auth-Semantik (?st=-Token) bleibt unverändert.
func TestHLS_MasterCORSHeader(t *testing.T) {
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
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if got := res.Header.Get("Access-Control-Allow-Methods"); got != "GET" {
		t.Errorf("Access-Control-Allow-Methods = %q, want GET", got)
	}
}

// TestHLS_MasterCORSPreflight: OPTIONS-Preflight läuft ohne ?st= und liefert
// 204 mit den CORS-Headern. Der Preflight ist per RFC credential-frei; Auth
// erfolgt beim nachgelagerten GET.
func TestHLS_MasterCORSPreflight(t *testing.T) {
	h, vid, _ := setupStreamFixture(t)
	srv := newStreamServer(t, h)

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/videos/"+strconv.Itoa(vid)+"/hls/master.m3u8", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("preflight status %d, want 204", res.StatusCode)
	}
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("preflight ACAO = %q, want *", got)
	}
}

// TestHLS_SegmentCORSHeader: Auch Segmente tragen den CORS-Header (defensive
// Symmetrie, damit Chromecast-Firmwares mit strengerem Verhalten nicht in
// Fallback-Modi kippen).
func TestHLS_SegmentCORSHeader(t *testing.T) {
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
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("segment ACAO = %q, want *", got)
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
	body, _ := io.ReadAll(res.Body)
	s := string(body)
	// Segment-Referenzen müssen ?st=<tok> bekommen, sonst scheitert der
	// Segment-GET an der StreamTokenMiddleware (403).
	if !strings.Contains(s, "seg_001.ts?st="+tok) {
		t.Errorf("index ohne Token-Anhang an Segment: %q", s)
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
