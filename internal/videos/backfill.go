package videos

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

// RunTVCompatBackfill migriert Bestandsvideos einmalig auf das neue
// tv-kompatible HLS-Format (video-tv-streaming):
//   - `videos.codecs` per ffprobe aus seg_001.ts der 720p-Rendition ermitteln
//     (Fallback 360p, falls das Video vor dem 360p-Wegfall angelegt wurde)
//   - `master.m3u8` mit CODECS + `#EXT-X-INDEPENDENT-SEGMENTS` neu schreiben,
//     nur noch 720p referenzieren
//   - vorhandenes `processed/{id}/360p/`-Verzeichnis löschen (Speicherersparnis)
//
// Idempotent via `codecs IS NULL AND status='ready'`. Einzelfehler (z.B.
// fehlende seg_001.ts nach altem manuellem Cleanup) werden geloggt und
// übersprungen — der Gesamtlauf bricht nicht ab. Harte Fehler (DB, Storage-Root
// nicht lesbar) bubbeln hoch, damit der Aufrufer sie behandeln kann.
func RunTVCompatBackfill(ctx context.Context, db *sql.DB, storageDir string) error {
	rows, err := db.Query(
		`SELECT id FROM videos WHERE status='ready' AND codecs IS NULL ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	slog.Info("video tv-compat backfill starting", "count", len(ids))

	migrated := 0
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := backfillOne(ctx, db, storageDir, id); err != nil {
			slog.Error("video tv-compat backfill: skipping video",
				"video_id", id, "error", err)
			continue
		}
		migrated++
	}
	slog.Info("video tv-compat backfill done", "migrated", migrated, "skipped", len(ids)-migrated)
	return nil
}

// probeSegmentCodecsFn ist die für Tests injizierbare ffprobe-Naht. In
// Produktion ist sie `probeSegmentCodecs`; Backfill-Tests injizieren einen
// deterministischen Fake, damit die Suite ohne installiertes ffprobe läuft.
var probeSegmentCodecsFn = probeSegmentCodecs

// backfillOne migriert genau ein Video. Wählt seg_001.ts aus 720p (bevorzugt)
// oder 360p (Fallback), probet Codecs, schreibt DB + master.m3u8 und löscht
// den 360p-Ordner. Wird bereits erfolgreich migrierte Videos nie erneut
// anfassen, weil sie durch die Query gefiltert werden.
func backfillOne(ctx context.Context, db *sql.DB, storageDir string, id int) error {
	processedDir := ProcessedDir(storageDir, id)
	seg := chooseCodecProbeSegment(processedDir)
	if seg == "" {
		return errors.New("no seg_001.ts found in 720p or 360p rendition")
	}
	codecs, err := probeSegmentCodecsFn(ctx, seg)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE videos SET codecs=? WHERE id=?`, codecs, id); err != nil {
		return err
	}
	if err := writeMasterManifest(processedDir, codecs); err != nil {
		return err
	}
	// 360p-Reste löschen — Speicher-Ersparnis für Bestandsvideos zieht die
	// Rendition-Entfernung aus dem Neu-Transcode-Pfad nach.
	old360 := filepath.Join(processedDir, "360p")
	if _, err := os.Stat(old360); err == nil {
		if err := os.RemoveAll(old360); err != nil {
			slog.Warn("video tv-compat backfill: could not remove old 360p dir",
				"video_id", id, "error", err)
		}
	}
	return nil
}

// chooseCodecProbeSegment sucht seg_001.ts zuerst in 720p (Prod-Standard nach
// video-tv-streaming), dann in 360p (Bestand vor Backfill). Liefert "" wenn
// beides fehlt — Backfill überspringt das Video mit Log.
func chooseCodecProbeSegment(processedDir string) string {
	for _, r := range []string{"720p", "360p"} {
		p := filepath.Join(processedDir, r, "seg_001.ts")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
