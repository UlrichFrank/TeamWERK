// Package beitragssaetze stellt CRUD für die Beitragsmatrix bereit
// (Kategorie × Betrag × valid_from mit Historie).
package beitragssaetze

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

var validKategorien = map[string]bool{
	"aktiv_ohne": true,
	"aktiv_mit":  true,
	"passiv":     true,
}

type Satz struct {
	ID         int    `json:"id"`
	Kategorie  string `json:"kategorie"`
	BetragCent int    `json:"betrag_cent"`
	ValidFrom  string `json:"valid_from"`
	CreatedAt  string `json:"created_at,omitempty"`
}

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

// GET /api/fee-rates
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, kategorie, betrag_eur, valid_from FROM beitrags_saetze
		 ORDER BY kategorie, valid_from DESC`)
	if err != nil {
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := []Satz{}
	for rows.Next() {
		var s Satz
		var validFrom string
		if err := rows.Scan(&s.ID, &s.Kategorie, &s.BetragCent, &validFrom); err != nil {
			http.Error(w, "DB-Fehler", http.StatusInternalServerError)
			return
		}
		s.ValidFrom = validFrom[:min(10, len(validFrom))]
		items = append(items, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// POST /api/fee-rates
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kategorie  string `json:"kategorie"`
		BetragCent int    `json:"betrag_cent"`
		ValidFrom  string `json:"valid_from"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	if !validKategorien[req.Kategorie] {
		http.Error(w, "ungültige Kategorie", http.StatusBadRequest)
		return
	}
	if req.BetragCent <= 0 {
		http.Error(w, "Betrag muss größer als 0 sein", http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("2006-01-02", req.ValidFrom); err != nil {
		http.Error(w, "valid_from muss ein ISO-Datum sein (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO beitrags_saetze (kategorie, betrag_eur, valid_from) VALUES (?, ?, ?)`,
		req.Kategorie, req.BetragCent, req.ValidFrom)
	if err != nil {
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	if h.hub != nil {
		h.hub.Broadcast("beitragssatz-changed")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Satz{
		ID: int(id), Kategorie: req.Kategorie, BetragCent: req.BetragCent, ValidFrom: req.ValidFrom,
	})
}

// DELETE /api/fee-rates/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "id ungültig", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM beitrags_saetze WHERE id=?`, id)
	if err != nil {
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "nicht gefunden", http.StatusNotFound)
		return
	}
	if h.hub != nil {
		h.hub.Broadcast("beitragssatz-changed")
	}
	w.WriteHeader(http.StatusNoContent)
}
