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

var imageTypes = []string{"image/jpeg", "image/jpg", "image/png", "image/webp"}
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

func sniffImageType(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xD8 {
		return "image/jpeg"
	}
	if len(b) >= 4 && b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47 {
		return "image/png"
	}
	if len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP" {
		return "image/webp"
	}
	return ""
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
	isAllowed := func(ct string) bool {
		for _, t := range allowedTypes {
			if t == ct {
				return true
			}
		}
		return false
	}
	if !isAllowed(contentType) {
		// Sniff from first bytes when Content-Type is absent or unrecognized
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		if s, ok := file.(io.Seeker); ok {
			s.Seek(0, io.SeekStart)
		}
		contentType = http.DetectContentType(buf[:n])
		if !isAllowed(contentType) {
			// Try magic bytes as final fallback
			contentType = sniffImageType(buf[:n])
		}
		if !isAllowed(contentType) {
			return "", fmt.Errorf("unsupported file type: %s", hdr.Header.Get("Content-Type"))
		}
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

// POST /api/profile/kind/:memberId/photo — parent auth
func (h *Handler) UploadChildPhoto(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID := r.PathValue("memberId")

	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?`,
		claims.UserID, memberID).Scan(&count)
	if count == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	filename, err := h.saveFile(r, "member-photos", imageTypes, maxPhotoBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var oldPath sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT photo_path FROM members WHERE id=?`, memberID).Scan(&oldPath)
	if oldPath.Valid && oldPath.String != "" {
		os.Remove(filepath.Join(h.uploadDir, oldPath.String))
	}

	if _, err := h.db.ExecContext(r.Context(), `UPDATE members SET photo_path=? WHERE id=?`, filename, memberID); err != nil {
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

// DELETE /api/upload/user-photo — own user
func (h *Handler) DeleteUserPhoto(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var path sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT photo_path FROM users WHERE id=?`, claims.UserID).Scan(&path)
	if path.Valid && path.String != "" {
		os.Remove(filepath.Join(h.uploadDir, path.String))
	}
	h.db.ExecContext(r.Context(), `UPDATE users SET photo_path=NULL WHERE id=?`, claims.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/profile/kind/{memberId}/photo — parent auth
func (h *Handler) DeleteChildPhoto(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID := r.PathValue("memberId")
	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?`,
		claims.UserID, memberID).Scan(&count)
	if count == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var path sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT photo_path FROM members WHERE id=?`, memberID).Scan(&path)
	if path.Valid && path.String != "" {
		os.Remove(filepath.Join(h.uploadDir, path.String))
	}
	h.db.ExecContext(r.Context(), `UPDATE members SET photo_path=NULL WHERE id=?`, memberID)
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/upload/member-photo/{id} — Admin only
func (h *Handler) DeleteMemberPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var path sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT photo_path FROM members WHERE id=?`, id).Scan(&path)
	if path.Valid && path.String != "" {
		os.Remove(filepath.Join(h.uploadDir, path.String))
	}
	h.db.ExecContext(r.Context(), `UPDATE members SET photo_path=NULL WHERE id=?`, id)
	w.WriteHeader(http.StatusNoContent)
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
