package videos

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// writeLegacyProcessed simuliert einen Bestandsvideo-Ordner vor Backfill:
// beide Renditions (720p + 360p) mit seg_001.ts, master.m3u8 ohne CODECS.
func writeLegacyProcessed(t *testing.T, root string, id int, with360 bool) {
	t.Helper()
	pd := ProcessedDir(root, id)
	renditions := []string{"720p"}
	if with360 {
		renditions = append(renditions, "360p")
	}
	for _, r := range renditions {
		dir := filepath.Join(pd, r)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "seg_001.ts"), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.m3u8"), []byte("#EXTM3U\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	master := "#EXTM3U\n#EXT-X-VERSION:3\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=2800000,RESOLUTION=1280x720\n720p/index.m3u8\n"
	if with360 {
		master += "#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=640x360\n360p/index.m3u8\n"
	}
	if err := os.WriteFile(MasterManifestPath(root, id), []byte(master), 0o644); err != nil {
		t.Fatal(err)
	}
}

// withFakeProbe ersetzt die ffprobe-Naht für die Dauer des Tests. Rückgabewert
// wird vom Backfill als Codec-String gelesen.
func withFakeProbe(t *testing.T, codecs string, err error) {
	t.Helper()
	prev := probeSegmentCodecsFn
	probeSegmentCodecsFn = func(_ context.Context, _ string) (string, error) {
		return codecs, err
	}
	t.Cleanup(func() { probeSegmentCodecsFn = prev })
}

// TestBackfill_Idempotent: erste Ausführung migriert das Video, zweite ist
// ein No-Op (Video ist nicht mehr in der Query).
func TestBackfill_Idempotent(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	v := testutil.CreateVideo(t, db, team, season, user, "ready")

	root := t.TempDir()
	writeLegacyProcessed(t, root, v, true)
	withFakeProbe(t, "avc1.640028,mp4a.40.2", nil)

	if err := RunTVCompatBackfill(context.Background(), db, root); err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	// codecs in DB
	var codecs sql.NullString
	if err := db.QueryRow(`SELECT codecs FROM videos WHERE id=?`, v).Scan(&codecs); err != nil {
		t.Fatal(err)
	}
	if !codecs.Valid || codecs.String != "avc1.640028,mp4a.40.2" {
		t.Fatalf("codecs = %+v, want avc1.640028,mp4a.40.2", codecs)
	}
	// master.m3u8 hat jetzt CODECS + INDEPENDENT-SEGMENTS, kein 360p
	master, _ := os.ReadFile(MasterManifestPath(root, v))
	s := string(master)
	if !strings.Contains(s, "CODECS=") {
		t.Errorf("master.m3u8 missing CODECS after backfill:\n%s", s)
	}
	if !strings.Contains(s, "#EXT-X-INDEPENDENT-SEGMENTS") {
		t.Errorf("master.m3u8 missing INDEPENDENT-SEGMENTS after backfill:\n%s", s)
	}
	if strings.Contains(s, "360p/index.m3u8") {
		t.Errorf("master.m3u8 still references 360p after backfill:\n%s", s)
	}
	// 360p-Dir gelöscht
	if _, err := os.Stat(filepath.Join(ProcessedDir(root, v), "360p")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("360p dir should be gone, stat err=%v", err)
	}

	// zweiter Lauf: kein Probe-Aufruf mehr (Video ist nicht mehr in der Query)
	probeCalled := false
	prev := probeSegmentCodecsFn
	probeSegmentCodecsFn = func(_ context.Context, _ string) (string, error) {
		probeCalled = true
		return "should-not-be-called", nil
	}
	t.Cleanup(func() { probeSegmentCodecsFn = prev })

	if err := RunTVCompatBackfill(context.Background(), db, root); err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if probeCalled {
		t.Error("second backfill run must not call probe (idempotency)")
	}
}

// TestBackfill_UeberspringtFehlerhaftesVideoOhneAbbruch: bei zwei Videos, eines
// mit fehlender seg_001.ts, migriert der Lauf das gute Video und überspringt
// das kaputte mit Log.
func TestBackfill_UeberspringtFehlerhaftesVideoOhneAbbruch(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	vBad := testutil.CreateVideo(t, db, team, season, user, "ready")
	vGood := testutil.CreateVideo(t, db, team, season, user, "ready")

	root := t.TempDir()
	// vBad hat keinen processed/-Ordner → seg_001.ts fehlt
	writeLegacyProcessed(t, root, vGood, false)
	withFakeProbe(t, "avc1.640028,mp4a.40.2", nil)

	if err := RunTVCompatBackfill(context.Background(), db, root); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var badCodecs, goodCodecs sql.NullString
	_ = db.QueryRow(`SELECT codecs FROM videos WHERE id=?`, vBad).Scan(&badCodecs)
	_ = db.QueryRow(`SELECT codecs FROM videos WHERE id=?`, vGood).Scan(&goodCodecs)
	if badCodecs.Valid {
		t.Errorf("broken video should not have codecs set, got %q", badCodecs.String)
	}
	if !goodCodecs.Valid || goodCodecs.String == "" {
		t.Errorf("good video should have codecs set, got %+v", goodCodecs)
	}
}

