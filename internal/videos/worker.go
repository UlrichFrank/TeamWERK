package videos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/push"
)

// renditions beschreibt die zwei erzeugten HLS-Varianten. Die Verzeichnisnamen
// (720p/360p) und das Segment-/Manifest-Schema MÜSSEN byte-kompatibel zur
// Streaming-Schicht bleiben (stream.go: renditionRe `^[0-9]{3,4}p$`, segmentRe
// `^(index\.m3u8|seg_[0-9]{1,6}\.ts)$`, ServeMaster erwartet `{rendition}/index.m3u8`).
type rendition struct {
	name      string // Verzeichnisname, z.B. "720p"
	height    int    // Skalierungs-Zielhöhe
	maxrate   string // -maxrate
	bufsize   string // -bufsize
	bandwidth int    // EXT-X-STREAM-INF BANDWIDTH (bit/s, grobe Schätzung)
	width     int    // EXT-X-STREAM-INF RESOLUTION-Breite (16:9-Annahme)
}

var workerRenditions = []rendition{
	{name: "720p", height: 720, maxrate: "2800k", bufsize: "5600k", bandwidth: 2_800_000, width: 1280},
	{name: "360p", height: 360, maxrate: "800k", bufsize: "1600k", bandwidth: 800_000, width: 640},
}

const (
	// defaultPollInterval ist der Idle-Schlaf, wenn keine Jobs anstehen (4.1).
	defaultPollInterval = 30 * time.Second
	// defaultDiskRetryInterval ist der Schlaf bei Disk-Mangel vor erneutem
	// Versuch — Video bleibt 'queued', wird NICHT 'failed' (4.3).
	defaultDiskRetryInterval = time.Hour
	// fallbackEstimateBytes wird als geschätzte Output-Größe genutzt, wenn
	// size_bytes unbekannt ist (NULL/0). Bewusst großzügig.
	fallbackEstimateBytes uint64 = 2 << 30 // 2 GiB
)

// transcodeFunc ist die injizierbare Naht (4.4/4.10): Produktion ruft echtes
// ffmpeg, Tests injizieren eine Fake, die nur HLS-Dummy-Dateien schreibt. Sie
// transcodiert rawPath in das ProcessedDir des Videos (HLS 720p+360p + master).
type transcodeFunc func(ctx context.Context, rawPath, processedDir string) error

// Worker zieht serielle (eine Goroutine) Transcode-Jobs aus der DB.
type Worker struct {
	db  *sql.DB
	hub broadcaster
	cfg workerConfig

	// transcode ist die ffmpeg-Naht; in NewWorker auf realFFmpegTranscode gesetzt.
	transcode transcodeFunc

	// pollInterval / diskRetryInterval sind für Tests injizierbar.
	pollInterval      time.Duration
	diskRetryInterval time.Duration

	// now ist für Tests injizierbar (Default time.Now).
	now func() time.Time

	// sleep wartet auf d oder ctx-Ende; für Tests injizierbar.
	sleep func(ctx context.Context, d time.Duration)
}

// broadcaster ist die kleine Teilmenge von *hub.EventHub, die der Worker nutzt —
// erleichtert Tests (Fake-Hub) ohne das ganze Hub-Paket nachzubauen.
type broadcaster interface {
	Broadcast(event string)
}

// workerConfig ist die Teilmenge der App-Config, die der Worker braucht.
type workerConfig interface {
	storageDir() string
	reservedBytes() uint64
	pushSend(userIDs []int, title, body, url string)
}

// NewWorker baut einen Produktions-Worker mit echtem ffmpeg und den Defaults für
// Poll-/Retry-Intervalle. h liefert DB, Hub und Config.
func NewWorker(h *Handler) *Worker {
	return &Worker{
		db:                h.db,
		hub:               h.hub,
		cfg:               handlerWorkerConfig{h},
		transcode:         realFFmpegTranscode,
		pollInterval:      defaultPollInterval,
		diskRetryInterval: defaultDiskRetryInterval,
		now:               time.Now,
		sleep:             ctxSleep,
	}
}

// handlerWorkerConfig adaptiert den Handler/Config an workerConfig (Produktion).
type handlerWorkerConfig struct{ h *Handler }

