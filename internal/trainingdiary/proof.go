package trainingdiary

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"database/sql"

	"github.com/google/uuid"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// maxProofBytes ist die harte Obergrenze je Nachweis. Das Frontend drückt
// Bilder vorher clientseitig auf ~150 KB (imageCompress.ts mit targetBytes
// 150 KB / maxEdge 1280) — dieses Limit ist der Backstop, nicht das Ziel.
const maxProofBytes = 1 << 20 // 1 MB

// multipartHeadroom deckt den Overhead der multipart-Kodierung ab, damit eine
// Datei von exakt 1 MB nicht schon am Body-Limit scheitert.
const multipartHeadroom = 64 << 10 // 64 KB

// extByMime ist die Whitelist erlaubter, per Content-Sniffing erkannter Typen.
// Bewusst ohne HEIC/HEIF: Chrome und Firefox können HEVC-Stills nicht
// dekodieren, die clientseitige Wandlung schlägt dort fehl und die Datei käme
// unkomprimiert an. Ein klares HTTP 400 ist ehrlicher als ein stiller
// Riesen-Blob — in der Praxis wandelt iOS beim Datei-Picker ohnehin nach JPEG.
var extByMime = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

// UploadProof — POST /api/training-diary/{id}/proof (multipart, Feld "proof")
// Auch lange nach dem Anlegen des Eintrags erlaubt; ersetzt einen bestehenden
// Nachweis. Nur der Eigentümer.
func (h *Handler) UploadProof(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	owner, oldDisk, _, err := h.entryOwner(r.Context(), id)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !h.isOwner(r.Context(), claims, owner) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxProofBytes+multipartHeadroom)
	if err := r.ParseMultipartForm(maxProofBytes + multipartHeadroom); err != nil {
		http.Error(w, "Datei zu groß oder ungültiges Formular", http.StatusRequestEntityTooLarge)
		return
	}
	file, _, err := r.FormFile("proof")
	if err != nil {
		http.Error(w, "Feld 'proof' fehlt", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	if len(data) > maxProofBytes {
		http.Error(w, "Datei zu groß", http.StatusRequestEntityTooLarge)
		return
	}

	// Typ aus dem Inhalt, nicht aus Endung oder Client-Header.
	mimeType := http.DetectContentType(data)
	ext, allowed := extByMime[mimeType]
	if !allowed {
		http.Error(w, "Format nicht unterstützt", http.StatusBadRequest)
		return
	}

	// Verzeichnis vor jedem Schreiben sicherstellen: NewHandler toleriert einen
	// Fehlschlag beim Start (siehe dort), damit ein falsch konfigurierter Pfad
	// nicht die ganze App abschießt. Ein nachträglich korrigierter Pfad greift
	// damit ohne Neustart — und ein von außen gelöschtes Verzeichnis heilt.
	if err := os.MkdirAll(h.dir, 0755); err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	diskName := uuid.New().String() + ext
	dst := filepath.Join(h.dir, diskName)
	if err := os.WriteFile(dst, data, 0644); err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	// proof_purged_at wird zurückgesetzt: der Eintrag hat wieder einen
	// echten Nachweis, der Retention-Marker wäre sonst irreführend.
	_, err = h.db.ExecContext(r.Context(), `
		UPDATE training_diary_entries
		   SET proof_disk_name = ?, proof_mime = ?, proof_size = ?,
		       proof_uploaded_at = CURRENT_TIMESTAMP, proof_purged_at = NULL,
		       updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		diskName, mimeType, len(data), id)
	if err != nil {
		// Kein Waisen-Blob zurücklassen (Muster media.Upload).
		os.Remove(dst)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	// Erst nach erfolgreichem Schreiben die Vorgängerdatei entfernen.
	if oldDisk != "" && oldDisk != diskName {
		h.removeProofFile(oldDisk)
	}

	e, err := h.loadEntry(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast(diaryEvent)
	writeJSON(w, http.StatusCreated, e)
}

// DeleteProof — DELETE /api/training-diary/{id}/proof
// Entfernt den Nachweis, lässt den Eintrag im Übrigen unverändert. Nur der
// Eigentümer. proof_purged_at bleibt ungesetzt — das ist der Retention-Marker,
// eine manuelle Löschung ist etwas anderes.
func (h *Handler) DeleteProof(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	owner, diskName, _, err := h.entryOwner(r.Context(), id)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !h.isOwner(r.Context(), claims, owner) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_, err = h.db.ExecContext(r.Context(), `
		UPDATE training_diary_entries
		   SET proof_disk_name = NULL, proof_mime = NULL, proof_size = NULL,
		       proof_uploaded_at = NULL, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	h.removeProofFile(diskName)

	h.hub.Broadcast(diaryEvent)
	w.WriteHeader(http.StatusNoContent)
}

// ServeProof — GET /api/training-diary/{id}/proof
// Liefert die Bytes an alle, die den Eintrag lesen dürfen. 410 grenzt
// „von der Retention gelöscht" von 404 („hatte nie einen Nachweis") ab; das
// Frontend zeigt darauf einen Hinweis statt eines defekten Bildes.
func (h *Handler) ServeProof(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var memberID int
	var diskName, mimeType, purgedAt sql.NullString
	err := h.db.QueryRowContext(r.Context(),
		`SELECT member_id, proof_disk_name, proof_mime, proof_purged_at
		   FROM training_diary_entries WHERE id = ?`, id).
		Scan(&memberID, &diskName, &mimeType, &purgedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	allowed, err := h.canReadMemberDiary(r.Context(), claims, memberID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if !diskName.Valid || diskName.String == "" {
		if purgedAt.Valid && purgedAt.String != "" {
			http.Error(w, "Nachweis wurde nach Saisonende gelöscht", http.StatusGone)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	f, err := os.Open(filepath.Join(h.dir, diskName.String))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", mimeType.String)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, diskName.String, time.Time{}, f)
}
