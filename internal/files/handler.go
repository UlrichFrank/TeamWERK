package files

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/policy"
)

type Handler struct {
	db       *sql.DB
	filesDir string
	secret   string
}

func NewHandler(db *sql.DB, filesDir, secret string) *Handler {
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		panic(fmt.Sprintf("files: cannot create storage dir %s: %v", filesDir, err))
	}
	return &Handler{db: db, filesDir: filesDir, secret: secret}
}

// folderPath returns [folderID, parentID, grandparentID, ...] up to the root.
func folderPath(db *sql.DB, folderID int) ([]int, error) {
	path := []int{}
	current := folderID
	for {
		path = append(path, current)
		var parentID sql.NullInt64
		err := db.QueryRow(`SELECT parent_id FROM file_folders WHERE id = ?`, current).Scan(&parentID)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("folder %d not found", current)
		}
		if err != nil {
			return nil, err
		}
		if !parentID.Valid {
			break
		}
		current = int(parentID.Int64)
	}
	return path, nil
}

// fetchFamilyContext returns the user IDs and club functions of members linked to
// userID via family_links. Used to grant Elternteil users their children's access.
func fetchFamilyContext(db *sql.DB, userID int) (linkedUserIDs []int, linkedFunctions []string) {
	rows, err := db.Query(`
		SELECT COALESCE(m.user_id, 0), COALESCE(mcf.function, '')
		  FROM family_links fl
		  JOIN members m ON m.id = fl.member_id
		  LEFT JOIN member_club_functions mcf ON mcf.member_id = m.id
		 WHERE fl.parent_user_id = ?`, userID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	for rows.Next() {
		var uid int
		var fn string
		if err := rows.Scan(&uid, &fn); err != nil {
			continue
		}
		if uid != 0 && !slices.Contains(linkedUserIDs, uid) {
			linkedUserIDs = append(linkedUserIDs, uid)
		}
		if fn != "" && !slices.Contains(linkedFunctions, fn) {
			linkedFunctions = append(linkedFunctions, fn)
		}
	}
	return linkedUserIDs, linkedFunctions
}

// resolveAccess returns the effective read/write access for the caller on folderID.
// Nearest-ancestor-wins: the closest folder in the hierarchy with explicit permissions
// is authoritative; ancestors beyond that point are ignored.
// Elternteil users also inherit the club_function and user-ID rights of their children.
func resolveAccess(db *sql.DB, claims *auth.Claims, folderID int) (canRead, canWrite bool, err error) {
	if claims.Role == "admin" {
		return true, true, nil
	}

	path, err := folderPath(db, folderID)
	if err != nil {
		return false, false, err
	}

	linkedUserIDs, linkedFunctions := fetchFamilyContext(db, claims.UserID)
	userIDStr := strconv.Itoa(claims.UserID)

	for _, id := range path {
		rows, err := db.Query(
			`SELECT principal_type, principal_ref, can_read, can_write
			   FROM folder_permissions WHERE folder_id = ?`, id)
		if err != nil {
			return false, false, err
		}

		var hasAny bool
		var cr, cw bool

		for rows.Next() {
			hasAny = true
			var pt, pr sql.NullString
			var r, w int
			if scanErr := rows.Scan(&pt, &pr, &r, &w); scanErr != nil {
				continue
			}
			matches := false
			switch pt.String {
			case "everyone":
				matches = true
			case "role":
				matches = pr.Valid && pr.String == claims.Role
			case "club_function":
				matches = pr.Valid && (claims.HasFunction(pr.String) || slices.Contains(linkedFunctions, pr.String))
			case "user":
				if pr.Valid && pr.String == userIDStr {
					matches = true
				} else if pr.Valid {
					if uid, parseErr := strconv.Atoi(pr.String); parseErr == nil {
						matches = slices.Contains(linkedUserIDs, uid)
					}
				}
			}
			if matches {
				if r == 1 {
					cr = true
				}
				if w == 1 {
					cw = true
				}
			}
		}
		rows.Close()

		if hasAny {
			return cr, cw, nil
		}
	}

	return false, false, nil
}

// checkAntiEscalation returns true if the caller is allowed to grant the requested rights.
// Admin may always grant anything. Others may only grant rights they themselves hold.
func checkAntiEscalation(db *sql.DB, claims *auth.Claims, folderID int, newRead, newWrite bool) (bool, error) {
	if claims.Role == "admin" {
		return true, nil
	}
	_, callerWrite, err := resolveAccess(db, claims, folderID)
	if err != nil {
		return false, err
	}
	// Caller needs can_write to manage permissions at all.
	// Additionally, can only grant write if they have write themselves.
	if !callerWrite {
		return false, nil
	}
	if newWrite && !callerWrite {
		return false, nil
	}
	return true, nil
}

// ─── Folder API ──────────────────────────────────────────────────────────────

type folderResponse struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	ParentID      *int   `json:"parent_id"`
	HasChildren   bool   `json:"has_children"`
	CanRead       bool   `json:"can_read"`
	CanWrite      bool   `json:"can_write"`
	CreatedAt     string `json:"created_at"`
	CreatedByName string `json:"created_by_name"`
}

