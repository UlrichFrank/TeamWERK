package videos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// remuxFunc ist die injizierbare Naht des Download-Remux (video-download-duty-
// upload): fügt die Segmente aus listFile (ffmpeg-Concat-Listendatei) per
// Stream-Copy zu einer MP4-Datei unter outPath zusammen. Produktion nutzt
// realFFmpegRemux, Tests eine Fake analog zu Worker.transcode.
type remuxFunc func(ctx context.Context, listFile, outPath string) error

// downloadRemuxTimeout deckelt den ffmpeg-Subprozess. Stream-Copy ist sehr
// günstig (kein Decode/Encode) — selbst ein Vollspiel (bis 4 h) remuxt in
// Sekunden bis wenige Minuten; der großzügige Deckel fängt nur hängende
// Prozesse ab, nicht den Normalfall.
const downloadRemuxTimeout = 5 * time.Minute

// downloadFilenameRe erlaubt im Download-Dateinamen nur Buchstaben (inkl.
// Unicode), Ziffern, Leerzeichen, Punkt, Bindestrich, Unterstrich — alles
// andere wird ersetzt. Vermeidet Header-Injection/Escaping-Fallstricke in
// Content-Disposition, ohne RFC-5987-Kodierung für Nicht-ASCII zu brauchen.
var downloadFilenameRe = regexp.MustCompile(`[^\p{L}\p{N} ._-]+`)

// sanitizeDownloadFilename leitet aus dem Video-Titel einen sicheren
// Dateinamen ab. Ein nach der Bereinigung leerer Titel fällt auf "video-{id}"
// zurück.
func sanitizeDownloadFilename(title string, id int) string {
	clean := strings.TrimSpace(downloadFilenameRe.ReplaceAllString(title, "-"))
	if clean == "" {
		return fmt.Sprintf("video-%d.mp4", id)
	}
	return clean + ".mp4"
}

// downloadVideo ist die für den Download-Endpoint relevante Teilmenge einer
// videos-Zeile.
type downloadVideo struct {
	ID     int
	TeamID int
	Status string
	Title  string
}

func (h *Handler) loadVideoForDownload(id int) (*downloadVideo, error) {
	v := &downloadVideo{ID: id}
	err := h.db.QueryRow(`SELECT team_id, status, title FROM videos WHERE id = ?`, id).
		Scan(&v.TeamID, &v.Status, &v.Title)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Download liefert ein fertig verarbeitetes Video als MP4-Datei zum
// Herunterladen. GET /api/videos/{id}/download (Authenticated-Tier).
// Berechtigung: exakt CanViewVideo — dieselbe Zielgruppe wie beim Streaming
// (video-download-duty-upload, keine eigene Sichtbarkeitsregel).
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	video, err := h.loadVideoForDownload(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if video == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ok, err := h.CanViewVideo(claims, &Video{ID: video.ID, TeamID: video.TeamID})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		// Existenz nicht preisgeben — wie GET /api/videos/{id}.
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if video.Status != "ready" {
		http.Error(w, "video not ready", http.StatusConflict)
		return
	}

	renditionDir := RenditionDir(h.cfg.VideoStorageDir, id, "720p")
	segments, err := readManifestSegmentOrder(filepath.Join(renditionDir, "index.m3u8"))
	if err != nil || len(segments) == 0 {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Disk-Guard (analog Upload): Remux-Output ist bei Stream-Copy ungefähr so
	// groß wie die Summe der Segmente.
	estimated, err := DirSize(renditionDir)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := RequireFreeBytes(h.cfg.VideoStorageDir, uint64(estimated), h.cfg.VideoReservedBytes); err != nil {
		if errors.Is(err, ErrInsufficientDiskSpace) {
			http.Error(w, "insufficient storage", http.StatusInsufficientStorage)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tmpDir := DownloadTempDir(h.cfg.VideoStorageDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	listFile, err := writeConcatListFile(tmpDir, renditionDir, segments)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer os.Remove(listFile)

	outFile, err := os.CreateTemp(tmpDir, fmt.Sprintf("download-%d-*.mp4", id))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	outPath := outFile.Name()
	outFile.Close()
	defer os.Remove(outPath)

	// Remux in die Temp-Datei statt direkt in die Response zu streamen: erst
	// nach erfolgreichem Abschluss kennen wir Content-Length und können bei
	// einem ffmpeg-Fehler noch sauber HTTP 500 antworten, statt eine bereits
	// mit 200 begonnene Antwort abzuschneiden (design.md Entscheidung 4).
	ctx, cancel := context.WithTimeout(r.Context(), downloadRemuxTimeout)
	defer cancel()
	if err := h.remux(ctx, listFile, outPath); err != nil {
		http.Error(w, "remux failed", http.StatusInternalServerError)
		return
	}

	f, err := os.Open(outPath)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	filename := sanitizeDownloadFilename(video.Title, id)
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	// Header sind zu diesem Zeitpunkt bereits gesendet (200 + Content-Length) —
	// ein Copy-Fehler (Client bricht ab, Netzfehler) kann nur noch geloggt,
	// nicht mehr als sauberer Statuscode beantwortet werden.
	if _, err := io.Copy(w, f); err != nil {
		slog.Error("video download: response copy failed", "video_id", id, "error", err)
	}
}

// readManifestSegmentOrder liest eine HLS-Rendition-Playlist (index.m3u8) und
// liefert die referenzierten Segment-Dateinamen in Abspielreihenfolge —
// NICHT die Verzeichnis-Sortierung, die ab Segment 1000 (seg_1000.ts,
// vierstellig statt der %03d-Nullpadding-Konvention) lexikographisch falsch
// sortiert. Nur Zeilen, die segmentRe (stream.go) als .ts-Segment matchen,
// zählen — Kommentare/Tags (#EXT…) werden übersprungen.
func readManifestSegmentOrder(manifestPath string) ([]string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var segments []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if segmentRe.MatchString(trimmed) && strings.HasSuffix(trimmed, ".ts") {
			segments = append(segments, trimmed)
		}
	}
	return segments, nil
}

// writeConcatListFile schreibt eine ffmpeg-Concat-Demuxer-Listendatei
// (`file '<absoluter Pfad>'` je Zeile) für segments (Dateinamen relativ zu
// renditionDir) in ein neues Temp-File unter tmpDir und liefert dessen Pfad.
func writeConcatListFile(tmpDir, renditionDir string, segments []string) (string, error) {
	f, err := os.CreateTemp(tmpDir, "concat-*.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()
	for _, seg := range segments {
		full := filepath.Join(renditionDir, seg)
		// ffmpeg-Concat-Syntax: einfache Anführungszeichen im Pfad müssten
		// escaped werden — unsere Segmentnamen kommen aus segmentRe
		// (`seg_[0-9]{1,6}\.ts`) und enthalten nie Sonderzeichen.
		if _, err := fmt.Fprintf(f, "file '%s'\n", full); err != nil {
			return "", err
		}
	}
	return f.Name(), nil
}

// realFFmpegRemux ist die Produktions-Naht: Stream-Copy-Concat der Segmente
// aus listFile in outPath (`nice -n 19 ffmpeg`, analog runFFmpegRendition).
func realFFmpegRemux(ctx context.Context, listFile, outPath string) error {
	cmd := exec.CommandContext(ctx, "nice",
		"-n", "19", "ffmpeg",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-movflags", "+faststart",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, lastLines(string(out), 5))
	}
	return nil
}
