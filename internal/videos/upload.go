package videos

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// maxUploadSize ist das 2-GB-Hard-Limit pro Datei (tusd Config.MaxSize).
const maxUploadSize int64 = 2 << 30 // 2 GiB

// uploadsDir liefert das tus-Session-Verzeichnis ({root}/uploads).
func uploadsDir(root string) string {
	return filepath.Join(root, "uploads")
}

// createUploadReq ist der Body von POST /api/videos (Pre-Upload-Init).
type createUploadReq struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	TeamID      int     `json:"team_id"`
	SeasonID    int     `json:"season_id"`
	GameID      *int    `json:"game_id"`
	SizeBytes   int64   `json:"size_bytes"`
}

// CreateUpload validiert die Metadaten, prüft Upload-Berechtigung und Disk-Guard,
// legt die videos-Zeile mit status='uploading' an und gibt video_id + die
// tus-Creation-URL zurück. POST /api/videos (Upload-Tier).
//
// Korrelation Upload↔Video: Der Client erzeugt die tus-Session mit Metadata
// `video_id=<id>`; der Finish-Hook (consumeCompletedUploads) liest diese ID und
// verschiebt die fertige Datei nach raw/{id}.mp4.
func (h *Handler) CreateUpload(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createUploadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		http.Error(w, "title must not be empty", http.StatusBadRequest)
		return
	}
	if req.TeamID <= 0 {
		http.Error(w, "invalid team_id", http.StatusBadRequest)
		return
	}
	if req.SeasonID <= 0 {
		http.Error(w, "invalid season_id", http.StatusBadRequest)
		return
	}
	if req.SizeBytes <= 0 {
		http.Error(w, "invalid size_bytes", http.StatusBadRequest)
		return
	}
	if req.SizeBytes > maxUploadSize {
		http.Error(w, "file too large (max 2 GB)", http.StatusBadRequest)
		return
	}

	ok, err := h.CanUploadToTeam(claims, req.TeamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Disk-Guard (Ebene 1): free ≥ size × 2.5 + RESERVED. Faktor 2.5 deckt
	// raw + processed-Peak (vor raw-Delete) + Sicherheit ab (siehe design.md).
	needed := uint64(float64(req.SizeBytes) * 2.5)
	if err := RequireFreeBytes(h.cfg.VideoStorageDir, needed, h.cfg.VideoReservedBytes); err != nil {
		if errors.Is(err, ErrInsufficientDiskSpace) {
			http.Error(w, "insufficient storage", http.StatusInsufficientStorage)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var desc any
	if req.Description != nil {
		desc = strings.TrimSpace(*req.Description)
	}
	var gameID any
	if req.GameID != nil {
		gameID = *req.GameID
	}

	res, err := h.db.Exec(
		`INSERT INTO videos (title, description, team_id, season_id, game_id, status, size_bytes, created_by)
		 VALUES (?, ?, ?, ?, ?, 'uploading', ?, ?)`,
		req.Title, desc, req.TeamID, req.SeasonID, gameID, req.SizeBytes, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"video_id":   id,
		"upload_url": "/api/videos/upload/",
	})
}

// tusHandler hält den gemounteten tus-Handler. Wird in NewTusHandler erzeugt und
// in router.go unter /api/videos/upload/ gemountet.
type tusHandler struct {
	h    *Handler
	tusd *tusd.Handler
}

// NewTusHandler baut den tusd-Handler über einem FileStore unter {root}/uploads,
// aktiviert die Completion-Notifications und startet die Hook-Goroutine, die
// fertige Uploads verarbeitet. Der Goroutine-Lebenszyklus ist an ctx gebunden.
func (h *Handler) NewTusHandler(ctx context.Context) (http.Handler, error) {
	dir := uploadsDir(h.cfg.VideoStorageDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	store := filestore.New(dir)
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	th, err := tusd.NewHandler(tusd.Config{
		BasePath:              "/api/videos/upload/",
		StoreComposer:         composer,
		MaxSize:               maxUploadSize,
		NotifyCompleteUploads: true,
	})
	if err != nil {
		return nil, err
	}

	go h.consumeCompletedUploads(ctx, th.CompleteUploads)

	return th, nil
}

// consumeCompletedUploads verarbeitet abgeschlossene tus-Uploads, bis ctx endet.
func (h *Handler) consumeCompletedUploads(ctx context.Context, ch <-chan tusd.HookEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			h.handleCompletedUpload(ev)
		}
	}
}

