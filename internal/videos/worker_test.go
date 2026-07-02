package videos

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// fakeBroadcaster zählt Broadcast-Aufrufe pro Event (thread-safe).
type fakeBroadcaster struct {
	mu     sync.Mutex
	events map[string]int
}

func newFakeBroadcaster() *fakeBroadcaster { return &fakeBroadcaster{events: map[string]int{}} }

func (f *fakeBroadcaster) Broadcast(event string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events[event]++
}

func (f *fakeBroadcaster) count(event string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.events[event]
}

// fakeConfig erfüllt workerConfig: liefert ein Temp-Storage-Root, einen kleinen
// Reserved-Wert und sammelt Push-Empfänger.
type fakeConfig struct {
	root     string
	reserved uint64

	mu       sync.Mutex
	pushUIDs [][]int
	pushBody []string
}

func (c *fakeConfig) storageDir() string    { return c.root }
func (c *fakeConfig) reservedBytes() uint64 { return c.reserved }
func (c *fakeConfig) pushSend(userIDs []int, _, body, _ string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pushUIDs = append(c.pushUIDs, userIDs)
	c.pushBody = append(c.pushBody, body)
}

func (c *fakeConfig) lastPush() ([]int, string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pushUIDs) == 0 {
		return nil, "", false
	}
	return c.pushUIDs[len(c.pushUIDs)-1], c.pushBody[len(c.pushBody)-1], true
}

// newTestWorker baut einen Worker mit Fakes; transcode wird vom Aufrufer gesetzt.
func newTestWorker(t *testing.T, db *sql.DB, transcode transcodeFunc) (*Worker, *fakeBroadcaster, *fakeConfig) {
	t.Helper()
	bc := newFakeBroadcaster()
	cfg := &fakeConfig{root: t.TempDir(), reserved: 0}
	// raw/-Verzeichnis anlegen, damit succeed() os.Remove ohne Verzeichnisfehler läuft.
	if err := os.MkdirAll(filepath.Join(cfg.root, "raw"), 0o755); err != nil {
		t.Fatal(err)
	}
	wk := &Worker{
		db:                db,
		hub:               bc,
		cfg:               cfg,
		transcode:         transcode,
		pollInterval:      time.Millisecond,
		diskRetryInterval: time.Millisecond,
		now:               time.Now,
		sleep:             ctxSleep,
	}
	return wk, bc, cfg
}

