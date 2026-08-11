package settings

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

// Handler bündelt die HTTP-Routen rund um system_settings (maintenance_mode,
// bewirtung_verhaeltnis).
type Handler struct {
	db    *sql.DB
	store *Store
	hub   *hub.EventHub
}

func NewHandler(db *sql.DB, store *Store, h *hub.EventHub) *Handler {
	return &Handler{db: db, store: store, hub: h}
}

// GetPublicStatus liefert `{"enabled": bool}` — ohne Auth erreichbar, damit
// der Banner auch auf der Login-Seite anzeigbar ist. Bewusst KEINE Metadaten
// (updated_by/updated_at) — kleiner Info-Leak vermeiden.
func (h *Handler) GetPublicStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"enabled": h.store.MaintenanceMode()})
}

// GetAdminStatus liefert den vollen Zustand inkl. `updated_at`/`updated_by_name`.
// Für die Admin-UI. Auth wird durch die vorgelagerte RequireRole("admin")-
// Middleware erzwungen.
func (h *Handler) GetAdminStatus(w http.ResponseWriter, r *http.Request) {
	snap, err := h.store.Snapshot(r.Context())
	if err != nil {
		slog.Error("settings: snapshot failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	resp := map[string]any{"enabled": snap.Enabled}
	if snap.UpdatedAt.Valid {
		resp["updated_at"] = snap.UpdatedAt.String
	}
	if snap.UpdatedByName.Valid {
		resp["updated_by_name"] = snap.UpdatedByName.String
	}
	writeJSON(w, http.StatusOK, resp)
}

// SetMaintenanceMode schaltet den Modus um. Erwartet Body `{"enabled": bool}`.
// Broadcast des SSE-Events "settings-changed" beendet den Roundtrip; Clients
// laden den Status daraufhin neu.
func (h *Handler) SetMaintenanceMode(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.store.SetMaintenanceMode(r.Context(), body.Enabled, claims.UserID); err != nil {
		slog.Error("settings: set failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if h.hub != nil {
		h.hub.Broadcast("settings-changed")
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": body.Enabled})
}

// GetBewirtung liefert das aktuelle Spiele-zu-Kuchen-Verhältnis. Authenticated-
// Tier — jeder eingeloggte Nutzer darf lesen (siehe spec bewirtungsrotation).
func (h *Handler) GetBewirtung(w http.ResponseWriter, r *http.Request) {
	v, err := GetBewirtungVerhaeltnis(r.Context(), h.db)
	if err != nil {
		slog.Error("settings: get bewirtung_verhaeltnis failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"verhaeltnis": v})
}

// SetBewirtung ändert das Spiele-zu-Kuchen-Verhältnis. Erwartet Body
// `{"verhaeltnis": number}`; Gating (vorstand/admin) sitzt in der Router-
// Middleware. Validiert > 0 (400 sonst, nichts wird persistiert) und
// broadcastet "settings-changed" (gleiches SSE-Event wie der Wartungsmodus-
// Toggle) nach erfolgreichem Schreiben.
func (h *Handler) SetBewirtung(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Verhaeltnis float64 `json:"verhaeltnis"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := SetBewirtungVerhaeltnis(r.Context(), h.db, body.Verhaeltnis, claims.UserID); err != nil {
		if err == ErrInvalidVerhaeltnis {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_verhaeltnis"})
			return
		}
		slog.Error("settings: set bewirtung_verhaeltnis failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if h.hub != nil {
		h.hub.Broadcast("settings-changed")
	}
	writeJSON(w, http.StatusOK, map[string]any{"verhaeltnis": body.Verhaeltnis})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
