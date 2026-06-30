package videos

import (
	"context"
	"database/sql"
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

// maxUploadSize ist das 2,5-GB-Hard-Limit pro Datei (tusd Config.MaxSize).
// 5 << 29 == 2.5 GiB == 2_684_354_560 Bytes.
const maxUploadSize int64 = 5 << 29 // 2.5 GiB

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
		http.Error(w, "file too large (max 2.5 GB)", http.StatusBadRequest)
		return
	}

	// Referenzielle Validierung VOR der Berechtigungsprüfung und dem INSERT:
	// team_id/season_id/game_id müssen existieren. Ohne diese Prüfung würde der
	// FK-Constraint erst beim INSERT zuschlagen und einen nichtssagenden 500
	// (Server-Fehler-Leak) erzeugen; bei admin/vorstand/sportliche_leitung
	// liefert CanUploadToTeam zudem für *jede* — auch nicht existente — Team-ID
	// `true`, sodass die Autorisierung sonst gegen ein Phantom-Team entschiede.
	if exists, err := h.rowExists("SELECT 1 FROM teams WHERE id = ?", req.TeamID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if !exists {
		http.Error(w, "unknown team_id", http.StatusBadRequest)
		return
	}
	if exists, err := h.rowExists("SELECT 1 FROM seasons WHERE id = ?", req.SeasonID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if !exists {
		http.Error(w, "unknown season_id", http.StatusBadRequest)
		return
	}
	if req.GameID != nil {
		if *req.GameID <= 0 {
			http.Error(w, "invalid game_id", http.StatusBadRequest)
			return
		}
		if exists, err := h.rowExists("SELECT 1 FROM games WHERE id = ?", *req.GameID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		} else if !exists {
			http.Error(w, "unknown game_id", http.StatusBadRequest)
			return
		}
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
		BasePath:      "/api/videos/upload/",
		StoreComposer: composer,
		MaxSize:       maxUploadSize,
		// PreUploadCreateCallback bindet die tus-Session an den authentifizierten
		// Eigentümer der vorab angelegten videos-Zeile und macht den Disk-Guard
		// gegen die *deklarierte* Upload-Länge autoritativ (Findings 1a + 2). Ein
		// non-nil error lässt tusd die Session-Erstellung ablehnen (siehe
		// unrouted_handler.PostFile → sendError).
		PreUploadCreateCallback: h.preUploadCreate,
		NotifyCompleteUploads:   true,
		// Hinter nginx (TLS-Terminierung → http://127.0.0.1:8080). Ohne diese
		// Option würde tusd Location-Header mit http://… bauen (r.TLS == nil),
		// die tus-js-client in localStorage speichert; ein Resume-HEAD ginge
		// dann an http://… → 301 → Fehler. Nginx setzt X-Forwarded-Proto und
		// Host bereits (deploy/nginx-intern.conf).
		RespectForwardedHeaders: true,
	})
	if err != nil {
		return nil, err
	}

	go h.consumeCompletedUploads(ctx, th.CompleteUploads)

	return th, nil
}