// writeRaw legt eine Dummy-Rohdatei für ein Video an.
func writeRaw(t *testing.T, root string, id int) string {
	t.Helper()
	p := RawPath(root, id)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("dummy raw"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// fakeCodecs ist der Codec-String, den die Fakes durchreichen — muss zum
// echten ffmpeg-Output-Format passen (`avc1.PPCCLL,mp4a.40.X`), damit
// Assertions gegen master.m3u8 realistisch bleiben.
const fakeCodecs = "avc1.640028,mp4a.40.2"

// fakeHLSTranscode simuliert ffmpeg: legt master.m3u8 + Rendition-Manifeste an
// und liefert einen deterministischen CODECS-String zurück (Naht analog zum
// echten realFFmpegTranscode, das den String aus ffprobe ableitet).
func fakeHLSTranscode(_ context.Context, _ string, processedDir string) (string, error) {
	if err := writeMasterManifest(ensureRenditions(processedDir), fakeCodecs); err != nil {
		return "", err
	}
	return fakeCodecs, nil
}

// ensureRenditions legt die Rendition-Verzeichnisse + Dummy-index.m3u8 an und
// gibt processedDir zurück (für fakeHLSTranscode).
func ensureRenditions(processedDir string) string {
	for _, rd := range workerRenditions {
		_ = os.MkdirAll(filepath.Join(processedDir, rd.name), 0o755)
		_ = os.WriteFile(filepath.Join(processedDir, rd.name, "index.m3u8"), []byte("#EXTM3U\n"), 0o644)
	}
	return processedDir
}

func statusOf(t *testing.T, db *sql.DB, id int) string {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT status FROM videos WHERE id=?`, id).Scan(&s); err != nil {
		t.Fatalf("statusOf(%d): %v", id, err)
	}
	return s
}

// waitFor pollt cond bis true oder Timeout (für Worker-Run im Hintergrund).
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("waitFor: condition not met within timeout")
}

// TestWorkerSerialProcessing: zwei queued Videos werden nacheinander verarbeitet
// (max. eines gleichzeitig 'processing') und beide landen auf 'ready'.
func TestWorkerSerialProcessing(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	v1 := testutil.CreateVideo(t, db, team, season, user, "queued")
	v2 := testutil.CreateVideo(t, db, team, season, user, "queued")

	var (
		mu         sync.Mutex
		concurrent int
		maxConc    int
	)
	tc := func(_ context.Context, raw, processedDir string) (string, error) {
		mu.Lock()
		concurrent++
		if concurrent > maxConc {
			maxConc = concurrent
		}
		mu.Unlock()
		time.Sleep(15 * time.Millisecond) // Überlappung sichtbar machen
		mu.Lock()
		concurrent--
		mu.Unlock()
		return fakeHLSTranscode(context.Background(), raw, processedDir)
	}

	wk, bc, cfg := newTestWorker(t, db, tc)
	writeRaw(t, cfg.root, v1)
	writeRaw(t, cfg.root, v2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go wk.Run(ctx)

	waitFor(t, func() bool {
		return bc.count("video-ready") == 2 &&
			statusOf(t, db, v1) == "ready" && statusOf(t, db, v2) == "ready"
	})
	cancel()

	if maxConc > 1 {
		t.Fatalf("expected serial processing (max 1 concurrent), got %d", maxConc)
	}
	if bc.count("video-ready") != 2 {
		t.Fatalf("expected 2 video-ready broadcasts, got %d", bc.count("video-ready"))
	}
	// Rohdateien müssen nach Erfolg gelöscht sein.
	if _, err := os.Stat(RawPath(cfg.root, v1)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected raw of v1 removed, stat err=%v", err)
	}
	if _, err := os.Stat(RawPath(cfg.root, v2)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected raw of v2 removed, stat err=%v", err)
	}
	// master.m3u8 muss eine streaming-kompatible Rendition-Zeile enthalten.
	master, err := os.ReadFile(MasterManifestPath(cfg.root, v1))
	if err != nil {
		t.Fatalf("read master: %v", err)
	}
	if !renditionLinePrefix.MatchString("720p/index.m3u8") {
		t.Fatal("sanity: renditionLinePrefix should match 720p/index.m3u8")
	}
	if !containsLine(string(master), "720p/index.m3u8") {
		t.Fatalf("master.m3u8 missing 720p rendition line:\n%s", master)
	}
	if containsLine(string(master), "360p/index.m3u8") {
		t.Fatalf("master.m3u8 unexpectedly contains 360p rendition line:\n%s", master)
	}
	if !strings.Contains(string(master), "CODECS=") {
		t.Fatalf("master.m3u8 missing CODECS attribute (AirPlay needs it):\n%s", master)
	}
	if !containsLine(string(master), "#EXT-X-INDEPENDENT-SEGMENTS") {
		t.Fatalf("master.m3u8 missing #EXT-X-INDEPENDENT-SEGMENTS:\n%s", master)
	}
	// Codecs sind in videos.codecs persistiert — Play-Route + Backfill lesen daraus.
	var codecs sql.NullString
	if err := db.QueryRow(`SELECT codecs FROM videos WHERE id=?`, v1).Scan(&codecs); err != nil {
		t.Fatalf("select codecs: %v", err)
	}
	if !codecs.Valid || codecs.String != fakeCodecs {
		t.Fatalf("expected codecs=%q, got %+v", fakeCodecs, codecs)
	}
}

// containsLine prüft, ob s eine exakt getrimmte Zeile == want enthält.
func containsLine(s, want string) bool {
	for _, line := range splitLines(s) {
		if line == want {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

// TestBuildFFmpegRenditionArgs_ContainsPixFmt: `-pix_fmt yuv420p` MUSS in der
// Arg-Liste stehen — sonst produziert libx264 bei 10-bit-Quellen yuv420p10, das
// AppleTV/tvOS nicht dekodiert (Ton ohne Bild).
func TestBuildFFmpegRenditionArgs_ContainsPixFmt(t *testing.T) {
	rd := workerRenditions[0]
	for _, aacSource := range []bool{true, false} {
		args := buildFFmpegRenditionArgs("/raw/1.mp4", "/dir", rd, aacSource)
		if !hasArgSequence(args, "-pix_fmt", "yuv420p") {
			t.Fatalf("aacSource=%v: -pix_fmt yuv420p missing in %v", aacSource, args)
		}
		// Reihenfolge: -pix_fmt vor -c:v libx264, damit ffmpeg das Ausgabeformat
		// vor der Codec-Wahl anwendet.
		if idxOf(args, "-pix_fmt") > idxOf(args, "-c:v") {
			t.Fatalf("aacSource=%v: -pix_fmt must come before -c:v in %v", aacSource, args)
		}
	}
}

// hasArgSequence prüft, ob a und b als benachbarte Elemente im Slice stehen.
func hasArgSequence(args []string, a, b string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == a && args[i+1] == b {
			return true
		}
	}
	return false
}

// idxOf liefert den ersten Index von needle in args oder -1.
func idxOf(args []string, needle string) int {
	for i, s := range args {
		if s == needle {
			return i
		}
	}
	return -1
}

// TestWorkerFailurePath: transcode liefert Fehler → status=failed, raw bleibt.
func TestWorkerFailurePath(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	v := testutil.CreateVideo(t, db, team, season, user, "queued")

	tc := func(_ context.Context, _, _ string) (string, error) {
		return "", errors.New("ffmpeg exploded")
	}
	wk, bc, cfg := newTestWorker(t, db, tc)
	rawPath := writeRaw(t, cfg.root, v)

	wk.process(context.Background(), v)

	if got := statusOf(t, db, v); got != "failed" {
		t.Fatalf("expected status=failed, got %q", got)
	}
	var reason sql.NullString
	if err := db.QueryRow(`SELECT failure_reason FROM videos WHERE id=?`, v).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if !reason.Valid || reason.String == "" {
		t.Fatal("expected failure_reason to be set")
	}
	// Rohdatei muss für Debug erhalten bleiben.
	if _, err := os.Stat(rawPath); err != nil {
		t.Fatalf("expected raw kept after failure, stat err=%v", err)
	}
	if bc.count("video-ready") != 0 {
		t.Fatal("expected no video-ready broadcast on failure")
	}
}

// TestWorkerDiskShortage: free < needed → bleibt queued (nicht failed), transcode
// wird nicht aufgerufen.
func TestWorkerDiskShortage(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	// size_bytes riesig setzen, damit estimateNeeded > freier Platz wird.
	v := testutil.CreateVideo(t, db, team, season, user, "queued")
	hugeBytes := int64(1) << 60 // 1 EiB → garantiert mehr als jeder reale Free-Space
	if _, err := db.Exec(`UPDATE videos SET size_bytes=? WHERE id=?`, hugeBytes, v); err != nil {
		t.Fatal(err)
	}

	called := false
	tc := func(_ context.Context, _, _ string) (string, error) {
		called = true
		return "", nil
	}
	wk, _, cfg := newTestWorker(t, db, tc)
	writeRaw(t, cfg.root, v)

	wk.process(context.Background(), v)

	if called {
		t.Fatal("transcode must not run when disk is insufficient")
	}
	if got := statusOf(t, db, v); got != "queued" {
		t.Fatalf("expected status to stay queued on disk shortage, got %q", got)
	}
}

// TestWorkerCrashRecovery: hängende 'processing'-Jobs werden beim Start auf
// 'queued' zurückgesetzt.
func TestWorkerCrashRecovery(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	v := testutil.CreateVideo(t, db, team, season, user, "processing")

	wk, _, _ := newTestWorker(t, db, fakeHLSTranscode)
	wk.recoverStuck()

	if got := statusOf(t, db, v); got != "queued" {
		t.Fatalf("expected stuck processing job reset to queued, got %q", got)
	}
}

// TestWorkerClaimGuard: ein bereits 'processing'-Video wird von process() nicht
// erneut übernommen (Claim-Guard schützt gegen Doppel-Pick).
func TestWorkerClaimGuard(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	v := testutil.CreateVideo(t, db, team, season, user, "processing")

	called := false
	tc := func(_ context.Context, _, _ string) (string, error) {
		called = true
		return "", nil
	}
	wk, _, cfg := newTestWorker(t, db, tc)
	writeRaw(t, cfg.root, v)

	wk.process(context.Background(), v)

	if called {
		t.Fatal("transcode must not run for a video not in 'queued' state")
	}
	if got := statusOf(t, db, v); got != "processing" {
		t.Fatalf("expected status unchanged (processing), got %q", got)
	}
}

// TestWorkerPushRecipients: Ready-Push erreicht Hochladenden + Team-Spieler +
// dessen Eltern + Trainer (distinkt).
func TestWorkerPushRecipients(t *testing.T) {
	db := testutil.NewDB(t)

	uploader := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")

	// Kader des Teams (player_memberships/trainer_memberships sind Views darüber).
	kaderID := testutil.CreateKader(t, db, team, season)

	// aktiver Spieler mit Account (über kader_members → player_memberships-View).
	playerUser := testutil.CreateUser(t, db, "standard")
	playerMember := testutil.CreateMember(t, db, playerUser)
	if _, err := db.Exec(
		`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, playerMember); err != nil {
		t.Fatal(err)
	}

	// Elternteil des Spielers
	parentUser := testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUser, playerMember); err != nil {
		t.Fatal(err)
	}

	// Trainer des Teams (über kader_trainers → trainer_memberships-View).
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMember)

	v := testutil.CreateVideo(t, db, team, season, uploader, "queued")
	wk, _, cfg := newTestWorker(t, db, fakeHLSTranscode)
	writeRaw(t, cfg.root, v)

	wk.process(context.Background(), v)
	waitFor(t, func() bool { _, _, ok := cfg.lastPush(); return ok })

	uids, body, _ := cfg.lastPush()
	want := map[int]bool{uploader: true, playerUser: true, parentUser: true, trainerUser: true}
	got := map[int]bool{}
	for _, u := range uids {
		got[u] = true
	}
	for u := range want {
		if !got[u] {
			t.Fatalf("expected push recipient %d in %v", u, uids)
		}
	}
	// keine NULL/0-IDs
	for _, u := range uids {
		if u == 0 {
			t.Fatalf("unexpected zero user id in recipients %v", uids)
		}
	}
	if body == "" {
		t.Fatal("expected non-empty push body")
	}
}
