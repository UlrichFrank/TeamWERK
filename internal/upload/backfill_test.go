package upload

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TC: Der Cleanup entfernt nur die Dateien, die in keiner users.photo_path
// mehr referenziert werden — referenzierte bleiben unangetastet, ein zweiter
// Lauf ist ein No-Op.
func TestRunPhotoCleanupBackfill_RemovesOrphans(t *testing.T) {
	tmp := t.TempDir()
	memberDir := filepath.Join(tmp, "member-photos")
	if err := os.MkdirAll(memberDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	referenced := filepath.Join(memberDir, "keep.jpg")
	orphan := filepath.Join(memberDir, "orphan.jpg")
	for _, p := range []string{referenced, orphan} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, photo_path TEXT)`); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, photo_path) VALUES (1, 'member-photos/keep.jpg'), (2, NULL)`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := RunPhotoCleanupBackfill(context.Background(), db, tmp); err != nil {
		t.Fatalf("backfill 1: %v", err)
	}
	if _, err := os.Stat(referenced); err != nil {
		t.Errorf("referenzierte Datei sollte erhalten sein: %v", err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Errorf("verwaiste Datei sollte entfernt sein, err=%v", err)
	}

	// Zweiter Lauf: idempotent.
	if err := RunPhotoCleanupBackfill(context.Background(), db, tmp); err != nil {
		t.Fatalf("backfill 2: %v", err)
	}
}

// TC: Ohne member-photos-Verzeichnis (frische Installation) bricht der
// Backfill nicht ab, sondern loggt und beendet sauber.
func TestRunPhotoCleanupBackfill_MissingDir(t *testing.T) {
	tmp := t.TempDir()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, photo_path TEXT)`); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := RunPhotoCleanupBackfill(context.Background(), db, tmp); err != nil {
		t.Errorf("erwartet kein Fehler bei fehlendem member-photos, bekam: %v", err)
	}
}
