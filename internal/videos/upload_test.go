package videos

import (
	"database/sql"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// newUploadServer mounts POST /api/videos behind the same auth tier as
// internal/app/router.go (JWT + RequireClubFunction upload tier). CreateUpload
// additionally enforces CanUploadToTeam.
func newUploadServer(t *testing.T, h *Handler) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Use(auth.Middleware(testutil.TestJWTSecret))
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
		r.Post("/api/videos", h.CreateUpload)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// uploadHandler builds a handler with storage at a temp dir and a real hub.
// reserved controls VideoReservedBytes (set huge to force the disk guard to trip).
func uploadHandler(t *testing.T, db *sql.DB, reserved uint64) (*Handler, string) {
	t.Helper()
	root := t.TempDir()
	cfg := &appconfig.Config{
		JWTSecret:          testutil.TestJWTSecret,
		VideoStorageDir:    root,
		VideoReservedBytes: reserved,
	}
	return NewHandler(db, hub.NewHub(), cfg), root
}

// --- POST /api/videos (Pre-Upload-Init) ------------------------------------

func TestCreateUpload_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024) // tiny reserve → disk guard passes
	srv := newUploadServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kader, trainerMember)

	tok := testutil.Token(t, trainerUser, "standard", []string{"trainer"})
	body := map[string]any{
		"title":      "Spiel gegen X",
		"team_id":    team,
		"season_id":  season,
		"size_bytes": 1024,
	}
	res := testutil.Post(t, srv, "/api/videos", tok, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", res.StatusCode)
	}

	// Row exists with status='uploading' and created_by set.
	var status string
	var createdBy int
	if err := db.QueryRow(
		`SELECT status, created_by FROM videos WHERE team_id = ? AND season_id = ?`,
		team, season).Scan(&status, &createdBy); err != nil {
		t.Fatalf("query video row: %v", err)
	}
	if status != "uploading" {
		t.Errorf("status = %q, want uploading", status)
	}
	if createdBy != trainerUser {
		t.Errorf("created_by = %d, want %d", createdBy, trainerUser)
	}
}

// TestCreateUpload_BroadcastsVideoQueued belegt, dass ein erfolgreicher
// Upload-Init das "video-queued"-Event auslöst, damit die Videos-Seite
// (useLiveUpdates('video-queued')) die neue Platzhalterzeile sofort nachlädt.
// Sichert die CLAUDE.md-Broadcast-Regel für CreateUpload ab.
func TestCreateUpload_BroadcastsVideoQueued(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)
	srv := newUploadServer(t, h)

	// Global subscriben — Broadcast erreicht sowohl clients als auch per-user
	// Streams (siehe hub.Broadcast).
	ch := h.hub.Subscribe()
	defer h.hub.Unsubscribe(ch)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kader, trainerMember)

	tok := testutil.Token(t, trainerUser, "standard", []string{"trainer"})
	body := map[string]any{
		"title":      "Spiel gegen X",
		"team_id":    team,
		"season_id":  season,
		"size_bytes": 1024,
	}
	res := testutil.Post(t, srv, "/api/videos", tok, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", res.StatusCode)
	}

	select {
	case ev := <-ch:
		if ev != "video-queued" {
			t.Errorf("event = %q, want video-queued", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("no broadcast received after CreateUpload; want video-queued")
	}
}

// Leerer Titel + Spiel-ID → Titel wird aus Datum + Gegner abgeleitet.
// CreateGame nutzt opponent="Test Opponent", date wie übergeben.
func TestCreateUpload_DerivesTitleFromGame(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)
	srv := newUploadServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)
	game := testutil.CreateGame(t, db, season, team, "2026-03-15")

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kader, trainerMember)

	tok := testutil.Token(t, trainerUser, "standard", []string{"trainer"})
	body := map[string]any{
		"team_id":    team,
		"season_id":  season,
		"game_id":    game,
		"size_bytes": 1024,
	}
	res := testutil.Post(t, srv, "/api/videos", tok, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", res.StatusCode)
	}

	var title string
	if err := db.QueryRow(`SELECT title FROM videos WHERE game_id = ?`, game).Scan(&title); err != nil {
		t.Fatalf("query video title: %v", err)
	}
	const want = "15.03.2026 · Test Opponent"
	if title != want {
		t.Errorf("derived title = %q, want %q", title, want)
	}
}

