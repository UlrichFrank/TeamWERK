// Package media ist ein gemeinsamer, JWT-geschützter Bild-Store für Chat-
// Nachrichten und Mitteilungen. Bilder liegen als <uuid>.<ext> unter mediaDir;
// die media-Tabelle hält Metadaten (disk_name, mime_type, size). Upload begrenzt
// auf 1 MB und eine MIME-Whitelist; das Frontend verkleinert vorher clientseitig.
package media

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// maxImageBytes ist die harte Obergrenze pro Bild (Backstop; das Frontend
// verkleinert bereits auf ≤ 1 MB).
const maxImageBytes = 1 << 20 // 1 MB

// multipartHeadroom deckt den Overhead der multipart-Kodierung ab, damit ein
// Bild von exakt 1 MB nicht schon am Body-Limit scheitert.
const multipartHeadroom = 64 << 10 // 64 KB

// extByMime bildet erlaubte (per Content-Sniffing erkannte) MIME-Types auf die
// gespeicherte Dateiendung ab.
var extByMime = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

type Handler struct {
	db       *sql.DB
	mediaDir string
}

func NewHandler(db *sql.DB, mediaDir string) *Handler {
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		panic(fmt.Sprintf("media: cannot create storage dir %s: %v", mediaDir, err))
	}
	return &Handler{db: db, mediaDir: mediaDir}
}

// Upload nimmt ein Bild (multipart, Feld "image") entgegen, prüft MIME + Größe,
// speichert es als <uuid>.<ext> und legt eine media-Zeile an.
// POST /api/media/upload
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserIDFromCtx(r.Context())
	if uid == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+multipartHeadroom)
	if err := r.ParseMultipartForm(maxImageBytes + multipartHeadroom); err != nil {
		http.Error(w, "file too large or invalid form", http.StatusRequestEntityTooLarge)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "missing image field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	if len(data) > maxImageBytes {
		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		return
	}

	mimeType := http.DetectContentType(data)
	ext, allowed := extByMime[mimeType]
	if !allowed {
		http.Error(w, "unsupported image type", http.StatusBadRequest)
		return
	}

	// Header-Probe für Bild-Dimensionen (Aspect-Ratio ab dem ersten Frame
	// im Frontend, kein Layout-Shift beim späteren Bild-Load). Scheitert die
	// Probe, wird der Upload trotzdem akzeptiert; Dims bleiben NULL und der
	// AuthImage-Client-Probe greift als Fallback.
	width, height, dimsOK := decodeDimensions(data, mimeType)

	diskName := uuid.New().String() + ext
	dst := filepath.Join(h.mediaDir, diskName)
	if err := os.WriteFile(dst, data, 0644); err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	var widthArg, heightArg any
	if dimsOK {
		widthArg, heightArg = width, height
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO media (disk_name, mime_type, size, uploaded_by, width, height) VALUES (?, ?, ?, ?, ?, ?)`,
		diskName, mimeType, len(data), uid, widthArg, heightArg)
	if err != nil {
		os.Remove(dst)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	resp := map[string]any{
		"mediaId": id,
		"url":     fmt.Sprintf("/media/%d", id),
	}
	if dimsOK {
		resp["width"] = width
		resp["height"] = height
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// canSee beantwortet die Sichtbarkeit eines Bildes über das referenzierende
// Objekt: Hochladender, Mitglied der Konversation, in der eine Nachricht das
// Bild trägt, oder Empfänger einer Mitteilung mit diesem Bild.
//
// Die Konversations-Bedingung fragt bewusst OHNE `left_at IS NULL` — analog
// chat.isMember: wer die Gruppe verlassen hat, behält den Verlauf und damit
// auch dessen Bilder. Ein Rollen-Bypass (admin/vorstand) fehlt bewusst: der
// Store liegt hinter privaten Unterhaltungen, eine Vereinsfunktion verschafft
// dort keinen Zugang.
func (h *Handler) canSee(ctx context.Context, mediaID, userID int) (bool, error) {
	var ok bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM media WHERE id = ? AND uploaded_by = ?)
		    OR EXISTS(SELECT 1 FROM messages m
		              JOIN conversation_members cm ON cm.conversation_id = m.conversation_id
		              WHERE m.media_id = ? AND cm.user_id = ?)
		    OR EXISTS(SELECT 1 FROM broadcasts b
		              JOIN broadcast_reads br ON br.broadcast_id = b.id
		              WHERE b.media_id = ? AND br.user_id = ?)`,
		mediaID, userID, mediaID, userID, mediaID, userID).Scan(&ok)
	return ok, err
}

// Serve liefert die Bild-Bytes anhand der media-ID aus.
// GET /api/media/{id}
func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Objekt-Gate VOR dem Dateizugriff. Fehlende Berechtigung antwortet 404 wie
	// eine unbekannte ID — sonst wäre über den Statuscode erratbar, welche
	// media-IDs existieren.
	visible, err := h.canSee(r.Context(), id, claims.UserID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !visible {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var diskName, mimeType string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT disk_name, mime_type FROM media WHERE id = ?`, id).
		Scan(&diskName, &mimeType)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	f, err := os.Open(filepath.Join(h.mediaDir, diskName))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, diskName, time.Time{}, f)
}
