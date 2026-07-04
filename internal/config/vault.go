package config

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/httpcache"
)

// Tresor-Verwaltung (Zero-Knowledge-Vault, Modell B — asymmetrisches Gruppen-Keypair).
// Gespeichert werden der öffentliche Schlüssel (nicht geheim), der passphrase-verschlüsselte
// private Schlüssel sowie Salt + Key-Check. Die Passphrase und der Klartext-Privatschlüssel
// verlassen den Browser nie und werden hier weder entgegengenommen noch abgelegt.

// GET /api/encryption-pubkey — öffentlicher Gruppen-Schlüssel zum Verschlüsseln (Schreiben).
// Nicht geheim; auch für das öffentliche Beitritts-Formular nutzbar.
func (h *Handler) GetGroupPublicKey(w http.ResponseWriter, r *http.Request) {
	var pub sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT group_public_key FROM clubs LIMIT 1`).Scan(&pub)
	// Quasi-unveränderlich (ändert sich nur bei Keypair-Rotation) und nicht
	// geheim → ETag aus dem Key-Material + langer public-Cache; If-None-Match
	// wird mit 304 beantwortet (Change efficient-data-loading-quickwins).
	etag := httpcache.ETagFor([]byte(pub.String))
	httpcache.Serve(w, r, etag, "public, max-age=86400", func() any {
		return map[string]any{
			"configured":       pub.Valid && pub.String != "",
			"group_public_key": pub.String,
		}
	})
}

// GET /api/admin/encryption-config — Konfiguration zum Entsperren (Lesen): öffentlicher +
// verschlüsselter privater Schlüssel, Salt, Key-Check. Alles nicht-geheim (privater
// Schlüssel ist passphrase-verschlüsselt).
func (h *Handler) GetEncryptionConfig(w http.ResponseWriter, r *http.Request) {
	var pub, privEnc, salt, keyCheck sql.NullString
	h.db.QueryRowContext(r.Context(),
		`SELECT group_public_key, group_private_key_enc, vorstand_kdf_salt, vorstand_key_check FROM clubs LIMIT 1`).
		Scan(&pub, &privEnc, &salt, &keyCheck)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"configured":            pub.Valid && pub.String != "" && privEnc.Valid && privEnc.String != "",
		"group_public_key":      pub.String,
		"group_private_key_enc": privEnc.String,
		"vorstand_kdf_salt":     salt.String,
		"vorstand_key_check":    keyCheck.String,
	})
}

type vaultConfigReq struct {
	GroupPublicKey     string `json:"group_public_key"`
	GroupPrivateKeyEnc string `json:"group_private_key_enc"`
	Salt               string `json:"vorstand_kdf_salt"`
	KeyCheck           string `json:"vorstand_key_check"`
}

func (req vaultConfigReq) complete() bool {
	return req.GroupPublicKey != "" && req.GroupPrivateKeyEnc != "" && req.Salt != "" && req.KeyCheck != ""
}

// PUT /api/admin/encryption-config — einmalige Tresor-Einrichtung. Bei bereits vorhandener
// Konfiguration HTTP 409 (Rotation verwenden).
func (h *Handler) SetEncryptionConfig(w http.ResponseWriter, r *http.Request) {
	var req vaultConfigReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	if !req.complete() {
		http.Error(w, "öffentlicher + privater Schlüssel, Salt und Key-Check erforderlich", http.StatusBadRequest)
		return
	}
	var existing sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT group_public_key FROM clubs LIMIT 1`).Scan(&existing)
	if existing.Valid && existing.String != "" {
		http.Error(w, "Tresor bereits eingerichtet — Rotation verwenden", http.StatusConflict)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE clubs SET group_public_key=?, group_private_key_enc=?, vorstand_kdf_salt=?, vorstand_key_check=?, updated_at=?
		 WHERE id=(SELECT id FROM clubs LIMIT 1)`,
		req.GroupPublicKey, req.GroupPrivateKeyEnc, req.Salt, req.KeyCheck, time.Now()); err != nil {
		http.Error(w, "Speicherfehler", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("settings")
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/admin/rotate-encryption — Passphrase-Rotation (Normalfall, O(1)): neuer Salt +
// Key-Check + der mit der neuen Passphrase verschlüsselte private Schlüssel; der öffentliche
// Schlüssel und die DEKs bleiben unverändert. Bei Keypair-Rotation (Schlüssel-Leak) wird
// zusätzlich ein neuer öffentlicher Schlüssel und eine Batch re-gewrappter DEKs mitgesendet
// und atomar geschrieben. Der Server erfährt keine Passphrase.
func (h *Handler) RotateEncryption(w http.ResponseWriter, r *http.Request) {
	var req struct {
		vaultConfigReq // GroupPublicKey hier optional (nur bei Keypair-Rotation)
		Wraps          []struct {
			MemberID       int    `json:"member_id"`
			DekEncVorstand string `json:"dek_enc_vorstand"`
		} `json:"wraps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	if req.GroupPrivateKeyEnc == "" || req.Salt == "" || req.KeyCheck == "" {
		http.Error(w, "privater Schlüssel, Salt und Key-Check erforderlich", http.StatusBadRequest)
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Transaktionsfehler", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() //nolint:errcheck

	// Keypair-Rotation: neuer öffentlicher Schlüssel + neu gewrappte DEKs.
	if len(req.Wraps) > 0 || req.GroupPublicKey != "" {
		if req.GroupPublicKey == "" {
			http.Error(w, "Keypair-Rotation erfordert group_public_key", http.StatusBadRequest)
			return
		}
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
			`UPDATE clubs SET group_public_key=? WHERE id=(SELECT id FROM clubs LIMIT 1)`,
			req.GroupPublicKey); err != nil {
			http.Error(w, "Speicherfehler", http.StatusInternalServerError)
			return
		}
	}

	// Immer: privaten Schlüssel + Salt + Key-Check ersetzen (Passphrase-Rotation).
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE clubs SET group_private_key_enc=?, vorstand_kdf_salt=?, vorstand_key_check=?, updated_at=?
		 WHERE id=(SELECT id FROM clubs LIMIT 1)`,
		req.GroupPrivateKeyEnc, req.Salt, req.KeyCheck, time.Now()); err != nil {
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
