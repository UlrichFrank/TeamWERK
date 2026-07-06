package members

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// GetChangeRequestsHandler GET /api/members/{id}/change-drafts
func (h *Handler) GetChangeRequestsHandler(w http.ResponseWriter, r *http.Request) {
	memberID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	// Ownership-Gate: nur Eigentümer/Eltern/admin/vorstand/kassierer dürfen die Anträge
	// (inkl. old_value-Snapshot der aktuellen Mitglieds-PII) eines Mitglieds lesen.
	claims := auth.ClaimsFromCtx(r.Context())
	if !h.canAccessMember(r.Context(), claims, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	drafts, err := h.GetChangeDrafts(memberID)
	if err != nil {
		http.Error(w, "Failed to retrieve drafts", http.StatusInternalServerError)
		return
	}

	if drafts == nil {
		drafts = []ChangeDraft{}
	}

	// Bankdaten-Entwürfe sind clientseitige Envelopes: Der Server entschlüsselt nicht; er
	// reicht den Envelope nur an die Finance-Gruppe durch (admin/vorstand/kassierer) und
	// schwärzt ihn sonst (G2 — Eigentümer/Eltern lesen Bankdaten nicht). Die eigentliche
	// Entschlüsselung passiert clientseitig mit dem Tresor-Schlüssel.
	revealBank := claims.Role == "admin" || claims.HasFunction("vorstand") || claims.HasFunction("kassierer")
	redactBankDrafts(drafts, revealBank)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]ChangeDraft{"drafts": drafts})
}

// CreateChangeRequestHandler POST /api/members/{id}/change-request
// User requests a change to their member data
func (h *Handler) CreateChangeRequestHandler(w http.ResponseWriter, r *http.Request) {
	memberID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Ownership-Gate: nur Eigentümer/Eltern/admin/vorstand/kassierer dürfen für ein
	// Mitglied einen Antrag anlegen, überschreiben oder einen pending-Antrag verdrängen.
	if !h.canAccessMember(r.Context(), claims, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req ChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation: allowed field names
	allowedFields := map[string]bool{
		"name": true, "address": true, "phones": true, "email": true,
		"photo_url": true, "bankdaten": true, "sepa_mandat": true, "dsgvo": true,
		"profil": true,
	}
	if !allowedFields[req.FieldName] {
		http.Error(w, "Invalid field name", http.StatusBadRequest)
		return
	}

	// Bankdaten-Anträge folgen dem Selbstbedienungsmodell: nur Eigentümer/Eltern dürfen
	// einen (clientseitig verschlüsselten) Envelope einreichen. Damit kann niemand einen
	// fremden Bankdaten-Envelope unter dem Namen eines anderen Mitglieds hinterlegen.
	if req.FieldName == "bankdaten" && !h.canSubmitBankDraft(r.Context(), claims, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	draft, err := h.CreateOrUpdateDraft(memberID, int(claims.UserID), req)
	if err != nil {
		http.Error(w, "Failed to create draft", http.StatusInternalServerError)
		return
	}

	// Draft geht an die Finance-Gruppe (prüft/genehmigt) + die Audience des Mitglieds
	// (Team-Roster/Profil) + den Einreicher selbst.
	h.broadcastMembers(r.Context(), []int{memberID}, int(claims.UserID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(draft)
}

// AcceptChangeRequestHandler POST /api/members/{id}/change-drafts/{draftId}/accept
// Admin accepts a change request
func (h *Handler) AcceptChangeRequestHandler(w http.ResponseWriter, r *http.Request) {
	draftID, err := strconv.Atoi(chi.URLParam(r, "draftId"))
	if err != nil {
		http.Error(w, "Invalid draft ID", http.StatusBadRequest)
		return
	}

	if err := h.AcceptDraft(draftID); err != nil {
		http.Error(w, "Failed to accept draft", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

// RejectChangeRequestHandler DELETE /api/members/{id}/change-drafts/{draftId}
// Admin rejects a change request
func (h *Handler) RejectChangeRequestHandler(w http.ResponseWriter, r *http.Request) {
	draftID, err := strconv.Atoi(chi.URLParam(r, "draftId"))
	if err != nil {
		http.Error(w, "Invalid draft ID", http.StatusBadRequest)
		return
	}

	if err := h.RejectDraft(draftID); err != nil {
		http.Error(w, "Failed to reject draft", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}