func (c handlerWorkerConfig) storageDir() string    { return c.h.cfg.VideoStorageDir }
func (c handlerWorkerConfig) reservedBytes() uint64 { return c.h.cfg.VideoReservedBytes }
func (c handlerWorkerConfig) pushSend(userIDs []int, title, body, url string) {
	push.SendToUsers(c.h.db, c.h.cfg, userIDs, title, body, url)
}

// ctxSleep schläft d lang oder kehrt früher zurück, wenn ctx endet.
func ctxSleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// Run startet die serielle Worker-Schleife (genau EINE Goroutine pro Prozess).
// Bei Start werden hängende 'processing'-Jobs auf 'queued' zurückgesetzt
// (Crash-Recovery, 4.2). Die Schleife endet, wenn ctx abgebrochen wird (4.9).
func (wk *Worker) Run(ctx context.Context) {
	wk.recoverStuck()
	slog.Info("video transcode worker started")
	for {
		if ctx.Err() != nil {
			slog.Info("video transcode worker stopped")
			return
		}
		id, ok, err := wk.pickNextQueued()
		if err != nil {
			slog.Error("video worker pickNextQueued failed", "error", err)
			wk.sleep(ctx, wk.pollInterval)
			continue
		}
		if !ok {
			wk.sleep(ctx, wk.pollInterval)
			continue
		}
		wk.process(ctx, id)
	}
}

