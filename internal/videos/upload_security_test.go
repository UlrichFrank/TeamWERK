package videos

import (
	"context"
	"database/sql"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// hookFor builds a tusd.HookEvent carrying a context with the given claims (as
// the auth middleware would inject them) plus the supplied tus metadata and the
// declared upload length. uploaderClaims may be nil to simulate a missing-auth
// session.
func hookFor(uploaderClaims *auth.Claims, meta map[string]string, declaredSize int64) tusd.HookEvent {
	ctx := context.Background()
	if uploaderClaims != nil {
		ctx = auth.ContextWithClaims(ctx, uploaderClaims)
	}
	return tusd.HookEvent{
		Context: ctx,
		Upload: tusd.FileInfo{
			ID:       "tus-session-id",
			Size:     declaredSize,
			MetaData: meta,
		},
	}
}

// --- Finding 1a: PreUploadCreateCallback ownership binding -------------------

func TestPreUploadCreate_AcceptsLegitOwner(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024) // tiny reserve → disk guard passes

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	owner := testutil.CreateUser(t, db, "standard")
	videoID := testutil.CreateVideo(t, db, team, season, owner, "uploading")

	c := claims(owner, "standard", "trainer")
	_, _, err := h.preUploadCreate(hookFor(c, map[string]string{"video_id": itoa(videoID)}, 1000))
	if err != nil {
		t.Fatalf("legit owner should be accepted, got error: %v", err)
	}
}

func TestPreUploadCreate_RejectsForeignOwner(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	owner := testutil.CreateUser(t, db, "standard")
	attacker := testutil.CreateUser(t, db, "standard")
	videoID := testutil.CreateVideo(t, db, team, season, owner, "uploading")

	// Attacker (a valid trainer) tries to bind a session to the owner's row.
	c := claims(attacker, "standard", "trainer")
	_, _, err := h.preUploadCreate(hookFor(c, map[string]string{"video_id": itoa(videoID)}, 1000))
	if err == nil {
		t.Fatal("session for a foreign-owned uploading row must be rejected")
	}
}

func TestPreUploadCreate_RejectsNonUploadingStatus(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	owner := testutil.CreateUser(t, db, "standard")
	// Same owner, but the row is already 'ready' → must not be re-bound/overwritten.
	videoID := testutil.CreateVideo(t, db, team, season, owner, "ready")

	c := claims(owner, "standard", "trainer")
	_, _, err := h.preUploadCreate(hookFor(c, map[string]string{"video_id": itoa(videoID)}, 1000))
	if err == nil {
		t.Fatal("session for an already-ready row must be rejected")
	}
}

func TestPreUploadCreate_RejectsMissingClaims(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	owner := testutil.CreateUser(t, db, "standard")
	videoID := testutil.CreateVideo(t, db, team, season, owner, "uploading")

	_, _, err := h.preUploadCreate(hookFor(nil, map[string]string{"video_id": itoa(videoID)}, 1000))
	if err == nil {
		t.Fatal("session without claims must be rejected")
	}
}

func TestPreUploadCreate_RejectsBadVideoID(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)
	owner := testutil.CreateUser(t, db, "standard")
	c := claims(owner, "standard", "trainer")

	cases := []struct {
		name string
		meta map[string]string
	}{
		{"missing", map[string]string{}},
		{"non-numeric", map[string]string{"video_id": "abc"}},
		{"zero", map[string]string{"video_id": "0"}},
		{"nonexistent", map[string]string{"video_id": "999999"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := h.preUploadCreate(hookFor(c, tc.meta, 1000))
			if err == nil {
				t.Fatalf("video_id %v must be rejected", tc.meta)
			}
		})
	}
}

// --- Finding 2: disk guard against the DECLARED upload length ----------------

func TestPreUploadCreate_DiskGuardUsesDeclaredLength(t *testing.T) {
	db := testutil.NewDB(t)
	// Reserve an absurd amount → RequireFreeBytes always trips.
	h, _ := uploadHandler(t, db, math.MaxUint64/2)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	owner := testutil.CreateUser(t, db, "standard")
	// The declared Upload-Length is what tusd enforces; the disk guard must run
	// against it. (announced size_bytes is set equal so the plausibility check
	// passes and the disk guard is the actual rejector.)
	const declared = int64(1 << 20)
	videoID := createVideoWithSize(t, db, team, season, owner, "uploading", declared)

	c := claims(owner, "standard", "trainer")
	_, _, err := h.preUploadCreate(hookFor(c, map[string]string{"video_id": itoa(videoID)}, declared))
	if err == nil {
		t.Fatal("disk guard must reject when declared length exceeds free space")
	}
}

func TestPreUploadCreate_RejectsDeferredLength(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	owner := testutil.CreateUser(t, db, "standard")
	videoID := testutil.CreateVideo(t, db, team, season, owner, "uploading")

	c := claims(owner, "standard", "trainer")
	ev := hookFor(c, map[string]string{"video_id": itoa(videoID)}, 0)
	ev.Upload.SizeIsDeferred = true
	if _, _, err := h.preUploadCreate(ev); err == nil {
		t.Fatal("deferred upload length must be rejected so the disk guard can apply")
	}
}

// --- Finding 1b: finishUpload conditional/atomic transition + cleanup --------

