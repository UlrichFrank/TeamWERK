package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// writeTempFileWithAge writes a file under dir and sets its mtime to
// ageFromNow relative to now (negative = past).
func writeTempFileWithAge(t *testing.T, dir, name string, ageFromNow time.Duration) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mtime := time.Now().Add(ageFromNow)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

// TestCleanStaleVideoDownloadTemp prüft, dass nur Remux-Temp-Dateien älter als
// 1 h gelöscht werden — frische Dateien (laufender Download) bleiben stehen.
func TestCleanStaleVideoDownloadTemp(t *testing.T) {
	db := testutil.NewDB(t)
	root := t.TempDir()
	tmpDir := filepath.Join(root, "tmp")

	writeTempFileWithAge(t, tmpDir, "download-1-old.mp4", -2*time.Hour)
	writeTempFileWithAge(t, tmpDir, "download-2-fresh.mp4", -1*time.Minute)

	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VideoStorageDir: root}
	s := New(db, cfg, nil)
	s.cleanStaleVideoDownloadTemp()

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read tmp dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "download-2-fresh.mp4" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("remaining files = %v, want only download-2-fresh.mp4", names)
	}
}

// TestCleanStaleVideoDownloadTemp_MissingDirIsNoError prüft, dass ein
// (noch) nicht existentes tmp/-Verzeichnis nicht zum Fehler führt (Video-
// Feature noch nie genutzt / Storage nicht angelegt).
func TestCleanStaleVideoDownloadTemp_MissingDirIsNoError(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VideoStorageDir: t.TempDir()}
	s := New(db, cfg, nil)
	s.cleanStaleVideoDownloadTemp() // must not panic
}