// recoverStuck setzt beim Start hängende 'processing'-Jobs zurück auf 'queued'
// (4.2). Ein vorheriger Crash/Restart mitten im Transcode darf ein Video nicht
// dauerhaft blockieren.
func (wk *Worker) recoverStuck() {
	res, err := wk.db.Exec(`UPDATE videos SET status='queued' WHERE status='processing'`)
	if err != nil {
		slog.Error("video worker crash-recovery failed", "error", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		slog.Info("video worker recovered stuck jobs", "count", n)
	}
}

// pickNextQueued liefert die ID des ältesten wartenden Videos (4.1). ok=false,
// wenn keines wartet.
func (wk *Worker) pickNextQueued() (int, bool, error) {
	var id int
	err := wk.db.QueryRow(
		`SELECT id FROM videos WHERE status='queued' ORDER BY created_at LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// process verarbeitet genau ein Video: Disk-Check → Claim → Transcode →
// ready/failed. Bei Disk-Mangel bleibt das Video 'queued' und der Worker schläft
// (4.3); es wird NICHT 'failed'.
func (wk *Worker) process(ctx context.Context, id int) {
	root := wk.cfg.storageDir()

	// 4.3 Pre-Transcode-Disk-Check: free ≥ geschätzte Output-Größe × 1.5.
	needed := wk.estimateNeeded(id)
	if free, err := FreeBytes(root); err != nil {
		slog.Error("video worker disk check failed", "video_id", id, "error", err)
		wk.sleep(ctx, wk.diskRetryInterval)
		return
	} else if free < needed {
		slog.Warn("video worker: insufficient disk for transcode, staying queued",
			"video_id", id, "free", free, "needed", needed)
		wk.sleep(ctx, wk.diskRetryInterval)
		return
	}

	// 4.4 Claim: nur übernehmen, wenn noch 'queued' (Schutz gegen Doppel-Pick).
	res, err := wk.db.Exec(
		`UPDATE videos SET status='processing' WHERE id=? AND status='queued'`, id)
	if err != nil {
		slog.Error("video worker claim failed", "video_id", id, "error", err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Schon von woanders übernommen oder Status geändert — nichts tun.
		return
	}

	rawPath := RawPath(root, id)
	processedDir := ProcessedDir(root, id)

	if err := wk.transcode(ctx, rawPath, processedDir); err != nil {
		// Abbruch durch Shutdown ist kein Failure: zurück auf 'queued', damit der
		// nächste Prozess-Start es erneut versucht.
		if ctx.Err() != nil {
			_, _ = wk.db.Exec(`UPDATE videos SET status='queued' WHERE id=? AND status='processing'`, id)
			return
		}
		wk.fail(id, err.Error())
		return
	}

	wk.succeed(id)
}

// estimateNeeded schätzt den nötigen freien Platz für den Transcode (4.3):
// geschätzte Output-Größe × 1.5. Die Output-Größe wird grob aus size_bytes der
// Quelle abgeleitet (HLS H.264 ist meist kleiner, aber wir sind konservativ und
// rechnen mit der vollen Quellgröße als Output-Schätzung). Ist size_bytes
// unbekannt, gilt fallbackEstimateBytes.
func (wk *Worker) estimateNeeded(id int) uint64 {
	var sb sql.NullInt64
	_ = wk.db.QueryRow(`SELECT size_bytes FROM videos WHERE id=?`, id).Scan(&sb)
	est := fallbackEstimateBytes
	if sb.Valid && sb.Int64 > 0 {
		est = uint64(sb.Int64)
	}
	return est + est/2 // × 1.5
}

// succeed markiert ein Video als 'ready', löscht die Rohdatei, broadcastet
// "video-ready" und stößt die Push-Notification an (4.6/4.8).
func (wk *Worker) succeed(id int) {
	if _, err := wk.db.Exec(
		`UPDATE videos SET status='ready', ready_at=CURRENT_TIMESTAMP, failure_reason=NULL WHERE id=?`,
		id); err != nil {
		slog.Error("video worker mark ready failed", "video_id", id, "error", err)
		return
	}
	// 4.6 Rohdatei löschen — Original wird nach Transcode nicht behalten.
	if err := os.Remove(RawPath(wk.cfg.storageDir(), id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("video worker: could not delete raw file", "video_id", id, "error", err)
	}
	wk.hub.Broadcast("video-ready")
	wk.notifyReady(id)
	slog.Info("video transcoded", "video_id", id)
}

// fail markiert ein Video als 'failed' mit Begründung; die Rohdatei BLEIBT für
// Debug erhalten (Cleanup nach 7 Tagen via Scheduler, 4.7).
func (wk *Worker) fail(id int, reason string) {
	if _, err := wk.db.Exec(
		`UPDATE videos SET status='failed', failure_reason=? WHERE id=?`, reason, id); err != nil {
		slog.Error("video worker mark failed failed", "video_id", id, "error", err)
	}
	slog.Error("video transcode failed", "video_id", id, "reason", reason)
}

// notifyReady sammelt die Empfänger und schickt die Push nicht-blockierend (4.8):
// Hochladender + aktive Team-Spieler + deren Eltern + Team-Trainer.
func (wk *Worker) notifyReady(id int) {
	var title, teamName string
	if err := wk.db.QueryRow(
		`SELECT v.title, t.name FROM videos v JOIN teams t ON t.id = v.team_id WHERE v.id=?`,
		id).Scan(&title, &teamName); err != nil {
		slog.Error("video worker: load push meta failed", "video_id", id, "error", err)
		return
	}
	uids, err := wk.pushRecipients(id)
	if err != nil {
		slog.Error("video worker: collect push recipients failed", "video_id", id, "error", err)
		return
	}
	if len(uids) == 0 {
		return
	}
	body := fmt.Sprintf("Neues Video: %s — %s", teamName, title)
	url := "/videos/" + strconv.Itoa(id)
	go wk.cfg.pushSend(uids, "Neues Video", body, url)
}

// pushRecipients liefert die distinkten User-IDs für die Ready-Push (4.8):
// Hochladender, aktive Spieler des Teams, deren Eltern, Team-Trainer — jeweils
// in der aktiven Saison. NULL user_ids (Spieler ohne Account) fallen via
// IS NOT NULL heraus.
func (wk *Worker) pushRecipients(id int) ([]int, error) {
	rows, err := wk.db.Query(`
		SELECT u FROM (
			-- Hochladender
			SELECT v.created_by AS u FROM videos v WHERE v.id = ?1
			UNION
			-- aktive Spieler des Teams
			SELECT m.user_id AS u
			FROM videos v
			JOIN player_memberships pm ON pm.team_id = v.team_id AND pm.season_id = v.season_id
			JOIN members m ON m.id = pm.member_id AND m.status = 'aktiv'
			WHERE v.id = ?1 AND m.user_id IS NOT NULL
			UNION
			-- Eltern aktiver Spieler des Teams
			SELECT fl.parent_user_id AS u
			FROM videos v
			JOIN player_memberships pm ON pm.team_id = v.team_id AND pm.season_id = v.season_id
			JOIN members m ON m.id = pm.member_id AND m.status = 'aktiv'
			JOIN family_links fl ON fl.member_id = m.id
			WHERE v.id = ?1
			UNION
			-- Trainer des Teams
			SELECT m.user_id AS u
			FROM videos v
			JOIN trainer_memberships tm ON tm.team_id = v.team_id AND tm.season_id = v.season_id
			JOIN members m ON m.id = tm.member_id
			WHERE v.id = ?1 AND m.user_id IS NOT NULL
		) WHERE u IS NOT NULL`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var uids []int
	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		uids = append(uids, uid)
	}
	return uids, rows.Err()
}

// realFFmpegTranscode ist die Produktions-Naht (4.4/4.5): erzeugt für jede
// Rendition serielle ein HLS-Set via `nice -n 19 ffmpeg` und schreibt danach die
// master.m3u8. CRF 26, preset medium, H.264; Audio `-c:a copy` wenn Quelle AAC,
// sonst `-c:a aac -b:a 128k`.
func realFFmpegTranscode(ctx context.Context, rawPath, processedDir string) error {
	if err := os.MkdirAll(processedDir, 0o755); err != nil {
		return err
	}
	aacSource, err := sourceIsAAC(ctx, rawPath)
	if err != nil {
		return fmt.Errorf("ffprobe audio codec: %w", err)
	}
	for _, rd := range workerRenditions {
		dir := filepath.Join(processedDir, rd.name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := runFFmpegRendition(ctx, rawPath, dir, rd, aacSource); err != nil {
			return fmt.Errorf("ffmpeg %s: %w", rd.name, err)
		}
	}
	return writeMasterManifest(processedDir)
}

// runFFmpegRendition führt einen einzelnen ffmpeg-Lauf für eine Rendition aus.
// Segmente: seg_%03d.ts, Manifest: index.m3u8 (MUSS zur Streaming-Schicht passen).
func runFFmpegRendition(ctx context.Context, rawPath, dir string, rd rendition, aacSource bool) error {
	args := []string{
		"-n", "19", "ffmpeg",
		"-y",
		"-i", rawPath,
		"-vf", fmt.Sprintf("scale=-2:%d", rd.height),
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "26",
		"-maxrate", rd.maxrate,
		"-bufsize", rd.bufsize,
	}
	if aacSource {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac", "-b:a", "128k")
	}
	args = append(args,
		"-f", "hls",
		"-hls_time", "10",
		"-hls_list_size", "0",
		"-hls_segment_filename", filepath.Join(dir, "seg_%03d.ts"),
		filepath.Join(dir, "index.m3u8"),
	)
	cmd := exec.CommandContext(ctx, "nice", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, lastLines(string(out), 5))
	}
	return nil
}

// sourceIsAAC probet den Audio-Codec der Quelle. Liefert true, wenn der erste
// Audio-Stream AAC ist (dann `-c:a copy`).
func sourceIsAAC(ctx context.Context, rawPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		rawPath)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(string(out)), "aac"), nil
}

// writeMasterManifest schreibt die master.m3u8 mit beiden Renditions (4.5). Die
// Rendition-Referenzen MÜSSEN exakt `{rendition}/index.m3u8` lauten (stream.go
// renditionLinePrefix `^[0-9]{3,4}p/index\.m3u8$`), sonst hängt ServeMaster den
// ?st=-Token nicht an → Playback-404.
func writeMasterManifest(processedDir string) error {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")
	for _, rd := range workerRenditions {
		fmt.Fprintf(&b, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n",
			rd.bandwidth, rd.width, rd.height)
		fmt.Fprintf(&b, "%s/index.m3u8\n", rd.name)
	}
	return os.WriteFile(filepath.Join(processedDir, "master.m3u8"), []byte(b.String()), 0o644)
}

// lastLines liefert die letzten n Zeilen von s (für kompakte ffmpeg-Fehlermeldungen).
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}
