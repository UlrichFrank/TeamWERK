package media

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
)

// Backfill füllt für Bestandsbilder in der media-Tabelle die width/height-
// Spalten aus dem Datei-Header. Idempotent via `WHERE width IS NULL` — beim
// zweiten Start ohne neue NULL-Zeilen wird kein Datei-I/O gemacht. Läuft
// sequenziell (VPS-Speicher schonen), Fehler pro Datei (fehlend, korrupter
// Header, unbekanntes Format) werden geloggt und übersprungen; der Gesamtlauf
// bricht nicht ab. Muster kopiert aus internal/videos/backfill.go.
func Backfill(ctx context.Context, db *sql.DB, mediaDir string) error {
	rows, err := db.QueryContext(ctx,
		`SELECT id, disk_name, mime_type FROM media WHERE width IS NULL ORDER BY id`)
	if err != nil {
		return err
	}
	type row struct {
		id       int
		diskName string
		mimeType string
	}
	var items []row
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			rows.Close()
			return err
		}
		var it row
		if err := rows.Scan(&it.id, &it.diskName, &it.mimeType); err != nil {
			rows.Close()
			return err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(items) == 0 {
		slog.Info("media dimensions backfill: nothing to do")
		return nil
	}
	slog.Info("media dimensions backfill starting", "count", len(items))

	migrated := 0
	for _, it := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := backfillOne(ctx, db, mediaDir, it.id, it.diskName, it.mimeType); err != nil {
			slog.Error("media dimensions backfill: skipping row",
				"media_id", it.id, "disk_name", it.diskName, "error", err)
			continue
		}
		migrated++
	}
	slog.Info("media dimensions backfill done",
		"migrated", migrated, "skipped", len(items)-migrated)
	return nil
}

func backfillOne(ctx context.Context, db *sql.DB, mediaDir string, id int, diskName, mimeType string) error {
	path := filepath.Join(mediaDir, diskName)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Cap auf maxImageBytes; korrupte Dateien mit falscher Größe können den
	// Prozess sonst mit einem unerwartet großen Read ausbremsen. Ein Bild
	// über 1 MB dürfte per Upload-Limit nie in die Tabelle gekommen sein.
	buf := make([]byte, maxImageBytes)
	n, readErr := f.Read(buf)
	if n == 0 && readErr != nil {
		return readErr
	}
	w, h, ok := decodeDimensions(buf[:n], mimeType)
	if !ok {
		return errUndecodable
	}
	_, err = db.ExecContext(ctx,
		`UPDATE media SET width=?, height=? WHERE id=?`, w, h, id)
	return err
}

// errUndecodable wird geworfen, wenn decodeDimensions den Header nicht liest
// (z. B. bei einem unbekannten Format). Als eigener Fehler, damit er in den
// Logs klar von I/O-Problemen unterscheidbar ist.
var errUndecodable = &backfillError{"cannot decode image dimensions"}

type backfillError struct{ msg string }

func (e *backfillError) Error() string { return e.msg }
