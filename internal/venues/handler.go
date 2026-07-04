package venues

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/httpcache"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

type Venue struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Street      string `json:"street"`
	City        string `json:"city"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"`
	Note        string `json:"note"`
	IsHomeVenue bool   `json:"is_home_venue"`
}

// GET /api/admin/venues
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, name, street, city, postal_code, country, note, is_home_venue
		 FROM venues ORDER BY is_home_venue DESC, name`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "venues List: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []Venue{}
	for rows.Next() {
		var v Venue
		rows.Scan(&v.ID, &v.Name, &v.Street, &v.City, &v.PostalCode, &v.Country, &v.Note, &v.IsHomeVenue)
		result = append(result, v)
	}
	// Referenzdaten: ETag/304-Revalidierung, kein geteilter max-age.
	httpcache.ServeJSON(w, r, "private, no-cache", result)
}

// POST /api/admin/venues
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Street      string `json:"street"`
		City        string `json:"city"`
		PostalCode  string `json:"postal_code"`
		Country     string `json:"country"`
		Note        string `json:"note"`
		IsHomeVenue bool   `json:"is_home_venue"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Street == "" || req.City == "" || req.PostalCode == "" {
		http.Error(w, "name, street, city, postal_code required", http.StatusBadRequest)
		return
	}
	if req.Country == "" {
		req.Country = "DE"
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if req.IsHomeVenue {
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE venues SET is_home_venue=0 WHERE is_home_venue=1`); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	res, err := tx.ExecContext(r.Context(),
		`INSERT INTO venues (name, street, city, postal_code, country, note, is_home_venue)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Street, req.City, req.PostalCode, req.Country, req.Note, req.IsHomeVenue)
	if err != nil {
		fmt.Fprintf(os.Stderr, "venues Create: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	// "venues" ist ein vereinsweites Referenzdaten-Topic (Spielstätten,
	// niederfrequent) → bewusst global (siehe scoped-live-updates).
	h.hub.Broadcast("venues")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// PUT /api/admin/venues/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Street      string `json:"street"`
		City        string `json:"city"`
		PostalCode  string `json:"postal_code"`
		Country     string `json:"country"`
		Note        string `json:"note"`
		IsHomeVenue bool   `json:"is_home_venue"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Street == "" || req.City == "" || req.PostalCode == "" {
		http.Error(w, "name, street, city, postal_code required", http.StatusBadRequest)
		return
	}
	if req.Country == "" {
		req.Country = "DE"
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if req.IsHomeVenue {
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE venues SET is_home_venue=0 WHERE is_home_venue=1 AND id != ?`, id); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	res, err := tx.ExecContext(r.Context(),
		`UPDATE venues SET name=?, street=?, city=?, postal_code=?, country=?, note=?, is_home_venue=? WHERE id=?`,
		req.Name, req.Street, req.City, req.PostalCode, req.Country, req.Note, req.IsHomeVenue, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "venues Update: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("venues")
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/admin/venues/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM venues WHERE id=?`, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "venues Delete: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.hub.Broadcast("venues")
	w.WriteHeader(http.StatusNoContent)
}

type importError struct {
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

type importResult struct {
	Imported int           `json:"imported"`
	Updated  int           `json:"updated"`
	Skipped  int           `json:"skipped"`
	Errors   []importError `json:"errors"`
}

// POST /api/admin/venues/import
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	// Skip until header row (first cell == "Name" after stripping BOM)
	headerFound := false
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) == 0 {
			continue
		}
		if strings.TrimPrefix(strings.TrimSpace(row[0]), "\xEF\xBB\xBF") == "Name" {
			headerFound = true
			break
		}
	}
	if !headerFound {
		http.Error(w, "header row not found", http.StatusBadRequest)
		return
	}

	type venueRow struct {
		name, street, postalCode, city, note string
	}

	var rows []venueRow
	result := importResult{Errors: []importError{}}
	lineNum := 4 // 3 preamble lines + header = line 4 is first data line

	for {
		row, err := reader.Read()
		lineNum++
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, importError{Line: lineNum, Reason: err.Error()})
			continue
		}
		if len(row) == 0 {
			continue
		}
		name := strings.TrimSpace(row[0])
		if name == "" {
			result.Errors = append(result.Errors, importError{Line: lineNum, Reason: "kein Name"})
			result.Skipped++
			continue
		}
		vr := venueRow{name: name}
		if len(row) > 2 {
			vr.street = strings.TrimSpace(row[2])
		}
		if len(row) > 3 {
			vr.postalCode = strings.TrimSpace(row[3])
		}
		if len(row) > 4 {
			vr.city = strings.TrimSpace(row[4])
		}
		if len(row) > 5 {
			vr.note = strings.TrimSpace(row[5])
		}
		if len(row) > 6 && strings.TrimSpace(row[6]) != "" {
			extra := strings.TrimSpace(row[6])
			if vr.note != "" {
				vr.note += " — " + extra
			} else {
				vr.note = extra
			}
		}
		rows = append(rows, vr)
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, vr := range rows {
		var existingID int
		err := tx.QueryRowContext(r.Context(),
			`SELECT id FROM venues WHERE name = ? AND city = ?`, vr.name, vr.city).Scan(&existingID)
		if err == sql.ErrNoRows {
			_, err = tx.ExecContext(r.Context(),
				`INSERT INTO venues (name, street, city, postal_code, country, note, is_home_venue)
				 VALUES (?, ?, ?, ?, 'DE', ?, 0)`,
				vr.name, vr.street, vr.city, vr.postalCode, vr.note)
			if err != nil {
				fmt.Fprintf(os.Stderr, "venues Import insert: %v\n", err)
				result.Skipped++
				continue
			}
			result.Imported++
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "venues Import lookup: %v\n", err)
			result.Skipped++
		} else {
			_, err = tx.ExecContext(r.Context(),
				`UPDATE venues SET street=?, postal_code=?, note=? WHERE id=?`,
				vr.street, vr.postalCode, vr.note, existingID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "venues Import update: %v\n", err)
				result.Skipped++
				continue
			}
			result.Updated++
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("venues")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DELETE /api/admin/venues
func (h *Handler) DeleteAll(w http.ResponseWriter, r *http.Request) {
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM venues WHERE is_home_venue = 0`); err != nil {
		fmt.Fprintf(os.Stderr, "venues DeleteAll: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("venues")
	w.WriteHeader(http.StatusNoContent)
}
