package media

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// mediaRow legt eine media-Zeile an und liefert die vergebene ID.
func mediaRow(t *testing.T, db *sql.DB, diskName, mimeType string, width, height sql.NullInt64) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO media (disk_name, mime_type, size, uploaded_by, width, height) VALUES (?, ?, ?, ?, ?, ?)`,
		diskName, mimeType, 100, 1, width, height)
	if err != nil {
		t.Fatalf("insert media: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func TestBackfill_UpdatesNullRows(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateUser(t, db, "standard")
	dir := t.TempDir()

	// Realistisches PNG in mediaDir ablegen und passende media-Zeile mit
	// NULL-Dimensionen anlegen.
	diskName := "backfill-real.png"
	pngData := realPNGBytes(t, 4, 3)
	if err := os.WriteFile(filepath.Join(dir, diskName), pngData, 0644); err != nil {
		t.Fatalf("write png: %v", err)
	}
	id := mediaRow(t, db, diskName, "image/png", sql.NullInt64{}, sql.NullInt64{})

	if err := Backfill(context.Background(), db, dir); err != nil {
		t.Fatalf("Backfill: %v", err)
	}

	var w, h sql.NullInt64
	db.QueryRow(`SELECT width, height FROM media WHERE id = ?`, id).Scan(&w, &h)
	if !w.Valid || !h.Valid || w.Int64 != 4 || h.Int64 != 3 {
		t.Errorf("expected width=4 height=3, got %v %v", w, h)
	}
}

func TestBackfill_SkipsAlreadyFilledRows(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateUser(t, db, "standard")
	dir := t.TempDir()

	// Zeile mit bereits gesetzten Dimensionen; wir legen bewusst KEINE Datei
	// auf Disk an — wenn Backfill fälschlich auf sie zugriffe, würde
	// os.Open fehlschlagen und der Test schlägt (durch fehlerhaftes Verhalten
	// ansonsten schwierig zu erkennen).
	id := mediaRow(t, db, "not-on-disk.png", "image/png",
		sql.NullInt64{Int64: 100, Valid: true},
		sql.NullInt64{Int64: 200, Valid: true})

	if err := Backfill(context.Background(), db, dir); err != nil {
		t.Fatalf("Backfill: %v", err)
	}

	var w, h sql.NullInt64
	db.QueryRow(`SELECT width, height FROM media WHERE id = ?`, id).Scan(&w, &h)
	if w.Int64 != 100 || h.Int64 != 200 {
		t.Errorf("expected unchanged 100x200, got %dx%d", w.Int64, h.Int64)
	}
}

func TestBackfill_MissingFileContinues(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateUser(t, db, "standard")
	dir := t.TempDir()

	// Zwei Zeilen mit NULL-Dimensionen: erste ohne Datei (fehlt), zweite mit
	// echter Datei. Backfill muss die zweite trotzdem verarbeiten.
	missingID := mediaRow(t, db, "gone.png", "image/png", sql.NullInt64{}, sql.NullInt64{})

	goodDisk := "good.png"
	if err := os.WriteFile(filepath.Join(dir, goodDisk), realPNGBytes(t, 5, 5), 0644); err != nil {
		t.Fatalf("write good png: %v", err)
	}
	goodID := mediaRow(t, db, goodDisk, "image/png", sql.NullInt64{}, sql.NullInt64{})

	if err := Backfill(context.Background(), db, dir); err != nil {
		t.Fatalf("Backfill: %v", err)
	}

	var w, h sql.NullInt64
	db.QueryRow(`SELECT width, height FROM media WHERE id = ?`, missingID).Scan(&w, &h)
	if w.Valid || h.Valid {
		t.Errorf("missing-file row should stay NULL, got %v %v", w, h)
	}
	db.QueryRow(`SELECT width, height FROM media WHERE id = ?`, goodID).Scan(&w, &h)
	if !w.Valid || w.Int64 != 5 || h.Int64 != 5 {
		t.Errorf("good row should be filled 5x5, got %v %v", w, h)
	}
}