// preUploadCreate ist der tusd PreUploadCreateCallback. Er läuft bei der
// Erzeugung der tus-Session (POST an /api/videos/upload/) — nachdem die
// auth.Middleware + RequireClubFunction den Request passieren ließen, sodass
// hook.Context die Claims trägt (tusd kopiert den Request-Kontext werterhaltend,
// siehe handler/context.go). Er erzwingt zwei Invarianten und lehnt die Session
// mit einem Fehler ab, falls eine verletzt ist (tusd erstellt dann nichts):
//
//   - Finding 1a (IDOR-Bindung): Die in der tus-Metadata mitgegebene video_id
//     MUSS auf eine videos-Zeile mit status='uploading' zeigen, die der
//     authentifizierte Aufrufer selbst angelegt hat (created_by). So kann eine
//     Session nicht an eine fremde oder bereits fertige Zeile gebunden werden.
//   - Finding 2 (Disk-Guard autoritativ): Der Platz-Check läuft gegen die von
//     tusd erzwungene, *deklarierte* Upload-Länge (hook.Upload.Size), nicht
//     gegen das vom Client frei wählbare size_bytes der POST-Init.
//
// FileInfoChanges bleibt leer (keine Metadaten-/ID-Überschreibung).
func (h *Handler) preUploadCreate(hook tusd.HookEvent) (tusd.HTTPResponse, tusd.FileInfoChanges, error) {
	var noChanges tusd.FileInfoChanges

	claims := auth.ClaimsFromCtx(hook.Context)
	if claims == nil {
		return tusd.HTTPResponse{}, noChanges, tusd.NewError(
			"ERR_UPLOAD_UNAUTHENTICATED", "upload session requires authentication", http.StatusUnauthorized)
	}

	idStr := hook.Upload.MetaData["video_id"]
	videoID, err := strconv.Atoi(idStr)
	if err != nil || videoID <= 0 {
		return tusd.HTTPResponse{}, noChanges, tusd.NewError(
			"ERR_UPLOAD_BAD_VIDEO_ID", "missing or invalid video_id metadata", http.StatusBadRequest)
	}

	// Eigentums- und Status-Bindung: nur eine eigene, noch im Upload befindliche
	// Zeile darf bespielt werden. Liefert KEINE Zeile bei fremdem Besitzer,
	// falscher ID oder bereits abgeschlossenem (queued/processing/ready/failed)
	// Upload — in allen Fällen wird die Session verweigert.
	var rowSize sql.NullInt64
	err = h.db.QueryRowContext(hook.Context,
		`SELECT size_bytes FROM videos WHERE id = ? AND status = 'uploading' AND created_by = ?`,
		videoID, claims.UserID).Scan(&rowSize)
	if errors.Is(err, sql.ErrNoRows) {
		return tusd.HTTPResponse{}, noChanges, tusd.NewError(
			"ERR_UPLOAD_NOT_OWNED", "no owned uploading video for this id", http.StatusForbidden)
	}
	if err != nil {
		return tusd.HTTPResponse{}, noChanges, tusd.NewError(
			"ERR_UPLOAD_LOOKUP", "could not verify upload ownership", http.StatusInternalServerError)
	}

	// Deklarierte Upload-Länge (das, was tusd hart durchsetzt). Bei deferred
	// length ist Size beim Create 0 — wir verlangen eine vorab deklarierte Länge,
	// damit der Disk-Guard greifen kann.
	declared := hook.Upload.Size
	if hook.Upload.SizeIsDeferred || declared <= 0 {
		return tusd.HTTPResponse{}, noChanges, tusd.NewError(
			"ERR_UPLOAD_LENGTH_REQUIRED", "upload length must be declared", http.StatusBadRequest)
	}
	if declared > maxUploadSize {
		return tusd.HTTPResponse{}, noChanges, tusd.NewError(
			"ERR_UPLOAD_TOO_LARGE", "declared upload length exceeds maximum", http.StatusRequestEntityTooLarge)
	}
	// Plausibilität: die deklarierte Länge darf die bei der POST-Init notierte
	// Größe nicht grob übersteigen (kleiner Toleranzfaktor für Container-Overhead).
	if rowSize.Valid && rowSize.Int64 > 0 && declared > rowSize.Int64*2 {
		return tusd.HTTPResponse{}, noChanges, tusd.NewError(
			"ERR_UPLOAD_SIZE_MISMATCH", "declared upload length far exceeds announced size", http.StatusBadRequest)
	}

	// Disk-Guard (autoritativ): free ≥ declared × 2.5 + RESERVED. declared ist
	// durch maxUploadSize (2.5 GiB) begrenzt, also überläuft die float64-Konversion
	// nicht (2.5 GiB × 2.5 ≪ math.MaxUint64).
	needed := uint64(float64(declared) * 2.5)
	if err := RequireFreeBytes(h.cfg.VideoStorageDir, needed, h.cfg.VideoReservedBytes); err != nil {
		if errors.Is(err, ErrInsufficientDiskSpace) {
			return tusd.HTTPResponse{}, noChanges, tusd.NewError(
				"ERR_UPLOAD_INSUFFICIENT_STORAGE", "insufficient storage for declared upload", http.StatusInsufficientStorage)
		}
		return tusd.HTTPResponse{}, noChanges, tusd.NewError(
			"ERR_UPLOAD_DISK_CHECK", "could not verify free space", http.StatusInternalServerError)
	}

	return tusd.HTTPResponse{}, noChanges, nil
}

// rowExists meldet, ob die gegebene 1-Spalten-Existenz-Query eine Zeile liefert.
func (h *Handler) rowExists(query string, args ...any) (bool, error) {
	var one int
	err := h.db.QueryRow(query, args...).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
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
//
// Die Statustransition ist konditional und atomar (Finding 1b, Defense in
// Depth): das UPDATE greift nur, solange die Zeile noch status='uploading' hat.
// Trifft es null Zeilen — die Zeile existiert nicht (mehr), gehört einem anderen
// Upload oder ist bereits queued/processing/ready/failed — wird das als
// Hijack-/Race-Versuch gewertet: die soeben nach raw/{id}.mp4 verschobene Datei
// wird wieder entfernt (kein hängender Klau-Artefakt) und ein Fehler
// zurückgegeben. So bleibt das Ziel auch dann unangetastet, wenn der
// PreUploadCreateCallback umgangen würde.
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
		// raw-Datei nicht zurücklassen (sonst Müll bzw. Überschreibung eines
		// fremden raw/{id}.mp4, falls videoID gekapert wurde).
		_ = os.Remove(rawPath)
		return err
	}

	res, err := h.db.Exec(
		`UPDATE videos SET status='queued', size_bytes=?, duration_sec=?, upload_id=?, failure_reason=NULL
		 WHERE id=? AND status='uploading'`,
		size, duration, uploadID, videoID)
	if err != nil {
		_ = os.Remove(rawPath)
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		_ = os.Remove(rawPath)
		return err
	}
	if affected != 1 {
		// Kein passender 'uploading'-Datensatz: abgelehnt. Die verschobene Datei
		// entfernen, damit kein Video gekapert/überschrieben bleibt.
		_ = os.Remove(rawPath)
		return errors.New("finishUpload: no uploading video row for id (rejected)")
	}

	h.hub.Broadcast("video-queued")
	return nil
}

// markVideoFailed setzt ein Video auf status='failed' mit Begründung (best
// effort). Die Bedingung status='uploading' ist sicherheitskritisch: schlägt
// finishUpload für einen Hijack-Versuch (fremde/abgeschlossene Zeile) fehl,
// dürfen wir das Opfer NICHT auf 'failed' umschreiben. Nur die noch im Upload
// befindliche eigene Zeile wird angefasst.
func (h *Handler) markVideoFailed(videoID int, reason string) {
	if _, err := h.db.Exec(
		`UPDATE videos SET status='failed', failure_reason=? WHERE id=? AND status='uploading'`,
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
