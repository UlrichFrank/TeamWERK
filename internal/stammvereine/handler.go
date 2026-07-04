// Package stammvereine stellt CRUD für die verwaltbare Liste der Stammvereine
// bereit. Die Zuordnung eines Mitglieds (members.home_club_id) entscheidet im
// Beitragslauf über die Aktiv-Kategorie (aktiv_mit vs. aktiv_ohne).
package stammvereine

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

type Verein struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Aktiv     bool   `json:"aktiv"`
	SortOrder int    `json:"sort_order"`
}

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

// broadcast fires the "stammvereine" event. This is a club-wide reference-data
// topic (Stammvereine, low-frequency) → intentionally global, not team/role
// scoped (see scoped-live-updates).
func (h *Handler) broadcast() {
	if h.hub != nil {
		h.hub.Broadcast("stammvereine")
	}
}

// canSeeInactive: nur Vorstand/Admin dürfen deaktivierte Vereine sehen.
func canSeeInactive(r *http.Request) bool {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		return false
	}
	return claims.Role == "admin" || claims.HasFunction("vorstand")
}

// GET /api/stammvereine  (authenticated; ?include_inactive=1 nur vorstand/admin)
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("include_inactive") == "1" && canSeeInactive(r)
	query := `SELECT id, name, aktiv, sort_order FROM stammvereine`
	if !includeInactive {
		query += ` WHERE aktiv=1`
	}
	query += ` ORDER BY sort_order, name`
	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := []Verein{}
	for rows.Next() {
		var v Verein
		var aktiv int
		if err := rows.Scan(&v.ID, &v.Name, &aktiv, &v.SortOrder); err != nil {
			http.Error(w, "DB-Fehler", http.StatusInternalServerError)
			return
		}
		v.Aktiv = aktiv != 0
		items = append(items, v)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// POST /api/stammvereine  (vorstand)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		http.Error(w, "name ist erforderlich", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO stammvereine (name, sort_order) VALUES (?, ?)`, req.Name, req.SortOrder)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "Stammverein mit diesem Namen existiert bereits", http.StatusConflict)
			return
		}
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	h.broadcast()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Verein{ID: int(id), Name: req.Name, Aktiv: true, SortOrder: req.SortOrder})
}

// PUT /api/stammvereine/{id}  (vorstand) — umbenennen und/oder aktiv-Flag setzen.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "ungültige id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name      *string `json:"name"`
		Aktiv     *bool   `json:"aktiv"`
		SortOrder *int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			http.Error(w, "name darf nicht leer sein", http.StatusBadRequest)
			return
		}
		if _, err := h.db.ExecContext(r.Context(),
			`UPDATE stammvereine SET name=? WHERE id=?`, name, id); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				http.Error(w, "Stammverein mit diesem Namen existiert bereits", http.StatusConflict)
				return
			}
			http.Error(w, "DB-Fehler", http.StatusInternalServerError)
			return
		}
	}
	if req.Aktiv != nil {
		h.db.ExecContext(r.Context(), `UPDATE stammvereine SET aktiv=? WHERE id=?`, boolToInt(*req.Aktiv), id)
	}
	if req.SortOrder != nil {
		h.db.ExecContext(r.Context(), `UPDATE stammvereine SET sort_order=? WHERE id=?`, *req.SortOrder, id)
	}
	h.broadcast()
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/stammvereine/{id}  (vorstand) — Soft-Delete (aktiv=0), niemals
// physisch löschen, damit referenzierende members.home_club_id intakt bleiben.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "ungültige id", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `UPDATE stammvereine SET aktiv=0 WHERE id=?`, id)
	if err != nil {
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "nicht gefunden", http.StatusNotFound)
		return
	}
	h.broadcast()
	w.WriteHeader(http.StatusNoContent)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
