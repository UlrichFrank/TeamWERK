package upload

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
)

// RunPhotoCleanupBackfill entfernt Foto-Dateien aus dem `member-photos`-
// Unterverzeichnis des uploadDir, die nach Migration 029 nicht mehr aus
// `users.photo_path` referenziert werden. Hintergrund: Vor 029 gab es
// `members.photo_path` als parallelen Slot. Beim Migrationslauf werden
// Dateien von Members ohne User-Account (kein Ziel-User) verwaist; ebenso
// die alten `members.photo_path`-Dateien in Fällen, in denen der Konflikt
// mit `users.photo_path` zugunsten des Users aufgelöst wurde.
//
// Der Backfill läuft idempotent: er scannt das Verzeichnis, sammelt alle
// aktiven `users.photo_path`-Werte, löscht alle Dateien, die dort nicht
// vorkommen. Ein zweiter Lauf ist ein No-Op.
//
// Einzelfehler beim Löschen werden geloggt und übersprungen — der Gesamtlauf
// bricht nicht ab. Harte Fehler (DB, Verzeichnis nicht lesbar) bubbeln hoch.
func RunPhotoCleanupBackfill(ctx context.Context, db *sql.DB, uploadDir string) error {
	if uploadDir == "" {
		return nil
	}
	dir := filepath.Join(uploadDir, "member-photos")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	referenced, err := loadReferencedPhotos(ctx, db)
	if err != nil {
		return err
	}

	var removed, kept int
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.IsDir() {
			continue
		}
		name := e.Name()
		key := filepath.Join("member-photos", name)
		if _, ok := referenced[key]; ok {
			kept++
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			slog.Warn("photo cleanup: remove failed", "file", name, "error", err)
			continue
		}
		removed++
	}
	slog.Info("photo cleanup backfill done", "kept", kept, "removed", removed)
	return nil
}

func loadReferencedPhotos(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT photo_path FROM users WHERE photo_path IS NOT NULL AND photo_path != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out[p] = struct{}{}
	}
	return out, rows.Err()
}
