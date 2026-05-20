package upload

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

var imageTypes = []string{"image/jpeg", "image/png", "image/webp"}
var pdfAndImageTypes = []string{"application/pdf", "image/jpeg", "image/png", "image/webp"}

const (
	maxPhotoBytes = 5 << 20  // 5 MB
	maxSepaBytes  = 10 << 20 // 10 MB
)

type Handler struct {
	db        *sql.DB
	uploadDir string
}

func NewHandler(db *sql.DB, uploadDir string) *Handler {
	return &Handler{db: db, uploadDir: uploadDir}
}

// saveFile reads a multipart upload, validates type/size, writes to uploadDir/subdir, returns filename.
func (h *Handler) saveFile(r *http.Request, subdir string, allowedTypes []string, maxBytes int64) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes+1024)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		return "", fmt.Errorf("file too large or invalid form")
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("missing file field")
	}
	defer file.Close()

	contentType := hdr.Header.Get("Content-Type")
	allowed := false
	for _, t := range allowedTypes {
		if t == contentType {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("unsupported file type: %s", contentType)
	}

	ext := filepath.Ext(hdr.Filename)
	filename := uuid.NewString() + ext

	dir := filepath.Join(h.uploadDir, subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create directory")
	}

	dst, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return "", fmt.Errorf("cannot create file")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("cannot write file")
	}

	return subdir + "/" + filename, nil
}

func (h *Handler) photoURL(path string) string {
	if path == "" {
		return ""
	}
	return "/api/uploads/" + path
}

// POST /api/upload/member-photo/{id} — Admin only
func (h *Handler) UploadMemberPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	filename, err := h.saveFile(r, "member-photos", imageTypes, maxPhotoBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var oldPath sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT photo_path FROM members WHERE id=?`, id).Scan(&oldPath)
	if oldPath.Valid && oldPath.String != "" {
		os.Remove(filepath.Join(h.uploadDir, oldPath.String))
	}

	if _, err := h.db.ExecContext(r.Context(), `UPDATE members SET photo_path=? WHERE id=?`, filename, id); err != nil {
		os.Remove(filepath.Join(h.uploadDir, filename))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"photo_url": h.photoURL(filename)})
}

// POST /api/upload/user-photo — own user
func (h *Handler) UploadUserPhoto(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	filename, err := h.saveFile(r, "user-photos", imageTypes, maxPhotoBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var oldPath sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT photo_path FROM users WHERE id=?`, claims.UserID).Scan(&oldPath)
	if oldPath.Valid && oldPath.String != "" {
		os.Remove(filepath.Join(h.uploadDir, oldPath.String))
	}

	if _, err := h.db.ExecContext(r.Context(), `UPDATE users SET photo_path=? WHERE id=?`, filename, claims.UserID); err != nil {
		os.Remove(filepath.Join(h.uploadDir, filename))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"photo_url": h.photoURL(filename)})
}

// POST /api/upload/sepa-mandat/{id} — Admin only
func (h *Handler) UploadSepaMandat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	filename, err := h.saveFile(r, "sepa-mandats", pdfAndImageTypes, maxSepaBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var oldPath sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT sepa_mandat_path FROM members WHERE id=?`, id).Scan(&oldPath)
	if oldPath.Valid && oldPath.String != "" {
		os.Remove(filepath.Join(h.uploadDir, oldPath.String))
	}

	if _, err := h.db.ExecContext(r.Context(), `UPDATE members SET sepa_mandat_path=? WHERE id=?`, filename, id); err != nil {
		os.Remove(filepath.Join(h.uploadDir, filename))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"sepa_mandat_url": "/api/uploads/" + filename})
}

// GET /api/uploads/* — Auth required
func (h *Handler) ServeUpload(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	if strings.Contains(rawPath, "..") || rawPath == "" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, filepath.Join(h.uploadDir, rawPath))
}