// GET /api/folders — root folders visible to the caller
func (h *Handler) ListRootFolders(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT f.id, f.name, f.created_at, u.first_name || ' ' || u.last_name
		   FROM file_folders f JOIN users u ON u.id = f.created_by
		  WHERE f.parent_id IS NULL ORDER BY f.name`)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []folderResponse{}
	for rows.Next() {
		var f folderResponse
		if err := rows.Scan(&f.ID, &f.Name, &f.CreatedAt, &f.CreatedByName); err != nil {
			continue
		}
		cr, cw, _ := resolveAccess(h.db, claims, f.ID)
		if !cr {
			continue
		}
		f.CanRead, f.CanWrite = cr, cw
		var childCount int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM file_folders WHERE parent_id = ?`, f.ID).Scan(&childCount)
		f.HasChildren = childCount > 0
		result = append(result, f)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/folders
func (h *Handler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		Name     string `json:"name"`
		ParentID *int   `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.ParentID != nil {
		_, cw, err := resolveAccess(h.db, claims, *req.ParentID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !cw {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	} else if claims.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var parentVal any
	if req.ParentID != nil {
		parentVal = *req.ParentID
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO file_folders (name, parent_id, created_by) VALUES (?, ?, ?)`,
		strings.TrimSpace(req.Name), parentVal, claims.UserID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

type contentsResponse struct {
	Folders  []folderResponse `json:"folders"`
	Files    []fileResponse   `json:"files"`
	CanRead  bool             `json:"can_read"`
	CanWrite bool             `json:"can_write"`
}

// GET /api/folders/{id}/contents
func (h *Handler) FolderContents(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	p := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions, IsParent: claims.IsParent}
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	cr, cw, err := policy.FolderAccess(h.db, p, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !cr {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Sub-folders
	frows, err := h.db.QueryContext(r.Context(),
		`SELECT f.id, f.name, f.created_at, u.first_name || ' ' || u.last_name
		   FROM file_folders f JOIN users u ON u.id = f.created_by
		  WHERE f.parent_id = ? ORDER BY f.name`, id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer frows.Close()

	folders := []folderResponse{}
	for frows.Next() {
		var f folderResponse
		f.ParentID = &id
		if err := frows.Scan(&f.ID, &f.Name, &f.CreatedAt, &f.CreatedByName); err != nil {
			continue
		}
		fcr, fcw, _ := resolveAccess(h.db, claims, f.ID)
		if !fcr {
			continue
		}
		f.CanRead, f.CanWrite = fcr, fcw
		var cc int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM file_folders WHERE parent_id = ?`, f.ID).Scan(&cc)
		f.HasChildren = cc > 0
		folders = append(folders, f)
	}

	// Files
	fls, err := h.listFiles(r, id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contentsResponse{
		Folders:  folders,
		Files:    fls,
		CanRead:  cr,
		CanWrite: cw,
	})
}

// PUT /api/folders/{id}
func (h *Handler) RenameFolder(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	_, cw, err := resolveAccess(h.db, claims, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !cw {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE file_folders SET name = ? WHERE id = ?`, strings.TrimSpace(req.Name), id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/folders/{id}
func (h *Handler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	_, cw, err := resolveAccess(h.db, claims, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !cw {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Only delete if empty
	var childCount, fileCount int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM file_folders WHERE parent_id = ?`, id).Scan(&childCount)
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM files WHERE folder_id = ?`, id).Scan(&fileCount)
	if childCount > 0 || fileCount > 0 {
		http.Error(w, "folder not empty", http.StatusConflict)
		return
	}

	h.db.ExecContext(r.Context(), `DELETE FROM file_folders WHERE id = ?`, id)
	w.WriteHeader(http.StatusNoContent)
}

// ─── Permission API ──────────────────────────────────────────────────────────

type permResponse struct {
	ID            int    `json:"id"`
	PrincipalType string `json:"principal_type"`
	PrincipalRef  string `json:"principal_ref"`
	DisplayName   string `json:"display_name,omitempty"`
	CanRead       bool   `json:"can_read"`
	CanWrite      bool   `json:"can_write"`
}

// GET /api/folders/{id}/permissions
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	folderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	_, cw, err := resolveAccess(h.db, claims, folderID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !cw {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, principal_type, COALESCE(principal_ref,''), can_read, can_write
		   FROM folder_permissions WHERE folder_id = ?`, folderID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []permResponse{}
	for rows.Next() {
		var p permResponse
		var cr, cw int
		if err := rows.Scan(&p.ID, &p.PrincipalType, &p.PrincipalRef, &cr, &cw); err != nil {
			continue
		}
		p.CanRead = cr == 1
		p.CanWrite = cw == 1
		if p.PrincipalType == "user" && p.PrincipalRef != "" {
			var name string
			err := h.db.QueryRowContext(r.Context(),
				`SELECT first_name || ' ' || last_name FROM users WHERE id = ?`, p.PrincipalRef).Scan(&name)
			if err == nil {
				p.DisplayName = name
			} else {
				p.DisplayName = p.PrincipalRef
			}
		}
		result = append(result, p)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/folders/{id}/permissions
func (h *Handler) AddPermission(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	folderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		PrincipalType string `json:"principal_type"`
		PrincipalRef  string `json:"principal_ref"`
		CanRead       bool   `json:"can_read"`
		CanWrite      bool   `json:"can_write"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	validTypes := map[string]bool{"everyone": true, "role": true, "club_function": true, "user": true}
	if !validTypes[req.PrincipalType] {
		http.Error(w, "invalid principal_type", http.StatusBadRequest)
		return
	}

	ok, err := checkAntiEscalation(h.db, claims, folderID, req.CanRead, req.CanWrite)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var ref any
	if req.PrincipalType != "everyone" && req.PrincipalRef != "" {
		ref = req.PrincipalRef
	}
	cr, cw := 0, 0
	if req.CanRead {
		cr = 1
	}
	if req.CanWrite {
		cw = 1
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO folder_permissions (folder_id, principal_type, principal_ref, can_read, can_write)
		 VALUES (?, ?, ?, ?, ?)`,
		folderID, req.PrincipalType, ref, cr, cw)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// DELETE /api/folders/{id}/permissions/{permId}
func (h *Handler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	folderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	permID, err := strconv.Atoi(chi.URLParam(r, "permId"))
	if err != nil {
		http.Error(w, "invalid permId", http.StatusBadRequest)
		return
	}

	_, cw, err := resolveAccess(h.db, claims, folderID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !cw {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	h.db.ExecContext(r.Context(),
		`DELETE FROM folder_permissions WHERE id = ? AND folder_id = ?`, permID, folderID)
	w.WriteHeader(http.StatusNoContent)
}

// ─── File API ────────────────────────────────────────────────────────────────

type fileResponse struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	MimeType       string `json:"mime_type"`
	UploadedByName string `json:"uploaded_by_name"`
	CreatedAt      string `json:"created_at"`
}

func (h *Handler) listFiles(r *http.Request, folderID int) ([]fileResponse, error) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT f.id, f.original_name, f.size, f.mime_type, f.created_at,
		        COALESCE(u.first_name||' '||u.last_name, u.email)
		   FROM files f
		   JOIN users u ON u.id = f.uploaded_by
		  WHERE f.folder_id = ?
		  ORDER BY f.created_at DESC`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []fileResponse{}
	for rows.Next() {
		var f fileResponse
		if err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.MimeType, &f.CreatedAt, &f.UploadedByName); err != nil {
			continue
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

// POST /api/folders/{folderId}/files
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	folderID, err := strconv.Atoi(chi.URLParam(r, "folderId"))
	if err != nil {
		http.Error(w, "invalid folderId", http.StatusBadRequest)
		return
	}

	_, cw, err := resolveAccess(h.db, claims, folderID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !cw {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	const maxSize = 50 << 20 // 50 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "file too large or invalid form", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	diskName := uuid.New().String() + ext
	dst := filepath.Join(h.filesDir, diskName)

	out, err := os.Create(dst)
	if err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		os.Remove(dst)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension(ext)
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO files (folder_id, original_name, disk_name, size, mime_type, uploaded_by)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		folderID, header.Filename, diskName, written, mimeType, claims.UserID)
	if err != nil {
		os.Remove(dst)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "name": header.Filename, "size": written})
}

// GET /api/files/{id}/download-token
func (h *Handler) HandleDownloadToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var folderID int
	err = h.db.QueryRowContext(r.Context(), `SELECT folder_id FROM files WHERE id = ?`, id).Scan(&folderID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	cr, _, err := resolveAccess(h.db, claims, folderID)
	if err != nil || !cr {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	token := generateDownloadToken(id, claims.UserID, h.secret)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// GET /api/files/{id}/download
// Accepts either a Bearer JWT (via auth middleware context) or a ?token= query param.
func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var folderID int
	var originalName, diskName, mimeType string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT folder_id, original_name, disk_name, mime_type FROM files WHERE id = ?`, id).
		Scan(&folderID, &originalName, &diskName, &mimeType)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	inline := false
	tokenParam := r.URL.Query().Get("token")
	if tokenParam != "" {
		userID, err := validateDownloadToken(tokenParam, id, h.secret)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := h.claimsForUser(r, userID)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		cr, _, err := resolveAccess(h.db, claims, folderID)
		if err != nil || !cr {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		inline = true
	} else {
		claims := auth.ClaimsFromCtx(r.Context())
		if claims == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		cr, _, err := resolveAccess(h.db, claims, folderID)
		if err != nil || !cr {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	path := filepath.Join(h.filesDir, diskName)
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", disposition+`; filename="`+sanitizeFilename(originalName)+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, originalName, time.Time{}, f)
}

// claimsForUser builds auth.Claims from the DB for token-based download auth.
func (h *Handler) claimsForUser(r *http.Request, userID int) (*auth.Claims, error) {
	var role string
	if err := h.db.QueryRowContext(r.Context(), `SELECT role FROM users WHERE id = ?`, userID).Scan(&role); err != nil {
		return nil, err
	}
	var functionsStr string
	h.db.QueryRowContext(r.Context(),
		`SELECT COALESCE(GROUP_CONCAT(mcf.function, ','), '')
		   FROM member_club_functions mcf
		   JOIN members m ON m.id = mcf.member_id
		  WHERE m.user_id = ?`, userID).Scan(&functionsStr)
	var clubFunctions []string
	if functionsStr != "" {
		clubFunctions = strings.Split(functionsStr, ",")
	}
	return &auth.Claims{UserID: userID, Role: role, ClubFunctions: clubFunctions}, nil
}

// PUT /api/files/{id}
func (h *Handler) RenameFile(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	var folderID int
	err = h.db.QueryRowContext(r.Context(), `SELECT folder_id FROM files WHERE id = ?`, id).Scan(&folderID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_, cw, err := resolveAccess(h.db, claims, folderID)
	if err != nil || !cw {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE files SET original_name = ? WHERE id = ?`, strings.TrimSpace(req.Name), id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/files/{id}
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var folderID int
	var diskName string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT folder_id, disk_name FROM files WHERE id = ?`, id).
		Scan(&folderID, &diskName)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	_, cw, err := resolveAccess(h.db, claims, folderID)
	if err != nil || !cw {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	h.db.ExecContext(r.Context(), `DELETE FROM files WHERE id = ?`, id)
	os.Remove(filepath.Join(h.filesDir, diskName))
	w.WriteHeader(http.StatusNoContent)
}

func sanitizeFilename(name string) string {
	return strings.NewReplacer(`"`, `'`, `\`, `-`).Replace(name)
}