func TestCreateUpload_ForbiddenForeignTeam(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)
	srv := newUploadServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	// Trainer of team A tries to upload to team B.
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMember)

	tok := testutil.Token(t, trainerUser, "standard", []string{"trainer"})
	body := map[string]any{
		"title":      "Fremdes Team",
		"team_id":    teamB,
		"season_id":  season,
		"size_bytes": 1024,
	}
	res := testutil.Post(t, srv, "/api/videos", tok, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM videos WHERE team_id = ?`, teamB).Scan(&n)
	if n != 0 {
		t.Errorf("forbidden upload created %d rows, want 0", n)
	}
}

func TestCreateUpload_InsufficientStorage(t *testing.T) {
	db := testutil.NewDB(t)
	// Reserve an absurd amount → RequireFreeBytes always trips → HTTP 507.
	h, _ := uploadHandler(t, db, math.MaxUint64/2)
	srv := newUploadServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kader, trainerMember)

	tok := testutil.Token(t, trainerUser, "standard", []string{"trainer"})
	body := map[string]any{
		"title":      "Zu groß",
		"team_id":    team,
		"season_id":  season,
		"size_bytes": 1024,
	}
	res := testutil.Post(t, srv, "/api/videos", tok, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusInsufficientStorage {
		t.Fatalf("status = %d, want 507", res.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&n)
	if n != 0 {
		t.Errorf("rejected upload created %d rows, want 0", n)
	}
}

func TestCreateUpload_BadRequest(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)
	srv := newUploadServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kader, trainerMember)
	tok := testutil.Token(t, trainerUser, "standard", []string{"trainer"})

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing title and game", map[string]any{"team_id": team, "season_id": season, "size_bytes": 1024}},
		{"missing team_id", map[string]any{"title": "x", "season_id": season, "size_bytes": 1024}},
		{"invalid team_id zero", map[string]any{"title": "x", "team_id": 0, "season_id": season, "size_bytes": 1024}},
		{"missing season_id", map[string]any{"title": "x", "team_id": team, "size_bytes": 1024}},
		{"missing size_bytes", map[string]any{"title": "x", "team_id": team, "season_id": season}},
		{"over 2GB", map[string]any{"title": "x", "team_id": team, "season_id": season, "size_bytes": int64(3) << 30}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.Post(t, srv, "/api/videos", tok, tc.body)
			defer res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", res.StatusCode)
			}
		})
	}
}

// --- finishUpload (extracted finish logic) ----------------------------------

// writeProbeableMP4 produces a tiny valid mp4 via ffmpeg so ffprobe yields a
// duration. Skips the test if ffmpeg is unavailable.
func writeProbeableMP4(t *testing.T, path string) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not in PATH — skipping finishUpload integration test")
	}
	// The destination has no .mp4 extension (it mimics a tus bin file), so the
	// output muxer must be named explicitly via -f mp4 with faststart so the
	// moov atom precedes the data (otherwise mp4 needs a seekable output).
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=128x96:rate=10",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-f", "mp4", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg generate source failed: %v\n%s", err, out)
	}
}

func TestFinishUpload_MovesFileProbesAndQueues(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not in PATH — skipping finishUpload integration test")
	}
	db := testutil.NewDB(t)
	h, root := uploadHandler(t, db, 1024)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	uploader := testutil.CreateUser(t, db, "standard")
	videoID := testutil.CreateVideo(t, db, team, season, uploader, "uploading")

	// Simulate the finished tus bin file under uploads/.
	if err := os.MkdirAll(uploadsDir(root), 0o755); err != nil {
		t.Fatalf("mkdir uploads: %v", err)
	}
	src := filepath.Join(uploadsDir(root), "abc123")
	writeProbeableMP4(t, src)

	if err := h.finishUpload(videoID, src, "abc123", 4242); err != nil {
		t.Fatalf("finishUpload: %v", err)
	}

	// raw/{id}.mp4 exists; source consumed.
	if _, err := os.Stat(RawPath(root, videoID)); err != nil {
		t.Errorf("raw file missing after finishUpload: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source file should be gone after move, stat err = %v", err)
	}

	var status string
	var size sql.NullInt64
	var duration sql.NullInt64
	var uploadID sql.NullString
	if err := db.QueryRow(
		`SELECT status, size_bytes, duration_sec, upload_id FROM videos WHERE id = ?`,
		videoID).Scan(&status, &size, &duration, &uploadID); err != nil {
		t.Fatalf("query video: %v", err)
	}
	if status != "queued" {
		t.Errorf("status = %q, want queued", status)
	}
	if !size.Valid || size.Int64 != 4242 {
		t.Errorf("size_bytes = %v, want 4242", size)
	}
	if !duration.Valid || duration.Int64 < 1 {
		t.Errorf("duration_sec = %v, want >= 1", duration)
	}
	if !uploadID.Valid || uploadID.String != "abc123" {
		t.Errorf("upload_id = %v, want abc123", uploadID)
	}
}

func TestFinishUpload_BrokenSourceFails(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not in PATH")
	}
	db := testutil.NewDB(t)
	h, root := uploadHandler(t, db, 1024)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	uploader := testutil.CreateUser(t, db, "standard")
	videoID := testutil.CreateVideo(t, db, team, season, uploader, "uploading")

	if err := os.MkdirAll(uploadsDir(root), 0o755); err != nil {
		t.Fatalf("mkdir uploads: %v", err)
	}
	src := filepath.Join(uploadsDir(root), "broken")
	if err := os.WriteFile(src, []byte("not a video"), 0o644); err != nil {
		t.Fatalf("write broken src: %v", err)
	}

	if err := h.finishUpload(videoID, src, "broken", 11); err == nil {
		t.Fatal("finishUpload on a non-video source should fail")
	}
}

// --- CleanupStaleUploads ----------------------------------------------------

func TestCleanupStaleUploads(t *testing.T) {
	root := t.TempDir()
	dir := uploadsDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stale := filepath.Join(dir, "old.bin")
	fresh := filepath.Join(dir, "new.bin")
	for _, p := range []string{stale, fresh} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	// Backdate the stale file by 48h.
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	n, err := CleanupStaleUploads(root, 24*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleUploads: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1", n)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale file should be removed")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh file should be kept: %v", err)
	}
}

func TestCleanupStaleUploads_MissingDirIsNoError(t *testing.T) {
	n, err := CleanupStaleUploads(t.TempDir(), 24*time.Hour)
	if err != nil {
		t.Fatalf("missing uploads dir should not be an error, got %v", err)
	}
	if n != 0 {
		t.Errorf("removed = %d, want 0", n)
	}
}
