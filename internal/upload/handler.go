package upload

import (
	"context"
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

// broadcastMembers sends the "members" live-update event to the finance group
// (vorstand/vorstand_beisitzer/kassierer + admin) and — when memberIDs are given
// — additionally to each affected member's audience (its teams' players/trainers/
// parents/staff plus the member's own linked user), so a photo change surfaces
// live on rosters and profiles that subscribe to "members" but are not finance.
// Replaces the former global Broadcast("members"); topic string and Frontend
// contract unchanged. extraUserIDs are always included (e.g. the acting user).
func (h *Handler) broadcastMembers(ctx context.Context, memberIDs []int, extraUserIDs ...int) {
	if h.hub == nil {
		return
	}
	a := hub.NewAudience(h.db)
	seen := make(map[int]struct{})
	var ids []int
	add := func(id int) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, id := range a.FinanceGroup(ctx, extraUserIDs...) {
		add(id)
	}
	if len(memberIDs) > 0 {
		for _, id := range a.MembersAudience(ctx, memberIDs) {
			add(id)
		}
	}
	h.hub.BroadcastToUsers(ids, "members")
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
		return "", fmt.Errorf("too_large")
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("missing file field")
	}
	defer file.Close()
	return h.persistMultipartFile(file, hdr, subdir, allowedTypes, maxBytes)
}

// persistMultipartFile validates and persists a single multipart part (im Klartext;
// Zero-Knowledge: Bank-/SEPA-PII wird ausschließlich clientseitig verschlüsselt — siehe
// UploadSepaMandat/saveEncryptedBlob). Size enforcement happens here: a part larger than
// maxBytes returns "too_large".
func (h *Handler) persistMultipartFile(file multipart.File, hdr *multipart.FileHeader, subdir string, allowedTypes []string, maxBytes int64) (string, error) {
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

	if _, err := io.Copy(dst, file); err != nil {
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

	h.broadcastMembers(r.Context(), memberIDs(id))
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

	h.broadcastMembers(r.Context(), memberIDs(memberID), claims.UserID)
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

	h.broadcastMembers(r.Context(), h.memberIDsForUser(r.Context(), claims.UserID), claims.UserID)
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
	h.broadcastMembers(r.Context(), h.memberIDsForUser(r.Context(), claims.UserID), claims.UserID)
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
	h.broadcastMembers(r.Context(), memberIDs(memberID), claims.UserID)
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
	h.broadcastMembers(r.Context(), memberIDs(id))
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/upload/sepa-mandat/{id} — Admin only
//
// Zero-Knowledge (Modell B): Das PDF kommt bereits clientseitig verschlüsselt an
// (Datei-Feld = Ciphertext-Blob mit "TWENC1"-Magic), der gewrappte DEK im Feld `dek_enc`.
// Der Server speichert beides unverändert und entschlüsselt nie.
func (h *Handler) UploadSepaMandat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	memberID, _ := strconv.Atoi(id)

	filename, err := h.saveEncryptedBlob(r, "sepa-mandats", maxSepaBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dekEnc := r.FormValue("dek_enc")
	if dekEnc == "" {
		os.Remove(filepath.Join(h.uploadDir, filename))
		http.Error(w, "dek_enc fehlt (clientseitig verschlüsseln)", http.StatusBadRequest)
		return
	}

	var oldPath sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT sepa_mandat_path FROM members WHERE id=?`, id).Scan(&oldPath)
	if oldPath.Valid && oldPath.String != "" {
		os.Remove(filepath.Join(h.uploadDir, oldPath.String))
	}

	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE members SET sepa_mandat_path=?, sepa_mandat_dek_enc=? WHERE id=?`, filename, dekEnc, id); err != nil {
		os.Remove(filepath.Join(h.uploadDir, filename))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.broadcastMembers(r.Context(), []int{memberID})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"sepa_mandat_url": "/api/uploads/" + filename})
}

// saveEncryptedBlob speichert einen bereits clientseitig verschlüsselten Datei-Blob roh
// (keine Typ-Prüfung — Ciphertext ist kein PDF; keine Server-Verschlüsselung). Verlangt
// den Client-Magic-Header, damit kein Klartext-PDF durchrutscht.
func (h *Handler) saveEncryptedBlob(r *http.Request, subdir string, maxBytes int64) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes+4096)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		return "", fmt.Errorf("too_large")
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("missing file field")
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("cannot read file")
	}
	if !crypto.IsClientEncryptedBytes(raw) {
		return "", fmt.Errorf("kein clientseitig verschlüsselter Blob")
	}
	dir := filepath.Join(h.uploadDir, subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create directory")
	}
	filename := uuid.NewString() + ".bin"
	if err := os.WriteFile(filepath.Join(dir, filename), raw, 0644); err != nil {
		return "", fmt.Errorf("cannot write file")
	}
	return filepath.Join(subdir, filename), nil
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

	var path, dekEnc sql.NullString
	h.db.QueryRowContext(r.Context(),
		`SELECT sepa_mandat_path, sepa_mandat_dek_enc FROM members WHERE id=?`, memberID).Scan(&path, &dekEnc)
	if !path.Valid || path.String == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// dek_enc mitliefern, damit der Client den heruntergeladenen Ciphertext-Blob mit dem
	// Tresor-Schlüssel entschlüsseln kann (Server entschlüsselt das PDF nicht).
	token := generateSepaToken(memberID, claims.UserID, h.secret)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token, "dek_enc": dekEnc.String})
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

	h.streamFile(w, r, filepath.Join(h.uploadDir, path.String), filepath.Base(path.String))
}

// streamFile liefert eine Datei direkt aus (Range-Support, kein Voll-Read in den RAM).
// Zero-Knowledge: SEPA-Mandat-Blobs sind clientseitig verschlüsselt; der Server streamt nur
// den Ciphertext, der Browser entschlüsselt mit dem gewrappten DEK (kein Server-Decrypt).
func (h *Handler) streamFile(w http.ResponseWriter, r *http.Request, full, name string) {
	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()
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

// memberIDs converts a string path-value member ID into a single-element slice
// for broadcastMembers. A non-numeric ID yields nil (finance-only broadcast).
func memberIDs(id string) []int {
	n, err := strconv.Atoi(id)
	if err != nil {
		return nil
	}
	return []int{n}
}

// memberIDsForUser returns the member IDs linked to a user (a user photo may
// surface via that user's own member row on rosters/profiles). Empty when the
// user has no linked member — the broadcast then reaches the finance group and
// the acting user (passed as extraUserID).
func (h *Handler) memberIDsForUser(ctx context.Context, userID int) []int {
	rows, err := h.db.QueryContext(ctx, `SELECT id FROM members WHERE user_id=?`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
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

// GET /api/uploads/* — erfordert Authentifizierung. Die Route ist unter
// auth.CookieMiddleware gemountet (Router), weil <img>-Requests kein Bearer-Token
// senden können — analog zu den SSE-Routen. Damit ist die Auslieferung nicht mehr
// unauthentifiziert erreichbar. Zusätzlich verhindern no-referrer/no-store, dass
// die (UUID-)URL über Referrer oder Caches weiterleakt.
func (h *Handler) ServeUpload(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	if strings.Contains(rawPath, "..") || rawPath == "" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "private, no-store")
	h.streamFile(w, r, filepath.Join(h.uploadDir, rawPath), filepath.Base(rawPath))
}
