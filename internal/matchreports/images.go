package matchreports

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// maxImageBytes ist das Upload-Limit pro Bild — 8 MB matcht typische
// Handy-JPGs nach Client-Downscale.
const maxImageBytes int64 = 8 << 20

// imagesDir liefert das Verzeichnis, in dem Bilder eines Berichts liegen.
func (h *Handler) imagesDir(reportID int) string {
	return filepath.Join(h.cfg.MatchReportImageDir, fmt.Sprintf("%d", reportID))
}

// UploadImage nimmt ein Bild (multipart) am Draft entgegen.
//
//	POST /api/match-reports/{id}/images
//	multipart: file (JPG/PNG), caption (Text, optional)
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reportID, ok := parsePathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_id")
		return
	}

	var authorID int
	var state string
	err := h.db.QueryRow(
		`SELECT author_user_id, state FROM match_reports WHERE id=?`, reportID,
	).Scan(&authorID, &state)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		logErr("matchreports.UploadImage select", err, "id", reportID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	if code, status := guardMutation(claims, authorID, state); code != "" {
		writeErr(w, status, code)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+1<<20)
	if err := r.ParseMultipartForm(maxImageBytes + 1<<20); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_multipart", err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "missing_file")
		return
	}
	defer file.Close()

	if header.Size > maxImageBytes {
		writeErr(w, http.StatusBadRequest, "image_too_large")
		return
	}

	ext, ok := allowedImageExt(header.Filename, header.Header.Get("Content-Type"))
	if !ok {
		writeErr(w, http.StatusBadRequest, "unsupported_mime")
		return
	}

	// Positions-Limit prüfen (max 10).
	var count int
	if err := h.db.QueryRow(
		`SELECT COUNT(*) FROM match_report_images WHERE report_id=?`, reportID,
	).Scan(&count); err != nil {
		logErr("matchreports.UploadImage count", err, "id", reportID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if count >= MaxImages {
		writeErr(w, http.StatusBadRequest, "too_many_images")
		return
	}

	// Datei ins Bericht-Verzeichnis schreiben.
	dir := h.imagesDir(reportID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		logErr("matchreports.UploadImage mkdir", err, "dir", dir)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	position := count + 1
	filename := fmt.Sprintf("%d.%s", position, ext)
	storagePath := filepath.Join(dir, filename)

	dst, err := os.Create(storagePath)
	if err != nil {
		logErr("matchreports.UploadImage create", err, "path", storagePath)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		os.Remove(storagePath)
		logErr("matchreports.UploadImage copy", err, "path", storagePath)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	dst.Close()

	caption := r.FormValue("caption")
	res, err := h.db.Exec(
		`INSERT INTO match_report_images (report_id, position, caption, storage_path)
		 VALUES (?, ?, ?, ?)`,
		reportID, position, caption, storagePath,
	)
	if err != nil {
		os.Remove(storagePath)
		logErr("matchreports.UploadImage insert", err, "id", reportID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	id, _ := res.LastInsertId()

	h.broadcast()
	writeJSON(w, http.StatusCreated, Image{
		ID:       int(id),
		Position: position,
		Caption:  caption,
		URL:      fmt.Sprintf("/api/match-reports/%d/images/%d/blob", reportID, id),
	})
}

// DeleteImage entfernt ein Bild vom Draft (nicht nach Publish).
//
//	DELETE /api/match-reports/{id}/images/{imgId}
func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reportID, ok := parsePathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_id")
		return
	}
	imageID, ok := parsePathID(r, "imgId")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_image_id")
		return
	}

	var authorID int
	var state, storagePath string
	err := h.db.QueryRow(
		`SELECT r.author_user_id, r.state, i.storage_path
		 FROM match_reports r
		 JOIN match_report_images i ON i.report_id = r.id
		 WHERE r.id=? AND i.id=?`,
		reportID, imageID,
	).Scan(&authorID, &state, &storagePath)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		logErr("matchreports.DeleteImage select", err, "id", imageID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	if code, status := guardMutation(claims, authorID, state); code != "" {
		writeErr(w, status, code)
		return
	}

	if _, err := h.db.Exec(`DELETE FROM match_report_images WHERE id=?`, imageID); err != nil {
		logErr("matchreports.DeleteImage db", err, "id", imageID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	// Datei-Löschen absichtlich nach DB-Löschen — falls die Datei schon weg
	// ist (Halbfehler), bleibt keine Zombie-Zeile hängen.
	if err := os.Remove(storagePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Nur loggen, DB ist konsistent.
		logErr("matchreports.DeleteImage remove file", err, "path", storagePath)
	}

	h.broadcast()
	w.WriteHeader(http.StatusNoContent)
}

// ServeImage streamt eine Bild-Datei (nur für den Autor oder Admin, im
// Draft/publish_failed-State — nach `published` sind die Bilder ohnehin weg).
//
//	GET /api/match-reports/{id}/images/{imgId}/blob
func (h *Handler) ServeImage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reportID, ok := parsePathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_id")
		return
	}
	imageID, ok := parsePathID(r, "imgId")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_image_id")
		return
	}

	var authorID int
	var storagePath string
	err := h.db.QueryRow(
		`SELECT r.author_user_id, i.storage_path
		 FROM match_reports r
		 JOIN match_report_images i ON i.report_id = r.id
		 WHERE r.id=? AND i.id=?`,
		reportID, imageID,
	).Scan(&authorID, &storagePath)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		logErr("matchreports.ServeImage select", err, "id", imageID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	// Autor darf sein eigenes Bild sehen. Freigeber (medien/vorstand/admin)
	// dürfen jedes Bild sehen — sie müssen den Bericht ja prüfen.
	if authorID != claims.UserID && !isReviewer(claims) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}

	http.ServeFile(w, r, storagePath)
}

// removeAllImageFiles löscht das gesamte Bilder-Verzeichnis eines Berichts.
// Wird vom Delete-Handler (Draft) und Publish-Cleanup aufgerufen.
func (h *Handler) removeAllImageFiles(reportID int) {
	dir := h.imagesDir(reportID)
	if err := os.RemoveAll(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
		logErr("matchreports.removeAllImageFiles", err, "dir", dir)
	}
}

// listImages liefert alle Bilder eines Berichts, sortiert nach position.
func (h *Handler) listImages(reportID int) ([]Image, error) {
	rows, err := h.db.Query(
		`SELECT id, position, caption FROM match_report_images
		 WHERE report_id=? ORDER BY position ASC`, reportID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Image
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.Position, &img.Caption); err != nil {
			return nil, err
		}
		img.URL = fmt.Sprintf("/api/match-reports/%d/images/%d/blob", reportID, img.ID)
		out = append(out, img)
	}
	return out, rows.Err()
}

// allowedImageExt validiert den Upload gegen erlaubte Mime/Extension-Kombis
// und liefert die kanonische Extension zurück.
func allowedImageExt(filename, contentType string) (string, bool) {
	ct := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch ct {
	case "image/jpeg", "image/jpg":
		return "jpg", true
	case "image/png":
		return "png", true
	}
	// Manche Clients senden keinen ordentlichen Content-Type — Fallback auf Extension.
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "jpg", true
	case ".png":
		return "png", true
	}
	return "", false
}
