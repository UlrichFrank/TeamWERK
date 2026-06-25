package config

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// Tresor-Verwaltung (Zero-Knowledge-Vault). Gespeichert werden ausschließlich die
// nicht-zurückrechenbaren Hilfswerte (Salt + Key-Check). Die geteilte Finance-Gruppen-
// Passphrase verlässt den Browser nie und wird hier weder entgegengenommen noch abgelegt.

// GET /api/admin/encryption-config — liefert Salt + Key-Check (nicht geheim) und ob der
// Tresor bereits eingerichtet ist. Clients brauchen Salt + Key-Check zum Entsperren.
func (h *Handler) GetEncryptionConfig(w http.ResponseWriter, r *http.Request) {
	var salt, keyCheck sql.NullString
	h.db.QueryRowContext(r.Context(),
		`SELECT vorstand_kdf_salt, vorstand_key_check FROM clubs LIMIT 1`).
		Scan(&salt, &keyCheck)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"configured":         salt.Valid && salt.String != "" && keyCheck.Valid && keyCheck.String != "",
		"vorstand_kdf_salt":  salt.String,
		"vorstand_key_check": keyCheck.String,
	})
}

// PUT /api/admin/encryption-config — einmalige Tresor-Einrichtung. Bei bereits
// vorhandener Konfiguration HTTP 409 (Rotation verwenden).
func (h *Handler) SetEncryptionConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Salt     string `json:"vorstand_kdf_salt"`
		KeyCheck string `json:"vorstand_key_check"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	if req.Salt == "" || req.KeyCheck == "" {
		http.Error(w, "Salt und Key-Check erforderlich", http.StatusBadRequest)
		return
	}
	var existing sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT vorstand_kdf_salt FROM clubs LIMIT 1`).Scan(&existing)
	if existing.Valid && existing.String != "" {
		http.Error(w, "Tresor bereits eingerichtet — Rotation verwenden", http.StatusConflict)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE clubs SET vorstand_kdf_salt=?, vorstand_key_check=?, updated_at=? WHERE id=(SELECT id FROM clubs LIMIT 1)`,
		req.Salt, req.KeyCheck, time.Now()); err != nil {
		http.Error(w, "Speicherfehler", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("settings")
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/admin/rotate-encryption — Passphrase-Rotation: neuer Salt + Key-Check und
// die Batch der mit dem neuen Schlüssel re-gewrappten DEKs, atomar in einer Transaktion.
// Der Server erfährt weder alte noch neue Passphrase.
func (h *Handler) RotateEncryption(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Salt     string `json:"vorstand_kdf_salt"`
		KeyCheck string `json:"vorstand_key_check"`
		Wraps    []struct {
			MemberID       int    `json:"member_id"`
			DekEncVorstand string `json:"dek_enc_vorstand"`
		} `json:"wraps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	if req.Salt == "" || req.KeyCheck == "" {
		http.Error(w, "Salt und Key-Check erforderlich", http.StatusBadRequest)
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Transaktionsfehler", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() //nolint:errcheck
	for _, wp := range req.Wraps {
		if wp.DekEncVorstand == "" {
			http.Error(w, "leerer Wrap", http.StatusBadRequest)
			return
		}
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE member_sensitive SET dek_enc_vorstand=? WHERE member_id=?`,
			wp.DekEncVorstand, wp.MemberID); err != nil {
			http.Error(w, "Speicherfehler", http.StatusInternalServerError)
			return
		}
	}
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE clubs SET vorstand_kdf_salt=?, vorstand_key_check=?, updated_at=? WHERE id=(SELECT id FROM clubs LIMIT 1)`,
		req.Salt, req.KeyCheck, time.Now()); err != nil {
		http.Error(w, "Speicherfehler", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "Commit-Fehler", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("settings")
	w.WriteHeader(http.StatusNoContent)
}