// handleCompletedUpload korreliert ein fertiges tus-Upload mit seiner videos-Zeile
// (über Metadata video_id) und ruft finishUpload. Fehler werden geloggt und als
// status='failed' am Video vermerkt.
func (h *Handler) handleCompletedUpload(ev tusd.HookEvent) {
	idStr := ev.Upload.MetaData["video_id"]
	videoID, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("video upload finished without valid video_id metadata",
			"upload_id", ev.Upload.ID, "video_id", idStr)
		return
	}
	srcPath := ev.Upload.Storage["Path"]
	if err := h.finishUpload(videoID, srcPath, ev.Upload.ID, ev.Upload.Size); err != nil {
		slog.Error("video finishUpload failed", "video_id", videoID, "error", err)
		h.markVideoFailed(videoID, err.Error())
	}
}

// finishUpload verschiebt die fertige Upload-Datei nach raw/{id}.mp4, ermittelt
// per ffprobe die Dauer, setzt status='queued' samt size/duration/upload_id und
// broadcastet "video-queued". Als testbare Einheit extrahiert.
func (h *Handler) finishUpload(videoID int, srcPath, uploadID string, size int64) error {
	root := h.cfg.VideoStorageDir
	rawPath := RawPath(root, videoID)
	if err := os.MkdirAll(filepath.Dir(rawPath), 0o755); err != nil {
		return err
	}
	if err := moveFile(srcPath, rawPath); err != nil {
		return err
	}

	duration, err := probeDurationSec(rawPath)
	if err != nil {
		return err
	}

	if _, err := h.db.Exec(
		`UPDATE videos SET status='queued', size_bytes=?, duration_sec=?, upload_id=?, failure_reason=NULL
		 WHERE id=?`,
		size, duration, uploadID, videoID); err != nil {
		return err
	}

	h.hub.Broadcast("video-queued")
	return nil
}

// markVideoFailed setzt ein Video auf status='failed' mit Begründung (best effort).
func (h *Handler) markVideoFailed(videoID int, reason string) {
	if _, err := h.db.Exec(
		`UPDATE videos SET status='failed', failure_reason=? WHERE id=?`,
		reason, videoID); err != nil {
		slog.Error("markVideoFailed: db update failed", "video_id", videoID, "error", err)
	}
}

// moveFile verschiebt src nach dst, mit Fallback auf Copy+Remove falls os.Rename
// scheitert (z.B. über Dateisystemgrenzen hinweg).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		in.Close()
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		in.Close()
		return err
	}
	if err := out.Close(); err != nil {
		in.Close()
		return err
	}
	in.Close()
	return os.Remove(src)
}

// probeDurationSec ruft ffprobe auf und liefert die (auf ganze Sekunden
// gerundete) Dauer der Datei. Liefert 0 mit Fehler, wenn ffprobe scheitert oder
// keine verwertbare Dauer ausgibt (z.B. Audio-only / kaputte Quelle).
func probeDurationSec(path string) (int, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" || s == "N/A" {
		return 0, errors.New("ffprobe: no duration in source")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if f <= 0 {
		return 0, errors.New("ffprobe: non-positive duration")
	}
	return int(f + 0.5), nil
}

// CleanupStaleUploads entfernt unfertige tus-Sessions im uploads/-Verzeichnis,
// die älter als maxAge sind (bin- und .info-Dateien). Idempotent und safe: ein
// fehlendes uploads/-Verzeichnis ist kein Fehler. Liefert die Anzahl gelöschter
// Dateien.
func CleanupStaleUploads(root string, maxAge time.Duration) (int, error) {
	dir := uploadsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}