func TestFinishUpload_IDOR_VictimRowUntouched(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not in PATH — skipping conditional-UPDATE IDOR test")
	}
	db := testutil.NewDB(t)
	h, root := uploadHandler(t, db, 1024)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	victimOwner := testutil.CreateUser(t, db, "standard")
	// Victim row is another user's already-'ready' video.
	victimID := testutil.CreateVideo(t, db, team, season, victimOwner, "ready")

	// Attacker's finished tus bin file, aimed at the victim id. It IS a valid
	// probeable mp4 so probeDurationSec succeeds and we reach the conditional
	// UPDATE — which must affect 0 rows (status != 'uploading') and clean up.
	if err := os.MkdirAll(uploadsDir(root), 0o755); err != nil {
		t.Fatalf("mkdir uploads: %v", err)
	}
	src := filepath.Join(uploadsDir(root), "attacker-bin")
	writeProbeableMP4(t, src)

	err := h.finishUpload(victimID, src, "attacker-bin", 7)
	if err == nil {
		t.Fatal("finishUpload targeting a non-uploading victim row must return an error")
	}

	// Victim row unchanged: still 'ready', no upload_id/size hijacked.
	var status string
	var uploadID sql.NullString
	var size sql.NullInt64
	if err := db.QueryRow(`SELECT status, upload_id, size_bytes FROM videos WHERE id=?`, victimID).
		Scan(&status, &uploadID, &size); err != nil {
		t.Fatalf("query victim: %v", err)
	}
	if status != "ready" {
		t.Errorf("victim status = %q, want ready (untouched)", status)
	}
	if uploadID.Valid {
		t.Errorf("victim upload_id hijacked: %v", uploadID.String)
	}

	// No hijacked raw file left behind at the victim's raw path.
	if _, statErr := os.Stat(RawPath(root, victimID)); !os.IsNotExist(statErr) {
		t.Errorf("hijacked raw file must be cleaned up, stat err = %v", statErr)
	}
}

func TestMarkVideoFailed_DoesNotTouchForeignReadyRow(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	owner := testutil.CreateUser(t, db, "standard")
	videoID := testutil.CreateVideo(t, db, team, season, owner, "ready")

	// A rejected hijack attempt eventually calls markVideoFailed; it must not
	// flip a ready row to failed.
	h.markVideoFailed(videoID, "rejected hijack")

	var status string
	if err := db.QueryRow(`SELECT status FROM videos WHERE id=?`, videoID).Scan(&status); err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "ready" {
		t.Errorf("status = %q, want ready (markVideoFailed must only touch uploading rows)", status)
	}
}

// --- Finding 3: CreateUpload validates FK references (no 500 leak) -----------

func TestCreateUpload_UnknownTeamIsBadRequest(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)
	srv := newUploadServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	// Admin passes CanUploadToTeam for ANY team id, so this exercises the FK
	// validation path (not the authorization path).
	admin := testutil.CreateUser(t, db, "admin")
	tok := testutil.Token(t, admin, "admin", nil)

	body := map[string]any{
		"title":      "Phantom-Team",
		"team_id":    999999, // does not exist
		"season_id":  season,
		"size_bytes": 1024,
	}
	res := testutil.Post(t, srv, "/api/videos", tok, body)
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatalf("status = %d, want 400 (unknown team_id must not 500)", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&n)
	if n != 0 {
		t.Errorf("unknown team_id created %d rows, want 0", n)
	}
}

func TestCreateUpload_UnknownSeasonIsBadRequest(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)
	srv := newUploadServer(t, h)

	team := testutil.CreateTeam(t, db, "Team A")
	admin := testutil.CreateUser(t, db, "admin")
	tok := testutil.Token(t, admin, "admin", nil)

	body := map[string]any{
		"title":      "Phantom-Saison",
		"team_id":    team,
		"season_id":  999999, // does not exist
		"size_bytes": 1024,
	}
	res := testutil.Post(t, srv, "/api/videos", tok, body)
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatalf("status = %d, want 400 (unknown season_id must not 500)", res.StatusCode)
	}
}

func TestCreateUpload_UnknownGameIsBadRequest(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := uploadHandler(t, db, 1024)
	srv := newUploadServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	admin := testutil.CreateUser(t, db, "admin")
	tok := testutil.Token(t, admin, "admin", nil)

	bogusGame := 999999
	body := map[string]any{
		"title":      "Phantom-Spiel",
		"team_id":    team,
		"season_id":  season,
		"game_id":    bogusGame,
		"size_bytes": 1024,
	}
	res := testutil.Post(t, srv, "/api/videos", tok, body)
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatalf("status = %d, want 400 (unknown game_id must not 500)", res.StatusCode)
	}
}

// createVideoWithSize inserts a video row with an explicit size_bytes so the
// disk-guard plausibility/announced-size logic can be exercised.
func createVideoWithSize(t *testing.T, db *sql.DB, teamID, seasonID, createdBy int, status string, sizeBytes int64) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO videos (title, team_id, season_id, status, size_bytes, created_by)
		 VALUES ('sec-test', ?, ?, ?, ?, ?)`,
		teamID, seasonID, status, sizeBytes, createdBy)
	if err != nil {
		t.Fatalf("createVideoWithSize: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}