// TestBackfill_LaesstNichtReadyUnberuehrt: uploading/queued/processing/failed
// werden nicht angefasst — nur ready mit codecs IS NULL.
func TestBackfill_LaesstNichtReadyUnberuehrt(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	vQueued := testutil.CreateVideo(t, db, team, season, user, "queued")

	root := t.TempDir()
	writeLegacyProcessed(t, root, vQueued, false)
	withFakeProbe(t, "avc1.640028,mp4a.40.2", nil)

	if err := RunTVCompatBackfill(context.Background(), db, root); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	var codecs sql.NullString
	_ = db.QueryRow(`SELECT codecs FROM videos WHERE id=?`, vQueued).Scan(&codecs)
	if codecs.Valid {
		t.Errorf("queued video should not be backfilled, got codecs=%q", codecs.String)
	}
}

// TestDiskUsageBackfill_MisstUndPersistiert: ein 'ready'-Video ohne
// disk_bytes bekommt die tatsächliche processed/{id}/-Größe eingetragen.
func TestDiskUsageBackfill_MisstUndPersistiert(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	v := testutil.CreateVideo(t, db, team, season, user, "ready")

	root := t.TempDir()
	writeLegacyProcessed(t, root, v, false) // "fake" (4 bytes) + index.m3u8 (8 bytes) in 720p/, plus master.m3u8

	if err := RunDiskUsageBackfill(context.Background(), db, root); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	want, err := DirSize(ProcessedDir(root, v))
	if err != nil {
		t.Fatalf("DirSize: %v", err)
	}
	if want == 0 {
		t.Fatal("test fixture produced an empty processed dir")
	}

	var got sql.NullInt64
	if err := db.QueryRow(`SELECT disk_bytes FROM videos WHERE id=?`, v).Scan(&got); err != nil {
		t.Fatalf("query disk_bytes: %v", err)
	}
	if !got.Valid || got.Int64 != want {
		t.Errorf("disk_bytes = %+v, want %d", got, want)
	}
}

// TestDiskUsageBackfill_Idempotent: ein zweiter Lauf lässt bereits befüllte
// Zeilen unangetastet (die Query filtert auf disk_bytes IS NULL).
func TestDiskUsageBackfill_Idempotent(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	v := testutil.CreateVideo(t, db, team, season, user, "ready")

	root := t.TempDir()
	writeLegacyProcessed(t, root, v, false)

	if err := RunDiskUsageBackfill(context.Background(), db, root); err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	// processed/ nach dem ersten Lauf wachsen lassen — ein zweiter Lauf darf
	// disk_bytes NICHT überschreiben, weil die Zeile nicht mehr NULL ist.
	if err := os.WriteFile(filepath.Join(ProcessedDir(root, v), "720p", "seg_002.ts"), []byte("mehr-daten"), 0o644); err != nil {
		t.Fatal(err)
	}
	var before sql.NullInt64
	_ = db.QueryRow(`SELECT disk_bytes FROM videos WHERE id=?`, v).Scan(&before)

	if err := RunDiskUsageBackfill(context.Background(), db, root); err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	var after sql.NullInt64
	_ = db.QueryRow(`SELECT disk_bytes FROM videos WHERE id=?`, v).Scan(&after)
	if after.Int64 != before.Int64 {
		t.Errorf("second backfill run must be a no-op, before=%d after=%d", before.Int64, after.Int64)
	}
}

// TestDiskUsageBackfill_LaesstNichtReadyUnberuehrt: nur 'ready' wird gemessen.
func TestDiskUsageBackfill_LaesstNichtReadyUnberuehrt(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	team := testutil.CreateTeam(t, db, "D1")
	season := testutil.CreateSeason(t, db, "2025/26")
	v := testutil.CreateVideo(t, db, team, season, user, "failed")

	root := t.TempDir()
	if err := RunDiskUsageBackfill(context.Background(), db, root); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	var got sql.NullInt64
	_ = db.QueryRow(`SELECT disk_bytes FROM videos WHERE id=?`, v).Scan(&got)
	if got.Valid {
		t.Errorf("failed video should not be measured, got disk_bytes=%d", got.Int64)
	}
}
