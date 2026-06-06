package venues

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
