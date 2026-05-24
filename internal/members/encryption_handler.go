package members

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// GET /api/admin/encryption-config
func (h *Handler) GetEncryptionConfig(w http.ResponseWriter, r *http.Request) {
	var salt, keyCheck sql.NullString
	err := h.db.QueryRowContext(r.Context(),
		`SELECT vorstand_kdf_salt, vorstand_key_check FROM clubs LIMIT 1`,
	).Scan(&salt, &keyCheck)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	configured := salt.Valid && salt.String != ""
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"vorstand_kdf_salt": salt.String,
		"vorstand_key_check": keyCheck.String,
		"configured":        configured,
	})
}

// PUT /api/admin/encryption-config
func (h *Handler) SetEncryptionConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VorstandKdfSalt  string `json:"vorstand_kdf_salt"`
		VorstandKeyCheck string `json:"vorstand_key_check"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.VorstandKdfSalt == "" || req.VorstandKeyCheck == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var existing sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT vorstand_kdf_salt FROM clubs LIMIT 1`).Scan(&existing)
	if existing.Valid && existing.String != "" {
		http.Error(w, "already configured", http.StatusConflict)
		return
	}

	_, err := h.db.ExecContext(r.Context(),
		`UPDATE clubs SET vorstand_kdf_salt=?, vorstand_key_check=? WHERE id=(SELECT id FROM clubs LIMIT 1)`,
		req.VorstandKdfSalt, req.VorstandKeyCheck)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/members/{id}/sensitive
func (h *Handler) GetSensitive(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID := r.PathValue("id")

	isVorstandOrAdmin := claims.Role == "vorstand" || claims.Role == "admin"

	if !isVorstandOrAdmin {
		// Check if the member belongs to the requesting user
		var linkedUserID sql.NullInt64
		err := h.db.QueryRowContext(r.Context(),
			`SELECT user_id FROM members WHERE id=?`, memberID).Scan(&linkedUserID)
		if err != nil || !linkedUserID.Valid || int(linkedUserID.Int64) != claims.UserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	var ciphertext, dekVorstand sql.NullString
	var dekMember, memberSalt sql.NullString

	err := h.db.QueryRowContext(r.Context(),
		`SELECT ciphertext, dek_enc_vorstand, dek_enc_member, member_salt FROM member_sensitive WHERE member_id=?`,
		memberID).Scan(&ciphertext, &dekVorstand, &dekMember, &memberSalt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"ciphertext":      ciphertext.String,
		"dek_enc_vorstand": dekVorstand.String,
	}
	if isVorstandOrAdmin {
		resp["dek_enc_member"] = dekMember.String
		resp["member_salt"] = memberSalt.String
	} else {
		// Member only gets their own DEK
		resp["dek_enc_member"] = dekMember.String
		resp["member_salt"] = memberSalt.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PUT /api/members/{id}/sensitive
func (h *Handler) PutSensitive(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims.Role != "vorstand" && claims.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	memberID := r.PathValue("id")
	var req struct {
		Ciphertext      string `json:"ciphertext"`
		DekEncVorstand  string `json:"dek_enc_vorstand"`
		DekEncMember    string `json:"dek_enc_member"`
		MemberSalt      string `json:"member_salt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Ciphertext == "" || req.DekEncVorstand == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand, dek_enc_member, member_salt)
		 VALUES (?,?,?,?,?)
		 ON CONFLICT(member_id) DO UPDATE SET
		   ciphertext=excluded.ciphertext,
		   dek_enc_vorstand=excluded.dek_enc_vorstand,
		   dek_enc_member=excluded.dek_enc_member,
		   member_salt=excluded.member_salt`,
		memberID,
		req.Ciphertext,
		req.DekEncVorstand,
		nullableString(req.DekEncMember),
		nullableString(req.MemberSalt),
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type rotateEntry struct {
	MemberID       int    `json:"member_id"`
	DekEncVorstand string `json:"dek_enc_vorstand"`
}

// PUT /api/admin/rotate-encryption
func (h *Handler) RotateEncryption(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NewSalt      string        `json:"new_salt"`
		NewKeyCheck  string        `json:"new_key_check"`
		Entries      []rotateEntry `json:"entries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewSalt == "" || req.NewKeyCheck == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(r.Context(),
		`UPDATE clubs SET vorstand_kdf_salt=?, vorstand_key_check=? WHERE id=(SELECT id FROM clubs LIMIT 1)`,
		req.NewSalt, req.NewKeyCheck)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	for _, entry := range req.Entries {
		_, err = tx.ExecContext(r.Context(),
			`UPDATE member_sensitive SET dek_enc_vorstand=? WHERE member_id=?`,
			entry.DekEncVorstand, entry.MemberID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type EncryptedMemberExport struct {
	ID              int     `json:"id"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	MemberNumber    string  `json:"member_number,omitempty"`
	PassNumber      string  `json:"pass_number,omitempty"`
	Gender          string  `json:"gender"`
	Status          string  `json:"status"`
	Position        string  `json:"position,omitempty"`
	Ciphertext      string  `json:"ciphertext,omitempty"`
	DekEncVorstand  string  `json:"dek_enc_vorstand,omitempty"`
}

// GET /api/members/export-encrypted
func (h *Handler) ExportEncrypted(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT m.id, m.first_name, m.last_name,
		        COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
		        COALESCE(m.gender,'u'), m.status, COALESCE(m.position,''),
		        COALESCE(ms.ciphertext,''), COALESCE(ms.dek_enc_vorstand,'')
		 FROM members m
		 LEFT JOIN member_sensitive ms ON ms.member_id = m.id
		 ORDER BY m.last_name, m.first_name`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []EncryptedMemberExport{}
	for rows.Next() {
		var e EncryptedMemberExport
		if err := rows.Scan(&e.ID, &e.FirstName, &e.LastName, &e.MemberNumber, &e.PassNumber,
			&e.Gender, &e.Status, &e.Position, &e.Ciphertext, &e.DekEncVorstand); err != nil {
			continue
		}
		result = append(result, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
