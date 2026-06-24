package upload

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/crypto"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

var imageTypes = []string{"image/jpeg", "image/jpg", "image/png", "image/webp"}
var pdfAndImageTypes = []string{"application/pdf", "image/jpeg", "image/png", "image/webp"}
var pdfOnlyTypes = []string{"application/pdf"}

const (
	maxPhotoBytes        = 5 << 20   // 5 MB
	maxSepaBytes         = 2 << 20   // 2 MB (Einzel-Upload via Detail-Tab)
	maxBulkSepaFileBytes = 10 << 20  // 10 MB pro PDF im Bulk-Import (gescannte Mandate sind oft >2 MB)
	maxBulkSepaBytes     = 500 << 20 // 500 MB Multipart-Body-Cap (~250 Mandate à 2 MB oder ~50 à 10 MB)
)

type Handler struct {
	db        *sql.DB
	uploadDir string
	secret    string
	hub       *hub.EventHub
}

func NewHandler(db *sql.DB, uploadDir, secret string, h *hub.EventHub) *Handler {
	return &Handler{db: db, uploadDir: uploadDir, secret: secret, hub: h}
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
func (h *Handler) saveFile(r *http.Request, subdir string, allowedTypes []string, maxBytes int64, encrypt bool) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes+1024)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		return "", fmt.Errorf("too_large")
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("missing file field")
	}
	defer file.Close()
	return h.persistMultipartFile(file, hdr, subdir, allowedTypes, maxBytes, encrypt)
}

// persistMultipartFile validates and persists a single multipart part. It is
// shared between the single-file upload paths and BulkImportSepaMandate.
// Size enforcement happens here: a part larger than maxBytes returns "too_large".
// Mit encrypt=true wird der Dateiinhalt at-rest verschlüsselt abgelegt
// (Magic-Header) — verwendet für SEPA-Mandat-PDFs.
func (h *Handler) persistMultipartFile(file multipart.File, hdr *multipart.FileHeader, subdir string, allowedTypes []string, maxBytes int64, encrypt bool) (string, error) {
	if hdr.Size > maxBytes {
		return "", fmt.Errorf("too_large")
	}

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
			return "", fmt.Errorf("unsupported_type")
		}
	}

	ext := filepath.Ext(hdr.Filename)
	filename := uuid.NewString() + ext

	dir := filepath.Join(h.uploadDir, subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create directory")
	}

	fullPath := filepath.Join(dir, filename)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("cannot create file")
	}
	defer dst.Close()

	if encrypt {
		raw, err := io.ReadAll(file)
		if err != nil {
			os.Remove(fullPath)
			return "", fmt.Errorf("cannot read file")
		}
		enc, err := crypto.EncryptBytes(raw)
		if err != nil {
			os.Remove(fullPath)
			return "", fmt.Errorf("cannot encrypt file")
		}
		if _, err := dst.Write(enc); err != nil {
			os.Remove(fullPath)
			return "", fmt.Errorf("cannot write file")
		}
	} else if _, err := io.Copy(dst, file); err != nil {
		os.Remove(fullPath)
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

	filename, err := h.saveFile(r, "member-photos", imageTypes, maxPhotoBytes, false)
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

	filename, err := h.saveFile(r, "member-photos", imageTypes, maxPhotoBytes, false)
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

	filename, err := h.saveFile(r, "user-photos", imageTypes, maxPhotoBytes, false)
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

	filename, err := h.saveFile(r, "sepa-mandats", pdfAndImageTypes, maxSepaBytes, true)
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

	if h.hub != nil {
		h.hub.Broadcast("members")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"sepa_mandat_url": "/api/uploads/" + filename})
}

// GET /api/members/{id}/sepa-mandat/download-token — authenticated
func (h *Handler) SepaDownloadToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	isAdmin := claims.Role == "admin"
	isVorstand := claims.HasFunction("vorstand")
	isOwn := h.memberUserID(r, memberID) == claims.UserID
	isParent := h.isParentOf(r, claims.UserID, memberID)

	if !isAdmin && !isVorstand && !isOwn && !isParent {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var path sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT sepa_mandat_path FROM members WHERE id=?`, memberID).Scan(&path)
	if !path.Valid || path.String == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	token := generateSepaToken(memberID, claims.UserID, h.secret)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// GET /api/members/{id}/sepa-mandat/download?token=... — public (token-auth internally)
func (h *Handler) SepaDownload(w http.ResponseWriter, r *http.Request) {
	memberID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	token := r.URL.Query().Get("token")
	if _, err := validateSepaToken(token, memberID, h.secret); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var path sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT sepa_mandat_path FROM members WHERE id=?`, memberID).Scan(&path)
	if !path.Valid || path.String == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if strings.Contains(path.String, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	h.serveFileMaybeDecrypt(w, r, filepath.Join(h.uploadDir, path.String), filepath.Base(path.String))
}

// serveFileMaybeDecrypt liefert eine Datei aus. Trägt sie den Verschlüsselungs-
// Magic-Header, wird der Inhalt entschlüsselt ausgeliefert; sonst wird die Datei
// direkt gestreamt (Range-Support, kein Voll-Read in den RAM).
func (h *Handler) serveFileMaybeDecrypt(w http.ResponseWriter, r *http.Request, full, name string) {
	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	if crypto.IsEncryptedBytes(head[:n]) {
		rest, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		plain, err := crypto.DecryptBytes(append(head[:n], rest...))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(plain))
		return
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, name, time.Time{}, f)
}

// DELETE /api/members/{id}/sepa-mandat — authenticated
func (h *Handler) DeleteSepaMandat(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	isAdmin := claims.Role == "admin"
	isVorstand := claims.HasFunction("vorstand")
	isOwn := h.memberUserID(r, memberID) == claims.UserID
	isParent := h.isParentOf(r, claims.UserID, memberID)

	if !isAdmin && !isVorstand && !isOwn && !isParent {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var path sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT sepa_mandat_path FROM members WHERE id=?`, memberID).Scan(&path)
	if path.Valid && path.String != "" {
		os.Remove(filepath.Join(h.uploadDir, path.String))
	}

	h.db.ExecContext(r.Context(),
		`UPDATE members SET sepa_mandat_path=NULL, sepa_mandat=0, sepa_mandat_date=NULL WHERE id=?`, memberID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) memberUserID(r *http.Request, memberID int) int {
	var userID sql.NullInt64
	h.db.QueryRowContext(r.Context(), `SELECT user_id FROM members WHERE id=?`, memberID).Scan(&userID)
	if userID.Valid {
		return int(userID.Int64)
	}
	return -1
}

func (h *Handler) isParentOf(r *http.Request, parentUserID, memberID int) bool {
	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?`,
		parentUserID, memberID).Scan(&count)
	return count > 0
}

// GET /api/uploads/* — Auth required
func (h *Handler) ServeUpload(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	if strings.Contains(rawPath, "..") || rawPath == "" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	h.serveFileMaybeDecrypt(w, r, filepath.Join(h.uploadDir, rawPath), filepath.Base(rawPath))
}
