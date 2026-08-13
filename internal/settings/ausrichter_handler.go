package settings

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// canSeeAusrichterInactive: wie stammvereine.canSeeInactive — nur Vorstand/
// Admin dürfen deaktivierte Ausrichter in der Liste sehen. Standard-Nutzer
// (Kalender, Termin-Wizard) sollen nur die nutzbaren Einträge angeboten
// bekommen, nicht die von der Verwaltung ausgemusterten.
func canSeeAusrichterInactive(r *http.Request) bool {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		return false
	}
	return claims.Role == "admin" || claims.HasFunction("vorstand")
}

// ListAusrichter liefert `{"items": [...]}`. Authenticated-Tier: der
// Kalender und der Termin-Wizard brauchen die volle Liste für jeden
// eingeloggten Nutzer, nicht nur für den Vorstand (anders als bei der
// Mutation). `?include_inactive=1` wirkt nur, wenn der Aufrufer Vorstand
// oder Admin ist (Vorbild stammvereine.List).
func (h *Handler) ListAusrichter(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("include_inactive") == "1" && canSeeAusrichterInactive(r)
	items, err := ListAusrichter(r.Context(), h.db, includeInactive)
	if err != nil {
		slog.Error("settings: list ausrichter failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// AusrichterUsage liefert die Vorab-Bilanz für den Löschen-Dialog: betroffene
// Spieltage (überleben das Löschen, fallen auf den Default zurück) und
// gebundene Vorlagen-Zeilen (werden mitgelöscht). Authenticated-Tier — die
// Route selbst ändert nichts, das eigentliche Löschen bleibt Vorstand-Tier.
func (h *Handler) AusrichterUsage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	report, err := AusrichterUsage(r.Context(), h.db, id)
	if err != nil {
		writeAusrichterError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// CreateAusrichter legt einen neuen Ausrichter an. Vorstand-Tier (Gating
// sitzt in der Router-Middleware). Erwartet Body
// `{"name": string, "sort_order": number, "is_default": bool}`.
func (h *Handler) CreateAusrichter(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
		IsDefault bool   `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	created, err := CreateAusrichter(r.Context(), h.db, AusrichterInput{
		Name:      body.Name,
		SortOrder: body.SortOrder,
		IsDefault: body.IsDefault,
	})
	if err != nil {
		writeAusrichterError(w, err)
		return
	}
	if h.hub != nil {
		h.hub.Broadcast("settings-changed")
	}
	writeJSON(w, http.StatusCreated, created)
}

// UpdateAusrichter ändert einen bestehenden Ausrichter. Vorstand-Tier.
// Pointer-Body — ein fehlendes Feld bleibt unverändert (siehe
// AusrichterUpdate in ausrichter.go).
func (h *Handler) UpdateAusrichter(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name      *string `json:"name"`
		Aktiv     *bool   `json:"aktiv"`
		SortOrder *int    `json:"sort_order"`
		IsDefault *bool   `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	updated, err := UpdateAusrichter(r.Context(), h.db, id, AusrichterUpdate{
		Name:      body.Name,
		Aktiv:     body.Aktiv,
		SortOrder: body.SortOrder,
		IsDefault: body.IsDefault,
	})
	if err != nil {
		writeAusrichterError(w, err)
		return
	}
	if h.hub != nil {
		h.hub.Broadcast("settings-changed")
	}
	writeJSON(w, http.StatusOK, updated)
}

// DeleteAusrichter löscht einen Ausrichter (Vorstand-Tier). Die Kaskade
// (Spieltage entkoppeln, gebundene Vorlagen-Zeilen mitlöschen) läuft
// vollständig in DeleteAusrichter (ausrichter.go) innerhalb einer
// Transaktion — der Handler übersetzt nur noch das Ergebnis.
func (h *Handler) DeleteAusrichter(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := DeleteAusrichter(r.Context(), h.db, id); err != nil {
		writeAusrichterError(w, err)
		return
	}
	if h.hub != nil {
		h.hub.Broadcast("settings-changed")
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// writeAusrichterError übersetzt die typisierten Fehler aus ausrichter.go in
// HTTP-Status + JSON-Fehlercode. ErrDefaultUndeletable trägt bewusst den in
// der Spec festgeschriebenen Code `default_ausrichter_undeletable` — das
// Frontend zeigt daraus einen spezifischen Hinweis statt einer generischen
// 409-Meldung.
func writeAusrichterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAusrichterNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ausrichter_not_found"})
	case errors.Is(err, ErrDuplicateName):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "duplicate_name"})
	case errors.Is(err, ErrEmptyName):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty_name"})
	case errors.Is(err, ErrDefaultUndeletable):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "default_ausrichter_undeletable"})
	case errors.Is(err, ErrDefaultRequired):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "default_required"})
	case errors.Is(err, ErrNoDefaultAusrichter):
		// Datenfehler, kein Betriebszustand (siehe Kommentar an
		// ErrNoDefaultAusrichter) — laut scheitern statt einen Ersatzwert zu
		// erfinden, der gegen kein Item matchen würde.
		slog.Error("settings: kein Default-Ausrichter gesetzt")
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		slog.Error("settings: ausrichter operation failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
