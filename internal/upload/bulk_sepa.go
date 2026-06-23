package upload

import (
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type bulkImportEntry struct {
	Filename   string  `json:"filename"`
	MemberID   *int    `json:"member_id,omitempty"`
	MemberName *string `json:"member_name,omitempty"`
	Reason     string  `json:"reason,omitempty"`
}

type bulkImportCandidate struct {
	MemberID   int    `json:"member_id"`
	MemberName string `json:"member_name"`
}

type bulkAmbiguousEntry struct {
	Filename   string                `json:"filename"`
	Candidates []bulkImportCandidate `json:"candidates"`
}

type bulkImportReport struct {
	Imported      []bulkImportEntry    `json:"imported"`
	AlreadyExists []bulkImportEntry    `json:"already_exists"`
	NoMatch       []bulkImportEntry    `json:"no_match"`
	Ambiguous     []bulkAmbiguousEntry `json:"ambiguous"`
}

// BulkImportSepaMandate accepts a multipart request with one or more PDF parts
// (form field "files") and matches each filename against members. Files matching
// exactly one member without an existing sepa_mandat_path are stored; the
// member's sepa_mandat_path is set and sepa_mandat is flipped to 1. Existing
// mandates are never overwritten; matchless and ambiguous files are reported.
func (h *Handler) BulkImportSepaMandate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBulkSepaBytes+1024)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "request_too_large",
			"limit": "500 MB pro Import — bitte in mehrere Tranchen aufteilen.",
		})
		return
	}

	report := bulkImportReport{
		Imported:      []bulkImportEntry{},
		AlreadyExists: []bulkImportEntry{},
		NoMatch:       []bulkImportEntry{},
		Ambiguous:     []bulkAmbiguousEntry{},
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
		return
	}

	anyImported := false
	for _, hdr := range files {
		entry := h.processBulkFile(r, hdr, &report)
		if entry {
			anyImported = true
		}
	}

	if anyImported && h.hub != nil {
		h.hub.Broadcast("members")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// processBulkFile returns true if the file was successfully imported.
func (h *Handler) processBulkFile(r *http.Request, hdr *multipart.FileHeader, report *bulkImportReport) bool {
	basename := strings.TrimSuffix(hdr.Filename, filepath.Ext(hdr.Filename))

	// Pre-validate: PDF extension required (cheap check before matching).
	if !strings.EqualFold(filepath.Ext(hdr.Filename), ".pdf") {
		report.NoMatch = append(report.NoMatch, bulkImportEntry{
			Filename: hdr.Filename, Reason: "kein PDF",
		})
		return false
	}

	if hdr.Size > maxBulkSepaFileBytes {
		report.NoMatch = append(report.NoMatch, bulkImportEntry{
			Filename: hdr.Filename, Reason: "zu groß (>10 MB)",
		})
		return false
	}

	matches, err := matchMemberByFilename(r.Context(), h.db, basename)
	if err != nil {
		report.NoMatch = append(report.NoMatch, bulkImportEntry{
			Filename: hdr.Filename, Reason: "lookup-fehler",
		})
		return false
	}

	switch len(matches) {
	case 0:
		report.NoMatch = append(report.NoMatch, bulkImportEntry{Filename: hdr.Filename})
		return false
	case 1:
		return h.tryStoreForMember(r, hdr, matches[0], report)
	default:
		candidates := make([]bulkImportCandidate, 0, len(matches))
		for _, id := range matches {
			name := h.lookupMemberName(r, id)
			candidates = append(candidates, bulkImportCandidate{MemberID: id, MemberName: name})
		}
		report.Ambiguous = append(report.Ambiguous, bulkAmbiguousEntry{
			Filename: hdr.Filename, Candidates: candidates,
		})
		return false
	}
}

func (h *Handler) tryStoreForMember(r *http.Request, hdr *multipart.FileHeader, memberID int, report *bulkImportReport) bool {
	var existing sql.NullString
	var first, last sql.NullString
	err := h.db.QueryRowContext(r.Context(),
		`SELECT sepa_mandat_path, first_name, last_name FROM members WHERE id=?`,
		memberID).Scan(&existing, &first, &last)
	if err != nil {
		report.NoMatch = append(report.NoMatch, bulkImportEntry{
			Filename: hdr.Filename, Reason: "member-lookup-fehler",
		})
		return false
	}
	memberName := strings.TrimSpace(first.String + " " + last.String)

	if existing.Valid && existing.String != "" {
		mid := memberID
		mn := memberName
		report.AlreadyExists = append(report.AlreadyExists, bulkImportEntry{
			Filename: hdr.Filename, MemberID: &mid, MemberName: &mn,
		})
		return false
	}

	file, err := hdr.Open()
	if err != nil {
		report.NoMatch = append(report.NoMatch, bulkImportEntry{
			Filename: hdr.Filename, Reason: "öffnen-fehler",
		})
		return false
	}
	defer file.Close()

	stored, err := h.persistMultipartFile(file, hdr, "sepa-mandats", pdfOnlyTypes, maxBulkSepaFileBytes)
	if err != nil {
		reason := err.Error()
		switch reason {
		case "too_large":
			reason = "zu groß (>10 MB)"
		case "unsupported_type":
			reason = "kein PDF"
		}
		report.NoMatch = append(report.NoMatch, bulkImportEntry{
			Filename: hdr.Filename, Reason: reason,
		})
		return false
	}

	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE members SET sepa_mandat_path=?, sepa_mandat=1 WHERE id=?`,
		stored, memberID); err != nil {
		os.Remove(filepath.Join(h.uploadDir, stored))
		report.NoMatch = append(report.NoMatch, bulkImportEntry{
			Filename: hdr.Filename, Reason: "db-fehler",
		})
		return false
	}

	mid := memberID
	mn := memberName
	report.Imported = append(report.Imported, bulkImportEntry{
		Filename: hdr.Filename, MemberID: &mid, MemberName: &mn,
	})
	return true
}

func (h *Handler) lookupMemberName(r *http.Request, memberID int) string {
	var first, last sql.NullString
	h.db.QueryRowContext(r.Context(),
		`SELECT first_name, last_name FROM members WHERE id=?`, memberID).
		Scan(&first, &last)
	return strings.TrimSpace(first.String + " " + last.String)
}
